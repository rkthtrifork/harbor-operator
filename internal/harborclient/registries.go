package harborclient

import (
	"context"
	"fmt"
	"net/url"
)

type Registry struct {
	ID            int                 `json:"id"`
	URL           string              `json:"url"`
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	Type          string              `json:"type"`
	Insecure      bool                `json:"insecure"`
	CACertificate string              `json:"ca_certificate,omitempty"`
	Credential    *RegistryCredential `json:"credential,omitempty"`
}

type RegistryCredential struct {
	Type         string `json:"type,omitempty"`
	AccessKey    string `json:"access_key,omitempty"`
	AccessSecret string `json:"access_secret,omitempty"`
}

type CreateRegistryRequest struct {
	URL           string              `json:"url,omitempty"`
	Name          string              `json:"name,omitempty"`
	Description   string              `json:"description,omitempty"`
	Type          string              `json:"type,omitempty"`
	Insecure      bool                `json:"insecure,omitempty"`
	CACertificate string              `json:"ca_certificate,omitempty"`
	Credential    *RegistryCredential `json:"credential,omitempty"`
}

type UpdateRegistryRequest struct {
	Name           string `json:"name,omitempty"`
	Description    string `json:"description,omitempty"`
	URL            string `json:"url,omitempty"`
	CredentialType string `json:"credential_type,omitempty"`
	AccessKey      string `json:"access_key,omitempty"`
	AccessSecret   string `json:"access_secret,omitempty"`
	Insecure       bool   `json:"insecure,omitempty"`
	CACertificate  string `json:"ca_certificate,omitempty"`
}

func (c *Client) FindRegistryByName(ctx context.Context, name string) (*Registry, error) {
	if name == "" {
		return nil, nil
	}
	escaped := url.QueryEscape(name)
	path := fmt.Sprintf("/api/v2.0/registries?page=1&page_size=100&q=name=%s", escaped)
	var regs []Registry
	if err := c.get(ctx, path, &regs); err != nil {
		return nil, err
	}
	for i := range regs {
		if regs[i].Name == name {
			return &regs[i], nil
		}
	}
	return nil, nil
}

func (c *Client) GetRegistryByID(ctx context.Context, id int) (*Registry, error) {
	var reg Registry
	err := c.get(ctx, fmt.Sprintf("/api/v2.0/registries/%d", id), &reg)
	return &reg, err
}

func (c *Client) CreateRegistry(ctx context.Context, in CreateRegistryRequest) (int, error) {
	return c.createWithNumericLocationID(ctx, "/api/v2.0/registries", &in)
}

func (c *Client) UpdateRegistry(ctx context.Context, id int, in UpdateRegistryRequest) error {
	return c.put(ctx, fmt.Sprintf("/api/v2.0/registries/%d", id), &in)
}

func (c *Client) DeleteRegistry(ctx context.Context, id int) error {
	return c.deleteIgnoringNotFound(ctx, fmt.Sprintf("/api/v2.0/registries/%d", id))
}
