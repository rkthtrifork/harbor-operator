// pkg/harborclient/ping.go
package harborclient

import (
	"context"
	"net/http"
)

// Ping calls /​api/v2.0/ping. It returns nil when Harbor is reachable.
// Both 200 OK and 401 Unauthorized are considered “reachable”.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.do(ctx, "GET", "/api/v2.0/ping", nil, nil)
	if he, ok := err.(*HTTPError); ok &&
		(he.StatusCode == http.StatusUnauthorized || he.StatusCode == http.StatusOK) {
		return nil
	}
	return err
}

type CurrentUser struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
}

// GetCurrentUser does a basic-auth GET /users/current.
func (c *Client) GetCurrentUser(ctx context.Context) (*CurrentUser, error) {
	var u CurrentUser
	_, err := c.do(ctx, "GET", "/api/v2.0/users/current", nil, &u)
	return &u, err
}
