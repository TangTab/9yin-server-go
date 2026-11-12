package main

// S5（设计文档 §5.1）面板代理的 ETag/304 行为测试：
//   - 后端 ETag 透传到浏览器
//   - 客户端 If-None-Match 命中缓存条目 → 304 空体（不再打后端）
//   - 30s 目录微缓存去抖（多请求只打一次后端）
//   - manifest 代理路由存在

import (
	"net/http"
	"testing"
)

// etagFakeBackend 给 catalog 端点装上 ETag/INM 语义的假后端。
func etagFakeBackend(t *testing.T) *fakeGameBackend {
	t.Helper()
	backend := newFakeGameBackend(t)
	backend.catalogETag = `W/"test-catalog-v1"`
	return backend
}

func panelGetWithHeader(t *testing.T, url, headerName, headerValue string) (string, string, int) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if headerValue != "" {
		request.Header.Set(headerName, headerValue)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer response.Body.Close()
	body := make([]byte, 0, 512)
	buf := make([]byte, 512)
	for {
		n, readErr := response.Body.Read(buf)
		body = append(body, buf[:n]...)
		if readErr != nil {
			break
		}
	}
	return string(body), response.Header.Get("ETag"), response.StatusCode
}

func TestPanelCatalogETagPassthrough(t *testing.T) {
	backend := etagFakeBackend(t)
	panel := newPanel(t, backend)

	// 首次：200 + 后端 ETag 透传
	_, etag, status := panelGetWithHeader(t, panel.URL+"/api/cards", "", "")
	if status != http.StatusOK {
		t.Fatalf("first status=%d", status)
	}
	if etag != backend.catalogETag {
		t.Fatalf("etag=%q want %q", etag, backend.catalogETag)
	}

	// 微缓存窗口内带 INM：面板直接 304，不再打后端
	hitsBefore := backend.requests.Load()
	_, _, status2 := panelGetWithHeader(t, panel.URL+"/api/cards", "If-None-Match", etag)
	if status2 != http.StatusNotModified {
		t.Fatalf("INM hit status=%d want=304", status2)
	}
	if backend.requests.Load() != hitsBefore {
		t.Fatalf("INM hit must not reach backend (hits=%d want=%d)", backend.requests.Load(), hitsBefore)
	}

	// 不匹配的 INM → 200（缓存体）
	_, _, status3 := panelGetWithHeader(t, panel.URL+"/api/cards", "If-None-Match", `W/"other"`)
	if status3 != http.StatusOK {
		t.Fatalf("mismatched INM status=%d want=200", status3)
	}
}

func TestPanelCatalogMicroCache(t *testing.T) {
	backend := etagFakeBackend(t)
	panel := newPanel(t, backend)
	for i := 0; i < 3; i++ {
		if _, _, status := panelGetWithHeader(t, panel.URL+"/api/titles", "", ""); status != http.StatusOK {
			t.Fatalf("request %d status=%d", i, status)
		}
	}
	if got := backend.requests.Load(); got != 1 {
		t.Fatalf("backend hits=%d want 1 (30s micro-cache)", got)
	}
	// 不同查询串是不同缓存键
	if _, _, status := panelGetWithHeader(t, panel.URL+"/api/items?q=刀&view=1", "", ""); status != http.StatusOK {
		t.Fatalf("items query status=%d", status)
	}
	if got := backend.requests.Load(); got != 2 {
		t.Fatalf("distinct query must be distinct cache key (hits=%d want=2)", got)
	}
}

func TestPanelManifestRoute(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)
	body, status := panelGet(t, panel.URL+"/api/catalog/manifest")
	if status != http.StatusOK {
		t.Fatalf("manifest status=%d", status)
	}
	if len(body) == 0 {
		t.Fatal("manifest body empty")
	}
}
