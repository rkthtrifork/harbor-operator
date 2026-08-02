package harborclient

import (
	"context"
)

// UserGroup represents a Harbor user group.
type UserGroup struct {
	ID          int    `json:"id,omitempty"`
	GroupName   string `json:"group_name,omitempty"`
	GroupType   int    `json:"group_type,omitempty"`
	LDAPGroupDN string `json:"ldap_group_dn,omitempty"`
}

// ListUserGroups lists all user groups.
func (c *Client) ListUserGroups(ctx context.Context) ([]UserGroup, error) {
	return getPaged[UserGroup](ctx, c, "/api/v2.0/usergroups", nil)
}

// CreateUserGroup creates a user group.
func (c *Client) CreateUserGroup(ctx context.Context, in UserGroup) (int, error) {
	return c.createWithNumericLocationID(ctx, "/api/v2.0/usergroups", &in)
}
