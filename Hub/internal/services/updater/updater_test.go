package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckForUpdatesSemver(t *testing.T) {
	releases := []struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Draft   bool   `json:"draft"`
		PreRel  bool   `json:"prerelease"`
	}{
		{"hub/v1.0.0", "http://github.com/v1.0.0", false, false},
		{"hub/v2.1.0", "http://github.com/v2.1.0", false, false},
		{"hub/v2.1.1-rc1", "http://github.com/v2.1.1-rc1", false, true},
		{"cli/v3.0.0", "http://github.com/v3.0.0", false, false},
	}

	b, _ := json.Marshal(releases)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	defer ts.Close()

	origURL := githubAPIURL
	githubAPIURL = ts.URL
	defer func() { githubAPIURL = origURL }()
}
