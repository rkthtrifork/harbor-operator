package harborclient

import (
	"context"
	"fmt"
	"net/url"
)

// Quota represents a Harbor quota.
type Quota struct {
	ID           int              `json:"id,omitempty"`
	Ref          map[string]any   `json:"ref,omitempty"`
	Hard         map[string]int64 `json:"hard,omitempty"`
	Used         map[string]int64 `json:"used,omitempty"`
	CreationTime string           `json:"creation_time,omitempty"`
	UpdateTime   string           `json:"update_time,omitempty"`
}

// ListQuotas lists quotas with optional reference filters.
func (c *Client) ListQuotas(ctx context.Context, reference, referenceID string) ([]Quota, error) {
	values := url.Values{}
	values.Set("page", "1")
	values.Set("page_size", "100")
	if reference != "" {
		values.Set("reference", reference)
	}
	if referenceID != "" {
		values.Set("reference_id", referenceID)
	}
	rel := "/api/v2.0/quotas"
	if len(values) > 0 {
		rel += "?" + values.Encode()
	}
	var out []Quota
	err := c.get(ctx, rel, &out)
	return out, err
}

// GetQuota retrieves a quota by ID.
func (c *Client) GetQuota(ctx context.Context, id int) (*Quota, error) {
	var out Quota
	err := c.get(ctx, fmt.Sprintf("/api/v2.0/quotas/%d", id), &out)
	return &out, err
}

// UpdateQuota updates a quota's hard limits.
func (c *Client) UpdateQuota(ctx context.Context, id int, hard map[string]int64) error {
	body := struct {
		Hard map[string]int64 `json:"hard,omitempty"`
	}{
		Hard: hard,
	}
	return c.put(ctx, fmt.Sprintf("/api/v2.0/quotas/%d", id), &body)
}
