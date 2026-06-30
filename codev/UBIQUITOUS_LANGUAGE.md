# Codev

Codev is an AI-assisted development framework that orchestrates human architects and autonomous AI builders through protocol-driven workflows.

## Language

**Architect**:
The human + primary AI agent that creates specs, plans, and reviews work. A workspace has one or more named architects (e.g., `main`, `ob-refine`).
_Avoid_: Operator, supervisor, coordinator

**Builder**:
An autonomous AI agent that implements a spec in an isolated git worktree. Spawned by an architect.
_Avoid_: Worker, runner, executor

**Session**:
A live PTY terminal managed by Tower, identified by a terminal ID. Architects, builders, and utility shells each have sessions.
_Avoid_: Terminal (when referring to the logical session, not the UI element), connection, stream

**Workspace**:
A project directory registered with Tower. The unit of scoping — all views, terminals, and data are scoped to the active workspace.
_Avoid_: Project (overloaded with porch project IDs), repo, folder

**Tower**:
The local HTTP server that manages terminals, workspaces, and builder orchestration. Single instance per machine on port 4100.
_Avoid_: Server (too generic), daemon, backend

**Gate**:
A human-approval checkpoint in a protocol workflow. Builders stop and wait at gates until a human approves. Examples: `spec-approval`, `plan-approval`, `pr`.
_Avoid_: Checkpoint, blocker (a gate is intentional, not a failure)

## Relationships

- A **Workspace** hosts one or more **Architects** and zero or more **Builders**
- An **Architect** spawns **Builders**
- Each **Architect**, **Builder**, and utility shell has exactly one **Session**
- **Tower** manages all **Workspaces**, **Sessions**, and inter-agent communication
- A **Builder** may be blocked at a **Gate**, which only a human can approve

## Example dialogue

> **Dev:** "The user pressed `a` but nothing happened."
> **Domain expert:** "Does the **Workspace** have an active **Architect** with a live **Session**? The UI should attach to the **Session**, not open a new shell."

> **Dev:** "Should we show blocked **Builders** in the needs-attention list?"
> **Domain expert:** "Only if they're blocked at a **Gate**. A **Builder** that's idle waiting on input is different — that's not a **Gate** block."

## Flagged ambiguities

- "terminal" was used to mean both the UI element (a pane/tab showing PTY output) and the logical **Session** (the Tower-managed PTY). Resolved: **Session** is the Tower-managed resource; "terminal" is the UI element that renders it.
