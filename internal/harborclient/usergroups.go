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
	_, err := c.do(ctx, "GET", "/api/v2.0/usergroups", nil, &out)
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
	_, err := c.do(ctx, "GET", rel, nil, &out)
	return out, err
}

// GetUserGroup retrieves a user group by ID.
func (c *Client) GetUserGroup(ctx context.Context, id int) (*UserGroup, error) {
	var out UserGroup
	_, err := c.do(ctx, "GET", fmt.Sprintf("/api/v2.0/usergroups/%d", id), nil, &out)
	return &out, err
}

// CreateUserGroup creates a user group.
func (c *Client) CreateUserGroup(ctx context.Context, in UserGroup) (int, error) {
	resp, err := c.do(ctx, "POST", "/api/v2.0/usergroups", &in, nil)
	if err != nil {
		return 0, err
	}
	return extractLocationID(resp)
}

// UpdateUserGroup updates a user group.
func (c *Client) UpdateUserGroup(ctx context.Context, id int, in UserGroup) error {
	_, err := c.do(ctx, "PUT", fmt.Sprintf("/api/v2.0/usergroups/%d", id), &in, nil)
	return err
}

// DeleteUserGroup deletes a user group.
func (c *Client) DeleteUserGroup(ctx context.Context, id int) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/v2.0/usergroups/%d", id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
