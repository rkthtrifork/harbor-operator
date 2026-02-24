package harborclient

import (
	"context"
	"encoding/json"
)

// ConfigurationItem represents a single configuration value with metadata.
type ConfigurationItem struct {
	Value    json.RawMessage `json:"value"`
	Editable bool            `json:"editable"`
}

// GetConfigurations retrieves Harbor system configurations.
func (c *Client) GetConfigurations(ctx context.Context) (map[string]ConfigurationItem, error) {
	var cfg map[string]ConfigurationItem
	_, err := c.do(ctx, "GET", "/api/v2.0/configurations", nil, &cfg)
	return cfg, err
}

// UpdateConfigurations updates Harbor system configurations.
// The input map may contain a subset of configuration keys.
func (c *Client) UpdateConfigurations(ctx context.Context, in map[string]any) error {
	_, err := c.do(ctx, "PUT", "/api/v2.0/configurations", in, nil)
	return err
}
