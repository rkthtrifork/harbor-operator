package harborclient

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Label represents a Harbor label.
type Label struct {
	ID           int    `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Color        string `json:"color,omitempty"`
	Scope        string `json:"scope,omitempty"`
	ProjectID    int    `json:"project_id,omitempty"`
	CreationTime string `json:"creation_time,omitempty"`
	UpdateTime   string `json:"update_time,omitempty"`
}

// ListLabels lists labels with optional filters.
func (c *Client) ListLabels(ctx context.Context, name, scope string, projectID *int) ([]Label, error) {
	values := url.Values{}
	values.Set("page", "1")
	values.Set("page_size", "100")
	if name != "" {
		values.Set("name", name)
	}
	if scope != "" {
		values.Set("scope", scope)
	}
	if projectID != nil {
		values.Set("project_id", strconv.Itoa(*projectID))
	}
	rel := "/api/v2.0/labels"
	if len(values) > 0 {
		rel += "?" + values.Encode()
	}
	var out []Label
	_, err := c.do(ctx, "GET", rel, nil, &out)
	return out, err
}

// GetLabel retrieves a label by ID.
func (c *Client) GetLabel(ctx context.Context, id int) (*Label, error) {
	var out Label
	_, err := c.do(ctx, "GET", fmt.Sprintf("/api/v2.0/labels/%d", id), nil, &out)
	return &out, err
}

// CreateLabel creates a label.
func (c *Client) CreateLabel(ctx context.Context, in Label) (int, error) {
	resp, err := c.do(ctx, "POST", "/api/v2.0/labels", &in, nil)
	if err != nil {
		return 0, err
	}
	return extractLocationID(resp)
}

// UpdateLabel updates a label.
func (c *Client) UpdateLabel(ctx context.Context, id int, in Label) error {
	_, err := c.do(ctx, "PUT", fmt.Sprintf("/api/v2.0/labels/%d", id), &in, nil)
	return err
}

// DeleteLabel deletes a label.
func (c *Client) DeleteLabel(ctx context.Context, id int) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/v2.0/labels/%d", id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
