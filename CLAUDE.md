# CLAUDE.md

## Coding guidelines for CLAUDE

#### TDD

Use TDD red-green loop when writing code

#### Comments
Where possible, code should speek for itself. Comments should fill in the gaps. 
Don't over comment because it makes code harder to read and comments easier to miss if comments are "empty" or meaningless

Delete comments that restate what the code says. Keep comments that explain why — a non-obvious constraint, a domain
rule, a deliberate tradeoff. Prefer renaming a function over documenting a bad name.

## Basic Memory (dev journal)

Per [docs/adr/0001-dev-journal-tool.md](docs/adr/0001-dev-journal-tool.md), this project uses Basic Memory (MCP) to journal cross-feature engineering discoveries, failed approaches, and open questions that don't belong to a specific OpenSpec change.

Conventions:

The `bojjaes-memory` project root is **`docs/`**, not `docs/basic-memory/`. Basic Memory therefore indexes the whole `docs/` tree — ADRs, `scoring.md`, probe write-ups — so they are searchable and linkable alongside journal notes.

Conventions:

- Write journal notes flat into `docs/basic-memory/` — pass `directory: "basic-memory"`, no nested subfolders.
- Title each note `yyyy-mm-dd-descriptive-title` (lowercase, hyphenated).
- Before writing a new note, search existing notes (`search_notes` / `recent_activity`) for related entries and link them with `[[wiki-links]]`. Retrieval relies on tags and links, not folder structure, so this step matters more than where the file lives.
- `[[wiki-links]]` resolve by note title, which for non-journal docs is the **filename without extension**. Link an ADR as `[[0002-live-scoreboard-backend]]`, not `[[adr-0002-...]]` and not by path. Wiki-links inside backticks are still parsed as relations — avoid writing `[[example]]` in code spans.
- Basic Memory entries are informative, not authoritative. Promote important findings to OpenSpec, ADRs, runbooks, or code docs when appropriate. A `[[link]]` to an ADR records a relationship; it does not update the ADR.

Config invariant: `ensure_frontmatter_on_sync` must stay `false` in `~/.basic-memory/config.json`. With it `true`, sync writes `title`/`type`/`permalink` frontmatter into every indexed file, including ADRs. Because these files carry no frontmatter, they have no permalink and cannot be fetched by `memory://` URI — reach them via search or by following a wiki-link to their `file_path`.
