package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	authToken  string
}

func NewClient(baseURL string) *Client {
	token := readAuthToken()
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		authToken:  token,
	}
}

func readAuthToken() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".agent-farm", "local-key"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func EncodeWorkspacePath(absPath string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(absPath))
}

func (c *Client) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.authToken != "" {
		req.Header.Set("codev-web-key", c.authToken)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) post(path string, payload interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest("POST", c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("codev-web-key", c.authToken)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) Health() (*HealthResponse, error) {
	data, err := c.get("/health")
	if err != nil {
		return nil, err
	}
	var resp HealthResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Version() (*VersionInfo, error) {
	data, err := c.get("/api/version")
	if err != nil {
		return nil, err
	}
	var resp VersionInfo
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Workspaces() ([]WorkspaceInfo, error) {
	data, err := c.get("/api/workspaces")
	if err != nil {
		return nil, err
	}
	var resp WorkspacesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Workspaces, nil
}

func (c *Client) Overview(workspace string) (*OverviewData, error) {
	path := "/api/overview"
	if workspace != "" {
		path += "?" + url.Values{"workspace": {workspace}}.Encode()
	}
	data, err := c.get(path)
	if err != nil {
		return nil, err
	}
	var resp OverviewData
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) WorkspaceState(workspace string) (*DashboardState, error) {
	encoded := EncodeWorkspacePath(workspace)
	data, err := c.get("/workspace/" + encoded + "/api/state")
	if err != nil {
		return nil, err
	}
	var resp DashboardState
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Send(req SendRequest) (*SendResponse, error) {
	data, err := c.post("/api/send", req)
	if err != nil {
		return nil, err
	}
	var resp SendResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) RefreshOverview(workspace string) error {
	if workspace != "" {
		encoded := EncodeWorkspacePath(workspace)
		_, err := c.post("/workspace/"+encoded+"/api/overview/refresh", nil)
		return err
	}
	_, err := c.post("/api/overview/refresh", nil)
	return err
}

func (c *Client) CreateShellTab(workspace string) (*ShellTabResponse, error) {
	encoded := EncodeWorkspacePath(workspace)
	data, err := c.post("/workspace/"+encoded+"/api/tabs/shell", nil)
	if err != nil {
		return nil, err
	}
	var resp ShellTabResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SSEEventsURL() string {
	return c.BaseURL + "/api/events"
}
