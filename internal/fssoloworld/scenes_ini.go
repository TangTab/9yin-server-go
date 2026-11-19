package fssoloworld

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LoadScenesINI 官方 main.lua load_scene_config（soloworld/solounite 同源）：
// scenes.ini（节=场景号，键 Member/Script/Config/Clonable）→ SceneSpec
// 装配面。文件缺失/读败返回 err（调用方决定跳过与否）。
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
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "//") ||
			strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			sec = strings.TrimSpace(line[1 : len(line)-1])
			kv = map[string]string{}
			continue
		}
		if i := strings.IndexByte(line, '='); i > 0 && sec != "" {
			kv[strings.ToLower(strings.TrimSpace(line[:i]))] = strings.TrimSpace(line[i+1:])
		}
	}
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
