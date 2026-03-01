package harborclient

import (
	"context"
	"fmt"
)

// ScannerRegistration represents a scanner registration.
type ScannerRegistration struct {
	UUID             string         `json:"uuid,omitempty"`
	Name             string         `json:"name,omitempty"`
	Description      string         `json:"description,omitempty"`
	URL              string         `json:"url,omitempty"`
	Disabled         bool           `json:"disabled,omitempty"`
	IsDefault        bool           `json:"is_default,omitempty"`
	Auth             string         `json:"auth,omitempty"`
	AccessCredential string         `json:"access_credential,omitempty"`
	SkipCertVerify   bool           `json:"skip_certVerify,omitempty"`
	UseInternalAddr  bool           `json:"use_internal_addr,omitempty"`
	CreateTime       string         `json:"create_time,omitempty"`
	UpdateTime       string         `json:"update_time,omitempty"`
	Adapter          string         `json:"adapter,omitempty"`
	Vendor           string         `json:"vendor,omitempty"`
	Version          string         `json:"version,omitempty"`
	Health           string         `json:"health,omitempty"`
	Capabilities     map[string]any `json:"capabilities,omitempty"`
}

// ScannerRegistrationReq is the payload for scanner registration create/update.
type ScannerRegistrationReq struct {
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	URL              string `json:"url,omitempty"`
	Auth             string `json:"auth,omitempty"`
	AccessCredential string `json:"access_credential,omitempty"`
	SkipCertVerify   bool   `json:"skip_certVerify,omitempty"`
	UseInternalAddr  bool   `json:"use_internal_addr,omitempty"`
	Disabled         bool   `json:"disabled,omitempty"`
}

// ListScanners lists scanner registrations.
func (c *Client) ListScanners(ctx context.Context) ([]ScannerRegistration, error) {
	var out []ScannerRegistration
	_, err := c.do(ctx, "GET", "/api/v2.0/scanners", nil, &out)
	return out, err
}

// GetScanner retrieves a scanner registration.
func (c *Client) GetScanner(ctx context.Context, id string) (*ScannerRegistration, error) {
	var out ScannerRegistration
	_, err := c.do(ctx, "GET", fmt.Sprintf("/api/v2.0/scanners/%s", id), nil, &out)
	return &out, err
}

// CreateScanner creates a scanner registration.
func (c *Client) CreateScanner(ctx context.Context, in ScannerRegistrationReq) (string, error) {
	resp, err := c.do(ctx, "POST", "/api/v2.0/scanners", &in, nil)
	if err != nil {
		return "", err
	}
	return extractLocationIDString(resp)
}

// UpdateScanner updates a scanner registration.
func (c *Client) UpdateScanner(ctx context.Context, id string, in ScannerRegistrationReq) error {
	_, err := c.do(ctx, "PUT", fmt.Sprintf("/api/v2.0/scanners/%s", id), &in, nil)
	return err
}

// DeleteScanner deletes a scanner registration.
func (c *Client) DeleteScanner(ctx context.Context, id string) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/v2.0/scanners/%s", id), nil, nil)
	if IsNotFound(err) {
		return nil
	}
	return err
}

// SetDefaultScanner sets the system default scanner registration.
func (c *Client) SetDefaultScanner(ctx context.Context, id string, isDefault bool) error {
	body := struct {
		IsDefault bool `json:"is_default"`
	}{
		IsDefault: isDefault,
	}
	_, err := c.do(ctx, "PATCH", fmt.Sprintf("/api/v2.0/scanners/%s", id), &body, nil)
	return err
}
