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

func (c *Client) ListUsers(ctx context.Context, query string) ([]User, error) {
	rel := "/api/v2.0/users"
	if query != "" {
		rel += "?q=" + query
	}
	var us []User
	err := c.get(ctx, rel, &us)
	return us, err
}

func (c *Client) GetUserByID(ctx context.Context, id int) (User, error) {
	var u User
	err := c.get(ctx, fmt.Sprintf("/api/v2.0/users/%d", id), &u)
	return u, err
}

func (c *Client) CreateUser(ctx context.Context, in CreateUserRequest) (int, error) {
	return c.createWithNumericLocationID(ctx, "/api/v2.0/users", &in)
}

func (c *Client) UpdateUser(ctx context.Context, id int, in CreateUserRequest) error {
	return c.put(ctx, fmt.Sprintf("/api/v2.0/users/%d", id), &in)
}

func (c *Client) DeleteUser(ctx context.Context, id int) error {
	return c.deleteIgnoringNotFound(ctx, fmt.Sprintf("/api/v2.0/users/%d", id))
}
