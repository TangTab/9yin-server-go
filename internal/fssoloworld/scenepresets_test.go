package fssoloworld

// scenepresets_test.go GM 场景目的地解析单测：补充键落点锚值 + 场景表
//（scenedata）回落 + 预设全集可解析守卫（中文别名表 / 面板下拉逐项）+
// 场景表场景的完整切图链。

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/scenedata"
)

// TestResolveSchool09Destination 移花宫：BornCJ 只登记 school01~08，别名/
// 面板预设的 school09 经场景表解析（表缺此场景 = gm scene 回 unknown key
// 无法传送）。落点逐值锚 relive_info.ini [force] force_yihua（场景号 281）。
func TestResolveSchool09Destination(t *testing.T) {
	dest, ok := resolveSceneDestination("school09")
	if !ok || dest.Resource != "school09" || dest.Config != `ini\scene\school09_YiHuaGong` {
		t.Fatalf("school09 resolve = %+v ok=%v", dest, ok)
	}
	if want := [4]float64{340.16, 47.979, 657.284, 2.588}; dest.Position != want {
		t.Fatalf("school09 position = %v, want %v", dest.Position, want)
	}
	alias, ok := resolveSceneDestination("移花宫")
	if !ok || alias != dest {
		t.Fatalf("移花宫 alias resolve = %+v ok=%v, want %+v", alias, ok, dest)
	}
}

// TestResolveSceneTableScenes 场景表回落：客户端有地形的每个世界场景都可
// 解析；BornCJ 登记的场景落点仍是出生点（既有行为不变），其余落点 = 表内
// 默认驿站 / NPC 落点。
func TestResolveSceneTableScenes(t *testing.T) {
	for _, sc := range scenedata.Scenes() {
		dest, ok := resolveSceneDestination(sc.Resource)
		if !ok {
			t.Errorf("%s does not resolve", sc.Resource)
			continue
		}
		if dest.Resource != sc.Resource || dest.Config != sc.Config {
			t.Errorf("%s resolve = %q/%q, want %q/%q", sc.Resource, dest.Config, dest.Resource, sc.Config, sc.Resource)
		}
		if dest.Position == [4]float64{} {
			t.Errorf("%s landing at origin", sc.Resource)
		}
		want := [4]float64{sc.Landing.X, sc.Landing.Y, sc.Landing.Z, sc.Landing.Orient}
		if born, ok := rolestate.FindSceneOption(sc.Resource); ok {
			want = born.Position
		}
		if dest.Position != want {
			t.Errorf("%s landing = %v, want %v", sc.Resource, dest.Position, want)
		}
	}
}

// TestResolveNewSceneSamples 此前 GM 列表完全没有的场景（八派之外的门派 /
// 江湖场景）按短名与唯一中文名解析；重名走显式别名。
func TestResolveNewSceneSamples(t *testing.T) {
	cases := []struct {
		key      string
		resource string
		config   string
	}{
		{"school24", "school24", `ini\scene\school24_kunlun`},
		{"昆仑派", "school24", `ini\scene\school24_kunlun`},
		{"SCHOOL10", "school10", `ini\scene\school10_taohuadao`},
		{"快活岛", "scene11", `ini\scene\scene11_kuaihuodao`},
		{"scene38", "scene38", `ini\scene\scene38`},
		{"向阳坡", "scene04", `ini\scene\scene04_XiangYangPo`},   // 官方写法
		{"襄阳坡", "scene04", `ini\scene\scene04_XiangYangPo`},   // 旧写法兼容
		{"凌霄城", "scene06", `ini\scene\scene06_LingXiaoCheng`}, // 重名：别名定 scene06 非 scene25
	}
	for _, c := range cases {
		dest, ok := resolveSceneDestination(c.key)
		if !ok || dest.Resource != c.resource || dest.Config != c.config {
			t.Errorf("%q resolve = %+v ok=%v, want %s %s", c.key, dest, ok, c.resource, c.config)
		}
	}
	// 无驿站场景：落点 = 摆放档复活药师旁（X 偏 2m）。
	if dest, _ := resolveSceneDestination("scene38"); dest.Position[0] != -824.238 || dest.Position[1] != 62.864 {
		t.Errorf("scene38 landing = %v", dest.Position)
	}
	if _, ok := resolveSceneDestination("念萝坝"); ok {
		t.Fatal("ambiguous name 念萝坝 (school15/scene05) must not resolve by name")
	}
	if _, ok := resolveSceneDestination("school16"); ok {
		t.Fatal("school16 (no client terrain) must not resolve")
	}
}

// TestScenePresetKeys `gm scene` 裸调用的清单：场景表逐场景一行，非 BornCJ
// 场景标默认落点。
func TestScenePresetKeys(t *testing.T) {
	out := scenePresetKeys()
	lines := strings.Split(out, "\n")
	var sceneLines int
	for _, sc := range scenedata.Scenes() {
		name := sc.Name
		if name == "" {
			name = sc.Resource
		}
		want := scenedata.CategoryName(sc.Category) + " " + name + "(" + sc.Resource + ")"
		_, born := rolestate.FindSceneOption(sc.Resource)
		found := false
		for _, line := range lines {
			if strings.HasPrefix(line, want) {
				found = true
				if !born && !strings.Contains(line, " → ") {
					t.Errorf("line %q lacks landing for non-BornCJ scene", line)
				}
				if born && strings.Contains(line, " → ") {
					t.Errorf("line %q marks landing on BornCJ scene", line)
				}
			}
		}
		if !found {
			t.Errorf("no line for %s", sc.Resource)
		}
		sceneLines++
	}
	if !strings.Contains(out, "门派 移花宫(school09) → 移花宫眠楼") {
		t.Errorf("school09 line missing:\n%s", out)
	}
	if !strings.Contains(out, "keys: born01 ") || !strings.Contains(out, " born_single_01\n") {
		t.Errorf("keys line missing BornCJ/extra keys:\n%s", out)
	}
}

// TestScenePresetKeysAllResolve 预设全集可解析守卫：中文别名表逐项 + 面板
// 「传送与场景」下拉 SCENE_PRESETS 逐项（gmpanel test.js——JS 表无 Go
// 镜像，按文件读）都须解析出目的地。任一缺席=面板列了场景而服务端回
// unknown key 无法传送。
func TestScenePresetKeysAllResolve(t *testing.T) {
	for name := range sceneAliases {
		if _, ok := resolveSceneDestination(name); !ok {
			t.Errorf("alias %q does not resolve", name)
		}
	}
	raw, err := os.ReadFile(filepath.Join("..", "gmpanel", "web", "static", "app", "panels", "test.js"))
	if err != nil {
		t.Fatalf("panel presets source: %v", err)
	}
	block := regexp.MustCompile(`(?s)const SCENE_PRESETS = \[(.*?)\];`).FindSubmatch(raw)
	if block == nil {
		t.Fatal("SCENE_PRESETS block not found in gmpanel test.js")
	}
	pairs := regexp.MustCompile(`\[\s*"([^"]+)"\s*,\s*"([^"]*)"\s*\]`).FindAllSubmatch(block[1], -1)
	if len(pairs) == 0 {
		t.Fatal("SCENE_PRESETS has no [key, label] pairs")
	}
	for _, p := range pairs {
		if _, ok := resolveSceneDestination(string(p[1])); !ok {
			t.Errorf("panel preset %q (%s) does not resolve", p[1], p[2])
		}
	}
}

// TestSceneSwitchToTableScene 场景表场景的完整切图链（真实 testdata）：此前
// GM 传不到的三类各取一个——在客户端 scenes.ini 里的（school10）、不在的
// （school24：开图无全局段号，走降级路径）、落点取摆放 NPC 的（scene38）。
// 0x0C → 0x0B（载荷含本次 Config/Resource）→ 0x0D → 0x09 后重入簇以 0x1F
// 开头且落点 = 表内默认落点，NPC 首拍 = 该场景摆放目录的物化条数。
func TestSceneSwitchToTableScene(t *testing.T) {
	prevGap := sceneSwitchGap
	sceneSwitchGap = time.Millisecond
	t.Cleanup(func() { sceneSwitchGap = prevGap })

	for _, key := range []string{"school10", "school24", "scene38"} {
		s, pl := newCustomTestServer(t)
		dest, ok := resolveSceneDestination(key)
		if !ok {
			t.Fatalf("%s unresolved", key)
		}
		frames := captureFrames(t)
		if err := s.transferPlayer(pl, dest, "test"); err != nil {
			t.Fatalf("%s transfer: %v", key, err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for s.pendingSceneSwitch(pl) == nil && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		pre := (*frames)[:]
		if len(pre) < 3 || pre[0][0] != 0x0C || pre[1][0] != 0x0B || pre[2][0] != 0x0D {
			t.Fatalf("%s pre-ready ops = % x, want 0c 0b 0d", key, opOf(pre))
		}
		if !bytes.Contains(pre[1], append([]byte(dest.Config), 0)) || !bytes.Contains(pre[1], append([]byte(dest.Resource), 0)) {
			t.Fatalf("%s 0x0B lacks config/resource: % x", key, pre[1])
		}
		if pl.roleState.Scene != dest.Config || pl.roleState.Resource != dest.Resource {
			t.Fatalf("%s role state = %q / %q", key, pl.roleState.Scene, pl.roleState.Resource)
		}
		dir := s.resolveSceneDir(dest.Config, dest.Resource)
		spawns := len(s.sceneNPCCatalog(dir).spawns)

		mark := len(*frames)
		s.clientReady(pl)
		reentry := (*frames)[mark:]
		if len(reentry) == 0 || reentry[0][0] != 0x1F {
			t.Fatalf("%s reentry must lead with 0x1F", key)
		}
		// 0x1F：[op][obj u32][own u32][x f32][y f32][z f32][orient f32]
		loc := reentry[0]
		for i := 0; i < 4; i++ {
			got := float64(math.Float32frombits(binary.LittleEndian.Uint32(loc[9+4*i:])))
			if math.Abs(got-dest.Position[i]) > 0.001 {
				t.Fatalf("%s 0x1F coord[%d] = %v, want %v", key, i, got, dest.Position[i])
			}
		}
		npc := 0
		for _, f := range reentry {
			if f[0] == 0x28 {
				npc++
			}
		}
		if spawns == 0 || npc != spawns {
			t.Fatalf("%s NPC frames = %d, catalog spawns = %d (dir %s)", key, npc, spawns, dir)
		}
	}
}
