# Plan: Tower SSE Thundering Herd Fix

## Metadata
- **ID**: 3-tower-sse-thundering-herd
- **Status**: draft
- **Created**: 2026-06-30
- **Spec**: `codev/specs/3-tower-sse-thundering-herd.md`
- **GitHub Issue**: [#3](https://github.com/amustafa/codev/issues/3)

## Overview

Fix the SSE thundering herd by replacing eviction-on-cap with rejection-on-cap, raising the cap, adding reconnection jitter, and reducing log noise. Five changes, ordered by blast radius (smallest first):

1. **Throttle logging** — Remove per-connection log spam
2. **Add `retry:` directive** — Space out browser reconnections
3. **Add max-age jitter** — Prevent synchronized eviction bursts
4. **Raise cap** — 50 → 200
5. **Reject instead of evict** — The critical fix that breaks the cascade

## Phase 1: Throttle SSE Logging

**tower-routes.ts** — SSE connect handler (`handleSSEEvents`):
- Remove or downgrade the per-connection `SSE client connected` and `SSE client disconnected` log lines from INFO to DEBUG
- The existing heartbeat log (`SSE heartbeat: N active client(s)` every 30s) remains at INFO and provides sufficient operational visibility

This phase is risk-free and immediately reduces log file growth by ~90% for SSE-related entries.

## Phase 2: SSE `retry:` Directive

**tower-routes.ts** — After writing the initial `connected` SSE event, send:
```
retry: 5000\n\n
```

Browser `EventSource` clients honor this field and use it as the reconnection delay. VSCode's fetch-based SSE client ignores it (already has `backoffDelayMs`), so this is additive, not disruptive.

## Phase 3: Max-Age Jitter

**tower-types.ts** — Add an optional `maxAge: number` field to the `SSEClient` type.

**tower-server.ts** — In the heartbeat handler that performs age-based eviction:
- When creating a new `SSEClient`, assign `maxAge = SSE_MAX_AGE_MS + randomJitter(-60000, +60000)` (4–6 minute range)
- In the heartbeat sweep, compare each client's age against its individual `maxAge` instead of the fixed `SSE_MAX_AGE_MS`

This spreads eviction of simultaneously-connected clients over a 2-minute window.

## Phase 4: Raise Cap

**tower-server.ts** — Change `SSE_MAX_CLIENTS` from `50` to `200`.

SSE connections are lightweight — one long-lived HTTP response, one file descriptor. The heartbeat max-age eviction already handles leaked connections. 50 is too low for workstations running multiple builders + dashboard tabs + VSCode windows.

## Phase 5: Reject Instead of Evict (Critical)

**tower-server.ts** — `addSseClient`:
- Change return type from `void` to `boolean`
- Remove the `while (sseClients.length >= SSE_MAX_CLIENTS)` eviction loop (the `sseClients.shift()` + `res.end()` that causes the cascade)
- Return `false` when `sseClients.length >= SSE_MAX_CLIENTS`

**tower-routes.ts** — `handleSSEEvents`:
- Move the capacity check **before** `res.writeHead(200, ...)`
- When `addSseClient` returns `false`, respond with:
  - Status `503 Service Unavailable`
  - `Retry-After: 5` header
  - JSON body: `{"error": "SSE capacity reached", "retryAfter": 5}`
- Do not write SSE headers or call `res.writeHead(200)` — the response is a normal HTTP 503

**RouteContext interface** — Update `addSseClient` return type from `void` to `boolean`.

This is the critical fix. Rejection is a dead end (client backs off), while eviction is a chain reaction (each eviction causes a reconnection that causes another eviction).

## Files Modified

| File | Change |
|------|--------|
| `packages/codev/src/agent-farm/servers/tower-server.ts:321-334` | Remove eviction loop, return boolean, raise cap to 200 |
| `packages/codev/src/agent-farm/servers/tower-server.ts:246-272` | Add per-client jitter to max-age eviction |
| `packages/codev/src/agent-farm/servers/tower-routes.ts:1176-1208` | Add 503 rejection, `retry:` directive, throttle logging |
| `packages/codev/src/agent-farm/servers/tower-routes.ts:134-148` | Update `RouteContext.addSseClient` return type |
| `packages/codev/src/agent-farm/servers/tower-types.ts:48-53` | Add `maxAge` field to `SSEClient` |

## Verification

```bash
# TIME_WAIT should stay under 100 after fix
watch -n 5 'ss -tn state time-wait dst 127.0.0.1:4100 | wc -l'

# SSE should show stable client set, not rapid churn
tail -f ~/.agent-farm/tower.log | grep -E 'SSE (client|cap|heartbeat)'

# Verify 503 at capacity
for i in $(seq 201); do curl -s -o /dev/null -w "%{http_code}" http://localhost:4100/api/events & done
# First 200 should return 200 (streaming), 201st should return 503
```

Regression checks:
- Dashboard SSE connection works normally
- VSCode SSE connection works normally
- Builder terminal events stream correctly
- `afx status` reflects correct builder state
- Heartbeat logs show stable client count (no churn)
