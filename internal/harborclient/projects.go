package harborclient

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type ProjectMetadata struct {
	Public                   string `json:"public,omitempty"`
	EnableContentTrust       string `json:"enable_content_trust,omitempty"`
	EnableContentTrustCosign string `json:"enable_content_trust_cosign,omitempty"`
	PreventVul               string `json:"prevent_vul,omitempty"`
	Severity                 string `json:"severity,omitempty"`
	AutoScan                 string `json:"auto_scan,omitempty"`
	AutoSBOMGeneration       string `json:"auto_sbom_generation,omitempty"`
	ReuseSysCVEAllowlist     string `json:"reuse_sys_cve_allowlist,omitempty"`
	RetentionID              string `json:"retention_id,omitempty"`
	ProxySpeedKB             string `json:"proxy_speed_kb,omitempty"`
}

type CVEAllowlistItem struct {
	CveID string `json:"cve_id,omitempty"`
}

type CVEAllowlist struct {
	ID           int                `json:"id,omitempty"`
	ProjectID    int                `json:"project_id,omitempty"`
	ExpiresAt    int                `json:"expires_at,omitempty"`
	Items        []CVEAllowlistItem `json:"items,omitempty"`
	CreationTime string             `json:"creation_time,omitempty"`
	UpdateTime   string             `json:"update_time,omitempty"`
}

type Project struct {
	ProjectID  int             `json:"project_id"`
	Name       string          `json:"name"`
	RegistryID int             `json:"registry_id"`
	OwnerName  string          `json:"owner_name"`
	Metadata   ProjectMetadata `json:"metadata"`
	CVEAllowlist
}

type CreateProjectRequest struct {
	ProjectName  string          `json:"project_name,omitempty"`
	Public       bool            `json:"public,omitempty"`
	Owner        string          `json:"owner,omitempty"`
	Metadata     ProjectMetadata `json:"metadata"`
	CVEAllowlist CVEAllowlist    `json:"cve_allowlist"`
	StorageLimit *int            `json:"storage_limit,omitempty"`
	RegistryID   *int            `json:"registry_id,omitempty"`
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	return getPaged[Project](ctx, c, "/api/v2.0/projects", nil)
}

func (c *Client) FindProjectByName(ctx context.Context, name string) (*Project, error) {
	if name == "" {
		return nil, nil
	}
	values := url.Values{}
	values.Set("q", "name="+name)
	projects, err := getPaged[Project](ctx, c, "/api/v2.0/projects", values)
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if strings.EqualFold(projects[i].Name, name) {
			return &projects[i], nil
		}
	}
	return nil, nil
}

func (c *Client) GetProjectByID(ctx context.Context, id int) (*Project, error) {
	var p Project
	err := c.get(ctx, fmt.Sprintf("/api/v2.0/projects/%d", id), &p)
	return &p, err
}

func (c *Client) CreateProject(ctx context.Context, in CreateProjectRequest) (int, error) {
	return c.createWithNumericLocationID(ctx, "/api/v2.0/projects", &in)
}

func (c *Client) UpdateProject(ctx context.Context, id int, in CreateProjectRequest) error {
	return c.put(ctx, fmt.Sprintf("/api/v2.0/projects/%d", id), &in)
}

func (c *Client) DeleteProject(ctx context.Context, id int) error {
	return c.deleteIgnoringNotFound(ctx, fmt.Sprintf("/api/v2.0/projects/%d", id))
}
