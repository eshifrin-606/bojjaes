## Why

[ADR 0005](../../../docs/adr/0005-writable-lineups-on-a-volume.md) moves the in-flight week's
lineup off `//go:embed` and onto a Fly volume so it can be set from a phone. Every later step of
that rollout — persisting a submission, the passphrase gate, the phone form, archiving — needs the
server to already *read* a week from the volume. This change is that read side alone, the first
line under "Writable lineups from a phone" in `backlog.md`: it ships with no visible difference
and no write path, so it can go out (and be proven on a real volume) before anything can write.

## What Changes

- The server's lineup tree becomes two layers: the embedded archive plus an optional volume
  directory at `<path>/lineups`, where `<path>` comes from an environment variable. The app owns
  the `lineups/` subdirectory; the mount root (which holds `lost+found` on a fresh ext4 volume) is
  never read.
- With the variable unset, the server reads the embedded tree alone, exactly as today.
- The volume holds at most one week. That week is served from the volume when it is **at or after**
  the embedded tree's latest week — the same week (an edit of the latest archived week, which
  shadows the embedded copy whole) or a later one (a new week, including the first week of a new
  season, e.g. embedded latest 2025/16 and volume 2026/1). Every other week is served from the
  embedded tree.
- A volume week *older* than the embedded latest is ignored and logged — never deleted, since it
  may hold edits that never reached git.
- A missing or empty `lineups/` directory reads identically to the embedded tree alone. Nothing is
  seeded onto the volume.
- `Tree`, `Read`, `Matchup`, `HasWeek`, `LatestWeek`, the page, and every URL are unchanged: the
  layering is an `fs.FS` handed to `lineup.New`.

Out of scope, each its own backlog line: any write to the volume, the passphrase, the phone form,
clearing the volume week once git matches it, export, and the `fly.toml` volume mount /
`min_machines_running = 1`. No `fly.toml` change is made here — with no mount configured the
variable stays unset in production and the deployed server behaves exactly as it does today.

## Capabilities

### New Capabilities

None. Which tree the server reads is already `lineup-source`'s concern.

### Modified Capabilities

- `lineup-source`: "The deployed binary carries its lineup tree" is narrowed — the embedded tree
  is still always read and still the archive, but the server may additionally read one week from
  a configured volume directory. New requirements state how the two layers compose: which volume
  week wins, whole-week shadowing, the ignored stale week, and an absent/empty volume reading as
  the embedded tree alone. The existing requirements on the tree (path resolution, matchup
  resolution, latest week) are unchanged and now hold for the composed tree.

## Impact

- `internal/lineup`: a new layered `fs.FS` constructor over two `fs.FS` values (embedded, volume).
  No change to `Tree` or its methods.
- `cmd/server`: a `resolveAddr`-style resolver for the lineup volume env var, and `main.go` builds
  the tree from it instead of passing `lineup.Embedded` directly.
- No new dependencies. No change to `internal/web`, templates, routes, Sleeper, or `fly.toml`.
