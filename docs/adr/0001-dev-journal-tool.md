# ADR: Use Basic Memory for Agent Development Journaling

**Status:** Accepted

## Context

We need a durable place for agents to save discoveries made during `opsx explore`, feature investigation, and other agentic development work that does not belong to a specific OpenSpec change.

The solution should:

- Preserve cross-feature discoveries, failed approaches, operational knowledge, and open questions.
- Be searchable and usable by agents without requiring regular manual editing.
- Store canonical knowledge in portable Markdown files.
- Complement OpenSpec rather than become a second specification system.

Basic Memory and ProjectMem were considered.

## Decision

Use **Basic Memory** for agent development journaling.

OpenSpec remains the system for feature-specific requirements and implementation intent. Basic Memory will hold broader engineering knowledge discovered during exploration and development.

## Rationale

Basic Memory was selected because:

- It has greater relative maturity and adoption.
- Markdown files are the canonical source of truth.
- Its built-in index supports agent search and retrieval.
- Its flexible knowledge model fits discoveries that may not relate to a specific feature.
- We do not specifically need ProjectMem’s opinionated development-event model for issues, attempts, fixes, and decisions.

## Consequences

Agents should save reusable engineering discoveries to Basic Memory and retrieve relevant notes during later exploration or implementation.

Basic Memory entries are informative rather than authoritative requirements. Important findings should be promoted to OpenSpec, ADRs, runbooks, code documentation, or other authoritative documentation when appropriate.
