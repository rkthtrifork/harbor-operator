package harborclient

import (
	"context"
	"fmt"
)

// MemberUser defines a user-based member as exposed by Harbor.
type MemberUser struct {
	// If the user already exists in Harbor, set UserID.
	UserID int `json:"user_id,omitempty"`
	// Username is used to onboard a user if not already present.
	Username string `json:"username,omitempty"`
}

// MemberGroup defines a group-based member as exposed by Harbor.
type MemberGroup struct {
	// If the group already exists in Harbor, set its ID.
	ID int `json:"id,omitempty"`
	// GroupName is the name of the group.
	GroupName string `json:"group_name,omitempty"`
	// GroupType is the type of the group (e.g. Harbor, LDAP, etc.).
	GroupType int `json:"group_type,omitempty"`
	// LDAPGroupDN is used for LDAP groups.
	LDAPGroupDN string `json:"ldap_group_dn,omitempty"`
}

// Member represents a project member returned by Harbor.
type Member struct {
	ID          int          `json:"id"`
	RoleID      int          `json:"role_id"`
	MemberUser  *MemberUser  `json:"member_user,omitempty"`
	MemberGroup *MemberGroup `json:"member_group,omitempty"`
}

// CreateMemberRequest is the payload used when creating or updating a member.
type CreateMemberRequest struct {
	RoleID      int          `json:"role_id"`
	MemberUser  *MemberUser  `json:"member_user,omitempty"`
	MemberGroup *MemberGroup `json:"member_group,omitempty"`
}

// ListProjectMembers lists all members of the given project.
// projectRef is the Harbor project ID or name (as accepted by the API).
func (c *Client) ListProjectMembers(ctx context.Context, projectRef string) ([]Member, error) {
	var members []Member
	_, err := c.do(ctx, "GET",
		fmt.Sprintf("/api/v2.0/projects/%s/members", projectRef),
		nil, &members)
	return members, err
}

// GetProjectMember returns a single member by its Harbor member ID.
func (c *Client) GetProjectMember(ctx context.Context, projectRef string, memberID int) (*Member, error) {
	var m Member
	_, err := c.do(ctx, "GET",
		fmt.Sprintf("/api/v2.0/projects/%s/members/%d", projectRef, memberID),
		nil, &m)
	return &m, err
}

// CreateProjectMember creates a new member on the given project.
// It returns the created member's ID when successful.
// If Harbor returns 409 Conflict (member already exists), it returns (0, nil)
// so controllers can treat it as idempotent.
func (c *Client) CreateProjectMember(ctx context.Context, projectRef string, in CreateMemberRequest) (int, error) {
	resp, err := c.do(ctx, "POST",
		fmt.Sprintf("/api/v2.0/projects/%s/members", projectRef),
		&in, nil)
	if err != nil {
		if IsConflict(err) {
			// Member already exists – treat as success but we don't know the ID.
			return 0, nil
		}
		return 0, err
	}
	return extractLocationID(resp)
}

// UpdateProjectMember updates an existing member's role / mapping.
func (c *Client) UpdateProjectMember(ctx context.Context, projectRef string, memberID int, in CreateMemberRequest) error {
	_, err := c.do(ctx, "PUT",
		fmt.Sprintf("/api/v2.0/projects/%s/members/%d", projectRef, memberID),
		&in, nil)
	return err
}

// DeleteProjectMember deletes a member from the project.
// 404 Not Found is treated as success for idempotency.
func (c *Client) DeleteProjectMember(ctx context.Context, projectRef string, memberID int) error {
	_, err := c.do(ctx, "DELETE",
		fmt.Sprintf("/api/v2.0/projects/%s/members/%d", projectRef, memberID),
		nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
