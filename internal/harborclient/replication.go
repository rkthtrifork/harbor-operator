package harborclient

import (
	"context"
	"fmt"
	"net/url"
)

// ReplicationTriggerSettings defines settings for replication triggers.
type ReplicationTriggerSettings struct {
	Cron string `json:"cron,omitempty"`
}

// ReplicationTrigger defines when a replication policy runs.
type ReplicationTrigger struct {
	Type            string                      `json:"type,omitempty"`
	TriggerSettings *ReplicationTriggerSettings `json:"trigger_settings,omitempty"`
}

// ReplicationFilter defines a replication policy filter.
type ReplicationFilter struct {
	Type       string `json:"type,omitempty"`
	Value      any    `json:"value,omitempty"`
	Decoration string `json:"decoration,omitempty"`
}

// ReplicationPolicy represents a Harbor replication policy.
type ReplicationPolicy struct {
	ID                        int                 `json:"id,omitempty"`
	Name                      string              `json:"name,omitempty"`
	Description               string              `json:"description,omitempty"`
	SrcRegistry               *Registry           `json:"src_registry,omitempty"`
	DestRegistry              *Registry           `json:"dest_registry,omitempty"`
	DestNamespace             string              `json:"dest_namespace,omitempty"`
	DestNamespaceReplaceCount *int                `json:"dest_namespace_replace_count,omitempty"`
	Trigger                   *ReplicationTrigger `json:"trigger,omitempty"`
	Filters                   []ReplicationFilter `json:"filters,omitempty"`
	ReplicateDeletion         *bool               `json:"replicate_deletion,omitempty"`
	Deletion                  *bool               `json:"deletion,omitempty"`
	Override                  *bool               `json:"override,omitempty"`
	Enabled                   *bool               `json:"enabled,omitempty"`
	CreationTime              string              `json:"creation_time,omitempty"`
	UpdateTime                string              `json:"update_time,omitempty"`
	Speed                     *int                `json:"speed,omitempty"`
	CopyByChunk               *bool               `json:"copy_by_chunk,omitempty"`
	SingleActiveReplication   *bool               `json:"single_active_replication,omitempty"`
}

// ListReplicationPolicies lists replication policies.
func (c *Client) ListReplicationPolicies(ctx context.Context, name string) ([]ReplicationPolicy, error) {
	values := url.Values{}
	values.Set("page", "1")
	values.Set("page_size", "100")
	if name != "" {
		values.Set("name", name)
	}
	rel := "/api/v2.0/replication/policies"
	if len(values) > 0 {
		rel += "?" + values.Encode()
	}
	var out []ReplicationPolicy
	_, err := c.do(ctx, "GET", rel, nil, &out)
	return out, err
}

// GetReplicationPolicy retrieves a replication policy by ID.
func (c *Client) GetReplicationPolicy(ctx context.Context, id int) (*ReplicationPolicy, error) {
	var out ReplicationPolicy
	_, err := c.do(ctx, "GET", fmt.Sprintf("/api/v2.0/replication/policies/%d", id), nil, &out)
	return &out, err
}

// CreateReplicationPolicy creates a new replication policy.
func (c *Client) CreateReplicationPolicy(ctx context.Context, in ReplicationPolicy) (int, error) {
	resp, err := c.do(ctx, "POST", "/api/v2.0/replication/policies", &in, nil)
	if err != nil {
		return 0, err
	}
	return extractLocationID(resp)
}

// UpdateReplicationPolicy updates an existing replication policy.
func (c *Client) UpdateReplicationPolicy(ctx context.Context, id int, in ReplicationPolicy) error {
	_, err := c.do(ctx, "PUT", fmt.Sprintf("/api/v2.0/replication/policies/%d", id), &in, nil)
	return err
}

// DeleteReplicationPolicy deletes a replication policy.
func (c *Client) DeleteReplicationPolicy(ctx context.Context, id int) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/v2.0/replication/policies/%d", id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}
