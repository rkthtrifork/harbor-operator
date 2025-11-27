package harborclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPError wraps a non-2xx response.
type HTTPError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("harbor API %d – %s", e.StatusCode, e.Message)
}

// Convenience testers.
func IsNotFound(err error) bool  { return isStatus(err, http.StatusNotFound) }
func IsConflict(err error) bool  { return isStatus(err, http.StatusConflict) }
func IsForbidden(err error) bool { return isStatus(err, http.StatusForbidden) }

func isStatus(err error, code int) bool {
	if he, ok := err.(*HTTPError); ok {
		return he.StatusCode == code
	}
	return false
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Username   string
	Password   string
}

var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

func New(baseURL, user, pass string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: defaultHTTPClient,
		Username:   user,
		Password:   pass,
	}
}

func (c *Client) do(ctx context.Context, method, relURL string, in, out any) (*http.Response, error) {
	// request body
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(b)
	}

	// build request
	req, err := http.NewRequestWithContext(ctx, method,
		c.BaseURL+relURL, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")

	// perform
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// non-2xx → wrap in *HTTPError
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    strings.TrimSpace(string(msg)),
		}
	}

	// decode
	if out != nil {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return nil, err
		}
	}
	return resp, nil
}
