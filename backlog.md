# Backlog

## Writable lineups from a phone

The rollout of [docs/adr/0005-writable-lineups-on-a-volume.md](docs/adr/0005-writable-lineups-on-a-volume.md) —
setting the current week's lineup from a phone, no laptop or deploy. Too big for one OpenSpec
change; sliced into independently shippable steps below. One line each, roughly in dependency
order. Not sized, not prioritized. Each row is its own `/opsx:propose` when picked up.

- [ ] Serve lineups from a writable store seeded from the embedded tree — `lineup.New` over
  `os.DirFS(<path>)`, seed-on-empty from `lineup.Embedded`, path via config. Read path and page
  unchanged; ships with no visible difference.
- [ ] App-enforced boundary: the volume is relied upon only for weeks not yet in the embedded
  archive. Past weeks resolve from embed, the latest from the volume. Structural, not editor
  discipline. No visible change.
- [ ] Persist a submitted lineup — validated (ADR 0004 §9 invariants), atomic temp-then-rename,
  latest-week-only, no UI. Rejects a non-latest week, a third file in a week directory, or an
  orphaned opponent.
- [ ] Gate writes behind a shared passphrase; reads stay public. Could fold into the persist step,
  kept separate to keep its tests about access rather than correctness.
- [ ] Phone lineup-submission form — renders the current lineup and submits it. The "from my
  phone" experience over the proven write handler.
- [ ] Deploy on a Fly volume with an always-on machine — `fly.toml` volume mount and
  `min_machines_running = 1`. Depends only on the writable-store step; a candidate to do early to
  derisk Fly before the write path.

## Open threads

- [ ] Lineup-lock timing — whether to refuse writes to a week whose games have kicked off. First
  integrity gap opened by the write path; deferred in ADR 0005 (single editor, current-week-only
  surface).
- [ ] Server-driven write-back-to-git as an automatic audit trail, instead of git being fed by
  ordinary commit-and-deploy. Out of scope for the initial rollout.
- [ ] Season-long stats — if it lands and Sleeper load justifies it, a *separate* stats cache
  (possibly SQLite) alongside file-based lineups, not a migration of lineups into a DB.
