package gmpanel

// jstest_test.go 前端子页逻辑回归（Node 在 PATH 才跑，缺席跳过）：真起
// 面板测试服务取 /api/gm/scenes，把嵌入的前端模块拷到临时目录，用最小
// 假 DOM（jstest/fakedom.mjs）真跑 panels/test.js 的子页（jstest/*.mjs）：
//   - scene_subpage.mjs「传送与场景」：下拉分组/重名/回退键、选驿站填
//     坐标、gm scene 命令行、接口失败回退；
//   - debug_subpage.mjs「调试工具」：怒气 / 移动参数的 gm rage / gm move
//     命令行。
//
// 面板此前零测试覆盖，而「面板列了场景、服务端解析不了」正是传送失败的
// 首发形态；命令行方言两端各写一份（test.js / gmcmd.go），这里钉对。

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSceneSubpageJS(t *testing.T) {
	runSubpageJS(t, "scene_subpage.mjs")
}

func TestDebugSubpageJS(t *testing.T) {
	runSubpageJS(t, "debug_subpage.mjs")
}

// runSubpageJS 跑包内 jstest/<script>（脚本的断言不过即非零退出）。
func runSubpageJS(t *testing.T, script string) {
	t.Helper()
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
	// 模块——脚本与数据分开放，共用的假 DOM fakedom.mjs 也按脚本目录解析）。
	// package.json type=module 让 .js 按 ESM 解析。
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

	scriptPath, err := filepath.Abs(filepath.Join("jstest", script))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, scriptPath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GM_JS_APP="+filepath.Join(dir, "app"),
		"GM_JS_SCENES="+filepath.Join(dir, "gm_scenes.json"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", script, err, out)
	}
	if !strings.Contains(string(out), "all assertions passed") {
		t.Fatalf("%s: 未见到通过标记\n%s", script, out)
	}
}
