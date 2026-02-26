package harborclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rkthtrifork/harbor-operator/internal/metrics"
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
	return NewWithHTTPClient(baseURL, user, pass, defaultHTTPClient)
}

func NewWithHTTPClient(baseURL, user, pass string, httpClient *http.Client) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: httpClient,
		Username:   user,
		Password:   pass,
	}
}

func (c *Client) do(ctx context.Context, method, relURL string, in, out any) (resp *http.Response, err error) {
	start := time.Now()
	endpointLabel := normalizeEndpoint(relURL)
	defer func() {
		if resp == nil || resp.Body == nil {
			return
		}
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

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
	resp, err = c.HTTPClient.Do(req)
	if err != nil {
		metrics.ObserveHarborRequest(method, endpointLabel, 0, time.Since(start).Seconds())
		return nil, err
	}

	// non-2xx → wrap in *HTTPError
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		metrics.ObserveHarborRequest(method, endpointLabel, resp.StatusCode, time.Since(start).Seconds())
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    strings.TrimSpace(string(msg)),
		}
	}

	// decode
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			metrics.ObserveHarborRequest(method, endpointLabel, resp.StatusCode, time.Since(start).Seconds())
			return nil, err
		}
	}
	metrics.ObserveHarborRequest(method, endpointLabel, resp.StatusCode, time.Since(start).Seconds())
	return resp, nil
}

var numberPathSegment = regexp.MustCompile(`/\\d+`)

func normalizeEndpoint(relURL string) string {
	endpoint := relURL
	if idx := strings.Index(endpoint, "?"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	endpoint = numberPathSegment.ReplaceAllString(endpoint, "/:id")
	return endpoint
}
