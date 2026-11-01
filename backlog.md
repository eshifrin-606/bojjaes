# Backlog

## Writable lineups from a phone

The rollout of [docs/adr/0005-writable-lineups-on-a-volume.md](docs/adr/0005-writable-lineups-on-a-volume.md) —
setting the current week's lineup from a phone, no laptop or deploy. Too big for one OpenSpec
change; sliced into independently shippable steps below. One line each, roughly in dependency
order. Not sized, not prioritized. Each row is its own `/opsx:propose` when picked up.

- [x] Read lineups from the volume only, seeded from git per week at boot — `os.DirFS(<path>/lineups)`
  as the sole read source; before listening, copy each embedded week whose content hash differs
  from `<path>/seed-manifest.json` (or has no entry) over the store's week whole, then record the
  hash; leave matching weeks alone, never delete a week git dropped, and log any phone edit the
  copy overwrites. ADR 0005 decision 4. Removed `internal/lineup/layered.go`; replaced the
  two-layer read below. Ships dark: `fly.toml` sets no `LINEUP_VOLUME`.
- [x] Serve lineups from two layers — the embedded archive plus an optional volume directory
  (`<path>/lineups`, path via config; unset means embedded only). The volume week wins when it
  is at or after the embedded latest; an older volume week is ignored and logged, never deleted.
  No seeding — an empty volume reads as embed alone. Read path and page unchanged; ships with no
  visible difference.
- [ ] Persist a submitted lineup — validated (ADR 0004 §9 invariants), atomic temp-then-rename,
  no UI. Writable weeks: the store's latest week or the week after it (2025/16 → 2026/1 explicit
  in the path); editing a week git already has is allowed. Rejects any other week, a third file
  in a week directory, or an orphaned opponent.
- [ ] Gate writes behind a shared passphrase; reads stay public. Could fold into the persist step,
  kept separate to keep its tests about access rather than correctness.
- [ ] Export unarchived weeks (ui-only or diverged, ADR 0005 decision 4) so they can be committed
  to git — `fly ssh sftp get` documented, or a download link. Without it the archive step has
  nothing to commit.
- [ ] Phone lineup-submission form — renders the current lineup and submits it; offers only the
  writable weeks. The "from my phone" experience over the proven write handler.
- [ ] Deploy on a Fly volume with exactly one machine — `fly.toml` volume mount,
  `fly scale count 1`; scale-to-zero (`min_machines_running = 0`) stays, since the volume
  persists across stops (ADR 0005 decision 6). Confirm that in the spike. Spike first: the image runs as distroless `nonroot`, and a Fly
  volume mount is likely root-owned, so the app may not be able to create `lineups/` or the
  manifest. Boot seeding writes to the volume, so this now blocks reads too, not just the write
  path — do the spike before, or with, setting `LINEUP_VOLUME` in `fly.toml`.

## Open threads

- [ ] Lineup-lock timing — whether to refuse writes to a week whose games have kicked off. First
  integrity gap opened by the write path; deferred in ADR 0005 (single editor, current-week-only
  surface).
- [ ] Server-driven write-back-to-git as an automatic audit trail, instead of git being fed by
  ordinary commit-and-deploy. Out of scope for the initial rollout.
- [ ] Season-long stats — if it lands and Sleeper load justifies it, a *separate* stats cache
  (possibly SQLite) alongside file-based lineups, not a migration of lineups into a DB.
