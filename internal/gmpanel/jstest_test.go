package gmpanel

// jstest_test.go 前端子页逻辑回归（Node 在 PATH 才跑，缺席跳过）：真起
// 面板测试服务取 /api/gm/scenes，把嵌入的前端模块拷到临时目录，用最小
// 假 DOM 跑 panels/test.js 的「传送与场景」子页（jstest/scene_subpage.mjs）
// ——下拉分组/重名/回退键、选驿站填坐标、gm scene 命令行、接口失败回退。
// 面板此前零测试覆盖，而「面板列了场景、服务端解析不了」正是传送失败的
// 首发形态。

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSceneSubpageJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node 不在 PATH：跳过前端子页逻辑测试")
	}
	world := fakeWorldEcho(t)
	s := newTestPanel(t, world, nil)
	code, body := get(t, "http://"+s.Addr().String()+"/api/gm/scenes")
	if code != 200 {
		t.Fatalf("scenes %d: %v", code, body)
	}
	scenes, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	// 前端模块拷到临时目录；脚本留在包内 jstest/（解释器按脚本目录加载
	// 模块——脚本与数据分开放）。package.json type=module 让 .js 按 ESM 解析。
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range []string{"ui.js", "api.js", "panels/test.js"} {
		content, err := assets.ReadFile("web/static/app/" + rel)
		if err != nil {
			t.Fatal(err)
		}
		write("app/"+rel, string(content))
	}
	write("package.json", `{"type":"module"}`)
	write("gm_scenes.json", string(scenes))

	script, err := filepath.Abs(filepath.Join("jstest", "scene_subpage.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GM_JS_APP="+filepath.Join(dir, "app"),
		"GM_JS_SCENES="+filepath.Join(dir, "gm_scenes.json"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scene subpage js: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "all assertions passed") {
		t.Fatalf("scene subpage js: 未见到通过标记\n%s", out)
	}
}
