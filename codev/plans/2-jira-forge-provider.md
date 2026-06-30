# Plan: Jira Forge Provider

## Metadata
- **ID**: 2-jira-forge-provider
- **Status**: draft
- **Created**: 2026-06-28
- **Spec**: `codev/specs/2-jira-forge-provider.md`
- **GitHub Issue**: [#2](https://github.com/amustafa/codev/issues/2)

## Overview

Add Jira Cloud as a forge provider preset, following the established pattern from Linear/GitLab/Gitea. Three work streams:

1. **Jira concept scripts** — 7 shell scripts in `packages/codev/scripts/forge/jira/`
2. **Provider registration** — One-line addition in `forge.ts`
3. **Prompt language fixes** — Replace GitHub-specific references with forge-agnostic language

## Phase 1: Jira Concept Scripts

Create `packages/codev/scripts/forge/jira/` with scripts matching the Linear provider pattern (curl + jq against Jira Cloud REST API v3).

Authentication via environment variables:
- `JIRA_BASE_URL` — e.g. `https://team.atlassian.net`
- `JIRA_USER_EMAIL` — Atlassian account email
- `JIRA_API_TOKEN` — API token from id.atlassian.com

Scripts to create:
- `issue-view.sh` — GET `/rest/api/3/issue/{key}`, map to IssueViewResult
- `issue-list.sh` — GET `/rest/api/3/search?jql=...`, map to IssueListItem[]
- `issue-search.sh` — Same endpoint, with text search JQL
- `issue-comment.sh` — POST `/rest/api/3/issue/{key}/comment`
- `recently-closed.sh` — JQL: `statusCategory = Done AND updated >= -7d`
- `auth-status.sh` — GET `/rest/api/3/myself`
- `user-identity.sh` — GET `/rest/api/3/myself`, extract displayName

Key decision: Jira descriptions use ADF (Atlassian Document Format). The `issue-view` script will extract text content from ADF using jq (traverse content nodes, concatenate text). This produces readable plaintext that LLMs handle well.

## Phase 2: Provider Registration

In `packages/codev/src/lib/forge.ts`, add `jira` to `getProviderPresets()`:
```
jira: buildPresetFromScripts('jira', ['team-activity', 'on-it-timestamps'])
```

Only `team-activity` and `on-it-timestamps` are explicitly disabled (they'd need Jira-specific batch implementations). PR concepts and other unsupported concepts are simply absent from the preset (no Jira scripts exist for them), so they fall through to GitHub defaults automatically. This is the correct behavior: `null` in a preset means "disabled," while absent means "use the default."

## Phase 3: Prompt Language

Fix GitHub-specific references in builder prompts and spawn code:

**spawn.ts line 825**: `"work for GitHub Issue #${issueNumber}"` → `"work for issue ${issueNumber}"`

**Builder prompts** (bugfix, air, pir, spir, aspir):
- `## Issue #{{issue.number}}` — keep as-is (the `#` prefix is conventional, not GitHub-specific)
- `"Fixes #{{issue.number}}"` in PR body → `"Fixes {{issue.number}}"` (drop the `#` — Jira keys like PROJ-123 don't use it)
- Message templates in notifications — use `{{issue.number}}` without `#` prefix

**experiment builder-prompt.md**: The PR body instructions mentioning GitHub auto-close — make conditional or generic.

## Files Modified

| File | Change |
|------|--------|
| `packages/codev/scripts/forge/jira/issue-view.sh` | New |
| `packages/codev/scripts/forge/jira/issue-list.sh` | New |
| `packages/codev/scripts/forge/jira/issue-search.sh` | New |
| `packages/codev/scripts/forge/jira/issue-comment.sh` | New |
| `packages/codev/scripts/forge/jira/recently-closed.sh` | New |
| `packages/codev/scripts/forge/jira/auth-status.sh` | New |
| `packages/codev/scripts/forge/jira/user-identity.sh` | New |
| `packages/codev/src/lib/forge.ts` | Add jira preset |
| `packages/codev/src/agent-farm/commands/spawn.ts` | Fix "GitHub Issue" string |
| `codev-skeleton/protocols/bugfix/builder-prompt.md` | Fix prompt language |
| `codev-skeleton/protocols/air/builder-prompt.md` | Fix prompt language |
| `codev-skeleton/protocols/pir/builder-prompt.md` | Fix prompt language |
| `codev-skeleton/protocols/experiment/builder-prompt.md` | Fix prompt language |
