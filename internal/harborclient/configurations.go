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
	err := c.get(ctx, "/api/v2.0/configurations", &cfg)
	return cfg, err
}

// UpdateConfigurations updates Harbor system configurations.
// The input map may contain a subset of configuration keys.
func (c *Client) UpdateConfigurations(ctx context.Context, in map[string]any) error {
	return c.put(ctx, "/api/v2.0/configurations", in)
}
