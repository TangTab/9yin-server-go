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
// 嵌入资源（index.html / console.html / static/）确实能被
// /、/console.html 与 /static/ 提供。
func gmPanelTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(newGMMux(newGMHub(), nil))
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

// TestGMIndexIsLocalServerOverview 验证概览页与本服 GM API 的对话面。
// 2026-11-07 二次修订：页面新增「记录服务器」登记册（Java gm-web index
// 同款，localStorage 列表 + /api/manage/status 同源探活代理 + /api/panel-info
// 自述）后，网关时代禁令里仍成立的是 __s=/?s= 多服跳转参数——探活与
// 登记册是面板自身实现的端点，不再是网关转发残留。
func TestGMIndexIsLocalServerOverview(t *testing.T) {
	server := gmPanelTestServer(t)
	index, contentType, status := gmGetBody(t, server.URL+"/")
	if status != http.StatusOK {
		t.Fatalf("index status=%d want=%d", status, http.StatusOK)
	}
	if !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("index content-type=%q want text/html", contentType)
	}
	for _, reference := range []string{
		`fetch("/api/sessions")`,
		`location.href='console.html'`,
		"服务器概览",
		"记录服务器",
		`fetch("/api/manage/status?s="`,
		`fetch("/api/panel-info")`,
	} {
		if !strings.Contains(index, reference) {
			t.Fatalf("index.html missing %q", reference)
		}
	}
	for _, stale := range []string{"__s=", "console.html?s="} {
		if strings.Contains(index, stale) {
			t.Fatalf("index.html still references gateway-era %q", stale)
		}
	}
}

// TestGMConsolePageServedAndAdapted 验证控制台页可经 /console.html 取到，
// 功能板块全量保留（服务端未实现的命令点击时提示等待支持），且不再有
// 网关时代的 ?s= 入口门禁与 __s 转发参数。
func TestGMConsolePageServedAndAdapted(t *testing.T) {
	server := gmPanelTestServer(t)
	page, contentType, status := gmGetBody(t, server.URL+"/console.html")
	if status != http.StatusOK {
		t.Fatalf("console status=%d want=%d", status, http.StatusOK)
	}
	if !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("console content-type=%q want text/html", contentType)
	}
	for _, reference := range []string{
		`href="/static/console.css"`,
		`src="/static/console-core.js"`,
		`src="/static/console-features.js"`,
		"九阴本地 GM",
		// 功能面抽查：面板功能全量保留，后续服务端实现后直接可用。
		`onclick="preset('yanjing')"`, // 常用传送
		`id="recast-target-slot"`,     // 洗练
		`id="clone-config"`,           // 副本
		`id="mount-result"`,           // 坐骑
		`id="guanyin"`,                // 官银
		`id="faction"`,                // 门派
	} {
		if !strings.Contains(page, reference) {
			t.Fatalf("console.html missing %q", reference)
		}
	}
	for _, stale := range []string{"/api/manage/", "__s=", "location.search"} {
		if strings.Contains(page, stale) {
			t.Fatalf("console.html still references gateway-era %q", stale)
		}
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
		"css":     {"/static/console.css", "text/css", "--accent"},
		"core-js": {"/static/console-core.js", "javascript", "async function command("},
		"feat-js": {"/static/console-features.js", "javascript", "function preset(key)"},
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
	for _, path := range []string{"/nope", "/static/missing.js", "/static/gm.js"} {
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

// TestGMWeatherCatalogAPI 面板天气下拉目录（set_weather 闭名单同源）：
// 大写字段名 Key/Label/TXID 与 Java gm-web 目录同形；静态表无需 DB。
func TestGMWeatherCatalogAPI(t *testing.T) {
	server := gmPanelTestServer(t)
	body, _, status := gmGetBody(t, server.URL+"/api/weather")
	if status != http.StatusOK {
		t.Fatalf("weather status=%d want=%d", status, http.StatusOK)
	}
	var rows []struct {
		Key   string `json:"Key"`
		Label string `json:"Label"`
		TXID  string `json:"TXID"`
	}
	if err := json.Unmarshal([]byte(body), &rows); err != nil {
		t.Fatalf("weather body not JSON: %v (%s)", err, body)
	}
	if len(rows) != 49 {
		t.Fatalf("weather rows=%d want 49", len(rows))
	}
	first := rows[0]
	if first.Key != "light_rain_001" || first.Label == "" || first.TXID != "" {
		t.Fatalf("first row=%+v", first)
	}
}
