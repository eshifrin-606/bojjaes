# CLAUDE.md

## Basic Memory (dev journal)

Per [docs/adr/0001-dev-journal-tool.md](docs/adr/0001-dev-journal-tool.md), this project uses Basic Memory (MCP) to journal cross-feature engineering discoveries, failed approaches, and open questions that don't belong to a specific OpenSpec change.

Conventions:

- Write notes flat into the root of the `bojjaes-memory` project (`docs/basic-memory/`) — no subfolders, and don't pass a `directory` of `basic-memory` (the project root already lives there, so that nests `basic-memory/basic-memory`). Use `directory: "/"` or omit it.
- Title each note `yyyy-mm-dd-descriptive-title` (lowercase, hyphenated).
- Before writing a new note, search existing notes (`search_notes` / `recent_activity`) for related entries and link them with `[[wiki-links]]`. Retrieval relies on tags and links, not folder structure, so this step matters more than where the file lives.
- Basic Memory entries are informative, not authoritative. Promote important findings to OpenSpec, ADRs, runbooks, or code docs when appropriate.
