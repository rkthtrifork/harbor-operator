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

// CheckAuthentication verifies credentials against an endpoint supported by
// both user and robot security contexts.
func (c *Client) CheckAuthentication(ctx context.Context) error {
	var permissions []struct{}
	return c.get(ctx, "/api/v2.0/users/current/permissions", &permissions)
}
