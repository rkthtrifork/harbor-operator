package harborclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestListProjectsFetchesAllPages(t *testing.T) {
	t.Parallel()

	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2.0/projects" {
			http.NotFound(w, r)
			return
		}
		page := r.URL.Query().Get("page")
		if r.URL.Query().Get("page_size") != strconv.Itoa(defaultListPageSize) {
			http.Error(w, "unexpected page size", http.StatusBadRequest)
			return
		}
		pages = append(pages, page)
		w.Header().Set("X-Total-Count", "101")
		w.Header().Set("Content-Type", "application/json")

		projects := make([]Project, 0, defaultListPageSize)
		switch page {
		case "1":
			for i := 1; i <= defaultListPageSize; i++ {
				projects = append(projects, Project{ProjectID: i, Name: fmt.Sprintf("project-%d", i)})
			}
		case "2":
			projects = append(projects, Project{ProjectID: 101, Name: "project-101"})
		default:
			http.Error(w, "unexpected page", http.StatusBadRequest)
			return
		}
		if err := json.NewEncoder(w).Encode(projects); err != nil {
			t.Fatalf("encode projects: %v", err)
		}
	}))
	defer server.Close()

	client := New(server.URL, "user", "pass")
	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}
	if len(projects) != 101 {
		t.Fatalf("ListProjects returned %d projects, want 101", len(projects))
	}
	if got, want := fmt.Sprint(pages), "[1 2]"; got != want {
		t.Fatalf("requested pages %s, want %s", got, want)
	}
}
