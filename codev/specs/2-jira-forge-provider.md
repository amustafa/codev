# Specification: Jira Forge Provider

<!--
SPEC vs PLAN BOUNDARY:
This spec defines WHAT and WHY. The plan defines HOW and WHEN.

DO NOT include in this spec:
- Implementation phases or steps
- File paths to modify
- Code examples or pseudocode
- "First we will... then we will..."

These belong in codev/plans/2-jira-forge-provider.md
-->

## Metadata
- **ID**: 2-jira-forge-provider
- **Status**: draft
- **Created**: 2026-06-28
- **GitHub Issue**: [#2](https://github.com/amustafa/codev/issues/2)

## Problem Statement

Teams using Jira as their project tracker cannot use Codev's issue-driven protocols (BUGFIX, AIR, PIR) without manually mirroring work items into GitHub Issues. The forge abstraction (Spec 589) already supports pluggable providers — GitHub, GitLab, Gitea, and Linear all have provider presets — but Jira is missing despite being the most widely-used issue tracker in enterprise engineering teams.

The forge infrastructure is already in place: concept commands, provider presets, opaque string IDs, JSON output contracts. Adding Jira is not an architectural change — it's a new provider preset following established patterns.

## Current State

- **Forge abstraction is implemented**: `forge.ts` dispatches concept commands through a resolution chain (manual override → provider preset → GitHub default). Four provider presets exist.
- **Alphanumeric issue IDs already work**: The CLI accepts `PROJ-123` format via `/^[A-Z]+-\d+$/i` regex. The spawn flow passes IDs as opaque strings.
- **Concept command contracts are defined**: `forge-contracts.ts` defines `IssueViewResult`, `IssueListItem`, `PrListResult`, etc. All contracts treat issue numbers as `number | string`.
- **Linear provider is the closest precedent**: It uses GraphQL against an external API (not a local CLI tool), authenticates via API key in an environment variable, and implements a subset of concepts.
- **No Jira provider preset exists**: No `scripts/forge/jira/` directory, no `jira` entry in `getProviderPresets()`.

## Desired State

- A `jira` provider preset ships with Codev, following the same pattern as `linear`
- Teams configure `forge.provider: "jira"` in `.codev/config.json` and set authentication credentials
- `afx spawn PROJ-123 --protocol bugfix` fetches the Jira issue, injects its title/description into the builder prompt, and proceeds normally
- Issue comments ("On it!", PR links) are posted back to Jira
- Concepts that have no Jira equivalent (PR operations) are explicitly disabled — Codev falls through to the VCS provider (GitHub/GitLab) for PR concepts
- Builder prompts use forge-agnostic language (no "GitHub Issue #N", no "Fixes #N")

## Stakeholders

- **Primary**: Engineering teams using Jira Cloud for project tracking alongside GitHub/GitLab for source control
- **Secondary**: Codev maintainers (ongoing maintenance of the provider preset)
- **Out of scope**: Jira Server/Data Center users (API differences are significant; Cloud-first, with Server noted as a future extension)

## Scope

### In Scope

**1. Jira issue concepts** — Shell scripts implementing forge concepts against the Jira Cloud REST API:
  - `issue-view` — Fetch issue by key (title, description, status, comments)
  - `issue-list` — List open issues in a project (for backlog/overview)
  - `issue-search` — Search issues by text (for backlog matching)
  - `issue-comment` — Post a comment on an issue ("On it!", PR links)
  - `recently-closed` — List recently resolved/closed issues
  - `auth-status` — Validate Jira credentials

**2. Authentication** — Jira Cloud requires two credentials:
  - `JIRA_BASE_URL` — The Atlassian instance URL (e.g. `https://team.atlassian.net`)
  - `JIRA_API_TOKEN` — API token (generated at https://id.atlassian.com/manage-profile/security/api-tokens)
  - `JIRA_USER_EMAIL` — The user's Atlassian email (used with API token for Basic Auth)

**3. Provider preset registration** — Add `jira` to `getProviderPresets()` with PR concepts explicitly disabled (Jira is an issue tracker, not a VCS).

**4. Forge-agnostic prompt language** — Replace GitHub-specific references in builder prompts and spawn code:
  - `"work for GitHub Issue #N"` → `"work for issue N"` (or use forge-derived title)
  - `"Fixes #N"` in PR body instructions → provider-aware closing-keyword syntax
  - `#{{issue.number}}` display format → configurable or neutral

**5. Configuration documentation** — How to set up the Jira provider in `.codev/config.json` and environment variables.

### Out of Scope

- **PR concepts for Jira**: Jira is not a VCS — PR operations (`pr-list`, `pr-exists`, `pr-merge`, etc.) remain delegated to the VCS forge (GitHub/GitLab). This spec does not introduce multi-provider composition (issue tracker + VCS provider); the `forge.provider` setting controls all concepts, with unsupported concepts set to `null` and falling through to defaults.
- **Jira Server/Data Center**: API differences (REST API v2 vs v3, authentication schemes) make Server support a separate effort. The scripts should be structured so a Server variant is feasible later.
- **Jira workflow transitions**: Automatically transitioning Jira issues through workflow states (e.g. "To Do" → "In Progress" → "Done") is desirable but complex — Jira workflows are project-specific and customizable. Deferred to a follow-up.
- **Sprint/epic/board management**: Codev does not manage Jira sprint planning, epic hierarchies, or board configurations.
- **Bidirectional sync**: No syncing between Jira issues and GitHub Issues.
- **Custom field mapping**: Jira custom fields are project-specific; mapping them into Codev's label/priority model is out of scope.

## Design Decisions

### Issue-only provider with VCS fallthrough

Jira is an issue tracker, not a code host. When `forge.provider: "jira"` is set, the Jira preset supplies issue concepts and explicitly sets PR concepts to `null`. Since the forge resolution chain falls through `null` preset entries to the default GitHub commands, PR operations continue to work via `gh` with no additional configuration.

This means a Jira-using team with GitHub repos needs only:
```json
{
  "forge": {
    "provider": "jira"
  }
}
```

Issue operations route to Jira. PR operations fall through to GitHub defaults. No multi-provider composition required.

**Trade-off**: Teams using Jira + GitLab (not GitHub) would need to explicitly override PR concepts to use GitLab scripts. This is already how non-GitHub VCS works today and is not a new limitation.

### API token authentication (not OAuth)

Jira Cloud supports both OAuth 2.0 (3LO) and API tokens with Basic Auth. API tokens are chosen because:
- They follow the same pattern as the Linear provider (`LINEAR_API_KEY`)
- They don't require a redirect URI or OAuth app registration
- They work in CI/headless environments
- They are simpler to document and configure

OAuth can be added later as an alternative authentication method without changing the concept scripts' interface.

### curl-based scripts (not Jira CLI)

Unlike GitHub (`gh`) and GitLab (`glab`), Jira has no widely-adopted official CLI. The Atlassian `atlas` CLI exists but is not focused on issue operations. Using `curl` directly against the Jira Cloud REST API v3:
- Avoids requiring users to install an additional CLI tool
- `curl` is universally available
- Follows the same pattern as the Linear provider
- Makes the API contract explicit and inspectable

### Jira project scoping

Jira issues are scoped to projects (e.g. `PROJ-123`). The issue key itself contains the project prefix, so Codev does not need a separate project configuration. The scripts extract the project key from the issue key when needed for list/search operations.

For `issue-list` and `issue-search` (which need a project scope), the environment variable `CODEV_JIRA_PROJECT` is set by the scripts from the most recently viewed issue key's prefix, or can be configured explicitly in `.codev/config.json`:

```json
{
  "forge": {
    "provider": "jira",
    "jira-project": "PROJ"
  }
}
```

## Success Criteria

- [ ] `afx spawn PROJ-123 --protocol bugfix` fetches the Jira issue and spawns a builder with correct context
- [ ] `afx spawn PROJ-123 --protocol air` works identically
- [ ] `afx spawn PROJ-123 --protocol pir` works identically
- [ ] `afx spawn PROJ-123 --protocol spir` works (issue context enriches spec phase)
- [ ] Issue comments ("On it!" and PR links) are posted to the Jira issue
- [ ] `codev doctor` validates Jira credentials when `forge.provider: "jira"` is configured
- [ ] Builder prompts contain no GitHub-specific language when using the Jira provider
- [ ] PR operations (list, exists, merge, diff) continue working via GitHub defaults when only `forge.provider: "jira"` is set
- [ ] Existing GitHub-only projects see zero behavior change (no regressions)
- [ ] Linear provider continues working (no regressions to existing non-GitHub providers)
- [ ] The Jira concept scripts conform to the existing JSON output contracts in `forge-contracts.ts`

## Constraints

### Technical

- Jira Cloud REST API v3 is the target; v2 differences are not handled
- Scripts must work with `curl` and `jq` (no additional dependencies)
- Authentication uses Basic Auth with API token (same trust model as `gh auth` / `LINEAR_API_KEY`)
- Issue IDs are always the full Jira key (e.g. `PROJ-123`), never just the number
- The `forge.provider` setting remains single-valued — no multi-provider composition in this spec

### Business

- Non-breaking: GitHub-only projects require zero configuration changes
- The Jira provider is a best-effort preset (same caveat as GitLab/Gitea/Linear): output schemas should match contracts but consumers handle `null` gracefully

## Assumptions

- Users have `curl` and `jq` installed (standard on macOS/Linux; available on Windows via Git Bash)
- Users can generate Jira API tokens from their Atlassian account settings
- Jira Cloud REST API v3 endpoints are stable and available
- The `issue-view` response from Jira can be mapped to the `IssueViewResult` contract (title, body → description, state → status name, comments)
- Teams using Jira typically also use GitHub or GitLab for source control (Jira is the issue tracker, not the VCS)

## Forge Concepts: Jira Coverage

| Concept | Jira Support | Notes |
|---------|-------------|-------|
| `issue-view` | Yes | `GET /rest/api/3/issue/{key}` — map to `IssueViewResult` |
| `issue-list` | Yes | `GET /rest/api/3/search?jql=project=PROJ AND status!=Done` |
| `issue-search` | Yes | JQL-based search: `text ~ "query"` |
| `issue-comment` | Yes | `POST /rest/api/3/issue/{key}/comment` |
| `recently-closed` | Yes | JQL: `status changed to Done AFTER -7d` |
| `auth-status` | Yes | `GET /rest/api/3/myself` — validates credentials |
| `user-identity` | Yes | Same as `auth-status` — extract email/displayName |
| `pr-list` | **null** | Jira is not a VCS — falls through to default |
| `pr-exists` | **null** | Falls through to default |
| `pr-merge` | **null** | Falls through to default |
| `pr-view` | **null** | Falls through to default |
| `pr-diff` | **null** | Falls through to default |
| `pr-search` | **null** | Falls through to default |
| `recently-merged` | **null** | Falls through to default |
| `on-it-timestamps` | **null** | Could be derived from issue comments; deferred |
| `team-activity` | **null** | Jira has activity APIs but different shape; deferred |
| `repo-archive` | **null** | Not applicable |

## Security Considerations

- Jira API tokens grant access to the user's Jira instance — same credential trust model as `gh auth login` or `LINEAR_API_KEY`
- Tokens are stored in environment variables, not in `.codev/config.json` (config is committed to git; env vars are not)
- The `CODEV_COMMENT_BODY` env var passed to `issue-comment` could contain shell metacharacters — same risk as all other forge providers, same mitigation (proper quoting in the script)
- Jira REST API responses may contain sensitive data (user emails, issue descriptions) — concept scripts output only the contracted fields

## Test Scenarios

1. **Jira provider, valid credentials**: `issue-view` returns correct `IssueViewResult` JSON for a known issue key
2. **Jira provider, invalid credentials**: `auth-status` exits non-zero; `issue-view` exits non-zero; spawn fails with helpful error
3. **Jira provider, non-existent issue**: `issue-view` exits non-zero; spawn fails with "issue not found"
4. **Jira provider, PR fallthrough**: With only `forge.provider: "jira"`, `pr-exists` falls through to GitHub default and works
5. **Jira provider, comment posting**: `issue-comment` posts an "On it!" comment visible in Jira
6. **Mixed config (Jira + GitLab overrides)**: `forge.provider: "jira"` with explicit `pr-list` override pointing to GitLab script
7. **Prompt language**: Builder prompts contain no "GitHub Issue" or "Fixes #N" when using Jira provider
8. **Regression**: GitHub-only project (no forge config) behaves identically to before

## Dependencies

- **Spec 589** (Non-GitHub Repository Support): The forge abstraction this spec builds on. Spec 589 is implemented — this spec adds a provider, not infrastructure.
- **No new runtime dependencies**: `curl` and `jq` are the only external tools required (same as Linear provider).

## Risks and Mitigation

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Jira REST API v3 response shape differs from `IssueViewResult` contract | Low | Medium | Map fields explicitly in the script (title→title, description→body, status.name→state); handle ADF→text conversion for description |
| Jira description uses Atlassian Document Format (ADF), not plain text/markdown | High | Medium | Convert ADF to markdown in the script (jq can extract text nodes) or fall back to rendered text endpoint |
| Teams have complex Jira workflows where "closed" is ambiguous | Medium | Low | Use JQL `statusCategory = Done` instead of specific status names |
| `curl`/`jq` not available on some systems | Low | Low | `codev doctor` checks; document prerequisites |
| Jira rate limiting on high-frequency operations | Low | Low | Concept commands are invoked infrequently (spawn, overview refresh); no batching needed |

## Open Questions

### Critical

- [x] Should Jira be a separate "backlog provider" abstraction or a forge provider preset? **Decision: Forge provider preset.** The infrastructure exists and handles the issue-only-with-VCS-fallthrough pattern cleanly.

### Important

- [ ] How should Jira's ADF (Atlassian Document Format) descriptions be converted for builder consumption? Options: (a) extract plain text nodes via `jq`, (b) use Jira's rendered content API endpoint, (c) pass raw ADF and let the LLM parse it. The LLM handles ADF reasonably well, but plain text/markdown is cleaner.
- [ ] Should `forge.provider` support a composite value (e.g. `"jira+github"`) or should mixed setups use per-concept overrides? **Leaning toward**: Per-concept overrides (simpler, already works). Composite providers add complexity for an edge case.

### Nice-to-Know

- [ ] Should Codev ship a `codev doctor` check that validates Jira API token scopes (read vs write)?
- [ ] Should the Jira provider preset include `on-it-timestamps` by mapping from issue comment history? (Lower priority, analytics feature.)

## References

- Spec 589: Non-GitHub Repository Support (`codev/specs/589-non-github-repository-support.md`)
- Jira Cloud REST API v3: https://developer.atlassian.com/cloud/jira/platform/rest/v3/
- Existing Linear provider: `packages/codev/scripts/forge/linear/`
- Forge contracts: `packages/codev/src/lib/forge-contracts.ts`
- Forge dispatcher: `packages/codev/src/lib/forge.ts`
