---
title: 2026-07-10-mcp-vs-skills-agents-commands-portability
type: note
permalink: bojjaes-memory/journal/mcp-vs-skills-agents-commands-and-basic-memorys-portability-model
tags:
- basic-memory
- mcp
- architecture
- tooling
---

## Context

Discovered while onboarding to Basic Memory, following [[0001-dev-journal-tool]] ADR. Captures reasoning about why Basic Memory is an MCP server rather than a skill/CLI, and a portability caveat not fully covered in the ADR.

## MCP vs Skills vs Agents vs Slash Commands

These operate at different layers, not just "same thing, different vendor lock-in":

- **MCP servers = new I/O/capabilities.** A separate process exposing typed tools/resources that the model calls into via a protocol. Execution happens outside the host agent (e.g. talks to a DB, an API, or in Basic Memory's case, maintains an indexed local store). Client-agnostic: any MCP-compatible client (Claude Code, Claude Desktop, Cursor, Windsurf, etc.) can use it without custom integration work.
- **Skills = instructions for using capabilities already available.** Markdown + optional scripts loaded into context when relevant. Teach a *procedure* ("when doing X, do these steps using Bash/Read/Edit"). No new schema, no new process — just prompt/context shaping.
- **Agents (subagents) = context isolation/delegation.** Own context window + tool access, for offloading exploratory or multi-step work without polluting the main conversation. Not a new capability, just orchestration.
- **Slash commands = canned prompt macros.** Simplest layer, just template expansion.

Rule of thumb: MCP adds new I/O to the world; skills/agents/commands all shape how the model uses I/O it already has.

## Basic Memory's "world" access is local, not external

Initially assumed MCP's "new I/O" framing implied reaching external systems (APIs, DBs over network). For Basic Memory specifically, the "world" it touches is local disk — no external network calls. What MCP actually buys here:

- A **persistent SQLite index / knowledge graph** built from the Markdown notes (backlinks, entity/relation parsing, full-text search) — maintained incrementally by a long-running process rather than rebuilt by grepping raw files on every agent turn.
- **Shared state across sessions/clients** — multiple Claude conversations (or other MCP clients) reading/writing the same memory store stay consistent through the server, rather than each agent independently re-parsing files and potentially disagreeing.
- **Structured retrieval** (`build_context`, graph traversal) beyond plain text search.

A skill + a Bash-invoked indexing script could technically approximate this, but would rebuild/maintain the index ad hoc per call with no schema guarantees and no live shared state across concurrent sessions — reinventing MCP informally.

## Portability caveat (SQLite index is a derived cache, not canonical)

The ADR's rationale leans on "Markdown files are the canonical source of truth" as a portability argument. That's true for note *content*, but incomplete:

- The **Markdown files are portable** — readable/greppable by any tool, human, or other agent.
- The **SQLite index/graph is a derived cache**, rebuilt from the Markdown via `basic-memory sync` (parses frontmatter + wiki-style `[[links]]`). Deleting the DB and re-syncing reconstructs it fully — so no unique data lives only in SQLite.
- Migrating to a different tool would carry over the *content* but lose the *graph structure* (backlinks, relations, search index) unless the new tool also understands Basic Memory's frontmatter/linking conventions. You'd end up with "a folder of Markdown notes," not "a knowledge graph."

This caveat was added to the Consequences section of [[0001-dev-journal-tool]].
