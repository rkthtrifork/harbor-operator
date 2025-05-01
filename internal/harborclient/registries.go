package harborclient

import (
	"context"
	"fmt"
)

type Registry struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Insecure    bool   `json:"insecure"`
}

type CreateRegistryRequest struct {
	URL         string `json:"url,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Insecure    bool   `json:"insecure,omitempty"`
}

// GET /registries
func (c *Client) ListRegistries(ctx context.Context) ([]Registry, error) {
	var regs []Registry
	_, err := c.do(ctx, "GET", "/api/v2.0/registries", nil, &regs)
	return regs, err
}

func (c *Client) GetRegistryByID(ctx context.Context, id int) (*Registry, error) {
	var reg Registry
	_, err := c.do(ctx, "GET",
		fmt.Sprintf("/api/v2.0/registries/%d", id), nil, &reg)
	return &reg, err
}

func (c *Client) CreateRegistry(ctx context.Context,
	in CreateRegistryRequest) (int, error) {

	resp, err := c.do(ctx, "POST", "/api/v2.0/registries", &in, nil)
	if err != nil {
		return 0, err
	}
	return extractLocationID(resp)
}

func (c *Client) UpdateRegistry(ctx context.Context, id int,
	in CreateRegistryRequest) error {

	_, err := c.do(ctx, "PUT",
		fmt.Sprintf("/api/v2.0/registries/%d", id), &in, nil)
	return err
}

func (c *Client) DeleteRegistry(ctx context.Context, id int) error {
	_, err := c.do(ctx, "DELETE",
		fmt.Sprintf("/api/v2.0/registries/%d", id), nil, nil)
	if IsNotFound(err) {
		return nil // idempotent
	}
	return err
}
