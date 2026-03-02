package harborclient

import (
	"context"
	"fmt"
)

// ImmutableSelector defines an immutable tag selector.
type ImmutableSelector struct {
	Kind       string `json:"kind,omitempty"`
	Decoration string `json:"decoration,omitempty"`
	Pattern    string `json:"pattern,omitempty"`
	Extras     string `json:"extras,omitempty"`
}

// ImmutableRule represents an immutable tag rule.
type ImmutableRule struct {
	ID             int                            `json:"id,omitempty"`
	Priority       int                            `json:"priority,omitempty"`
	Disabled       bool                           `json:"disabled,omitempty"`
	Action         string                         `json:"action,omitempty"`
	Template       string                         `json:"template,omitempty"`
	Params         map[string]any                 `json:"params,omitempty"`
	TagSelectors   []ImmutableSelector            `json:"tag_selectors,omitempty"`
	ScopeSelectors map[string][]ImmutableSelector `json:"scope_selectors,omitempty"`
}

// ListImmutableRules lists immutable tag rules for a project.
func (c *Client) ListImmutableRules(ctx context.Context, projectNameOrID string) ([]ImmutableRule, error) {
	var out []ImmutableRule
	_, err := c.do(ctx, "GET", fmt.Sprintf("/api/v2.0/projects/%s/immutabletagrules", projectNameOrID), nil, &out)
	return out, err
}

// CreateImmutableRule creates a new immutable tag rule.
func (c *Client) CreateImmutableRule(ctx context.Context, projectNameOrID string, in ImmutableRule) error {
	_, err := c.do(ctx, "POST", fmt.Sprintf("/api/v2.0/projects/%s/immutabletagrules", projectNameOrID), &in, nil)
	return err
}

// UpdateImmutableRule updates an immutable tag rule.
func (c *Client) UpdateImmutableRule(ctx context.Context, projectNameOrID string, id int, in ImmutableRule) error {
	_, err := c.do(ctx, "PUT", fmt.Sprintf("/api/v2.0/projects/%s/immutabletagrules/%d", projectNameOrID, id), &in, nil)
	return err
}

// DeleteImmutableRule deletes an immutable tag rule.
func (c *Client) DeleteImmutableRule(ctx context.Context, projectNameOrID string, id int) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/v2.0/projects/%s/immutabletagrules/%d", projectNameOrID, id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
