// Package npcfunc 读取服务端的 Npc_Func 表，并公开构建非通用菜单所需的
// 逐 NPC 功能配置。
package npcfunc

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Binding struct {
	NPCID  string
	FuncID int
	Extra  map[string]string
}

type Registry struct {
	byNPC map[string][]Binding
}

var (
	propertyTag = regexp.MustCompile(`(?is)<Property\b[^>]*>`)
	attribute   = regexp.MustCompile(`(?i)\b([A-Za-z][A-Za-z0-9_]*)\s*=\s*"([^"]*)"`)
)

// Load 直接解析 ASCII 属性名和属性值。这是有意为之：
// 在本安装中 Npc_Func.xml 为 GBK 编码，而协议决策所需的 ID 均为 ASCII。
// 这样可以避免转换无关的中文描述。
func Load(path string) (*Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("npcfunc: read %s: %w", path, err)
	}
	reg := &Registry{byNPC: make(map[string][]Binding)}
	for _, tag := range propertyTag.FindAllString(string(raw), -1) {
		attrs := make(map[string]string)
		for _, pair := range attribute.FindAllStringSubmatch(tag, -1) {
			attrs[strings.ToLower(pair[1])] = pair[2]
		}
		npcID := strings.TrimSpace(attrs["npcid"])
		funcText := strings.TrimSpace(attrs["funcid"])
		if npcID == "" || funcText == "" {
			continue
		}
		funcID, err := strconv.Atoi(funcText)
		if err != nil {
			return nil, fmt.Errorf("npcfunc: NpcID %q invalid FuncID %q: %w", npcID, funcText, err)
		}
		if enabled, exists := attrs["enablefunc"]; exists && (enabled == "0" || strings.EqualFold(enabled, "false")) {
			continue
		}
		extra := make(map[string]string)
		for key, value := range attrs {
			switch key {
			case "id", "name", "npcid", "npcname", "funcid", "enablefunc", "enabled":
			default:
				extra[key] = value
			}
		}
		key := normalize(npcID)
		reg.byNPC[key] = append(reg.byNPC[key], Binding{NPCID: npcID, FuncID: funcID, Extra: extra})
	}
	if len(reg.byNPC) == 0 {
		return nil, fmt.Errorf("npcfunc: %s contains no NpcID/FuncID bindings", path)
	}
	return reg, nil
}

func (r *Registry) FuncsForNPC(configID string) []Binding {
	if r == nil {
		return nil
	}
	source := r.byNPC[normalize(configID)]
	out := append([]Binding(nil), source...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].FuncID < out[j].FuncID })
	return out
}

func (r *Registry) Count() int {
	if r == nil {
		return 0
	}
	return len(r.byNPC)
}

func normalize(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
