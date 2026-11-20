package fssoloworld

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// LoadScenesINI 官方 main.lua load_scene_config（soloworld/solounite 同源）：
// scenes.ini（节=场景号，键 Member/Script/Config/Clonable）→ SceneSpec
// 装配面（分词统一走 fskit.ParseItems＝官方 CIniFile Build 方言，按文件
// 序成组）。文件缺失/读败返回 err（调用方决定跳过与否）。
func LoadScenesINI(path string) ([]SceneSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var specs []SceneSpec
	sec := ""
	kv := map[string]string{}
	flush := func() {
		if sec == "" {
			return
		}
		id, err := strconv.Atoi(strings.TrimSpace(sec))
		if err != nil {
			return
		}
		spec := SceneSpec{
			Index:  id,
			Member: atoiOr(kv["member"], 0),
			Script: kv["script"],
			Config: kv["config"],
		}
		if spec.Script == "" {
			spec.Script = "scene" // 官方 ReadString 第三参缺省
		}
		switch strings.ToLower(strings.TrimSpace(kv["clonable"])) {
		case "1", "true":
			spec.Clonable = true // 副本原型场景
		}
		specs = append(specs, spec)
	}
	_ = fskit.ParseItems(data, func(s, k, v string) error {
		if s == "" { // 节前裸键不入表（同旧解析器口径）
			return nil
		}
		if s != sec {
			flush()
			sec, kv = s, map[string]string{}
		}
		kv[strings.ToLower(k)] = v
		return nil
	})
	flush()
	return specs, nil
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return n
}

// FormatScenesCount 装载结果呈现行（官方无此输出；cmd 冒烟辅助）。
func FormatScenesCount(n int) string { return fmt.Sprintf("load scene config: %d scenes", n) }
