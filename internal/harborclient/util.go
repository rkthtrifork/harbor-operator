package harborclient

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
)

func extractLocationID(resp *http.Response) (int, error) {
	loc := resp.Header.Get("Location")
	if loc == "" {
		return 0, fmt.Errorf("no Location header")
	}
	id, err := strconv.Atoi(path.Base(loc))
	if err != nil {
		return 0, fmt.Errorf("parse Location %q: %w", loc, err)
	}
	return id, nil
}
