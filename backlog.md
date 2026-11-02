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
- [x] Deploy on a Fly volume with exactly one machine — `fly.toml` volume mount,
  `fly scale count 1`; scale-to-zero stays (ADR 0005 decision 6). The container starts as root,
  claims the mount, then drops to `nonroot` before seeding.
- [x] Lineup slots — OpenSpec change `lineup-slots`. Nine slots in canonical order
  QB, RB, WR, TE, FLEX, FLEX, K, D, D (FLEX = RB/WR/TE, FB counts as RB; D = any defensive
  position). Slot is assigned by position, no longer implied by file order; a file that isn't
  exactly 9 rows or can't fill every slot is refused. Bench removed and the four over-9-row files
  trimmed. Render back to CSV in slot order. Matchup page shows starters in slot order.
- [ ] Current NFL week from Sleeper `GET /v1/state/nfl`, cached. The week Sleeper reports is the
  week the form sets, even if it doesn't flip on Tuesday. Weeks past the league's 16 and the
  offseason have no settable week.
- [ ] Lineup form, read-only — `GET /lineup?team=…`; team picker (default bojjaes; changing it
  reloads), season and week are the current week as labels. Slots start from the team's file for
  that week, else its latest lineup. One dropdown per slot, filtered by position, showing NFL team
  and kickoff. "Check" is a GET, so the URL is a shareable proposed lineup (for texting a
  teammate); also shows a plain-text summary and a CSV download to commit to git. No write path.
- [ ] Home page — `GET /` links to the scoreboard by season and to the lineup form.
- [ ] Persist a submitted lineup — validated (ADR 0004 §9 invariants), atomic temp-then-rename,
  no UI. Writable weeks: the store's latest week or the week after it (2025/16 → 2026/1 explicit
  in the path); editing a week git already has is allowed. Rejects any other week, a third file
  in a week directory, or an orphaned opponent.
- [ ] Gate writes behind a shared passphrase; reads stay public. Could fold into the persist step,
  kept separate to keep its tests about access rather than correctness.
- [ ] Export unarchived weeks (ui-only or diverged, ADR 0005 decision 4) so they can be committed
  to git — `fly ssh sftp get` documented, or a download link. Without it the archive step has
  nothing to commit.
- [ ] Save from the lineup form — a `POST` button beside "Check" that submits to the persist
  handler. The "from my phone" experience over the proven write handler.

## Open threads

- [ ] Lineup-lock timing — whether to refuse writes to a week whose games have kicked off. First
  integrity gap opened by the write path; deferred in ADR 0005 (single editor, current-week-only
  surface).
- [ ] Server-driven write-back-to-git as an automatic audit trail, instead of git being fed by
  ordinary commit-and-deploy. Out of scope for the initial rollout.
- [ ] Season-long stats — if it lands and Sleeper load justifies it, a *separate* stats cache
  (possibly SQLite) alongside file-based lineups, not a migration of lineups into a DB.
- [ ] Writable weeks vs. current week — ADR 0005 decision 3 allows the store's latest week or the
  next; the form targets Sleeper's current week. Reconcile when the persist step is picked up.
- [ ] Roster transactions (adds/drops) — rosters stay git-edited and the form ignores drift.
  Not planned.
