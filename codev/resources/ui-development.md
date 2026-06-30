# Tower UI Development Reference

Complete API reference for building a new Codev Tower UI. Covers every HTTP endpoint, the WebSocket binary terminal protocol, Server-Sent Events, reconnection policy, and all response types.

**Source of truth**: `packages/types/src/` (TypeScript type contracts), `packages/codev/src/agent-farm/servers/tower-routes.ts` (route dispatch table).

---

## 1. Architecture Overview

Tower is a local HTTP server that manages AI agent terminals, workspaces, and builder orchestration. It runs on **port 4100** by default, bound to **localhost only**.

```
┌─────────────────────────────────────────────┐
│                   Tower                      │
│  HTTP server (port 4100)                     │
│  ├── REST API (JSON)                         │
│  ├── WebSocket (binary terminal I/O)         │
│  ├── SSE (real-time push notifications)      │
│  └── Static file serving (React dashboard)   │
├─────────────────────────────────────────────┤
│  State: SQLite (state.db + global.db)        │
│  Terminals: PTY sessions via shellper        │
│  Workspaces: git worktrees in .builders/     │
└─────────────────────────────────────────────┘
```

**Consumers**: React dashboard (`packages/dashboard/`), VSCode extension (`packages/vscode/`), CLI tools (`afx`, `porch`).

---

## 2. Authentication & Security

### Localhost binding

Tower binds to `127.0.0.1:4100` by default. All non-localhost requests are rejected with `403 Forbidden` via the `isRequestAllowed()` check on every request.

**Bridge mode**: Set `BRIDGE_MODE=1` and `BRIDGE_TOWER_HOST` to allow non-localhost binding (for remote access via tunnel).

### Auth token (optional)

Clients may store a token in `localStorage['codev-web-key']` and send it as:

```
Authorization: Bearer <token>
```

The server does not enforce this by default — it's application-level.

### CORS

```
Access-Control-Allow-Origin: (localhost:* | 127.0.0.1:* | https://*)
Access-Control-Allow-Methods: GET, POST, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
Cache-Control: no-store
```

### Rate limiting

Workspace activations: 10 per minute per client IP. Returns `429 Too Many Requests`.

---

## 3. Workspace Path Encoding

Many routes use an encoded workspace path in the URL. Encoding: **Base64URL (RFC 4648)** — no padding, URL-safe alphabet (`-` and `_` instead of `+` and `/`).

```
Filesystem path:  /home/user/my-project
Encoded:          L2hvbWUvdXNlci9teS1wcm9qZWN0
URL:              /workspace/L2hvbWUvdXNlci9teS1wcm9qZWN0/api/state
```

**Go**: `base64.RawURLEncoding.EncodeToString([]byte(path))`
**JS**: `btoa(path).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')`

Path validation: must start with `/` (POSIX) or match `[A-Za-z]:[\/]` (Windows). Normalized via `path.resolve()`.

---

## 4. HTTP API Reference

### 4.1 Health & Version

#### GET /health

Health check with readiness gate.

**Response** `200`:
```json
{
  "status": "healthy",
  "ready": true,
  "uptime": 3600.5,
  "activeWorkspaces": 2,
  "totalWorkspaces": 3,
  "memoryUsage": 85000000,
  "timestamp": "2026-06-29T12:00:00.000Z"
}
```

`ready` is `false` during startup until persistent sessions are rehydrated from SQLite.

#### GET /api/version

Running Tower process version.

**Response** `200`:
```json
{
  "version": "3.1.3",
  "startedAt": "2026-06-29T10:00:00.000Z"
}
```

### 4.2 Workspaces

#### GET /api/workspaces

List all known workspaces.

**Response** `200`:
```json
{
  "workspaces": [
    {
      "path": "/home/user/project",
      "name": "project",
      "active": true,
      "proxyUrl": "https://...",
      "terminals": 4
    }
  ]
}
```

#### POST /api/create

Create a new workspace directory.

**Request body**:
```json
{
  "parent": "/home/user",
  "name": "new-project"
}
```

`name` must match `^[a-zA-Z0-9_-]+$`. Target directory must not exist.

**Response** `200`: `{ "success": true, "workspacePath": "/home/user/new-project" }`
**Error** `400`: `{ "success": false, "error": "..." }`

#### POST /api/launch

Activate a workspace instance.

**Request body**: `{ "workspacePath": "/absolute/path" }`

Supports `~` expansion. No relative paths (`../`).

**Response** `200`: `{ "success": true, "adopted": false }`
**Error** `400`: `{ "success": false, "error": "..." }`
**Error** `429`: Rate-limited (10/min/IP)

#### POST /api/stop

Stop a workspace instance.

**Request body**: `{ "workspacePath": "/absolute/path" }`

**Response** `200`: instance stop result

#### GET /api/workspaces/:encodedPath/status

**Response** `200`:
```json
{
  "path": "/home/user/project",
  "name": "project",
  "active": true,
  "terminals": [...]
}
```

#### POST /api/workspaces/:encodedPath/activate

Activate a workspace.

**Response** `200`: activation result
**Error** `429`: Rate-limited

#### POST /api/workspaces/:encodedPath/deactivate

Deactivate a workspace.

**Response** `200`: deactivation result

#### POST /api/workspaces/:encodedPath/architects

Add a named architect to a workspace.

**Request body**: `{ "name": "ob-refine" }` (optional, auto-generated if omitted)

**Response** `200`: `{ "success": true, "name": "ob-refine", "terminalId": "..." }`
**Error** `400`/`404`: `{ "success": false, "error": "..." }`

#### DELETE /api/workspaces/:encodedPath/architects/:name

Remove a sibling architect. Cannot remove `main`.

**Response** `200`: `{ "success": true }`
**Error** `400`: `{ "success": false, "error": "Cannot remove main" }`

### 4.3 Terminals

#### POST /api/terminals

Create a new terminal session.

**Request body**:
```json
{
  "command": "bash",
  "args": [],
  "cols": 120,
  "rows": 30,
  "cwd": "/home/user/project",
  "env": { "TERM": "xterm-256color" },
  "label": "Builder 42",
  "workspacePath": "/home/user/project",
  "type": "builder",
  "roleId": "builder-spir-42",
  "persistent": true
}
```

All fields optional. `persistent: true` requests shellper backend (survives Tower restarts).

**Response** `201`:
```json
{
  "id": "uuid-here",
  "pid": 12345,
  "cols": 120,
  "rows": 30,
  "wsPath": "/ws/terminal/uuid-here",
  "persistent": true
}
```

#### GET /api/terminals

List all terminal sessions.

**Response** `200`: `{ "terminals": [...PtySessionInfo] }`

#### GET /api/terminals/:id

Get terminal info.

**Response** `200`:
```json
{
  "id": "uuid",
  "pid": 12345,
  "cols": 120,
  "rows": 30,
  "label": "Builder 42",
  "cwd": "/home/user/project",
  "lastDataAt": 1719655200000,
  "shellperSessionId": "..."
}
```
**Error** `404`: `{ "error": "NOT_FOUND", "message": "..." }`

#### DELETE /api/terminals/:id

Kill terminal and delete from registry.

**Response** `204` (no body)
**Error** `404`: `{ "error": "NOT_FOUND", "message": "..." }`

#### POST /api/terminals/:id/write

Write data to terminal.

**Request body**: `{ "data": "ls -la\n" }` (`data` must be a string)

**Response** `200`: `{ "ok": true }`
**Error** `400`/`404`

#### POST /api/terminals/:id/resize

Resize terminal PTY.

**Request body**: `{ "cols": 120, "rows": 30 }`

**Response** `200`: session info
**Error** `400`/`404`

#### GET /api/terminals/:id/output

Get terminal output buffer.

**Query params**: `?lines=100&offset=0`

**Response** `200`: `{ "lines": 100, "data": "...", "total": 500 }`
**Error** `404`

#### PATCH /api/terminals/:id/rename

Rename a shell terminal (builders/architects cannot be renamed).

**Request body**: `{ "name": "My Shell" }` (1-100 chars, control chars stripped)

**Response** `200`: `{ "id": "...", "name": "My Shell" }`
**Error** `403`: `{ "error": "Cannot rename builder/architect terminals" }`

### 4.4 Overview & Analytics

#### GET /api/overview

Primary data endpoint for the work view. Returns builders, PRs, backlog, and recently closed items.

**Query params**: `?workspace=/path` (optional, defaults to first non-builder workspace)

**Response** `200`: `OverviewData` (see [Response Types](#8-response-type-reference))

#### POST /api/overview/refresh

Invalidate the overview cache. Next `GET /api/overview` re-fetches from GitHub + filesystem.

**Response** `200`: `{ "ok": true }`

#### GET /api/analytics

Development metrics and consultation stats.

**Query params**:
- `?workspace=/path` (optional)
- `?range=1|7|30|all` (default `7`; invalid → `400`)
- `?refresh=1` (force cache refresh)

**Response** `200`: `AnalyticsResponse` (see [Response Types](#8-response-type-reference))
**Error** `400`: `{ "error": "Invalid range. Must be 1, 7, 30, or all." }`

### 4.5 Issues

#### GET /api/issue

Fetch a single issue by number.

**Query params**: `?workspace=/path&number=42`

**Response** `200`: `IssueView`
```json
{
  "title": "Add TUI support",
  "body": "## Description\n...",
  "state": "open",
  "comments": [
    { "body": "...", "createdAt": "2026-06-29T...", "author": { "login": "user" } }
  ]
}
```
**Error** `400`: Missing workspace or number
**Error** `404`: Issue not found or forge unavailable

#### GET /api/issue-search

Search issues with body content for filtering.

**Query params**:
- `?workspace=/path` (optional)
- `?state=open|closed|all` (default `open`)

**Response** `200`:
```json
{
  "items": [
    {
      "id": "42",
      "title": "Add TUI support",
      "url": "https://github.com/...",
      "area": "area/tower",
      "author": "user",
      "assignees": ["user"],
      "createdAt": "2026-06-01T...",
      "body": "## Description\n..."
    }
  ],
  "currentUser": "user",
  "error": null
}
```

### 4.6 Inter-Agent Messaging

#### POST /api/send

Send a message to a resolved agent terminal.

**Request body**:
```json
{
  "to": "architect",
  "message": "PR ready for review",
  "from": "builder-42",
  "workspace": "/path/to/project",
  "fromWorkspace": "/path/to/project",
  "options": {
    "raw": false,
    "noEnter": false,
    "interrupt": false
  }
}
```

Required: `to`, `message`. All other fields optional.

**Target addressing forms**:

| `to` value | Resolution |
|---|---|
| `<builder-id>` | Specific builder (e.g. `0042`) |
| `architect` | Spawning architect (from builder) or main architect |
| `architect:<name>` | Named architect. Builders: only their spawning architect. Architects: any sibling. |
| `<workspace>:architect` | Cross-workspace (e.g. `marketmaker:architect`) |

**Delivery modes**:
- **Immediate** (`deferred: false`): user is idle, written directly
- **Deferred** (`deferred: true`): user is typing, buffered until 500ms idle
- **Interrupt** (`options.interrupt: true`): sends Ctrl+C first, waits 100ms, then delivers

**Response** `200`:
```json
{
  "ok": true,
  "terminalId": "uuid",
  "resolvedTo": "architect",
  "deferred": false
}
```
**Error** `400`: `{ "error": "INVALID_PARAMS", "message": "..." }`
**Error** `404`: `{ "error": "NOT_FOUND", "message": "..." }`
**Error** `409`: `{ "error": "AMBIGUOUS", "message": "..." }`

### 4.7 Workspace-Scoped Routes

All routes under `/workspace/:encodedPath/api/...`. The workspace path is Base64URL-encoded.

#### GET .../api/state

Full dashboard state for a workspace.

**Response** `200`: `DashboardState` (see [Response Types](#8-response-type-reference))

#### POST .../api/tabs/shell

Create a new shell terminal tab.

**Response** `200`: `{ "id": "shell-1", "port": 0, "name": "Shell", "terminalId": "uuid", "persistent": true }`

#### POST .../api/tabs/file

Create a file viewer tab.

**Request body**:
```json
{
  "path": "src/main.ts",
  "line": 42,
  "terminalId": "uuid-for-resolving-relative-paths"
}
```

**Response** `200`: `{ "id": "file-uuid", "existing": false, "line": 42, "notFound": false }`
**Error** `400`: `{ "error": "Missing path parameter" }`

#### GET .../api/file/:id

Get file content and metadata.

**Response** `200` (text file):
```json
{
  "path": "/absolute/path/to/file.ts",
  "name": "file.ts",
  "content": "import ...",
  "language": "typescript",
  "isMarkdown": false,
  "isImage": false,
  "isVideo": false
}
```

**Response** `200` (binary file):
```json
{
  "path": "/absolute/path/to/image.png",
  "name": "image.png",
  "content": null,
  "language": "png",
  "isMarkdown": false,
  "isImage": true,
  "isVideo": false,
  "size": 102400
}
```

#### GET .../api/file/:id/raw

Raw binary file download. Returns the file with appropriate `Content-Type` and `Content-Length` headers.

#### POST .../api/file/:id/save

Save file content.

**Request body**: `{ "content": "new file content" }`

**Response** `200`: `{ "success": true }`

#### DELETE .../api/tabs/:id

Delete a terminal or file tab.

**Response** `204` (no body)

#### POST .../api/stop

Stop all terminals in the workspace.

**Response** `200`: `{ "ok": true }`

#### GET .../api/files

Directory tree for the workspace.

**Query params**: `?depth=3` (max directory depth, default 3)

**Response** `200`:
```json
[
  {
    "name": "src",
    "path": "src",
    "type": "directory",
    "children": [
      { "name": "main.ts", "path": "src/main.ts", "type": "file" }
    ]
  }
]
```

Ignored directories: `.git`, `node_modules`, `.builders`, `dist`, `.agent-farm`, `.next`, `.cache`, `__pycache__`

#### GET .../api/git/status

Git status for the workspace.

**Response** `200`:
```json
{
  "modified": ["src/main.ts"],
  "staged": [],
  "untracked": ["new-file.ts"],
  "error": null
}
```

Returns 200 with empty arrays + `error` field if not a git repo.

#### GET .../api/files/recent

Recently opened file tabs (max 10, most recent first).

**Response** `200`:
```json
[
  { "id": "file-uuid", "path": "/abs/path", "name": "main.ts", "relativePath": "src/main.ts" }
]
```

#### GET .../api/team

Team members and collaboration data.

**Response** `200` (team disabled): `{ "enabled": false }`

**Response** `200` (team enabled): `TeamApiResponse` (see [Response Types](#8-response-type-reference))

#### POST .../api/paste-image

Upload a pasted image.

**Request**: Binary body with `Content-Type: image/png|jpeg|gif|webp`. Max 10 MB.

**Response** `200`: `{ "path": "/tmp/codev-paste-abc123.png" }`
**Error** `413`: `{ "error": "Image too large (max 10 MB)" }`

#### GET .../api/overview

Workspace-scoped overview (same response as global `GET /api/overview`).

#### POST .../api/overview/refresh

Workspace-scoped cache invalidation.

#### GET .../api/analytics

Workspace-scoped analytics.

#### GET .../api/events

Workspace-scoped SSE event stream.

#### DELETE .../api/architects/:name

Remove a sibling architect from this workspace.

### 4.8 Cron Tasks

#### GET /api/cron/tasks

List all cron tasks.

**Query params**: `?workspace=/path` (optional filter)

**Response** `200`:
```json
[
  {
    "name": "daily-review",
    "schedule": "0 9 * * *",
    "enabled": true,
    "last_run": 1719655200,
    "last_result": "success",
    "workspacePath": "/home/user/project"
  }
]
```

#### GET /api/cron/tasks/:name/status

Detailed task status.

**Query params**: `?workspace=/path` (disambiguate if task exists in multiple workspaces)

**Response** `200`:
```json
{
  "name": "daily-review",
  "schedule": "0 9 * * *",
  "command": "...",
  "enabled": true,
  "last_run": 1719655200,
  "last_result": "success",
  "last_output": "...",
  "workspacePath": "/home/user/project",
  "target": "...",
  "timeout": 300
}
```
**Error** `409`: `{ "error": "AMBIGUOUS", "message": "...", "workspaces": [...] }`

#### POST /api/cron/tasks/:name/run

Execute a cron task immediately.

**Response** `200`: `{ "ok": true, "result": "...", "output": "..." }`

#### POST /api/cron/tasks/:name/enable

**Response** `200`: `{ "ok": true, "name": "...", "enabled": true }`

#### POST /api/cron/tasks/:name/disable

**Response** `200`: `{ "ok": true, "name": "...", "enabled": false }`

### 4.9 Tunnel

#### GET /api/tunnel/status

Cloud tunnel connection state.

**Response** `200`:
```json
{
  "registered": true,
  "state": "connected",
  "uptime": 3600,
  "towerId": "abc123",
  "towerName": "my-tower",
  "serverUrl": "https://...",
  "accessUrl": "https://..."
}
```

`state`: `disconnected | connecting | connected | auth_failed | error`

**Response** `404`: Tunnel not configured (return `null` client-side)

#### POST /api/tunnel/connect

Establish cloud tunnel connection.

#### POST /api/tunnel/disconnect

Close cloud tunnel connection.

### 4.10 Command Relay

#### POST /api/command

Relay a canonical command verb to the active editor provider (VSCode, dashboard).

**Request body**:
```json
{
  "verb": "view-diff",
  "args": ["builder-42"],
  "workspace": "/home/user/project"
}
```

**Response** `200`: `{ "ok": true }`
**Error**: `{ "ok": false, "error": "..." }`

The command is also broadcast as an SSE `command` event to all connected providers.

### 4.11 Browse

#### GET /api/browse

Directory suggestions for autocomplete.

**Query params**: `?path=/home/user/pro`

**Response** `200`: `{ "suggestions": ["/home/user/project", "/home/user/projects"] }`

---

## 5. WebSocket Binary Protocol

Terminal I/O uses a binary WebSocket protocol over raw frames.

### Connection URLs

```
ws://localhost:4100/ws/terminal/<terminal-id>
ws://localhost:4100/workspace/<base64-path>/ws/terminal/<terminal-id>
```

Optional reconnection: `?resume=<seq>` to resume from a sequence number.

### Frame format

Every WebSocket message is a binary frame with a 1-byte prefix:

| Prefix | Type | Payload |
|---|---|---|
| `0x00` | Control | UTF-8 JSON: `{ "type": "...", "payload": {...} }` |
| `0x01` | Data | Raw terminal bytes (PTY output or user input) |

### Control message types

| Type | Direction | Payload | Purpose |
|---|---|---|---|
| `resize` | Client → Server | `{ "cols": 120, "rows": 30 }` | Resize PTY |
| `ping` | Client → Server | `{}` | Keepalive |
| `pong` | Server → Client | `{}` | Keepalive response |
| `pause` | Server → Client | `{}` | Begin replay buffer (initial connect) |
| `resume` | Server → Client | `{}` | End replay buffer |
| `seq` | Server → Client | `{ "seq": 12345 }` | Current sequence number |
| `error` | Server → Client | `{ "message": "..." }` | Error notification |

### Handshake sequence

```
Client                              Server
  |                                   |
  +--- WebSocket upgrade -----------→ |
  |                                   | (lookup terminal, prepare replay)
  | ←--- [0x00] { type: "pause" } ---+
  | ←--- [0x01] <replay bytes> ------+  (buffered output since last seq)
  | ←--- [0x00] { type: "resume" } --+
  | ←--- [0x00] { type: "seq",       |
  |              payload: {seq: N} }--+
  |                                   |
  | (bidirectional data frames)       |
  +--- [0x01] <user input> ---------→ |
  | ←--- [0x01] <pty output> --------+
  |                                   |
  | ←--- [0x00] { type: "seq", ... } +  (every 10 seconds)
```

### Resume protocol

On reconnect, pass `?resume=<lastSeq>` where `lastSeq` is the most recent sequence number received. The server replays output from its ring buffer starting after that sequence.

### Backpressure

Server drops frames when `ws.bufferedAmount > 1 MB`. Terminal output is ephemeral; dropping is preferable to unbounded memory. The client can detect lost frames via sequence gaps and reconnect.

### Close codes

| Code | Meaning | Action |
|---|---|---|
| `4404` | Session unknown/gone (permanent) | Re-fetch state, find successor session ID |
| `1006` | Transport blip (transient) | Retry with exponential backoff |
| Other | Transient | Retry with backoff |

---

## 6. Server-Sent Events (SSE)

### Endpoint

`GET /api/events` (global) or `GET /workspace/:path/api/events` (workspace-scoped)

### Response headers

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

### Event format

**Initial connected event** (sent immediately on connect):
```
data: {"type":"connected","id":"a1b2c3d4"}

```

**Broadcast events**:
```
id: 42
data: {"type":"overview-changed"}

```

**Heartbeat** (every 30 seconds):
```
:heartbeat

```

### Event types

| Type | Payload | Trigger |
|---|---|---|
| `connected` | `{ "id": "<clientId>" }` | On SSE connection |
| `overview-changed` | (none) | Overview cache invalidated |
| `builder-spawned` | `{ "terminalId", "roleId", "workspacePath" }` | New builder created |
| `architects-updated` | `{ "workspace": "<path>" }` | Architect added/removed |
| `codev-config-updated` | (none) | `.codev/config.json` changed |
| `notification` | `{ "type", "title", "body", "workspace?" }` | Generic notification |
| `command` | `CommandRequest` | Command relay from controller |
| `heartbeat` | (none, `:heartbeat` comment) | Every 30s |

### Limits

- Max **50 concurrent SSE clients** per Tower process
- Oldest clients evicted when limit reached
- Connections older than **5 minutes** auto-evicted

### Client implementation notes

- Use a **singleton** connection per UI instance (browsers have a 6-connection-per-origin limit for HTTP/1.1)
- **Close when hidden** (tab background) to free the connection slot
- **Reconnect on focus** — the browser's `EventSource` auto-reconnects on error
- On any message, trigger a re-fetch of relevant data (state, overview)

---

## 7. Reconnection Policy

Shared across all clients (dashboard, VSCode, TUI).

### Exponential backoff

```
delay = min(baseMs * 2^attempt, capMs)
```

| Parameter | Value |
|---|---|
| `baseMs` | 1000 |
| `capMs` | 30000 |
| `maxAttempts` | 6 |

**Delay sequence**: 1s → 2s → 4s → 8s → 16s → 30s → give up.

### Error classification

| Error | Classification | Action |
|---|---|---|
| WebSocket close `4404` | Permanent | Re-fetch state, find successor session ID. Grace window: ~4s. |
| WebSocket close `1006` | Transient | Retry with backoff |
| HTTP 4xx | Permanent | Stop retrying |
| Network error | Transient | Retry with backoff |

### Persistent session recovery

After a Tower restart, persistent sessions are rehydrated from SQLite under new terminal IDs. The client:
1. Detects permanent close (4404)
2. Re-fetches workspace state
3. Finds successor session under new ID
4. Remounts terminal with new WebSocket path

---

## 8. Response Type Reference

Go-style struct definitions for all API response types. JSON field names use camelCase.

### Core State

```go
type DashboardState struct {
    Architect   *ArchitectState  `json:"architect"`    // main or first architect (backward-compat)
    Architects  []ArchitectState `json:"architects"`   // all architects, main-first
    Builders    []Builder        `json:"builders"`
    Utils       []UtilTerminal   `json:"utils"`
    Annotations []Annotation     `json:"annotations"`
    WorkspaceName string         `json:"workspaceName,omitempty"`
    Version     string           `json:"version,omitempty"`
    Hostname    string           `json:"hostname,omitempty"`
    TeamEnabled bool             `json:"teamEnabled,omitempty"`
}

type ArchitectState struct {
    Name       string `json:"name"`       // "main" or sibling name
    Port       int    `json:"port"`
    PID        int    `json:"pid"`
    TerminalID string `json:"terminalId,omitempty"`
    Persistent bool   `json:"persistent,omitempty"`
}

type Builder struct {
    ID                 string `json:"id"`
    Name               string `json:"name"`
    Port               int    `json:"port"`
    PID                int    `json:"pid"`
    Status             string `json:"status"`
    Phase              string `json:"phase"`
    Worktree           string `json:"worktree"`
    Branch             string `json:"branch"`
    Type               string `json:"type"`
    ProjectID          string `json:"projectId,omitempty"`
    TerminalID         string `json:"terminalId,omitempty"`
    Persistent         bool   `json:"persistent,omitempty"`
    SpawnedByArchitect string `json:"spawnedByArchitect,omitempty"`
}

type UtilTerminal struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    Port       int    `json:"port"`
    PID        int    `json:"pid"`
    TerminalID string `json:"terminalId,omitempty"`
    Persistent bool   `json:"persistent,omitempty"`
    LastDataAt int64  `json:"lastDataAt,omitempty"`
}

type Annotation struct {
    ID   string `json:"id"`
    File string `json:"file"`
    Port int    `json:"port"`
    PID  int    `json:"pid"`
}
```

### Overview

```go
type OverviewData struct {
    Builders       []OverviewBuilder       `json:"builders"`
    PendingPRs     []OverviewPR            `json:"pendingPRs"`
    Backlog        []OverviewBacklogItem   `json:"backlog"`
    RecentlyClosed []OverviewRecentlyClosed `json:"recentlyClosed"`
    Architects     []ArchitectState        `json:"architects"`
    CurrentUser    string                  `json:"currentUser,omitempty"`
    Errors         *OverviewErrors         `json:"errors,omitempty"`
}

type OverviewErrors struct {
    PRs    string `json:"prs,omitempty"`
    Issues string `json:"issues,omitempty"`
}

type OverviewBuilder struct {
    ID                 string            `json:"id"`
    IssueID            *string           `json:"issueId"`
    IssueTitle         *string           `json:"issueTitle"`
    Phase              string            `json:"phase"`
    ProtocolPhase      string            `json:"protocolPhase"`
    Mode               string            `json:"mode"`         // "strict" | "soft"
    Gates              map[string]string `json:"gates"`
    WorktreePath       string            `json:"worktreePath"`
    RoleID             *string           `json:"roleId"`
    Protocol           string            `json:"protocol"`
    PlanPhases         []PlanPhase       `json:"planPhases"`
    Progress           int               `json:"progress"`
    Blocked            *string           `json:"blocked"`
    BlockedGate        *string           `json:"blockedGate"`
    BlockedSince       *string           `json:"blockedSince"`
    StartedAt          *string           `json:"startedAt"`
    IdleMs             int64             `json:"idleMs"`
    LastDataAt         *string           `json:"lastDataAt"`
    SpawnedByArchitect *string           `json:"spawnedByArchitect"`
    Area               string            `json:"area"`
    PRReady            bool              `json:"prReady"`
}

type PlanPhase struct {
    ID     string `json:"id"`
    Title  string `json:"title"`
    Status string `json:"status"`
}

type OverviewPR struct {
    ID             string   `json:"id"`
    Title          string   `json:"title"`
    URL            string   `json:"url"`
    ReviewStatus   string   `json:"reviewStatus"`
    LinkedIssue    *string  `json:"linkedIssue"`
    CreatedAt      string   `json:"createdAt"`
    Author         string   `json:"author,omitempty"`
    ReviewRequests []string `json:"reviewRequests"`
    IsDraft        bool     `json:"isDraft"`
}

type OverviewBacklogItem struct {
    ID         string   `json:"id"`
    Title      string   `json:"title"`
    URL        string   `json:"url"`
    Type       string   `json:"type"`
    Priority   string   `json:"priority"`
    Area       string   `json:"area"`
    HasSpec    bool     `json:"hasSpec"`
    HasPlan    bool     `json:"hasPlan"`
    HasReview  bool     `json:"hasReview"`
    HasBuilder bool     `json:"hasBuilder"`
    CreatedAt  string   `json:"createdAt"`
    Author     string   `json:"author,omitempty"`
    Assignees  []string `json:"assignees,omitempty"`
    SpecPath   string   `json:"specPath,omitempty"`
    PlanPath   string   `json:"planPath,omitempty"`
    ReviewPath string   `json:"reviewPath,omitempty"`
}

type OverviewRecentlyClosed struct {
    ID         string `json:"id"`
    Title      string `json:"title"`
    URL        string `json:"url"`
    Type       string `json:"type"`
    ClosedAt   string `json:"closedAt"`
    PRUrl      string `json:"prUrl,omitempty"`
    SpecPath   string `json:"specPath,omitempty"`
    PlanPath   string `json:"planPath,omitempty"`
    ReviewPath string `json:"reviewPath,omitempty"`
}
```

### Analytics

```go
type AnalyticsResponse struct {
    TimeRange    string                 `json:"timeRange"` // "24h" | "7d" | "30d" | "all"
    Activity     ActivityMetrics        `json:"activity"`
    Consultation ConsultationMetrics    `json:"consultation"`
    Errors       *AnalyticsErrors       `json:"errors,omitempty"`
}

type ActivityMetrics struct {
    PRsMerged                int                       `json:"prsMerged"`
    MedianTimeToMergeHours   *float64                  `json:"medianTimeToMergeHours"`
    IssuesClosed             int                       `json:"issuesClosed"`
    MedianTimeToCloseBugsHrs *float64                  `json:"medianTimeToCloseBugsHours"`
    ProjectsByProtocol       map[string]ProtocolStats  `json:"projectsByProtocol"`
}

type ProtocolStats struct {
    Count             int      `json:"count"`
    AvgWallClockHours *float64 `json:"avgWallClockHours"`
    AvgAgentTimeHours *float64 `json:"avgAgentTimeHours"`
}

type ConsultationMetrics struct {
    TotalCount       int                `json:"totalCount"`
    TotalCostUSD     *float64           `json:"totalCostUsd"`
    CostByModel      map[string]float64 `json:"costByModel"`
    AvgLatencySecs   *float64           `json:"avgLatencySeconds"`
    SuccessRate      *float64           `json:"successRate"`
    ByModel          []ModelMetric      `json:"byModel"`
    ByReviewType     map[string]int     `json:"byReviewType"`
    ByProtocol       map[string]int     `json:"byProtocol"`
}

type ModelMetric struct {
    Model       string   `json:"model"`
    Count       int      `json:"count"`
    AvgLatency  float64  `json:"avgLatency"`
    TotalCost   *float64 `json:"totalCost"`
    SuccessRate float64  `json:"successRate"`
}
```

### Team

```go
type TeamApiResponse struct {
    Enabled     bool             `json:"enabled"`
    Members     []TeamApiMember  `json:"members,omitempty"`
    Messages    []TeamApiMessage `json:"messages,omitempty"`
    Warnings    []string         `json:"warnings,omitempty"`
    GithubError string           `json:"githubError,omitempty"`
}

type TeamApiMember struct {
    Name       string               `json:"name"`
    Github     string               `json:"github"`
    Role       string               `json:"role"`
    FilePath   string               `json:"filePath"`
    GithubData *TeamMemberGitHubData `json:"github_data"`
}

type TeamMemberGitHubData struct {
    AssignedIssues      []IssueRef `json:"assignedIssues"`
    AssignedIssuesCount int        `json:"assignedIssuesCount"`
    OpenPRs             []PRRef    `json:"openPRs"`
    OpenPRsCount        int        `json:"openPRsCount"`
    RecentActivity      struct {
        MergedPRs         []MergedPRRef   `json:"mergedPRs"`
        MergedPRsCount    int             `json:"mergedPRsCount"`
        ClosedIssues      []ClosedIssueRef `json:"closedIssues"`
        ClosedIssuesCount int              `json:"closedIssuesCount"`
    } `json:"recentActivity"`
    ReviewBlocking []ReviewBlockingEntry `json:"reviewBlocking"`
}

type ReviewBlockingEntry struct {
    Direction  string `json:"direction"` // "authored" | "reviewing"
    OtherName  string `json:"otherName"`
    OtherGithub string `json:"otherGithub"`
    PR         struct {
        Number    int    `json:"number"`
        Title     string `json:"title"`
        URL       string `json:"url"`
        CreatedAt string `json:"createdAt"`
    } `json:"pr"`
}
```

### Other Types

```go
type TowerVersionInfo struct {
    Version   string `json:"version"`
    StartedAt string `json:"startedAt"`
}

type TunnelStatus struct {
    Registered bool    `json:"registered"`
    State      string  `json:"state"` // disconnected|connecting|connected|auth_failed|error
    Uptime     *int64  `json:"uptime"`
    TowerID    *string `json:"towerId"`
    TowerName  *string `json:"towerName"`
    ServerURL  *string `json:"serverUrl"`
    AccessURL  *string `json:"accessUrl"`
}

type IssueSearchItem struct {
    ID        string   `json:"id"`
    Title     string   `json:"title"`
    URL       string   `json:"url"`
    Area      string   `json:"area"`
    Author    string   `json:"author,omitempty"`
    Assignees []string `json:"assignees,omitempty"`
    CreatedAt string   `json:"createdAt"`
    Body      string   `json:"body"`
}

type IssueSearchResponse struct {
    Items       []IssueSearchItem `json:"items"`
    CurrentUser string            `json:"currentUser,omitempty"`
    Error       string            `json:"error,omitempty"`
}

// SSE
type SSEEvent struct {
    Type  string `json:"type"`
    Title string `json:"title,omitempty"`
    Body  string `json:"body,omitempty"`
}

type BuilderSpawnedPayload struct {
    TerminalID    string `json:"terminalId"`
    RoleID        string `json:"roleId"`
    WorkspacePath string `json:"workspacePath"`
}

// Send
type SendRequest struct {
    To            string       `json:"to"`
    Message       string       `json:"message"`
    From          string       `json:"from,omitempty"`
    Workspace     string       `json:"workspace,omitempty"`
    FromWorkspace string       `json:"fromWorkspace,omitempty"`
    Options       *SendOptions `json:"options,omitempty"`
}

type SendOptions struct {
    Raw       bool `json:"raw,omitempty"`
    NoEnter   bool `json:"noEnter,omitempty"`
    Interrupt bool `json:"interrupt,omitempty"`
}

type SendResponse struct {
    OK         bool   `json:"ok"`
    TerminalID string `json:"terminalId,omitempty"`
    ResolvedTo string `json:"resolvedTo,omitempty"`
    Deferred   bool   `json:"deferred,omitempty"`
    Error      string `json:"error,omitempty"`
    Message    string `json:"message,omitempty"`
}

// Command relay
type CommandRequest struct {
    Verb      string        `json:"verb"`
    Args      []interface{} `json:"args,omitempty"`
    Workspace string        `json:"workspace,omitempty"`
}

type CommandResult struct {
    OK    bool   `json:"ok"`
    Error string `json:"error,omitempty"`
}

// Workspace list
type WorkspaceInfo struct {
    Path      string `json:"path"`
    Name      string `json:"name"`
    Active    bool   `json:"active"`
    ProxyURL  string `json:"proxyUrl,omitempty"`
    Terminals int    `json:"terminals"`
}

type WorkspacesResponse struct {
    Workspaces []WorkspaceInfo `json:"workspaces"`
}

// Health
type HealthResponse struct {
    Status           string  `json:"status"`
    Ready            bool    `json:"ready"`
    Uptime           float64 `json:"uptime"`
    ActiveWorkspaces int     `json:"activeWorkspaces"`
    TotalWorkspaces  int     `json:"totalWorkspaces"`
    MemoryUsage      int64   `json:"memoryUsage"`
    Timestamp        string  `json:"timestamp"`
}
```

---

## 9. Data Flow Patterns

### Polling intervals

| Data | Interval | Endpoint | Trigger |
|---|---|---|---|
| Dashboard state (terminals) | 1s | `GET .../api/state` | Poll + SSE |
| Overview (builders, PRs, backlog) | 2.5s | `GET /api/overview` | Poll + SSE |
| Analytics | On demand | `GET /api/analytics` | Tab focus + range change |
| Team | On demand | `GET .../api/team` | Tab focus |

### SSE-triggered re-fetch

On any SSE message, immediately re-fetch relevant data rather than waiting for the next poll interval. This provides near-instant updates when builders spawn, gates change, or PRs are created.

### Tab lifecycle

1. **Create**: API call (e.g., `POST tabs/shell`) → response includes `id` and `terminalId`
2. **Focus**: Auto-focus newly created tabs. Use deep links via URL params (`?tab=builder:42`).
3. **Render**: Terminal tabs connect via WebSocket. File tabs load content via GET.
4. **Close**: `DELETE tabs/:id` removes from server registry.

---

## 10. Error Response Patterns

### Standard JSON error

```json
{ "error": "ERROR_CODE", "message": "Human-readable description" }
```

### Success flag pattern (some endpoints)

```json
{ "success": false, "error": "description" }
{ "success": true, ... }
```

### Error codes

| Code | Meaning |
|---|---|
| `INVALID_PARAMS` | Body/query validation failed |
| `NOT_FOUND` | Resource doesn't exist |
| `AMBIGUOUS` | Multiple matches, needs `?workspace=` to disambiguate |
| `INTERNAL_ERROR` | Server-side exception |
| `EXECUTION_FAILED` | Cron task execution failed |

### HTTP status codes

| Status | Meaning |
|---|---|
| `200` | Success |
| `201` | Created (terminal create) |
| `204` | No content (delete success) |
| `400` | Bad request |
| `403` | Forbidden (host/origin, rename builder) |
| `404` | Not found |
| `405` | Method not allowed |
| `409` | Conflict (ambiguous target) |
| `413` | Payload too large |
| `429` | Rate limited |
| `500` | Internal server error |

---

## 11. Implementation Checklist

### Minimum viable UI

- [ ] HTTP client with JSON parsing and Base64URL workspace encoding
- [ ] `GET /api/workspaces` → workspace picker
- [ ] `GET /workspace/:path/api/state` → terminal listing (architects, builders, shells)
- [ ] `GET /api/overview` → builders table, PRs, backlog
- [ ] SSE connection to `/api/events` with reconnection
- [ ] `POST /api/send` → inter-agent messaging
- [ ] Terminal attachment (WebSocket binary protocol or delegate to external terminal)

### Full-featured UI (additional)

- [ ] `GET /api/analytics` → metrics dashboard
- [ ] `GET .../api/team` → team collaboration view
- [ ] `GET .../api/files` + `api/git/status` → file browser with git status
- [ ] File content viewing (`GET .../api/file/:id`)
- [ ] Terminal create/resize/rename/delete lifecycle
- [ ] Cron task management
- [ ] Tunnel status and control
- [ ] Command relay integration
