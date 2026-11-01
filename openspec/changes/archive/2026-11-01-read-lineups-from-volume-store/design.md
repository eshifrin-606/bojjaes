## Context

Today `cmd/server/volume.go`'s `lineupTree(volume)` returns `lineup.Embedded` when
`LINEUP_VOLUME` is empty and otherwise `lineup.Layered(lineup.Embedded, os.DirFS(<vol>/lineups),
log.Printf)`. `layered.go` routes at most one volume week (the greatest, iff at or after the
embedded latest) to the volume, merges season/root listings, and logs stale and superseded volume
weeks. It also holds the `weekDir`/`weekDirs`/`isNumberedDir` helpers that walk numbered
`<season>/<week>` directories.

ADR 0005, amended 2026-11-01, replaces that with one store (decisions 1, 2, 5): the volume's
`lineups/` is the sole read source, git seeds it per week at boot (decision 4), and the app runs on
one machine with scale-to-zero, so seeding runs on every cold start (decision 6). `lineup.New`
already reads any `fs.FS`, so the read side is `os.DirFS` and nothing more.

## Goals / Non-Goals

**Goals:**
- With a volume: read `os.DirFS(<vol>/lineups)` only; seed it from `lineup.Embedded` before
  listening.
- Seeding exactly as ADR 0005 decision 4: per-week hash vs manifest, whole-week replace,
  git-deleted weeks kept, overwritten phone edits logged, idempotent.
- Without a volume: `lineup.Embedded` directly, no writes — today's behaviour.
- Seeding testable against `fstest.MapFS` sources and `t.TempDir()` mounts; no real volume.

**Non-Goals:**
- The write handler, temp-then-rename for lineup submissions, passphrase, form, export,
  classifying weeks into synced/ui-only/diverged (decision 4's parsed-record comparison belongs to
  export), `fly.toml`, the volume-ownership spike.
- Validating roster contents during seeding — git's tree is already the read path's input today.

## Decisions

**1. Seeding lives in `internal/lineup` as `Seed(src fs.FS, mount string, logf func(string,
...any)) error`.** It walks `src` with the numbered-week walk moved out of `layered.go` (kept,
trimmed to what seeding needs), and writes under `mount/lineups` and `mount/seed-manifest.json`.
`src` is an `fs.FS` so tests pass a `fstest.MapFS`; `mount` is a path because `fs.FS` cannot write.
*Alternative:* put it in `cmd/server`. Rejected — the week walk and layout rules are
`lineup-source`'s, and `cmd/server` stays a composition root.

**2. Week hash = SHA-256 over the week's regular files, sorted by name, each as
`name NUL len(contents) NUL contents`.** Names are included: the ADR's "covers file contents only"
is read as "not mtimes or build metadata" — a renamed opponent with identical rows is a different
week. Length-prefixing keeps `a`+`bc` distinct from `ab`+`c`. Subdirectories inside a week are
ignored (none exist; the read path ignores them too). Encoded as lowercase hex.
*Alternative:* hash bytes of a tar/zip. Rejected — order and metadata leak in.

**3. Manifest = `{"weeks": {"<season>/<week>": "<hex>"}}`, written temp-then-rename.** A top-level
object leaves room for a version field later. Written after each replaced week (ADR: "copy, then
update the entry"), via `seed-manifest.json.tmp` in the mount root, then `os.Rename`. A week
needing no copy causes no write, so an unchanged boot writes nothing at all.

**4. Whole-week replacement = remove the store's week directory, then copy git's files in.**
Seeding runs before the listener opens, so no reader can observe the empty or half-filled week;
a crash mid-copy leaves no manifest entry, so the next boot re-copies (idempotence covers it).
*Alternative:* stage in a sibling temp dir and swap with renames. Rejected for now — more moving
parts for a hazard that cannot be observed, and leftover staging dirs would need cleanup. The
write handler's atomicity (decision 3 of the ADR) is a separate concern.

**5. Overwrite detection reuses the hash.** Before replacing an existing store week, hash it the
same way. If that hash equals neither the manifest entry (what git last seeded) nor git's new hash,
the week holds content git never seeded → log a header line naming `<season>/<week>`, then one line
per file with its name and full contents (`%q`-quoted so newlines stay on one log line and the log
reads back losslessly). A byte comparison, not parsed records: a CRLF-only difference logs a
harmless extra entry, never a missed one.

**6. Unreadable volume vs corrupt manifest.** Any I/O error (create `lineups/`, read/write
manifest, copy) returns from `Seed`; `main` logs it and exits non-zero before `run`. A manifest that
exists but does not parse is treated as empty with a log line — every week then re-seeds, and
decision 5 logs any phone edit it overwrites, so nothing is lost silently. (See open questions.)

**7. `lineupTree(volume string, logf) (fs.FS, error)`.** `""` → `lineup.Embedded, nil`, nothing
written. Otherwise `Seed(lineup.Embedded, volume, logf)` then `os.DirFS(filepath.Join(volume,
"lineups"))`. Returning the error forces `main` to handle it before building the server, which is
how "seed before listening" is made structural and testable without a listener.

**8. Delete `layered.go` and `layered_test.go`** once nothing references `Layered`. Volume-read
behaviour tests (`cmd/server/volume_test.go`) are rewritten to the new rule, not kept as guards for
the old one.

## Risks / Trade-offs

- [Distroless `nonroot` vs root-owned Fly mount] → Boot now writes, so an unwritable mount blocks
  all reads. Production sets no `LINEUP_VOLUME` in this change; the deploy backlog line's spike must
  land before or with enabling it.
- [Seeding cost on every cold start] → ~11 weeks × 2 small files hashed; negligible beside the
  Sleeper cold fetch.
- [Overwrite log size / leaking a lineup into logs] → Lineups are public data; a week is ~2 KB.
- [Remove-then-copy leaves a week missing if a crash lands mid-replace] → Not observable (no
  listener yet); next boot re-copies because the manifest entry was not written.
- [A git edit to the same week after a phone edit wins] → Accepted by the ADR; mitigated by the
  overwrite log.

## Migration Plan

Ships dark: `fly.toml` sets no `LINEUP_VOLUME`, so production reads the embedded tree directly.
Locally, pointing `LINEUP_VOLUME` at a scratch dir previously used with the two-layer build is
fine: no manifest → every embedded week is copied (any differing old volume week is logged), store-
only weeks are kept. Rollback is redeploying the previous binary; it reads the volume layered again
and ignores the manifest.

## Open Questions

- **Corrupt manifest:** treat as empty and re-seed (chosen; overwrites are logged) vs fail boot.
- **Hash includes file names** — chosen; ADR 0005 amended to match.
- **Overwrite trigger is byte-hash, not parsed records** — chosen for simplicity; the parsed
  comparison in ADR decision 4 is left to the export step's state classification.
- **Remove-then-copy vs staged swap** for replacing a week — chosen remove-then-copy because
  seeding precedes listening; revisit if seeding ever runs while serving.
- **Requirement title** "The server reads the in-flight week from a volume" is kept (so the
  MODIFIED delta matches) though its content is now "the volume only"; rename on archive if wanted.
- **Manifest written per replaced week** rather than once at the end — chosen to follow the ADR's
  copy-then-entry ordering literally; both are idempotent.
