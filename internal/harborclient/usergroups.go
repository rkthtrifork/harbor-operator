package harborclient

import (
	"context"
	"fmt"
	"net/url"
)

// UserGroup represents a Harbor user group.
type UserGroup struct {
	ID          int    `json:"id,omitempty"`
	GroupName   string `json:"group_name,omitempty"`
	GroupType   int    `json:"group_type,omitempty"`
	LDAPGroupDN string `json:"ldap_group_dn,omitempty"`
}

// UserGroupSearchItem represents a user group search result.
type UserGroupSearchItem struct {
	ID        int    `json:"id,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	GroupType int    `json:"group_type,omitempty"`
}

// ListUserGroups lists all user groups.
func (c *Client) ListUserGroups(ctx context.Context) ([]UserGroup, error) {
	var out []UserGroup
	err := c.get(ctx, "/api/v2.0/usergroups", &out)
	return out, err
}

// SearchUserGroups searches user groups by name.
func (c *Client) SearchUserGroups(ctx context.Context, groupName string) ([]UserGroupSearchItem, error) {
	values := url.Values{}
	values.Set("groupname", groupName)
	values.Set("page", "1")
	values.Set("page_size", "100")
	rel := "/api/v2.0/usergroups/search?" + values.Encode()
	var out []UserGroupSearchItem
	err := c.get(ctx, rel, &out)
	return out, err
}

// GetUserGroup retrieves a user group by ID.
func (c *Client) GetUserGroup(ctx context.Context, id int) (*UserGroup, error) {
	var out UserGroup
	err := c.get(ctx, fmt.Sprintf("/api/v2.0/usergroups/%d", id), &out)
	return &out, err
}

// CreateUserGroup creates a user group.
func (c *Client) CreateUserGroup(ctx context.Context, in UserGroup) (int, error) {
	return c.createWithNumericLocationID(ctx, "/api/v2.0/usergroups", &in)
}

// UpdateUserGroup updates a user group.
func (c *Client) UpdateUserGroup(ctx context.Context, id int, in UserGroup) error {
	return c.put(ctx, fmt.Sprintf("/api/v2.0/usergroups/%d", id), &in)
}

// DeleteUserGroup deletes a user group.
func (c *Client) DeleteUserGroup(ctx context.Context, id int) error {
	return c.deleteIgnoringNotFound(ctx, fmt.Sprintf("/api/v2.0/usergroups/%d", id))
}
