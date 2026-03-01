package harborclient

import (
	"context"
	"fmt"
	"strings"
)

type Schedule struct {
	Schedule   ScheduleObj    `json:"schedule"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type ScheduleObj struct {
	Type string `json:"type,omitempty"`
	Cron string `json:"cron,omitempty"`
}

func (c *Client) GetGCSchedule(ctx context.Context) (*Schedule, error) {
	var out Schedule
	_, err := c.do(ctx, "GET", "/api/v2.0/system/gc/schedule", nil, &out)
	return &out, err
}

func (c *Client) CreateGCSchedule(ctx context.Context, in Schedule) error {
	_, err := c.do(ctx, "POST", "/api/v2.0/system/gc/schedule", &in, nil)
	return err
}

func (c *Client) UpdateGCSchedule(ctx context.Context, in Schedule) error {
	_, err := c.do(ctx, "PUT", "/api/v2.0/system/gc/schedule", &in, nil)
	return err
}

func (c *Client) GetPurgeSchedule(ctx context.Context) (*Schedule, error) {
	var out Schedule
	_, err := c.do(ctx, "GET", "/api/v2.0/system/purgeaudit/schedule", nil, &out)
	return &out, err
}

func (c *Client) CreatePurgeSchedule(ctx context.Context, in Schedule) error {
	_, err := c.do(ctx, "POST", "/api/v2.0/system/purgeaudit/schedule", &in, nil)
	return err
}

func (c *Client) UpdatePurgeSchedule(ctx context.Context, in Schedule) error {
	_, err := c.do(ctx, "PUT", "/api/v2.0/system/purgeaudit/schedule", &in, nil)
	return err
}

func (c *Client) GetScanAllSchedule(ctx context.Context) (*Schedule, error) {
	var out Schedule
	_, err := c.do(ctx, "GET", "/api/v2.0/system/scanAll/schedule", nil, &out)
	return &out, err
}

func (c *Client) CreateScanAllSchedule(ctx context.Context, in Schedule) error {
	_, err := c.do(ctx, "POST", "/api/v2.0/system/scanAll/schedule", &in, nil)
	return err
}

func (c *Client) UpdateScanAllSchedule(ctx context.Context, in Schedule) error {
	_, err := c.do(ctx, "PUT", "/api/v2.0/system/scanAll/schedule", &in, nil)
	return err
}

func (c *Client) GetRetentionByID(ctx context.Context, id int) (*RetentionPolicy, error) {
	var out RetentionPolicy
	_, err := c.do(ctx, "GET", fmt.Sprintf("/api/v2.0/retentions/%d", id), nil, &out)
	return &out, err
}

func (c *Client) CreateRetention(ctx context.Context, in RetentionPolicy) (int, error) {
	resp, err := c.do(ctx, "POST", "/api/v2.0/retentions", &in, nil)
	if err != nil {
		return 0, err
	}
	return extractLocationID(resp)
}

func (c *Client) UpdateRetention(ctx context.Context, id int, in RetentionPolicy) error {
	_, err := c.do(ctx, "PUT", fmt.Sprintf("/api/v2.0/retentions/%d", id), &in, nil)
	return err
}

func (c *Client) DeleteRetention(ctx context.Context, id int) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/v2.0/retentions/%d", id), nil, nil)
	if IsNotFound(err) || isRetentionGone(err) {
		return nil
	}
	return err
}

func isRetentionGone(err error) bool {
	if err == nil {
		return false
	}
	if he, ok := err.(*HTTPError); ok && he.StatusCode == 400 {
		// Harbor sometimes returns 400 for missing retention policies.
		return strings.Contains(he.Message, "no such Retention policy")
	}
	return false
}

type RetentionPolicy struct {
	ID        int               `json:"id,omitempty"`
	Algorithm string            `json:"algorithm,omitempty"`
	Rules     []RetentionRule   `json:"rules,omitempty"`
	Trigger   *RetentionTrigger `json:"trigger,omitempty"`
	Scope     *RetentionScope   `json:"scope,omitempty"`
}

type RetentionRule struct {
	ID             int                            `json:"id,omitempty"`
	Priority       int                            `json:"priority,omitempty"`
	Disabled       bool                           `json:"disabled,omitempty"`
	Action         string                         `json:"action,omitempty"`
	Template       string                         `json:"template,omitempty"`
	Params         any                            `json:"params,omitempty"`
	TagSelectors   []RetentionSelector            `json:"tag_selectors,omitempty"`
	ScopeSelectors map[string][]RetentionSelector `json:"scope_selectors,omitempty"`
}

type RetentionSelector struct {
	Kind       string `json:"kind,omitempty"`
	Decoration string `json:"decoration,omitempty"`
	Pattern    string `json:"pattern,omitempty"`
	Extras     string `json:"extras,omitempty"`
}

type RetentionTrigger struct {
	Kind       string         `json:"kind,omitempty"`
	Settings   map[string]any `json:"settings,omitempty"`
	References map[string]any `json:"references,omitempty"`
}

type RetentionScope struct {
	Level string `json:"level,omitempty"`
	Ref   int    `json:"ref,omitempty"`
}
