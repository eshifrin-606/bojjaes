## Why

[ADR 0005](../../../docs/adr/0005-writable-lineups-on-a-volume.md), as amended 2026-11-01, makes
the volume the **single lineup store and sole read source**, with git and the phone form as two
writers to it. The two-layer read shipped by `serve-lineups-from-two-layers` answers "whose version
wins" on every read; the amended ADR moves that answer into the writers. This change is the git
writer and the switch to one read source — the first unchecked line under "Writable lineups from a
phone" in `backlog.md` — so every later step (persist a submission, passphrase, form, export) writes
into a store the server already reads whole.

## What Changes

- **BREAKING (read path):** with a volume configured, the server's lineup tree is
  `os.DirFS(<LINEUP_VOLUME>/lineups)` and nothing else. The layered read, its "at or after the
  embedded latest" rule, and its ignored-week logging are removed, along with
  `internal/lineup/layered.go`.
- **Boot seeding.** Before the server listens, each `<season>/<week>` of the embedded tree is
  compared with a per-week content hash recorded in `<LINEUP_VOLUME>/seed-manifest.json`:
  - no entry, or a different hash → the store's week directory is replaced whole with git's, then
    the manifest entry is recorded;
  - same hash → the store's week is left alone (a phone edit to it survives).
- A week the store holds and git does not is never deleted, and its manifest entry is left as is.
- When seeding replaces a store week that held content git never seeded (a phone edit), the
  overwritten files are logged, names and contents, so the lineup can be recovered.
- Seeding is idempotent: copy precedes the manifest write, so a crash between them re-copies
  identical content on the next boot.
- A seeding failure stops the process before it listens.
- With `LINEUP_VOLUME` unset or empty, the server reads the embedded tree directly, seeds nothing,
  and writes nothing — unchanged from today.

Out of scope, each its own backlog line: the write handler, the passphrase, the phone form, export
of unarchived weeks, and the `fly.toml` volume mount / `fly scale count 1` (including the
`nonroot`-vs-root-owned-mount spike). Production keeps no `LINEUP_VOLUME`, so this ships dark.

## Capabilities

### New Capabilities

None. Which tree the server reads, and how it is filled, is `lineup-source`'s concern.

### Modified Capabilities

- `lineup-source`: "The server reads the in-flight week from a volume" is rewritten to read the
  volume alone (title kept so the delta matches; see design open questions); "The deployed binary carries its lineup tree" is reworded so the embedded tree
  is git's copy and the seed source, read directly only with no volume. The three layering
  requirements are removed. New requirements cover the seed manifest, per-week replace-or-leave,
  whole-week replacement, kept git-deleted weeks, the overwrite log, and idempotence.
- `http-server-lifecycle`: a new requirement that lineup seeding completes before the listener
  opens, and that a seeding failure exits without listening.

## Impact

- `internal/lineup`: `layered.go` and `layered_test.go` deleted. A new seeding function writing to
  a mount directory from an `fs.FS` source (the embedded tree in production, `fstest.MapFS` in
  tests). `Tree`, `Read`, `Matchup`, `HasWeek`, `LatestWeek` unchanged.
- `cmd/server`: `lineupTree` seeds (when a volume is set) and returns the volume's `os.DirFS`;
  `main.go` exits on a seeding error before `run`.
- No new dependencies (`crypto/sha256`, `encoding/json` from the standard library). No change to
  `internal/web`, routes, templates, Sleeper, or `fly.toml`.
