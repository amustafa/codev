# ADR-0001: TUI Architect Terminal via Zellij Native Pane

**Status**: Accepted (revised)
**Date**: 2026-06-30
**Context**: The Go TUI (`packages/tui/`) is a Bubbletea status dashboard that does not embed a live terminal emulator. The web dashboard's left pane shows a live architect PTY stream via xterm.js over WebSocket — the TUI initially had no equivalent, and the first implementation (opening a new shell) did not attach to the actual Tower-managed PTY session.

## Decision

Use a **WebSocket-to-stdio bridge** (`codev-tui attach <terminalId>`) running inside a Zellij pane. The bridge connects to Tower's binary WebSocket protocol, strips/prepends the frame prefix bytes, and passes raw PTY data between the WebSocket and the pane's stdin/stdout. Zellij renders the terminal natively. The TUI fetches `DashboardState` to discover real terminal IDs for architects, builders, and shells, and opens Zellij panes via `zellij action new-pane -- codev-tui attach <id>`.

This gives the user the **actual live PTY session** — the same one visible in the web dashboard — rendered by Zellij with full scrollback, search, and copy/paste.

## Options Considered

### Option 1: New shell in workspace directory (original implementation, rejected)

Open a Zellij pane with `bash -c "cd <workspace> && exec $SHELL"`.

**Pros**: Zero protocol code, immediate.
**Cons**: Not the same session — it's a separate process, not the live architect/builder PTY from Tower. No visibility into existing open shells. The user explicitly reported this doesn't work correctly.

**Rejected because**: Doesn't solve the problem. The user needs the actual Tower-managed session.

### Option 2: WebSocket-to-stdio bridge in Zellij pane (chosen)

Build `codev-tui attach <terminalId>` which:
1. Connects to `ws://localhost:4100/ws/terminal/<terminalId>`
2. Sets the local terminal to raw mode
3. Strips `0x01` prefix from server data frames → writes to stdout
4. Reads stdin → prepends `0x01` → sends as binary WebSocket frame
5. Sends `0x00` resize control frames on `SIGWINCH`
6. Handles the handshake (pause/resume/seq)

Zellij renders the output as a native terminal pane. The TUI opens these panes via `zellij action new-pane -- codev-tui attach --tower-url <url> --workspace <path> <terminalId>`.

**Pros**:
- Shows the *actual* live Tower session (same PTY as the web dashboard)
- Zellij handles all terminal rendering, scrollback, search, copy/paste
- Works for all terminal types: architects, builders, and utility shells
- The bridge is ~120 lines of Go — thin and maintainable
- The TUI's Terminals view shows all live sessions with their IDs for one-click attach

**Cons**:
- Requires `gorilla/websocket` and `x/term` dependencies
- ~50ms latency per `zellij action` subprocess call to open a pane
- Standalone mode (`--no-zellij`) prints the command to run manually instead of opening a pane

### Option 3: Embed a Go terminal emulator in Bubbletea

Connect to Tower's WebSocket and render terminal output inside a Bubbletea viewport using a Go VT100/xterm parser.

**Pros**: Works without Zellij. Single-pane experience.
**Cons**: No mature Go terminal emulator library exists (xterm.js is ~30k LOC). Bubbletea's `View() string` rendering model conflicts with incremental terminal updates. Multi-month effort.

**Rejected because**: The implementation cost is disproportionate to the value.

### Option 4: Rust WASM Zellij plugin with `open_terminal()`

Use the Rust WASM plugin (`packages/tui-zellij/`) which can call Zellij's host functions directly.

**Pros**: Sub-millisecond pane control, event-driven tracking.
**Cons**: Requires maintaining a separate Rust implementation. Still wouldn't attach to the *Tower* session (Zellij's `open_terminal()` opens a new PTY, not Tower's).

**Rejected because**: Even the WASM plugin would need the same WebSocket bridge to show Tower's actual sessions. The Go bridge is simpler and works with both TUI implementations.

## Consequences

- `codev-tui attach <terminalId>` is a new subcommand that bridges Tower's WebSocket binary protocol to stdin/stdout
- The TUI fetches `DashboardState` to discover terminal IDs for all architects, builders, and shells
- A new **Terminals** tab lists all active sessions with type, name, and terminal ID — pressing Enter opens a Zellij pane attached to that session
- Pressing `a` opens the first architect's terminal (by terminal ID, not a new shell)
- Pressing Enter on a builder in the Builders tab opens its actual terminal session
- Standalone mode (`--no-zellij`) prints the `codev-tui attach` command for manual use
- The Zellij layout no longer needs a pre-opened architect pane — the user opens terminals on demand from the TUI
