## Context

`lineup.New(fsys fs.FS) *Tree` already reads through any `fs.FS`: `Read`/`Path` open
`<season>/<week>/<team>.csv`, `HasWeek` stats the week directory, `Matchup` lists it (skipping
dotfiles and non-`.csv`), and `LatestWeek` lists the season directory. `cmd/server/main.go` passes
`lineup.Embedded`, the `//go:embed` tree rooted at the season directories. Configuration is read
through injected lookups (`resolveAddr(getenv)` in `addr.go`).

ADR 0005 decisions 1, 2 and 5 fix the behaviour: CSVs on a volume under an app-owned `lineups/`
subdirectory, keyed exactly as the embedded tree; at most one volume week, served iff it is at or
after the embedded latest week; an older one ignored, logged, never deleted; no seeding, and an
empty volume reads as the embedded tree alone.

## Goals / Non-Goals

**Goals:**
- An `fs.FS` composing the embedded tree and a volume tree, so `Tree` and every caller are
  unchanged.
- A config switch: env var unset → embedded only, byte-for-byte today's behaviour.
- Tested entirely with `fstest.MapFS` for both layers; no real volume needed for `go test`.

**Non-Goals:**
- Any write, temp-then-rename, validation of volume files beyond what `Read` already does, auth,
  the phone form, boot-time clearing (ADR 0005 decision 4), export, and `fly.toml` changes.
- Enforcing "at most one week" on the volume — that is the write path's job.

## Decisions

**1. Compose at the `fs.FS` layer, not in `Tree`.** A new `lineup.Layered(archive, volume fs.FS,
logf func(string, ...any)) fs.FS` (name open to taste) implements `fs.FS` and `fs.ReadDirFS`.
*Alternative:* teach `Tree` about two sources. Rejected — every method would grow a branch, and
ADR 0005 already commits to "`lineup.New` is handed an `fs.FS` composing the two layers".

**2. Routing is by week directory, whole-week.** Each call resolves the *active volume week*
(below). A name at or under `<s>/<w>` for the active week is served from the volume; any other week
name from the archive. For the active week, the archive's copy is invisible — no per-file merge.
*Alternative:* per-file overlay. Rejected — a volume edit that changes the opponent would leave the
old opponent's CSV visible and turn the week into a three-roster error.

**3. Directory listings above the week are merged.** `ReadDir("<season>")` returns the archive's
entries plus the active volume week's directory entry (deduplicated, sorted by name as
`fs.ReadDir` promises). `Open`/`Stat` of a season directory that exists only on the volume
(`2026` when the archive ends at 2025/16) must succeed as a directory. The root listing merges the
same way. Only these listings need merging; `LatestWeek` then works unchanged.

**4. Active volume week = the greatest `<s>/<w>` directory on the volume, iff ≥ archive latest.**
Ordering is `(season, week)` lexicographic on integers, so 2026/1 > 2025/16. The archive latest is
computed once (the embedded FS is immutable). The volume is scanned on each call rather than
cached, so the next backlog step's writes are visible without a restart and there is no cache to
invalidate. The volume is a two-level listing of at most a couple of entries; the cost is
negligible beside a Sleeper fetch. Non-numeric names and non-directories are skipped, matching
`LatestWeek`.

**5. Absent or unreadable volume ≡ empty.** `os.DirFS` of a missing `lineups/` returns
`fs.ErrNotExist` on every call; the layer treats any error listing the volume root as "no volume
week". Nothing is created. *Alternative:* fail startup on a missing directory. Rejected — a fresh
volume has no `lineups/` until the first write, and ADR 0005 decision 5 says that state is valid.

**6. Logging the stale week happens once, at construction.** `Layered` scans the volume once when
built and calls `logf` for each volume week it will ignore (older than the archive latest). Logging
per request would flood the log on every page view. A week that goes stale *while running* is not
possible in this change (the archive is fixed per binary and nothing writes the volume).

**7. Config mirrors `resolveAddr`.** `resolveLineupVolume(getenv) string` in `cmd/server` reads
`LINEUP_VOLUME`; a small `lineupTree(volume string) fs.FS` returns `lineup.Embedded` for `""` and
otherwise `lineup.Layered(lineup.Embedded, os.DirFS(filepath.Join(volume, "lineups")), log.Printf)`.
The env var names the mount path, not the `lineups/` dir, so the app — not the operator — owns the
subdirectory name. `main.go` calls both. The `os.DirFS` wiring itself is the one line not covered
by a unit test beyond "non-empty path yields a layered tree"; section 9 of tasks verifies it by
hand.

## Risks / Trade-offs

- [More than one week on the volume, before the write path enforces one] → Take the greatest as the
  candidate and log the rest as ignored; never delete. This keeps the read side total rather than
  refusing to serve.
- [A symlink or `..` under the volume escaping `lineups/`] → `os.DirFS` does not follow names
  outside its root for `..`, and `Tree.Path` already refuses separators in team names. Symlinks on
  a volume only we write are not a concern at this stage.
- [Volume scanned per call] → Bounded to a tiny directory; revisit only if profiling says so.
- [The fs.FS contract (`fs.ValidPath`, `ReadDir` sorted, `*PathError`)] → Run `testing/fstest.TestFS`
  over a layered fixture as a final conformance check.

## Migration Plan

Ships dark: `fly.toml` mounts no volume and sets no `LINEUP_VOLUME`, so production reads the
embedded tree alone. Rollback is a redeploy of the previous binary. The mount itself is the later
"Deploy on a Fly volume" backlog line.

## Open Questions

- Env var name `LINEUP_VOLUME` is a proposal; rename freely before the `fly.toml` step.
- Multiple volume weeks: "greatest wins, others logged" (above) vs. treating it as a startup error.
  Chosen the former to keep reads total; revisit when the write path lands.
