# ADR: Writable Lineups on a Persistent Volume

**Status:** Accepted 2026-09-27; amended 2026-11-01.

Enables the "lineup submission" extension that [ADR 0004](0004-web-frontend-stack.md) kept the
door open for. It **reverses ADR 0004 decision #3** (the lineup tree is `//go:embed`'d, so "a
deploy is therefore also the lineup update") and **narrows its decision #7** ("writes do not
exist"). Everything else in ADR 0004 — server-rendered `html/template`, one binary/one origin,
the `/{season}/{week}` URL, client-polls/server-caches freshness, the no-margin honesty
constraint, and the CSV format of decision #9 — remains in force. ADR 0003's provider interface
and Sleeper-as-sole-provider are untouched.

## Context

Today a lineup is set by editing `internal/data/lineups/<season>/<week>/<team>.csv` in git and
deploying: the tree is compiled into the binary with `//go:embed`, so the running server reads
no mutable file and every request sees identical, build-time-frozen data. We want to **set a
lineup from a phone without a laptop, an editor, or a deploy**.

The rendering stack is not the obstacle — ADR 0004 chose `html/template` precisely so that "a
form and some tables" drop in without a framework. The obstacle is that the lineup data is a
build-time constant. Making it settable at runtime crosses one architectural line:

> **Immutable build-time data → mutable runtime data.**

Crossing it is the whole decision. Everything downstream — where writes land, backups,
concurrency, a single writer machine, write auth — follows from it and is the same regardless of how
the mutable data is *shaped*.

Two candidate shapes were weighed:

- **B — SQLite on a volume.** The traditional CRUD/relational store.
- **C — CSV files on a volume.** The current file format, moved from the binary to a writable disk.

The read side already accommodates either: `lineup.New(fsys fs.FS)` builds a `Tree` over any
`fs.FS`, and `Read`/matchup resolution go through `t.fsys`. Today we pass the embedded FS; we
could pass `os.DirFS(...)` without touching the read code. `fs.FS` is read-only *by interface*
(`Open` only), so the *write* path is genuinely new surface under either shape.

## Decision

Store lineups as **CSV files on a persistent Fly volume** that is the **single store and sole read
source**, read through `os.DirFS`. Git and a new passphrase-gated phone form in the same Go binary
are **two writers to that store**; the precedence between them lives in the writers, not in the
read path. **Option C.**

1. **Storage: CSV on a mounted volume.** Lineups live on a Fly volume mounted into the machine,
   under a `lineups/` subdirectory the app owns (the mount root is left alone — a fresh ext4
   volume already holds `lost+found`), keyed exactly as the embedded tree:
   `<mount>/lineups/<season>/<week>/<team>.csv`. The volume holds every week, archived or not.
   The week is always explicit in the path, never inferred from another week, so a deploy cannot
   relabel a phone-set lineup and a season rollover needs no special case. The CSV format of ADR
   0004 decision #9 is unchanged — header row `id,name,position,team`, closed sets for position
   and team, Sleeper-ID as sole identity, labels frozen at write time.

   Fly mounts a new volume root-owned, but the server runs as distroless `nonroot` (65532). So
   the container starts as root, and the server's first act is to `lchown` the mount root (not
   recursively) to `nonroot` and drop to it — groups, then gid, then uid — failing if still root.
   Seeding and listening happen only after the drop. This keeps the distroless image with no
   shell or entrypoint script, at the cost of a brief root window at the start of every boot.

2. **Read: the volume only.** `lineup.New` is handed `os.DirFS(<mount>/lineups)` and nothing
   else. There is no layering, no per-week choice between sources, and no volume week that is
   stale, superseded, or ignored: what the store holds is what is served. `Read`, matchup
   resolution, `LatestWeek`, and week-directory enforcement are untouched. Local development and
   tests with no volume configured read the embedded tree directly.

3. **Write: a new path outside `fs.FS`.** A lineup submission is `POST`ed from a phone form and
   written with `os.WriteFile`-style disk writes, since `fs.FS` cannot express a write. Writes
   are **atomic**: write to a temp file in the same directory, then rename over the target, so a
   reader never observes a half-written lineup and a crashed write leaves the previous lineup
   intact. The write validates the ADR 0004 decision #9 invariants (header, closed sets,
   Sleeper-ID lookup) *before* committing the file — an invalid submission is refused, not
   half-saved.

   **Writable weeks are the store's latest week or the week after it.** The latest week covers
   "the week is in git, but I need to change it and I'm away from a laptop" — a lineup change at
   11:55 on Sunday; the next week covers setting a lineup ahead of git. Editing a week git
   already has is allowed. At a season boundary the week after 2025/16 is 2026/1, chosen
   explicitly and stored in the path (decision 1).

   Any other week is refused. This is the product shape we want — set *this* week's lineup from
   wherever I am — and a guardrail: a settled past week cannot be silently rewritten from a
   phone. Past weeks are fixed through git only (decision 4).

4. **Git seeds the store at boot, per week; if git's copy of a week changed, git is right.**
   Before the server listens, the app walks the embedded tree and seeds the volume from it. A
   manifest at `<mount>/seed-manifest.json` — outside `lineups/`, so it never reads as a season —
   records, per `<season>/<week>`, a hash of git's version of that week at its last seed. The
   hash covers file names and contents but not metadata such as mtimes, so it is stable across
   rebuilds; names are included so a swapped opponent file counts as a change even if the bytes
   match. For each embedded week:

   - **git's hash differs from the manifest entry, or there is no entry** → the store's week is
     replaced **whole** with git's — the directory as a unit, never file by file, since if the
     opponent changed, merging would leave three lineups and break the week (ADR 0004's
     two-lineups-per-week invariant) — and then the manifest entry is updated;
   - **the hashes match** → the store's week is left alone.

   So a git edit to an old week does not touch a phone-edited week, and code-only deploys and
   restarts are no-ops for lineups. Copy precedes the manifest update, so a crash between them
   just re-copies identical content on the next boot: seeding is idempotent. A week deleted from
   git is **not** deleted from the store. When seeding overwrites a store week that differed from
   git — a phone edit — the overwritten lineup is logged so it can be recovered.

   Each week is then in one of four states:

   - **none** — neither git nor the store has it;
   - **synced** — the store matches git;
   - **ui-only** — the store has it, git does not;
   - **diverged** — the store differs from git, and git is unchanged since its last seed.

   A git change to a week makes it synced (seeding copies it). A phone edit makes it ui-only if
   git lacks the week, synced if the result matches git, and diverged otherwise. Matching
   compares parsed records, not bytes, since line endings differ between a phone write and an
   editor. **Unarchived** means ui-only or diverged; it is computable at boot, and it is what the
   export path lists.

   **Archiving a week is committing the store's version to git and deploying.** Getting that
   version out needs an export path (`fly ssh sftp get`, or a download link); once committed,
   seeding copies identical content and the week is synced. The known loss path is accepted: a
   git change to the *same* week merged after a phone edit wins over it. The overwrite log and
   making export-then-commit the normal archive step are the mitigations.

5. **The volume is the store of record, rebuildable from git.** Losing the volume — manifest
   included — re-seeds every week from git on the next boot, so the **blast radius is the
   unarchived weeks only**, re-enterable from a phone in minutes. That bound holds only while
   finished weeks are actually archived. Fly's automatic daily volume snapshots are an
   unrelied-upon convenience on top of that — not the backup strategy. A server-driven
   write-back-to-git (an automatic audit trail) remains a possible follow-up but is **not in this
   change**; it would reintroduce the git coupling C removes for no gain while a single person
   edits.

6. **Exactly one machine; scale-to-zero kept.** A Fly volume attaches to a single machine, so
   the app runs as one machine with one volume (`fly scale count 1`) — never two machines that
   could each hold a different copy of the store. That constraint is on machine *count*, not on
   uptime: a volume survives the machine stopping and starting, so ADR 0004's scale-to-zero
   (`auto_stop_machines = 'stop'`, `min_machines_running = 0`) stays. Seeding (decision 4) then
   runs on every cold start, not just on deploy — a hash comparison per week, a no-op when git
   is unchanged. An in-flight POST keeps the machine up, and atomic temp-then-rename (decision 3)
   covers a stop that lands mid-write anyway. `min = 1` remains an optional knob, not a
   requirement: it would buy a warm stats cache and no cold start on a phone edit, at the
   negligible cost of one always-on 256mb machine.

7. **Writes are gated by a shared passphrase; reads stay public.** Per ADR 0004's rationale, the
   write path does not require accounts — a single shared passphrase gates submission. Reads
   remain public and unauthenticated as before.

8. **Handler ordering stays load-bearing.** ADR 0004 decision #7 made "validate path, resolve
   week, read lineups, *then* call the provider" a security property, not an efficiency detail.
   The write handler inherits the same discipline: it authenticates, validates the submission and
   its `(season, week, team)`, and enforces the **two-lineups-per-week-directory** invariant
   (ADR 0004 consequence: a third file breaks that week's page) **before** touching disk, and it
   never calls Sleeper on the write path except the decision-#9 label lookup, which runs only
   after auth and validation. A write must not create a third file in a week directory or orphan
   an opponent.

9. **Lineup-lock timing is deferred, unrestricted in the interim.** A write path invents a
   question that did not exist when data was build-time-frozen: may a lineup be changed after its
   games kick off? For now, **no server-enforced lock** — in practice a single co-manager edits,
   the writable surface is already just the store's latest week or the next (decision 3), and ADR 0004's no-margin
   rule means the page never presents a week as settled. A time-based lock (reject writes to a
   week whose games have started) is a follow-up, called out here because it is a real integrity
   gap, not an oversight.

## Rationale

- **C crosses exactly the line that must be crossed, and no more.** The architectural threshold
  is "mutable runtime state" — concurrency, atomicity, a mutable working copy off git, a
  single writer machine, write auth. B and C incur that threshold identically. C adds nothing beyond
  it; B additionally adds a schema, migrations, and a query engine.

- **The mutable data is document-shaped, not relational.** A lineup is a small, self-contained
  thing keyed by `(season, week, team)` — nine starters, per team, with no join inside
  it. That is a file almost by definition. Modeling it as rows buys no relational benefit while
  costing schema and migrations, and it forfeits the genuinely useful property C keeps: lineups
  stay human-readable and git-diffable.

- **The relational win B seemed to offer is separable and not about lineups.** "Season-long
  stats" is NFL player stats transformed into fantasy score — **Sleeper's** data, not the lineup
  store's. If Sleeper load ever justifies caching that in SQLite, that cache can be added
  *alongside* file-based lineups, independently, without restructuring lineups into rows. It was
  never the same problem as lineup storage, so it does not pull lineup storage toward B.

- **C is the smallest honest step off `//go:embed`.** The read change is swapping the embedded
  FS for `os.DirFS`, invisible to `Tree`; the format is unchanged; git remains the archive. The
  new machinery is the write path, which any writable-lineup design requires, and boot seeding
  (decision 4).

- **One store is easier to reason about than read-time precedence between two.** With a single
  read source, what is served is simply what is on the volume. The question "whose version
  wins" is answered once, by each writer at write time — the phone form by its writable-weeks
  rule, seeding by "if git's copy changed, git is right" — instead of on every read.

- **A volume needs one machine, not an always-on one.** Persistence comes from the volume
  outliving machine stops, so the single-writer constraint costs nothing in uptime and ADR
  0004's "upstream load drops to zero when nobody is looking" survives. The cold-start Sleeper
  fetch auto-stop imposes is already bounded by the plays cold wait.

## Consequences

- **A lineup change no longer requires a commit or a deploy** — the reversal of ADR 0004 decision
  #3, and the point of this change. Deploys ship code, and lineups only for weeks git changed.

- **The volume is the store of record, rebuildable from git except unarchived weeks.** Losing it
  costs only ui-only and diverged weeks (decision 5); no volume-snapshot discipline is required
  to make that true.

- **Boot writes to the volume.** Seeding runs before the server listens, so a volume the app
  cannot write to blocks reads, not just phone writes. That is why the container starts as root
  and the server claims the mount before dropping to `nonroot` (decision 1).

- **New concurrency and atomicity surface.** Read-during-write and partial-write hazards exist
  for the first time; atomic temp-then-rename (decision 3) is a correctness requirement, not an
  optimisation.

- **The app is pinned to one machine** (decision 6). Scale-to-zero stays; `fly.toml` gains the
  volume mount, not a `min_machines_running` change. A cold start now also re-runs seeding.

- **A new write auth surface exists** — a shared passphrase to keep out of logs, commits, and
  screenshots. Small, but real, and it is the first secret the project carries.

- **Lineup edits are unbounded in time until a lock lands** (decision 9). A lineup can be changed
  after kickoff, rewriting what a reader saw. Accepted for now on trust; flagged as the first
  follow-up.

- **Git no longer always has the last word for a week.** A phone edit to a week git holds is
  served until git's copy of that week changes; a deploy that leaves the week alone leaves the
  phone version in place. This can surprise anyone who remembers the old model.

- **Archiving a week is a manual step.** Forgetting leaves the week unarchived — still served,
  but lost with the volume, and overwritten (with a log entry) if git later changes that same
  week (decision 4).

## Follow-ups

- **Lineup-lock timing.** Decide and implement whether writes to a week whose games have started
  are refused. First integrity gap opened by this change.
- **Write-back-to-git.** Optionally mirror each submission back to git as an audit trail /
  restore-diffable backup, without recoupling reads to git.
- **Season-long stats.** If it lands and Sleeper load justifies it, add a *separate* stats cache
  (possibly SQLite) alongside file-based lineups — not a migration of lineups into a DB.
- **Consider `min = 1`** if cold starts on a phone edit or a cold stats cache ever prove
  annoying; it is a config change, not a design change.
