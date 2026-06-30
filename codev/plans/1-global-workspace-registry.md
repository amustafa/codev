# Plan: Global Workspace Registry

## Metadata
- **ID**: 1-global-workspace-registry
- **Status**: draft
- **Created**: 2026-06-30
- **Spec**: `codev/specs/1-global-workspace-registry.md`
- **GitHub Issue**: [#1](https://github.com/amustafa/codev/issues/1)

## Overview

Add a global workspace registry to `~/.agent-farm/global.db` so users can list, discover, and prune all codev-managed workspaces from any directory. Three work streams:

1. **Database schema** — `workspaces` table in `global.db`
2. **Auto-registration** — Hook into `codev init`, `codev adopt`, `codev update`
3. **CLI command** — `codev workspaces` with list, add, prune, and rename subcommands

## Phase 1: Database Schema

Add a `workspaces` table to `~/.agent-farm/global.db`:

```sql
CREATE TABLE IF NOT EXISTS workspaces (
  path TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  registered_at TEXT NOT NULL DEFAULT (datetime('now')),
  last_active TEXT NOT NULL DEFAULT (datetime('now'))
);
```

- `path` is the canonical absolute path (resolved symlinks) — serves as the primary key for idempotent upserts
- `name` is derived from `package.json` name → `Cargo.toml` [package].name → directory basename
- `last_active` is updated on every `afx` or `porch` command via a shared middleware function

Create a `workspace-registry.ts` module in `packages/codev/src/lib/` exposing:
- `registerWorkspace(path, name)` — upsert into registry
- `listWorkspaces()` — return all entries with disk-existence check
- `pruneWorkspaces()` — delete entries where path no longer exists
- `touchWorkspace(path)` — update `last_active` timestamp
- `removeWorkspace(path)` — delete a single entry
- `renameWorkspace(path, newName)` — update the `name` field

## Phase 2: Auto-Registration

Hook `registerWorkspace()` into the exit path of three commands:

**`codev init`** — After successful initialization, register the workspace with the derived project name. The init flow already knows the project root and reads `package.json` — tap into that.

**`codev adopt`** — Same as init. Adopt runs in an existing project directory, so the path and name are already available.

**`codev update`** — Register if not already registered. This handles pre-existing workspaces that were set up before this feature shipped.

The registration call is fire-and-forget — if `global.db` is unavailable (first run, permissions), the command succeeds normally. The registry is informational, not load-bearing.

## Phase 3: Activity Tracking Middleware

Add a lightweight middleware that calls `touchWorkspace(cwd)` at the start of every `afx` and `porch` command invocation. This keeps the `last_active` timestamp current without modifying each command individually.

The middleware checks if the current directory is a registered workspace (by path lookup) and skips the touch if not registered. Cost: one SQLite read per CLI invocation — negligible.

## Phase 4: CLI — `codev workspaces`

Add the `workspaces` command to the `codev` CLI with the following subcommands:

**`codev workspaces`** (no subcommand, default: list)
- Table output: Name, Path, Builders, Last Active
- Mark workspaces whose path doesn't exist as `(missing)`
- Show active builder count by querying each workspace's `state.db` (if accessible)
- `--json` flag for machine-readable output

**`codev workspaces add <path>`**
- Manually register a workspace
- Resolve path to canonical absolute form
- Derive name from project files or directory basename
- Error if path doesn't exist on disk

**`codev workspaces prune`**
- Remove all entries where the path no longer exists
- Print what was removed
- `--dry-run` flag to preview without deleting

**`codev workspaces rename <path> <name>`**
- Update the display name of a registered workspace
- Error if path is not registered

## Files Modified

| File | Change |
|------|--------|
| `packages/codev/src/lib/workspace-registry.ts` | New — registry CRUD operations |
| `packages/codev/src/lib/global-db.ts` | Add `workspaces` table creation to schema init |
| `packages/codev/src/commands/init.ts` | Call `registerWorkspace()` after successful init |
| `packages/codev/src/commands/adopt.ts` | Call `registerWorkspace()` after successful adopt |
| `packages/codev/src/commands/update.ts` | Call `registerWorkspace()` if not already registered |
| `packages/codev/src/commands/workspaces.ts` | New — `codev workspaces` CLI command |
| `packages/codev/src/cli.ts` | Register the `workspaces` command |
| `packages/codev/src/agent-farm/middleware/activity-tracker.ts` | New — touch `last_active` on CLI invocation |

## Verification

- `codev init` in a fresh directory → `codev workspaces` shows it
- `codev adopt` in an existing project → appears in registry
- Run `afx status` in a registered workspace → `last_active` updates
- Delete a workspace directory → `codev workspaces` shows `(missing)`
- `codev workspaces prune` removes the `(missing)` entry
- `codev workspaces add /some/path` → appears in registry
- `codev workspaces --json` → valid JSON array output
