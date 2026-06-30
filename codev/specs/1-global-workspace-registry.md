# Specification: Global Workspace Registry

<!--
SPEC vs PLAN BOUNDARY:
This spec defines WHAT and WHY. The plan defines HOW and WHEN.

DO NOT include in this spec:
- Implementation phases or steps
- File paths to modify
- Code examples or pseudocode
- "First we will... then we will..."

These belong in codev/plans/1-global-workspace-registry.md
-->

## Metadata
- **ID**: 1-global-workspace-registry
- **Status**: draft
- **Created**: 2026-06-30
- **GitHub Issue**: [#1](https://github.com/amustafa/codev/issues/1)

## Problem Statement

Codev workspaces are isolated silos. Once a project is initialized with `codev init` or `codev adopt`, there is no central record of it. Users working across multiple repositories have no way to list all codev-managed workspaces, quickly switch between them, see which workspaces have active builders or pending gates, or get a cross-workspace view from Tower or the dashboard.

Each workspace must be discovered manually by remembering its filesystem path and `cd`-ing into it.

## Current State

- **Each workspace is self-contained**: State lives in `.agent-farm/state.db` per workspace and `~/.agent-farm/global.db` for global state, but `global.db` does not track which workspaces exist.
- **No workspace discovery**: There is no command to list all codev workspaces on the machine.
- **Tower is workspace-scoped**: The Tower dashboard (port 4100) shows only the workspace it was started from. Switching requires stopping Tower, navigating to another workspace, and restarting.
- **`afx status` is local**: Shows builders and architects for the current workspace only.

## Desired State

- A global registry of all codev workspaces stored in `~/.agent-farm/global.db`
- Auto-registration on `codev init`, `codev adopt`, and `codev update`
- A `codev workspaces` CLI command listing all known workspaces with status
- Stale detection for workspaces whose paths no longer exist on disk
- Tower API awareness of the registry for future cross-workspace dashboard views

## Stakeholders

- **Primary**: Developers managing multiple codev projects on one machine
- **Secondary**: Tower dashboard users who want cross-workspace visibility

## Scope

### In Scope

**1. Global registry table** — A `workspaces` table in `~/.agent-farm/global.db` tracking:
  - Absolute path to workspace root
  - Project name (derived from `package.json`, `Cargo.toml`, directory name, or user-provided)
  - Registration timestamp
  - Last activity timestamp (updated on any `afx` or `porch` command)
  - Disk existence flag (stale detection)

**2. Auto-registration** — `codev init`, `codev adopt`, and `codev update` register the current workspace automatically. Registration is idempotent — re-running on an already-registered workspace updates the entry, not duplicates it.

**3. CLI: `codev workspaces`** — List all known workspaces in a table:
  - Name, path, active builder count, last activity
  - Mark missing paths as `(missing)`
  - `--prune` flag to remove stale entries
  - `--json` flag for machine-readable output

**4. Manual registration** — `codev workspaces add <path>` for registering pre-existing workspaces that predate this feature.

### Out of Scope

- Cross-workspace `afx` commands (e.g., `afx status --all-workspaces`)
- Remote/cloud workspace tracking
- Workspace grouping or tagging
- Cross-workspace Tower dashboard views (future enhancement, depends on this registry)
- Automatic workspace switching or `cd` integration

## Design Decisions

### Store in `~/.agent-farm/global.db`

The global database already exists as the single source of truth for global state (per arch-critical.md). Adding a `workspaces` table follows the established pattern. No new files or databases needed.

### Registration is informational, not required

The registry is a convenience layer. Workspaces function fully without being registered — no features should break if a workspace is unregistered. This is graceful degradation by design.

### Project name derivation

Name is derived in priority order: `package.json` `name` field → `Cargo.toml` `[package].name` → directory basename. Users can override via `codev workspaces rename <path> <name>`.

### Last-activity tracking via middleware

Rather than modifying every `afx` and `porch` command individually, a middleware/hook updates the `last_active` timestamp whenever any codev CLI runs in a registered workspace. This is a single point of change.

## Success Criteria

- [ ] `codev init` registers the workspace in `~/.agent-farm/global.db`
- [ ] `codev adopt` registers the workspace in `~/.agent-farm/global.db`
- [ ] `codev update` registers the workspace if not already registered
- [ ] `codev workspaces` lists all registered workspaces with name, path, builder count, last activity
- [ ] Workspaces with deleted paths show `(missing)` status
- [ ] `codev workspaces --prune` removes entries for missing paths
- [ ] `codev workspaces add <path>` manually registers a workspace
- [ ] Re-running init/adopt on the same workspace updates, not duplicates
- [ ] Unregistered workspaces continue functioning normally (no regressions)
- [ ] `codev workspaces --json` outputs machine-readable JSON

## Constraints

### Technical

- `~/.agent-farm/global.db` is SQLite — use the existing database connection patterns
- Registration must be idempotent (unique constraint on path)
- Path storage must be absolute (no relative paths)
- The `last_active` timestamp update must not slow down CLI commands noticeably

### Business

- Non-breaking: existing workspaces without registration work identically
- The registry is local to the machine — no network calls, no cloud sync

## Assumptions

- `~/.agent-farm/global.db` exists and is writable (created by Tower on first start)
- SQLite supports the schema additions without migration complexity
- The number of workspaces per machine is small (< 100) — no pagination needed for the CLI listing

## Security Considerations

- The registry stores filesystem paths — no credentials or secrets
- Stale entries reveal historical workspace locations but contain no sensitive data
- The `--prune` operation only removes registry entries, never deletes workspace files

## Test Scenarios

1. **Init + list**: `codev init` in a new directory → `codev workspaces` shows it
2. **Adopt + list**: `codev adopt` in an existing project → workspace appears in registry
3. **Idempotent registration**: Run `codev init` twice → only one registry entry
4. **Stale detection**: Delete a registered workspace's directory → `codev workspaces` shows `(missing)`
5. **Prune**: `codev workspaces --prune` removes `(missing)` entries
6. **Manual add**: `codev workspaces add /path/to/existing` → workspace appears in registry
7. **Name derivation**: Init in a directory with `package.json` → name comes from `package.json`
8. **JSON output**: `codev workspaces --json` returns valid JSON array

## Dependencies

- `~/.agent-farm/global.db` SQLite database (already exists)
- No new runtime dependencies

## Risks and Mitigation

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| `global.db` doesn't exist on first run | Low | Medium | Create the database and table on first registration attempt |
| Path changes (symlinks, mounts) cause false stale detection | Low | Low | Resolve paths to canonical form before storing |
| Large number of workspaces degrades list performance | Very Low | Low | SQLite handles hundreds of rows trivially; add pagination if needed |

## References

- Architecture: `codev/resources/arch.md` — state management section
- `~/.agent-farm/global.db`: Global state database (per arch-critical.md)
