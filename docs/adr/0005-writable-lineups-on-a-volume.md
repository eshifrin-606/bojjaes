# ADR: Writable Lineups on a Persistent Volume

**Status:** Accepted 2026-09-27.

Enables the "lineup submission" extension that [ADR 0004](0004-web-frontend-stack.md) kept the
door open for. It **reverses ADR 0004 decision #3** (the lineup tree is `//go:embed`'d, so "a
deploy is therefore also the lineup update") and **narrows its decision #7** ("writes do not
exist"). Everything else in ADR 0004 — server-rendered `html/template`, one binary/one origin,
the `/{season}/{week}` URL, client-polls/server-caches freshness, the no-margin honesty
constraint, and the CSV format of decision #9 — remains in force. ADR 0003's provider interface
and Sleeper-as-sole-provider are untouched.

## Context

Today a lineup is set by editing `internal/lineup/data/<season>/<week>/<team>.csv` in git and
deploying: the tree is compiled into the binary with `//go:embed`, so the running server reads
no mutable file and every request sees identical, build-time-frozen data. We want to **set a
lineup from a phone without a laptop, an editor, or a deploy**.

The rendering stack is not the obstacle — ADR 0004 chose `html/template` precisely so that "a
form and some tables" drop in without a framework. The obstacle is that the lineup data is a
build-time constant. Making it settable at runtime crosses one architectural line:

> **Immutable build-time data → mutable runtime data.**

Crossing it is the whole decision. Everything downstream — where writes land, backups,
concurrency, an always-on writer, write auth — follows from it and is the same regardless of how
the mutable data is *shaped*.

Two candidate shapes were weighed:

- **B — SQLite on a volume.** The traditional CRUD/relational store.
- **C — CSV files on a volume.** The current file format, moved from the binary to a writable disk.

The read side already accommodates either: `lineup.New(fsys fs.FS)` builds a `Tree` over any
`fs.FS`, and `Read`/matchup resolution go through `t.fsys`. Today we pass the embedded FS; we
could pass `os.DirFS(...)` without touching the read code. `fs.FS` is read-only *by interface*
(`Open` only), so the *write* path is genuinely new surface under either shape.

## Decision

Store phone-set lineups as **CSV files on a persistent Fly volume**, read through `os.DirFS`
layered over the embedded archive, written by a new passphrase-gated form handler in the same Go
binary. **Option C.**

1. **Storage: CSV on a mounted volume.** Phone-set lineups live on a Fly volume mounted into the
   machine, under a `lineups/` subdirectory the app owns (the mount root is left alone — a fresh
   ext4 volume already holds `lost+found`), keyed exactly as the embedded tree:
   `<mount>/lineups/<season>/<week>/<team>.csv`. The week is always explicit in the path, never
   inferred from the embedded tree's latest week, so a deploy cannot relabel a phone-set lineup
   and a season rollover needs no special case. The CSV format of ADR
   0004 decision #9 is unchanged — header row `id,name,position,team`, closed sets for position
   and team, Sleeper-ID as sole identity, labels frozen at write time.

2. **Read: two layers, the volume winning only for the latest week.** The embedded tree remains
   a read source — it serves every archived week — and the volume holds **at most one week**.
   Per week:

   - the volume's week is served from the volume if it is **at or after** the embedded tree's
     latest week — whether it is a new week (latest + 1) or an edit of the latest archived week;
   - every other week is served from the embedded tree;
   - a volume week *older* than the embedded tree's latest is stale: it is **ignored, never
     deleted**, and logged, since it may hold edits that never reached git.

   The latest week overall is the greater of the two sources' latest weeks. `Read`, matchup
   resolution, and week-directory enforcement are untouched — `lineup.New` is handed an `fs.FS`
   composing the two layers, and only the season listing that `LatestWeek` reads needs merging.
   An empty or absent `lineups/` directory reads exactly like the embedded tree alone.

3. **Write: a new path outside `fs.FS`.** A lineup submission is `POST`ed from a phone form and
   written with `os.WriteFile`-style disk writes, since `fs.FS` cannot express a write. Writes
   are **atomic**: write to a temp file in the same directory, then rename over the target, so a
   reader never observes a half-written lineup and a crashed write leaves the previous lineup
   intact. The write validates the ADR 0004 decision #9 invariants (header, closed sets,
   Sleeper-ID lookup) *before* committing the file — an invalid submission is refused, not
   half-saved.

   **Which weeks are writable follows from what the volume holds.** With `G` the embedded tree's
   latest week:

   - **volume empty** → `G` or the week after it. `G` covers "the week is in git, but I need to
     change it and I'm away from a laptop"; the next week covers setting a lineup ahead of git.
     At a season boundary the week after 2025/16 is 2026/1, chosen explicitly and stored in the
     path (decision 1).
   - **volume holds week `v`** → `v` only. Nothing else is writable until `v` is reconciled with
     git (decision 4), so an unarchived week cannot be buried under the next one.

   Any other week is refused. This is the product shape we want — set *this* week's lineup from
   wherever I am — and a guardrail: a settled past week cannot be silently rewritten, and the
   volume never holds more than one week.

4. **Git stays the durable archive; the app — not a human — keeps the volume to just the
   in-flight week.** Every finalized week is committed to git and rides into the binary via the
   embedded tree; only the latest week is writable (decision 3). The reconciliation between the
   two — the volume only ever wins for the latest week (decision 2), and only one week is ever
   on it (decision 3) — is **enforced by the application, not by editor discipline.** A control
   that depends on someone remembering to sync is exactly the failure mode this avoids.

   **The volume is the source of truth for its week until git matches it.** Archiving a week is
   committing the volume's version to git and deploying; on boot, when the embedded tree holds
   the volume's week **and its parsed records match** (ids, in order — not bytes, since line
   endings differ between a phone write and an editor), the volume copy is cleared and git
   serves the week. The match is load-bearing: the app never observes a git update, only "both
   sources hold week `v`", which is also the state after any restart following a phone edit of an
   archived week. Clearing on presence alone would erase that edit on every deploy.

   When they differ, nothing is cleared: the volume keeps winning (if still the latest week) and
   the divergence is logged. Discarding a phone version in favour of a different git version is
   an explicit act, not an inference. Getting the volume's version out to commit it needs an
   export path (`fly ssh sftp get`, or a download link).

   Git itself cannot be blocked: a later week can be committed while the volume still holds an
   unarchived one. That week then falls behind the embedded tree's latest and is ignored, not
   deleted (decision 2) — hidden, but recoverable.

5. **Backup falls out of decision 4; snapshots are only a safety net.** Nothing is seeded: an
   empty volume is a valid state that reads as the embedded tree alone, so a fresh machine is
   never lineup-less. Durability then rests on the git archive, not on the volume: because the
   volume holds at most one not-yet-archived week, the **blast radius of losing the volume is
   at most that one week's edits**, re-enterable from a phone in minutes. That bound holds only
   while finished weeks are actually archived; a week left on the volume is the one exposure. Fly's automatic daily
   volume snapshots are an unrelied-upon convenience on top of that — not the backup strategy. A
   server-driven write-back-to-git (an automatic audit trail) remains a possible follow-up but is
   **not in this change**; it would reintroduce the git coupling C removes for no gain while a
   single person edits.

6. **`min_machines_running = 1`.** A Fly volume attaches to a single machine, so C wants one
   pinned, always-on writer rather than ADR 0004's scale-to-zero (`auto_stop_machines = 'stop'`,
   `min = 0`). This reverses ADR 0004's "upstream load drops to zero when nobody is looking," but
   the cost is small at our traffic and it *removes* the cold-start Sleeper fetch that
   auto-stop's cache discard imposes today (noted in `backlog.md`): the stats cache now stays
   warm. The dollar cost of one 256mb machine running continuously is negligible.

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
   the writable surface is already just the current week (decision 3), and ADR 0004's no-margin
   rule means the page never presents a week as settled. A time-based lock (reject writes to a
   week whose games have started) is a follow-up, called out here because it is a real integrity
   gap, not an oversight.

## Rationale

- **C crosses exactly the line that must be crossed, and no more.** The architectural threshold
  is "mutable runtime state" — concurrency, atomicity, a mutable working copy off git, an
  always-on writer, write auth. B and C incur that threshold identically. C adds nothing beyond
  it; B additionally adds a schema, migrations, and a query engine.

- **The mutable data is document-shaped, not relational.** A lineup is a small, self-contained
  thing keyed by `(season, week, team)` — nine starters plus bench, per team, with no join inside
  it. That is a file almost by definition. Modeling it as rows buys no relational benefit while
  costing schema and migrations, and it forfeits the genuinely useful property C keeps: lineups
  stay human-readable and git-diffable.

- **The relational win B seemed to offer is separable and not about lineups.** "Season-long
  stats" is NFL player stats transformed into fantasy score — **Sleeper's** data, not the lineup
  store's. If Sleeper load ever justifies caching that in SQLite, that cache can be added
  *alongside* file-based lineups, independently, without restructuring lineups into rows. It was
  never the same problem as lineup storage, so it does not pull lineup storage toward B.

- **C is the smallest honest step off `//go:embed`.** The read change is an `fs.FS` composed of
  two layers, invisible to `Tree`; the format is unchanged; git remains the archive and a read
  source. The new machinery is the write path, which any writable-lineup design requires, and
  the reconciliation of one volume week against git (decision 4).

- **`min = 1` is a mild net positive here.** It is required for single-machine volume ownership,
  and it happens to erase the cold-start Sleeper fetch ADR 0004 accepted. The property we give up
  — zero upstream load when idle — is bounded by the TTL cache regardless.

## Consequences

- **A lineup change no longer requires a commit or a deploy** — the reversal of ADR 0004 decision
  #3, and the point of this change. Deploys now ship code, not lineups.

- **The volume holds at most one week, so its loss is cheap.** Git remains the durable
  archive of finalized weeks (decision 4), so losing the volume costs at most that week's
  edits — re-enterable from a phone in minutes. This is what keeps C's "own a stateful volume"
  cost small: the volume is a buffer, not the store of record, and no volume-snapshot discipline
  is required to make that true.

- **New concurrency and atomicity surface.** Read-during-write and partial-write hazards exist
  for the first time; atomic temp-then-rename (decision 3) is a correctness requirement, not an
  optimisation.

- **Scale-to-zero is given up** (`min = 1`), trading ADR 0004's "zero when idle" for a warm cache
  and a single always-on writer. `fly.toml` changes accordingly.

- **A new write auth surface exists** — a shared passphrase to keep out of logs, commits, and
  screenshots. Small, but real, and it is the first secret the project carries.

- **Lineup edits are unbounded in time until a lock lands** (decision 9). A lineup can be changed
  after kickoff, rewriting what a reader saw. Accepted for now on trust; flagged as the first
  follow-up.

- **The embedded tree stays the archive and read source, but no longer always has the last
  word.** For the latest week a phone version on the volume wins over git, so a commit to that
  week is shown only once it matches what the volume holds (decision 4). A differing commit is
  logged, not served. This can surprise anyone who remembers the old model.

- **Archiving a week is a manual step with an app-enforced safety net.** Committing the volume's
  version clears it; forgetting leaves the week on the volume, still served, and blocks the next
  phone-set week (decision 3) until done. Committing a later week past it hides — but does not
  delete — the unarchived one.

## Follow-ups

- **Lineup-lock timing.** Decide and implement whether writes to a week whose games have started
  are refused. First integrity gap opened by this change.
- **Write-back-to-git.** Optionally mirror each submission back to git as an audit trail /
  restore-diffable backup, without recoupling reads to git.
- **Season-long stats.** If it lands and Sleeper load justifies it, add a *separate* stats cache
  (possibly SQLite) alongside file-based lineups — not a migration of lineups into a DB.
- **Revisit `min = 1`** if an always-on machine ever proves annoying; a moved-out-of-process
  cache or a read-replica-plus-single-writer split are the additive fixes.
