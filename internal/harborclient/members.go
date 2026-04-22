package harborclient

import (
	"context"
	"fmt"
)

// MemberUser is the Harbor API payload for a user member.
type MemberUser struct {
	Username string `json:"username,omitempty"`
}

// MemberGroup is the Harbor API payload for a group member.
type MemberGroup struct {
	GroupName   string `json:"group_name,omitempty"`
	GroupType   int    `json:"group_type,omitempty"`
	LDAPGroupDN string `json:"ldap_group_dn,omitempty"`
}

// CreateMemberRequest is the payload for creating a project member.
type CreateMemberRequest struct {
	RoleID      int          `json:"role_id"`
	MemberUser  *MemberUser  `json:"member_user,omitempty"`
	MemberGroup *MemberGroup `json:"member_group,omitempty"`
}

// ProjectMember represents the member object returned by Harbor.
type ProjectMember struct {
	ID         int    `json:"id"`
	ProjectID  int    `json:"project_id"`
	EntityName string `json:"entity_name"`
	EntityType string `json:"entity_type"` // "u" for user, "g" for group
	EntityID   int    `json:"entity_id"`
	RoleID     int    `json:"role_id"`
	RoleName   string `json:"role_name"`
}

// ListProjectMembers lists all members of a Harbor project.
// Harbor accepts a project path segment here; controllers pass the resolved project name.
func (c *Client) ListProjectMembers(ctx context.Context, projectRef string) ([]ProjectMember, error) {
	var ms []ProjectMember
	err := c.get(ctx, fmt.Sprintf("/api/v2.0/projects/%s/members", projectRef), &ms)
	return ms, err
}

// CreateProjectMember creates a new project member.
// Returns the numeric member ID parsed from the Location header, if present.
func (c *Client) CreateProjectMember(ctx context.Context, projectRef string, in CreateMemberRequest) (int, error) {
	return c.createWithNumericLocationID(ctx, fmt.Sprintf("/api/v2.0/projects/%s/members", projectRef), &in)
}

// GetProjectMember retrieves a specific project member by ID.
func (c *Client) GetProjectMember(ctx context.Context, projectRef string, memberID int) (*ProjectMember, error) {
	var m ProjectMember
	err := c.get(ctx, fmt.Sprintf("/api/v2.0/projects/%s/members/%d", projectRef, memberID), &m)
	return &m, err
}

// UpdateProjectMemberRole updates only the role_id of an existing project member.
func (c *Client) UpdateProjectMemberRole(ctx context.Context, projectRef string, memberID int, roleID int) error {
	body := struct {
		RoleID int `json:"role_id"`
	}{
		RoleID: roleID,
	}

	return c.put(ctx, fmt.Sprintf("/api/v2.0/projects/%s/members/%d", projectRef, memberID), &body)
}

// DeleteProjectMember deletes a project member.
// 404 is treated as success (already gone).
func (c *Client) DeleteProjectMember(ctx context.Context, projectRef string, memberID int) error {
	return c.deleteIgnoringNotFound(ctx, fmt.Sprintf("/api/v2.0/projects/%s/members/%d", projectRef, memberID))
}
