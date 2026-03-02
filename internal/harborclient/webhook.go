package harborclient

import (
	"context"
	"fmt"
)

// WebhookTarget defines a webhook target.
type WebhookTarget struct {
	Type           string `json:"type,omitempty"`
	Address        string `json:"address,omitempty"`
	AuthHeader     string `json:"auth_header,omitempty"`
	PayloadFormat  string `json:"payload_format,omitempty"`
	SkipCertVerify bool   `json:"skip_cert_verify,omitempty"`
}

// WebhookPolicy represents a webhook policy.
type WebhookPolicy struct {
	ID           int             `json:"id,omitempty"`
	Name         string          `json:"name,omitempty"`
	Description  string          `json:"description,omitempty"`
	ProjectID    int             `json:"project_id,omitempty"`
	Targets      []WebhookTarget `json:"targets,omitempty"`
	EventTypes   []string        `json:"event_types,omitempty"`
	Creator      string          `json:"creator,omitempty"`
	CreationTime string          `json:"creation_time,omitempty"`
	UpdateTime   string          `json:"update_time,omitempty"`
	Enabled      bool            `json:"enabled,omitempty"`
}

// ListWebhookPolicies lists webhook policies for a project.
func (c *Client) ListWebhookPolicies(ctx context.Context, projectNameOrID string) ([]WebhookPolicy, error) {
	var out []WebhookPolicy
	_, err := c.do(ctx, "GET", fmt.Sprintf("/api/v2.0/projects/%s/webhook/policies", projectNameOrID), nil, &out)
	return out, err
}

// GetWebhookPolicy retrieves a webhook policy by ID.
func (c *Client) GetWebhookPolicy(ctx context.Context, projectNameOrID string, id int) (*WebhookPolicy, error) {
	var out WebhookPolicy
	_, err := c.do(ctx, "GET", fmt.Sprintf("/api/v2.0/projects/%s/webhook/policies/%d", projectNameOrID, id), nil, &out)
	return &out, err
}

// CreateWebhookPolicy creates a webhook policy.
func (c *Client) CreateWebhookPolicy(ctx context.Context, projectNameOrID string, in WebhookPolicy) (int, error) {
	resp, err := c.do(ctx, "POST", fmt.Sprintf("/api/v2.0/projects/%s/webhook/policies", projectNameOrID), &in, nil)
	if err != nil {
		return 0, err
	}
	return extractLocationID(resp)
}

// UpdateWebhookPolicy updates a webhook policy.
func (c *Client) UpdateWebhookPolicy(ctx context.Context, projectNameOrID string, id int, in WebhookPolicy) error {
	_, err := c.do(ctx, "PUT", fmt.Sprintf("/api/v2.0/projects/%s/webhook/policies/%d", projectNameOrID, id), &in, nil)
	return err
}

// DeleteWebhookPolicy deletes a webhook policy.
func (c *Client) DeleteWebhookPolicy(ctx context.Context, projectNameOrID string, id int) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/v2.0/projects/%s/webhook/policies/%d", projectNameOrID, id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
