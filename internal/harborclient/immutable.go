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
func (c *Client) ListImmutableRules(ctx context.Context, projectRef string) ([]ImmutableRule, error) {
	var out []ImmutableRule
	err := c.get(ctx, fmt.Sprintf("/api/v2.0/projects/%s/immutabletagrules", projectRef), &out)
	return out, err
}

// CreateImmutableRule creates a new immutable tag rule.
func (c *Client) CreateImmutableRule(ctx context.Context, projectRef string, in ImmutableRule) error {
	return c.post(ctx, fmt.Sprintf("/api/v2.0/projects/%s/immutabletagrules", projectRef), &in, nil)
}

// UpdateImmutableRule updates an immutable tag rule.
func (c *Client) UpdateImmutableRule(ctx context.Context, projectRef string, id int, in ImmutableRule) error {
	return c.put(ctx, fmt.Sprintf("/api/v2.0/projects/%s/immutabletagrules/%d", projectRef, id), &in)
}

// DeleteImmutableRule deletes an immutable tag rule.
func (c *Client) DeleteImmutableRule(ctx context.Context, projectRef string, id int) error {
	return c.deleteIgnoringNotFound(ctx, fmt.Sprintf("/api/v2.0/projects/%s/immutabletagrules/%d", projectRef, id))
}
