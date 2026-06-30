package api

// Go struct equivalents of @cluesmith/codev-types (packages/types/src/api.ts).
// JSON tags use camelCase to match the Tower wire format.

// --- Health ---

type HealthResponse struct {
	Status           string  `json:"status"`
	Ready            bool    `json:"ready"`
	Uptime           float64 `json:"uptime"`
	ActiveWorkspaces int     `json:"activeWorkspaces"`
	TotalWorkspaces  int     `json:"totalWorkspaces"`
	MemoryUsage      int64   `json:"memoryUsage"`
	Timestamp        string  `json:"timestamp"`
}

// --- Version ---

type VersionInfo struct {
	Version   string `json:"version"`
	StartedAt string `json:"startedAt"`
}

// --- Workspaces ---

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

// --- Dashboard State ---

type ArchitectState struct {
	Name       string `json:"name"`
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

type DashboardState struct {
	Architect     *ArchitectState  `json:"architect"`
	Architects    []ArchitectState `json:"architects"`
	Builders      []Builder        `json:"builders"`
	Utils         []UtilTerminal   `json:"utils"`
	Annotations   []Annotation     `json:"annotations"`
	WorkspaceName string           `json:"workspaceName,omitempty"`
	Version       string           `json:"version,omitempty"`
	Hostname      string           `json:"hostname,omitempty"`
	TeamEnabled   bool             `json:"teamEnabled,omitempty"`
}

// --- Overview ---

type PlanPhase struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type OverviewBuilder struct {
	ID                 string            `json:"id"`
	IssueID            *string           `json:"issueId"`
	IssueTitle         *string           `json:"issueTitle"`
	Phase              string            `json:"phase"`
	ProtocolPhase      string            `json:"protocolPhase"`
	Mode               string            `json:"mode"`
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
}

type OverviewRecentlyClosed struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Type     string `json:"type"`
	ClosedAt string `json:"closedAt"`
	PRUrl    string `json:"prUrl,omitempty"`
}

type OverviewErrors struct {
	PRs    string `json:"prs,omitempty"`
	Issues string `json:"issues,omitempty"`
}

type OverviewData struct {
	Builders       []OverviewBuilder       `json:"builders"`
	PendingPRs     []OverviewPR            `json:"pendingPRs"`
	Backlog        []OverviewBacklogItem   `json:"backlog"`
	RecentlyClosed []OverviewRecentlyClosed `json:"recentlyClosed"`
	Architects     []ArchitectState        `json:"architects"`
	CurrentUser    string                  `json:"currentUser,omitempty"`
	Errors         *OverviewErrors         `json:"errors,omitempty"`
}

// --- Send ---

type SendOptions struct {
	Raw       bool `json:"raw,omitempty"`
	NoEnter   bool `json:"noEnter,omitempty"`
	Interrupt bool `json:"interrupt,omitempty"`
}

type SendRequest struct {
	To            string       `json:"to"`
	Message       string       `json:"message"`
	From          string       `json:"from,omitempty"`
	Workspace     string       `json:"workspace,omitempty"`
	FromWorkspace string       `json:"fromWorkspace,omitempty"`
	Options       *SendOptions `json:"options,omitempty"`
}

type SendResponse struct {
	OK         bool   `json:"ok"`
	TerminalID string `json:"terminalId,omitempty"`
	ResolvedTo string `json:"resolvedTo,omitempty"`
	Deferred   bool   `json:"deferred,omitempty"`
	Error      string `json:"error,omitempty"`
	Message    string `json:"message,omitempty"`
}

// --- Shell Tab ---

type ShellTabResponse struct {
	ID         string `json:"id"`
	Port       int    `json:"port"`
	Name       string `json:"name"`
	TerminalID string `json:"terminalId,omitempty"`
	Persistent bool   `json:"persistent,omitempty"`
}

// --- SSE ---

type SSEEvent struct {
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Body      string `json:"body,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}
