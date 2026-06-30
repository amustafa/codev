# Codev UI Behavior Specification

Platform-agnostic behavior checklist for any Codev Tower UI implementation.
Each scenario describes **what** the user must be able to do, not **how** the
UI presents it. Implementations choose their own interaction model (clicks,
keybindings, gestures) and layout.

Companion doc: `codev/resources/ui-development.md` (API reference).

---

## Feature: Workspace Management

```gherkin
Scenario: List available workspaces
  Given Tower is running
  When the user requests the workspace list
  Then all known workspaces are shown with name, path, and active/inactive status

Scenario: Select a workspace
  Given the workspace list is visible
  When the user selects a workspace
  Then the UI scopes all views to that workspace
  And the workspace name is displayed prominently

Scenario: Switch workspaces
  Given a workspace is already active
  When the user requests the workspace list
  Then they can select a different workspace
  And all views refresh to reflect the new workspace

Scenario: Filter workspaces
  Given the workspace list is visible
  And there are many workspaces
  When the user types a filter string
  Then only workspaces matching by name or path are shown

Scenario: Create a new workspace
  Given the user provides a parent directory and workspace name
  When they confirm creation
  Then the workspace is created on disk
  And it appears in the workspace list
```

---

## Feature: Architect Terminal

```gherkin
Scenario: Attach to live architect session
  Given the active workspace has an architect with a live terminal session
  When the user opens the architect terminal
  Then they see the actual Tower-managed PTY session
  And input they type is sent to that session
  And output from that session is displayed in real time

Scenario: Multiple architects
  Given the workspace has more than one registered architect
  When the user views architect terminals
  Then each architect is listed by name
  And the user can attach to any of them individually

Scenario: No architect session
  Given the active workspace has no live architect terminal
  When the user tries to open the architect terminal
  Then they see an appropriate empty state or message

Scenario: Remove a sibling architect
  Given the workspace has more than one architect
  And the target architect is not "main"
  When the user requests removal
  Then a confirmation is shown with the architect name
  And the count of in-flight builders spawned by that architect is displayed
  And on confirmation the architect is removed
```

---

## Feature: Builder Oversight

```gherkin
Scenario: View active builders
  Given the workspace has active builders
  When the user views the builders list
  Then each builder shows: ID, linked issue, protocol phase, protocol type, progress percentage, and area

Scenario: Identify blocked builders
  Given a builder is blocked at a gate
  When the user views the builders list
  Then the blocked builder is visually distinct from active builders
  And the blocking gate name is displayed
  And the duration of the block is shown

Scenario: Identify PR-ready builders
  Given a builder has completed review and its PR is ready for human review
  When the user views the builders list
  Then that builder is visually marked as PR-ready

Scenario: Identify idle builders
  Given a builder has not produced terminal output recently
  When the user views the builders list
  Then that builder is visually indicated as potentially waiting on input

Scenario: Attach to builder terminal
  Given the user selects an active builder
  When they open its terminal
  Then they see the actual Tower-managed PTY session for that builder

Scenario: Builder progress within plan phases
  Given a builder is executing a multi-phase plan
  When the user views that builder
  Then the current plan sub-phase is shown
  And a progress indicator reflects phase completion (e.g., "2/5")

Scenario: Builder architect attribution
  Given the workspace has multiple architects
  When the user views the builders list
  Then each builder shows which architect spawned it
```

---

## Feature: Terminal Management

```gherkin
Scenario: List all active terminal sessions
  Given the workspace has terminal sessions
  When the user views the terminal list
  Then all sessions are shown: architects, builders, and utility shells
  And each entry shows its type, name, and identifier

Scenario: Attach to any terminal session
  Given the terminal list is visible
  When the user selects a terminal
  Then they attach to that live PTY session

Scenario: Open a new shell
  Given a workspace is active
  When the user creates a new shell
  Then a new terminal session is created in that workspace
  And the user can immediately interact with it

Scenario: Shell activity indicator
  Given multiple shell sessions exist
  When the user views the terminal list
  Then recently active shells are visually distinct from idle ones
  And idle duration is shown for inactive shells
```

---

## Feature: Needs Attention

```gherkin
Scenario: Surface PRs awaiting human review
  Given builders have created PRs that are pending human review
  When the user views the needs-attention list
  Then those PRs are shown with: PR number, title, and waiting duration
  And the user can navigate to the PR

Scenario: Surface builders blocked at human gates
  Given builders are blocked at human-approval gates
  When the user views the needs-attention list
  Then blocked builders are shown with: builder ID, gate name, and block duration
  And the gate name shown is the canonical porch gate key (e.g., "plan-approval")

Scenario: Sort by waiting duration
  Given multiple items need attention
  When the user views the needs-attention list
  Then items are sorted oldest-waiting first

Scenario: Distinguish PR-ready from gate-blocked
  Given both PR-ready builders and gate-blocked builders exist
  When the user views the needs-attention list
  Then PR items and gate items are visually distinguishable
  And a builder blocked at the "pr" gate appears as a PR item, not a gate item
```

---

## Feature: Backlog

```gherkin
Scenario: View open issues
  Given the workspace has open GitHub issues
  When the user views the backlog
  Then issues are shown with: ID, title, type, priority, and area

Scenario: Group by area
  Given issues have area labels
  When the user views the backlog
  Then issues are grouped by their area label

Scenario: Show artifact status
  Given an issue has associated spec, plan, or review documents
  When the user views that issue in the backlog
  Then badges or indicators show which artifacts exist (spec, plan, review)
  And whether a builder is currently working on it

Scenario: Open artifact
  Given a backlog item has a spec, plan, or review artifact
  When the user opens that artifact
  Then the file content is displayed

Scenario: Navigate to issue
  Given a backlog item is linked to a GitHub issue
  When the user opens the issue link
  Then they are directed to the issue on GitHub

Scenario: Exclude items with active builders
  Given an issue has an active builder working on it
  When the user views the backlog
  Then that issue is not shown in the backlog list
```

---

## Feature: Recently Closed

```gherkin
Scenario: View recently closed items
  Given issues have been closed recently
  When the user views recently closed items
  Then each item shows: ID, title, type, closure time, and PR link if available

Scenario: Access closure artifacts
  Given a closed item has spec, plan, review, or PR artifacts
  When the user opens an artifact
  Then the file content or PR page is shown

Scenario: Hide when empty
  Given no issues have been closed recently
  Then the recently closed section is not displayed
```

---

## Feature: Inter-Agent Messaging

```gherkin
Scenario: Send message to architect
  Given a workspace is active
  When the user sends a message to "architect"
  Then the message is delivered to the architect terminal session

Scenario: Send message to specific builder
  Given a builder is active
  When the user sends a message to that builder by ID
  Then the message is delivered to that builder's terminal session

Scenario: Message delivery feedback
  Given the user sends a message
  When delivery succeeds
  Then the user sees confirmation with the resolved target
  When delivery fails
  Then the user sees an error with the reason

Scenario: Target addressing forms
  Given the messaging interface
  Then the user can address: a builder by ID, "architect", "architect:<name>", or "<workspace>:architect"
```

---

## Feature: Real-Time Updates

```gherkin
Scenario: Live data refresh
  Given the UI is connected to Tower's event stream
  When a builder spawns, a gate changes, or a PR is created
  Then the relevant views update automatically without manual refresh

Scenario: Connection status indicator
  Given the UI connects to Tower
  Then a visible indicator shows whether the connection is active or lost

Scenario: Reconnect after disconnect
  Given the event stream connection drops
  Then the UI automatically reconnects with backoff
  And data is refreshed on reconnection

Scenario: Degrade gracefully when Tower is unreachable
  Given Tower is not running or unreachable
  Then the UI shows a clear "disconnected" or "connecting" state
  And previously loaded data remains visible if available
```

---

## Feature: Overview Refresh

```gherkin
Scenario: Soft refresh
  Given data is displayed
  When the user triggers a soft refresh
  Then the latest data is fetched from Tower's cache

Scenario: Force refresh
  Given data is displayed and may be stale
  When the user triggers a force refresh
  Then Tower's overview cache is invalidated
  And fresh data is fetched from GitHub and filesystem
```

---

## Feature: Analytics

```gherkin
Scenario: View activity metrics
  Given the workspace has development history
  When the user views analytics
  Then they see: PRs merged count, issues closed count, median time to merge, and median time to close bugs

Scenario: View protocol breakdown
  Given projects have been run with different protocols
  When the user views analytics
  Then a breakdown by protocol shows: count, average wall clock time, and average agent time

Scenario: View consultation metrics
  Given AI consultations have been run
  When the user views analytics
  Then they see: total consultations, total cost, average latency, success rate, and per-model breakdown

Scenario: Select time range
  Given analytics are displayed
  When the user selects a time range (24h, 7d, 30d, or all)
  Then metrics recalculate for that range

Scenario: Refresh analytics
  Given analytics are displayed
  When the user triggers a refresh
  Then the latest metrics are fetched
```

---

## Feature: Team

```gherkin
Scenario: View team members
  Given the workspace has team configuration enabled
  When the user views the team
  Then each member shows: name, role, and GitHub handle

Scenario: View member workload
  Given team data has been fetched
  When the user views a team member
  Then they see: assigned issues, open PRs, and review-blocking relationships

Scenario: View review blocking
  Given a team member has review-blocking relationships
  When the user views that member
  Then blocking items show: direction (waiting for them / they're waiting), the other person, PR title, and waiting duration

Scenario: View recent team activity
  Given team members have recent GitHub activity
  When the user views team activity
  Then a chronological feed shows: who merged PRs and who closed issues, with timestamps

Scenario: View team messages
  Given team messages exist
  When the user views team messages
  Then messages are shown in chronological order with author and timestamp

Scenario: Hidden when disabled
  Given the workspace does not have team configuration
  Then the team view is not available
```

---

## Feature: File Browser

```gherkin
Scenario: Browse workspace files
  Given a workspace is active
  When the user opens the file browser
  Then a directory tree is shown with expandable folders

Scenario: Git status indicators
  Given the workspace is a git repository
  When the user views the file tree
  Then modified, staged, and untracked files have distinct visual indicators

Scenario: Search files
  Given the file tree is loaded
  When the user types a search query
  Then matching files are suggested
  And selecting a suggestion opens that file

Scenario: View recent files
  Given files have been opened previously
  When the user views recent files
  Then the most recently opened files are listed (up to 10)

Scenario: Open file from tree
  Given the file tree is visible
  When the user selects a file
  Then the file content is displayed
```

---

## Feature: File Viewing

```gherkin
Scenario: View text file
  Given the user opens a text file
  Then the content is displayed with syntax highlighting appropriate to the language

Scenario: View markdown
  Given the user opens a markdown file
  Then a rendered preview is available

Scenario: View image
  Given the user opens an image file
  Then the image is displayed

Scenario: View binary file
  Given the user opens a binary file (video, 3D model, PDF)
  Then an appropriate viewer is used or the file type and size are shown

Scenario: Save file
  Given the user is viewing a text file
  When they edit and save
  Then the changes are persisted to disk

Scenario: Navigate to line
  Given a file is opened with a line number reference (e.g., from a terminal click)
  Then the view scrolls to that line
```

---

## Feature: Cloud Tunnel

```gherkin
Scenario: View tunnel status
  Given the tunnel feature is configured
  When the user views the tunnel status
  Then the current state is shown: disconnected, connecting, connected, auth_failed, or error

Scenario: Connect tunnel
  Given the tunnel is disconnected
  When the user triggers connect
  Then the tunnel begins connecting
  And the status updates to reflect the connection progress

Scenario: Disconnect tunnel
  Given the tunnel is connected
  When the user triggers disconnect
  Then the tunnel disconnects
  And the status updates accordingly

Scenario: View access URL
  Given the tunnel is connected
  Then the public access URL is displayed
  And the user can navigate to it

Scenario: Show uptime
  Given the tunnel is connected
  Then the connection uptime is displayed

Scenario: Hidden when not configured
  Given the tunnel feature is not configured
  Then no tunnel status is shown
```

---

## Feature: Tab / View Navigation

```gherkin
Scenario: Switch between views
  Given multiple views are available (builders, backlog, terminals, analytics, etc.)
  When the user navigates between them
  Then the selected view is displayed
  And the previous view's state is preserved

Scenario: Close closable views
  Given a builder terminal, shell, or file tab is open
  When the user closes it
  Then the tab is removed from the UI
  And the underlying server resource is cleaned up

Scenario: Non-closable views
  Given core views like the work overview exist
  Then they cannot be closed by the user

Scenario: Persistent terminal state
  Given a terminal tab has been opened
  When the user navigates away and back
  Then the terminal session is still connected
  And output generated while away is visible

Scenario: Deep linking
  Given the UI supports URL-based navigation
  When a specific view or tab is encoded in the URL
  Then opening that URL navigates directly to that view
```

---

## Feature: Error Handling

```gherkin
Scenario: API error in a section
  Given a specific data fetch fails (e.g., PRs, issues, team)
  Then only the affected section shows an error message
  And other sections continue to function normally

Scenario: Network error
  Given Tower becomes unreachable
  Then the connection indicator reflects the disconnection
  And previously loaded data remains visible
  And the UI attempts to reconnect automatically

Scenario: Rate limiting
  Given the user triggers too many workspace activations
  Then the UI shows an appropriate rate-limit message
```
