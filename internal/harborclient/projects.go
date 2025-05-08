package harborclient

import (
	"context"
	"fmt"
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
	Metadata     ProjectMetadata `json:"metadata,omitempty"`
	CVEAllowlist CVEAllowlist    `json:"cve_allowlist,omitempty"`
	StorageLimit *int            `json:"storage_limit,omitempty"`
	RegistryID   *int            `json:"registry_id,omitempty"`
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var ps []Project
	_, err := c.do(ctx, "GET", "/api/v2.0/projects", nil, &ps)
	return ps, err
}

func (c *Client) GetProjectByID(ctx context.Context, id int) (*Project, error) {
	var p Project
	_, err := c.do(ctx, "GET",
		fmt.Sprintf("/api/v2.0/projects/%d", id), nil, &p)
	return &p, err
}

func (c *Client) CreateProject(ctx context.Context, in CreateProjectRequest) (int, error) {

	resp, err := c.do(ctx, "POST", "/api/v2.0/projects", &in, nil)
	if err != nil {
		return 0, err
	}
	return extractLocationID(resp)
}

func (c *Client) UpdateProject(ctx context.Context, id int, in CreateProjectRequest) error {

	_, err := c.do(ctx, "PUT",
		fmt.Sprintf("/api/v2.0/projects/%d", id), &in, nil)
	return err
}

func (c *Client) DeleteProject(ctx context.Context, id int) error {
	_, err := c.do(ctx, "DELETE",
		fmt.Sprintf("/api/v2.0/projects/%d", id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
