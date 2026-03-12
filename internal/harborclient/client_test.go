package harborclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoAllowsEmptyJSONBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/empty" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, "user", "pass")
	var out map[string]any
	if _, err := client.do(context.Background(), http.MethodGet, "/empty", nil, &out); err != nil {
		t.Fatalf("do returned error for empty body: %v", err)
	}
}
