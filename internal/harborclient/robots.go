package harborclient

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Robot represents a Harbor robot account.
type Robot struct {
	ID          int               `json:"id,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Secret      string            `json:"secret,omitempty"`
	Level       string            `json:"level,omitempty"`
	Duration    *int              `json:"duration,omitempty"`
	Editable    bool              `json:"editable,omitempty"`
	Disable     bool              `json:"disable,omitempty"`
	ExpiresAt   int               `json:"expires_at,omitempty"`
	Permissions []RobotPermission `json:"permissions,omitempty"`
}

// RobotPermission defines access control for a robot account.
type RobotPermission struct {
	Kind      string   `json:"kind,omitempty"`
	Namespace string   `json:"namespace,omitempty"`
	Access    []Access `json:"access,omitempty"`
}

// Access defines a resource/action/effect tuple.
type Access struct {
	Resource string `json:"resource,omitempty"`
	Action   string `json:"action,omitempty"`
	Effect   string `json:"effect,omitempty"`
}

// RobotCreateRequest is the payload for robot account creation.
type RobotCreateRequest struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Secret      string            `json:"secret,omitempty"`
	Level       string            `json:"level,omitempty"`
	Disable     *bool             `json:"disable,omitempty"`
	Duration    *int              `json:"duration,omitempty"`
	Permissions []RobotPermission `json:"permissions,omitempty"`
}

// RobotCreated is the response for robot account creation.
type RobotCreated struct {
	ID         int    `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Secret     string `json:"secret,omitempty"`
	ExpiresAt  int    `json:"expires_at,omitempty"`
	CreateTime string `json:"creation_time,omitempty"`
}

// RobotSec is the response for refreshing a robot secret.
type RobotSec struct {
	Secret string `json:"secret,omitempty"`
}

// ListRobots lists robot accounts with an optional query.
func (c *Client) ListRobots(ctx context.Context, query string) ([]Robot, error) {
	rel := "/api/v2.0/robots"
	values := url.Values{}
	if query != "" {
		values.Set("q", query)
	}
	values.Set("page", "1")
	values.Set("page_size", "100")
	if len(values) > 0 {
		rel = rel + "?" + values.Encode()
	}
	var robots []Robot
	err := c.get(ctx, rel, &robots)
	return robots, err
}

// GetRobotByID retrieves a robot account by ID.
func (c *Client) GetRobotByID(ctx context.Context, id int) (*Robot, error) {
	var robot Robot
	err := c.get(ctx, fmt.Sprintf("/api/v2.0/robots/%d", id), &robot)
	return &robot, err
}

// CreateRobot creates a new robot account.
func (c *Client) CreateRobot(ctx context.Context, in RobotCreateRequest) (*RobotCreated, error) {
	var created RobotCreated
	err := c.post(ctx, "/api/v2.0/robots", &in, &created)
	return &created, err
}

// UpdateRobot updates an existing robot account.
func (c *Client) UpdateRobot(ctx context.Context, id int, in Robot) error {
	return c.put(ctx, fmt.Sprintf("/api/v2.0/robots/%d", id), &in)
}

// RefreshRobotSecret refreshes or sets a robot account secret.
func (c *Client) RefreshRobotSecret(ctx context.Context, id int, secret string) (*RobotSec, error) {
	var sec RobotSec
	body := RobotSec{Secret: secret}
	err := c.patch(ctx, fmt.Sprintf("/api/v2.0/robots/%d", id), &body, &sec)
	return &sec, err
}

// DeleteRobot deletes a robot account.
func (c *Client) DeleteRobot(ctx context.Context, id int) error {
	return c.deleteIgnoringNotFound(ctx, fmt.Sprintf("/api/v2.0/robots/%d", id))
}

// ParseRobotID converts string robot IDs to int.
func ParseRobotID(id string) (int, error) {
	parsed, err := strconv.ParseInt(id, 10, 0)
	return int(parsed), err
}
