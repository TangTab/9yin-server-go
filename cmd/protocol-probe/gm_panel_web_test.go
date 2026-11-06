package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// gmPanelTestServer 挂载与生产完全相同的 GM 路由，用于验证
// 嵌入资源（index.html / gm.css / gm.js）确实能被 / 与 /static/ 提供。
func gmPanelTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(newGMMux(newGMHub()))
	t.Cleanup(server.Close)
	return server
}

func gmGetBody(t *testing.T, url string) (string, string, int) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("GET %s: read body: %v", url, err)
	}
	return string(body), response.Header.Get("Content-Type"), response.StatusCode
}

func TestGMIndexReferencesExternalAssets(t *testing.T) {
	server := gmPanelTestServer(t)
	index, contentType, status := gmGetBody(t, server.URL+"/")
	if status != http.StatusOK {
		t.Fatalf("index status=%d want=%d", status, http.StatusOK)
	}
	if !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("index content-type=%q want text/html", contentType)
	}
	for _, reference := range []string{
		`href="/static/gm.css"`,
		`src="/static/gm.js"`,
		"九阴本地 GM",
	} {
		if !strings.Contains(index, reference) {
			t.Fatalf("index.html missing %q", reference)
		}
	}
	// 页面不再携带内联样式或脚本。
	if strings.Contains(index, "<style>") || strings.Contains(index, "<script>") {
		t.Fatal("index.html still contains inline <style>/<script>")
	}
}

func TestGMServesEmbeddedStaticAssets(t *testing.T) {
	server := gmPanelTestServer(t)
	cases := map[string]struct {
		url         string
		contentType string
		marker      string
	}{
		// Go 1.23 的 mime 表把 .js 标为 application/javascript，
		// 与 text/javascript 等价，浏览器均接受。
		"css": {"/static/gm.css", "text/css", "body{font:15px"},
		"js":  {"/static/gm.js", "javascript", "async function refresh()"},
	}
	for name, testCase := range cases {
		body, contentType, status := gmGetBody(t, server.URL+testCase.url)
		if status != http.StatusOK {
			t.Fatalf("%s status=%d want=%d", name, status, http.StatusOK)
		}
		if !strings.Contains(contentType, testCase.contentType) {
			t.Fatalf("%s content-type=%q want to contain %q", name, contentType, testCase.contentType)
		}
		if !strings.Contains(body, testCase.marker) {
			t.Fatalf("%s body missing marker %q", name, testCase.marker)
		}
	}
}

func TestGMRejectsUnknownPaths(t *testing.T) {
	server := gmPanelTestServer(t)
	for _, path := range []string{"/nope", "/static/missing.js"} {
		_, _, status := gmGetBody(t, server.URL+path)
		if status != http.StatusNotFound {
			t.Fatalf("GET %s status=%d want=%d", path, status, http.StatusNotFound)
		}
	}
}

func TestGMSessionsAPIReturnsJSON(t *testing.T) {
	server := gmPanelTestServer(t)
	body, contentType, status := gmGetBody(t, server.URL+"/api/sessions")
	if status != http.StatusOK {
		t.Fatalf("sessions status=%d want=%d", status, http.StatusOK)
	}
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("sessions content-type=%q want application/json", contentType)
	}
	var sessions []gmPlayerState
	if err := json.Unmarshal([]byte(body), &sessions); err != nil {
		t.Fatalf("sessions body is not a JSON array: %v", err)
	}
}
