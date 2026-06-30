# Specification: Tower SSE Thundering Herd Fix

<!--
SPEC vs PLAN BOUNDARY:
This spec defines WHAT and WHY. The plan defines HOW and WHEN.

DO NOT include in this spec:
- Implementation phases or steps
- File paths to modify
- Code examples or pseudocode
- "First we will... then we will..."

These belong in codev/plans/3-tower-sse-thundering-herd.md
-->

## Metadata
- **ID**: 3-tower-sse-thundering-herd
- **Status**: draft
- **Created**: 2026-06-30
- **GitHub Issue**: [#3](https://github.com/amustafa/codev/issues/3)

## Problem Statement

The Tower server SSE endpoint (`GET /api/events`, port 4100) has a hard client cap of 50. When exceeded, the server evicts the oldest client to make room. The evicted client auto-reconnects, pushing the count back to the cap, evicting another client — creating a thundering herd loop at up to ~800 connections/second.

Each short-lived TCP connection leaves a socket in `TIME_WAIT` for ~60 seconds. At peak churn this accumulates 2,900+ `TIME_WAIT` sockets against `127.0.0.1:4100`, consuming ~10% of the ephemeral port range (32768–60999). This causes intermittent `EADDRNOTAVAIL` / connection-refused errors for **all** localhost services — not just Tower.

## Current State

- **SSE cap**: `SSE_MAX_CLIENTS = 50` in `tower-server.ts:325`
- **Eviction strategy**: When cap is reached, `sseClients.shift()` removes the oldest client and calls `res.end()`, forcing a disconnect
- **Client reconnection**: Browser `EventSource` reconnects ~3s after disconnect; VSCode reconnects via `backoffDelayMs` in `connection-manager.ts`
- **Connection identity**: Each new connection generates a random client ID — no session continuity
- **Logging**: Every SSE connect and disconnect is logged at INFO level, producing 774K log lines in 2.5 days
- **Max-age eviction**: A fixed 5-minute `SSE_MAX_AGE_MS` heartbeat evicts all connections older than 5 minutes simultaneously, causing synchronized reconnection bursts after Tower restarts

### Evidence

| Metric | Value |
|---|---|
| Peak connect rate | 49,524 connects/min (2026-06-30T12:17) |
| Unique client IDs in peak minute | 49,524 (new random ID per connection) |
| TIME_WAIT sockets at diagnosis | 2,888 against 127.0.0.1:4100 |
| Port utilization at peak | ~10% sustained, higher in bursts |
| Log file size (2.5 days) | 1.18M lines, 774K SSE connect/disconnect |

## Desired State

- SSE connections are stable — established connections are never evicted to make room for new ones
- New connections are rejected with `503 Retry-After` when at capacity, breaking the cascade chain
- The SSE cap is raised to accommodate realistic workstation usage (multiple builders + dashboard tabs + VSCode windows)
- Reconnection timing is staggered to prevent synchronized bursts
- SSE connect/disconnect logging is reduced to operational heartbeats only
- `TIME_WAIT` socket accumulation stays under 100 during normal operation

## Stakeholders

- **Primary**: Developers running Tower locally — the port exhaustion affects all localhost services, not just Tower
- **Secondary**: Codev maintainers (logging volume, support burden from mysterious connection failures)

## Scope

### In Scope

**1. Reject-instead-of-evict** — The critical fix. Change `addSseClient` from evicting the oldest client to returning `false` when at capacity. The route handler checks capacity **before** `res.writeHead()` and returns `503` with `Retry-After` header when full. Rejection is a dead end (client backs off independently), while eviction is a chain reaction.

**2. Raise SSE cap** — Increase `SSE_MAX_CLIENTS` from 50 to 200. SSE connections are lightweight (one long-lived HTTP response, one fd). The heartbeat already handles leaked connections. 50 is too low for workstations running multiple builders + dashboard tabs + VSCode windows.

**3. SSE `retry:` directive** — After the initial `connected` event, send `retry: 5000\n\n`. Browser `EventSource` clients honor this to space reconnection attempts. VSCode's fetch-based client ignores it (already has its own backoff).

**4. Throttle SSE logging** — Remove per-connection INFO-level connect/disconnect logging. The existing heartbeat logs `SSE heartbeat: N active client(s)` every 30s — that's sufficient for operational visibility.

**5. Stagger max-age eviction** — Add per-client jitter to the 5-minute `SSE_MAX_AGE_MS`. Store a randomized `maxAge` (4–6 min range) on each `SSEClient` object to prevent synchronized eviction bursts after Tower restarts.

### Out of Scope

- Client-side reconnection logic in VSCode extension (already has backoff)
- Connection deduplication (same browser tab reconnecting gets a new ID — fixing this is a larger change)
- HTTP/2 or WebSocket migration for the event stream
- Kernel tuning (`net.ipv4.ip_local_port_range`, `tcp_fin_timeout`) — these are workarounds, not fixes

## Design Decisions

### Reject > evict

Eviction creates a feedback loop: evict → reconnect → evict → reconnect. Rejection breaks the loop: the client sees 503 + `Retry-After`, backs off, and tries again later. The steady-state client set remains stable.

**Trade-off**: A legitimate new client that connects when at capacity will be temporarily rejected. This is acceptable because (a) the cap is being raised to 200, making this rare, and (b) the `Retry-After` header ensures the client retries rather than failing silently.

### 200 is the right cap

Back-of-envelope: 5 builder terminals × 2 SSE each (terminal + dashboard) + 3 dashboard browser tabs + 2 VSCode windows × 2 SSE each = ~19 connections per heavy session. 200 gives headroom for 10× that. The fd cost is negligible.

### Jitter range: 4–6 minutes

The current 5-minute fixed max-age means N clients that connect at the same time all expire at the same time. A uniform random distribution over [4, 6] minutes spreads the eviction window over 2 minutes, preventing synchronized bursts.

## Success Criteria

- [ ] No `TIME_WAIT` socket accumulation beyond 100 during normal operation
- [ ] New SSE connections receive `503 Retry-After` when at capacity (not eviction of existing connections)
- [ ] SSE client set is stable after initial connections are established (no churn visible in logs)
- [ ] Log file growth rate drops by >90% for SSE-related entries
- [ ] SSE max-age evictions are spread over a 2-minute window (not synchronized)
- [ ] Existing dashboard, VSCode, and terminal clients continue working with no behavior change
- [ ] The `retry: 5000` directive is sent to browser clients
- [ ] `SSE_MAX_CLIENTS` is 200

## Constraints

### Technical

- The fix must be backward-compatible — existing SSE clients must not need changes
- The `503` response must include `Retry-After` to prevent aggressive reconnection
- The heartbeat-based max-age eviction must remain (it cleans up leaked connections)
- Changes are confined to Tower server code — no client-side changes required

### Business

- Non-breaking: Normal workstation usage (< 200 SSE clients) sees zero behavior change
- The fix must eliminate the thundering herd under all observed conditions, not just reduce it

## Assumptions

- Browser `EventSource` clients honor the `Retry-After` header on 503 responses
- SSE connections are truly lightweight on Node.js (one fd + one HTTP response object per connection)
- The 200-client cap is sufficient for any reasonable single-workstation usage pattern
- Clients handle 503 gracefully (retry with backoff, not crash or show error UI)

## Security Considerations

- The SSE cap change does not introduce new attack surface — an attacker could already open 50 connections
- 200 connections from localhost is not a meaningful DoS vector
- No authentication changes — SSE endpoint remains localhost-only

## Test Scenarios

1. **Cap reached, new client**: Connect 200 SSE clients → 201st receives `503` with `Retry-After: 5`
2. **No eviction cascade**: Connect 200 clients, attempt 201st → existing 200 remain connected (no shift/disconnect)
3. **Retry-After honored**: Client receiving 503 retries after the specified delay
4. **Jittered max-age**: Start 10 clients simultaneously → they expire over a 2-minute window, not all at once
5. **Log volume**: Under normal operation, no per-connection connect/disconnect log entries (only heartbeat)
6. **TIME_WAIT recovery**: After fix, run normal workload for 10 minutes → `ss -tn state time-wait dst 127.0.0.1:4100 | wc -l` < 100
7. **Regression**: Dashboard, VSCode, and terminal SSE features work identically after the fix

## Dependencies

- No new dependencies — all changes are in existing Tower server code
- The `SSEClient` type in `tower-types.ts` needs a `maxAge` field added

## Risks and Mitigation

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Browser `EventSource` ignores `Retry-After` on 503 | Medium | Medium | Send `retry:` SSE directive as secondary signal; browser clients respect this even if they don't respect HTTP `Retry-After` |
| VSCode SSE client doesn't handle 503 gracefully | Low | Medium | VSCode already has `backoffDelayMs` — a failed connection triggers the existing backoff |
| 200 cap still too low for extreme multi-monitor setups | Very Low | Low | Cap is configurable if needed; 200 is 10× observed peak usage |
| Removing connect/disconnect logging loses debugging signal | Low | Low | Heartbeat log every 30s with active count is sufficient; add debug-level logging for connect/disconnect if needed |

## Code Locations

- **Cap + eviction logic**: `packages/codev/src/agent-farm/servers/tower-server.ts:321-334`
- **SSE handler**: `packages/codev/src/agent-farm/servers/tower-routes.ts:1176-1208`
- **RouteContext interface**: `packages/codev/src/agent-farm/servers/tower-routes.ts:134-148`
- **Age-based eviction (heartbeat)**: `packages/codev/src/agent-farm/servers/tower-server.ts:246-272`
- **SSEClient type**: `packages/codev/src/agent-farm/servers/tower-types.ts:48-53`

## References

- Issue: [#3](https://github.com/amustafa/codev/issues/3) — original bug report with full diagnosis
- Architecture: `codev/resources/arch.md` — Tower server section
