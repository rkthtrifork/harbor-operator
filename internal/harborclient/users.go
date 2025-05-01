package harborclient

import (
	"context"
	"fmt"
)

type User struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Realname     string `json:"realname"`
	Comment      string `json:"comment"`
	SysadminFlag bool   `json:"sysadmin_flag"`
}

type CreateUserRequest struct {
	Email    string `json:"email,omitempty"`
	Realname string `json:"realname,omitempty"`
	Comment  string `json:"comment,omitempty"`
	Password string `json:"password,omitempty"`
	Username string `json:"username,omitempty"`
}

type UpdateUserRequest struct {
	Email    string `json:"email,omitempty"`
	Realname string `json:"realname,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// ListUsers GET /users
func (c *Client) ListUsers(ctx context.Context, query string) ([]User, error) {
	rel := "/api/v2.0/users"
	if query != "" {
		rel += "?q=" + query
	}
	var us []User
	_, err := c.do(ctx, "GET", rel, nil, &us)
	return us, err
}

func (c *Client) GetUserByID(ctx context.Context, id int) (*User, error) {
	var u User
	_, err := c.do(ctx, "GET",
		fmt.Sprintf("/api/v2.0/users/%d", id), nil, &u)
	return &u, err
}

func (c *Client) CreateUser(ctx context.Context,
	in CreateUserRequest) (int, error) {

	resp, err := c.do(ctx, "POST", "/api/v2.0/users", &in, nil)
	if err != nil {
		return 0, err
	}
	return extractLocationID(resp)
}

func (c *Client) UpdateUser(ctx context.Context, id int,
	in UpdateUserRequest) error {

	_, err := c.do(ctx, "PUT",
		fmt.Sprintf("/api/v2.0/users/%d", id), &in, nil)
	return err
}

func (c *Client) DeleteUser(ctx context.Context, id int) error {
	_, err := c.do(ctx, "DELETE",
		fmt.Sprintf("/api/v2.0/users/%d", id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
