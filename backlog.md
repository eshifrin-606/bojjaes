# Backlog

## Writable lineups from a phone

The rollout of [docs/adr/0005-writable-lineups-on-a-volume.md](docs/adr/0005-writable-lineups-on-a-volume.md) —
setting the current week's lineup from a phone, no laptop or deploy. Too big for one OpenSpec
change; sliced into independently shippable steps below. One line each, roughly in dependency
order. Not sized, not prioritized. Each row is its own `/opsx:propose` when picked up.

- [x] Serve lineups from two layers — the embedded archive plus an optional volume directory
  (`<path>/lineups`, path via config; unset means embedded only). The volume week wins when it
  is at or after the embedded latest; an older volume week is ignored and logged, never deleted.
  No seeding — an empty volume reads as embed alone. Read path and page unchanged; ships with no
  visible difference.
- [ ] Persist a submitted lineup — validated (ADR 0004 §9 invariants), atomic temp-then-rename,
  no UI. Writable weeks: if the volume holds a week, only that one; otherwise the embedded latest
  or the week after it. Rejects any other week, a third file in a week directory, or an orphaned
  opponent.
- [ ] Gate writes behind a shared passphrase; reads stay public. Could fold into the persist step,
  kept separate to keep its tests about access rather than correctness.
- [ ] Clear the volume week on boot once git's copy matches it (parsed records, not bytes); log,
  never clear, on a mismatch. The archive step of ADR 0005 decision 4.
- [ ] Export the volume's week so it can be committed to git — `fly ssh sftp get` documented, or
  a download link. Without it the archive step has nothing to commit.
- [ ] Phone lineup-submission form — renders the current lineup and submits it; offers only the
  writable weeks. The "from my phone" experience over the proven write handler.
- [ ] Deploy on a Fly volume with an always-on machine — `fly.toml` volume mount and
  `min_machines_running = 1`. Depends only on the two-layer read step; a candidate to do early to
  derisk Fly before the write path. Spike first: the image runs as distroless `nonroot`, and a
  Fly volume mount is likely root-owned, so the app may not be able to create `lineups/`.

## Open threads

- [ ] Lineup-lock timing — whether to refuse writes to a week whose games have kicked off. First
  integrity gap opened by the write path; deferred in ADR 0005 (single editor, current-week-only
  surface).
- [ ] Server-driven write-back-to-git as an automatic audit trail, instead of git being fed by
  ordinary commit-and-deploy. Out of scope for the initial rollout.
- [ ] Season-long stats — if it lands and Sleeper load justifies it, a *separate* stats cache
  (possibly SQLite) alongside file-based lineups, not a migration of lineups into a DB.
