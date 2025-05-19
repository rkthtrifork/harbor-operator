package harborclient

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// -----------------------------------------------------------------------------
// API payloads
// -----------------------------------------------------------------------------

// Member represents the wire format returned by
//
//	GET /projects/{project_name_or_id}/members[/{id}]
type Member struct {
	ID         int    `json:"id"`
	ProjectID  int    `json:"project_id"`
	EntityName string `json:"entity_name"`
	RoleName   string `json:"role_name"`
	RoleID     int    `json:"role_id"`
	EntityID   int    `json:"entity_id"`
	EntityType string `json:"entity_type"` // user / group / robot
}

// --- create ---------------------------------------------------------------

type MemberUser struct {
	UserID   int    `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
}

type MemberGroup struct {
	ID          int    `json:"id,omitempty"`
	GroupName   string `json:"group_name,omitempty"`
	GroupType   int    `json:"group_type,omitempty"`
	LDAPGroupDN string `json:"ldap_group_dn,omitempty"`
}

type CreateMemberRequest struct {
	RoleID      int          `json:"role_id"`
	MemberUser  *MemberUser  `json:"member_user,omitempty"`
	MemberGroup *MemberGroup `json:"member_group,omitempty"`
}

// --- update (role only) ---------------------------------------------------

type updateMemberRoleRequest struct {
	RoleID int `json:"role_id"`
}

// -----------------------------------------------------------------------------
// Client helpers
// -----------------------------------------------------------------------------

// ListProjectMembers returns **all** members (no paging) by following `Link` headers.
func (c *Client) ListProjectMembers(ctx context.Context, project string) ([]Member, error) {
	var all []Member
	page := 1
	for {
		var m []Member
		q := url.Values{}
		q.Set("page", fmt.Sprint(page))
		q.Set("page_size", "100")
		path := fmt.Sprintf("/api/v2.0/projects/%s/members?%s", url.PathEscape(project), q.Encode())

		resp, err := c.do(ctx, "GET", path, nil, &m)
		if err != nil {
			return nil, err
		}
		all = append(all, m...)

		// Check if there's a 'next' link header; if not, we're done.
		if !hasNextLink(resp.Header.Get("Link")) {
			break
		}
		page++
	}
	return all, nil
}

func (c *Client) GetProjectMember(ctx context.Context, project string, id int) (*Member, error) {
	var m Member
	_, err := c.do(ctx, "GET",
		fmt.Sprintf("/api/v2.0/projects/%s/members/%d", url.PathEscape(project), id), nil, &m)
	return &m, err
}

func (c *Client) CreateProjectMember(ctx context.Context, project string, in CreateMemberRequest) (int, error) {
	resp, err := c.do(ctx, "POST",
		fmt.Sprintf("/api/v2.0/projects/%s/members", url.PathEscape(project)), &in, nil)
	if err != nil {
		return 0, err
	}
	return extractLocationID(resp)
}

func (c *Client) UpdateProjectMemberRole(ctx context.Context, project string, id, roleID int) error {
	in := updateMemberRoleRequest{RoleID: roleID}
	_, err := c.do(ctx, "PUT",
		fmt.Sprintf("/api/v2.0/projects/%s/members/%d", url.PathEscape(project), id), &in, nil)
	return err
}

func (c *Client) DeleteProjectMember(ctx context.Context, project string, id int) error {
	_, err := c.do(ctx, "DELETE",
		fmt.Sprintf("/api/v2.0/projects/%s/members/%d", url.PathEscape(project), id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// Harbor uses the RFC5988 Link header for paging. We only care if a `rel="next"` exists.
func hasNextLink(linkHeader string) bool {
	// naive but sufficient: look for rel="next"
	return strings.Contains(linkHeader, `rel="next"`)
}
