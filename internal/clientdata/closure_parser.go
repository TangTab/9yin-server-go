package clientdata

import (
	"bufio"
	"bytes"
	"path"
	"strings"
)

// dependencyKey 描述现代 composite/action 加载器所识别的资源引用。
// 将该解析器与文件系统索引分开维护，使得无需包文件
// 即可测试这些展开规则。
type dependencyKey struct {
	Key      string
	Relation string
	Kind     ResourceKind
}

var compositeDependencyKeys = map[string]dependencyKey{
	"action":     {Key: "Action", Relation: "action", Kind: ResourceAction},
	"model":      {Key: "Model", Relation: "model", Kind: ResourceModel},
	"main_model": {Key: "main_model", Relation: "main_model", Kind: ResourceModel},
}

var actionDependencyKeys = map[string]dependencyKey{
	"action_base_file":  {Key: "ACTION_BASE_FILE", Relation: "action_base", Kind: ResourceAction},
	"action_child_file": {Key: "ACTION_CHILD_FILE", Relation: "action_child", Kind: ResourceAction},
	"skeleton":          {Key: "Skeleton", Relation: "skeleton", Kind: ResourceSkeleton},
}

type parsedReference struct {
	Key      string
	Relation string
	Raw      string
	Kind     ResourceKind
}

// parseINIReferences 有意保留重复键。某些 action 文件含有多个
// ACTION_CHILD_FILE 条目，基于 map 的 INI 读取器会悄悄丢弃
// 其中除一条之外的全部条目。
func parseINIReferences(data []byte, keys map[string]dependencyKey) []parsedReference {
	var refs []parsedReference
	s := bufio.NewScanner(bytes.NewReader(stripUTF8BOM(data)))
	// 资源 INI 行可能包含很长的 action 列表。不要继承 Scanner
	// 较小的默认 token 上限。
	s.Buffer(make([]byte, 4096), 4<<20)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i < 1 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		spec, ok := keys[strings.ToLower(key)]
		if !ok {
			continue
		}
		value := trimINIValue(line[i+1:])
		if value == "" {
			continue
		}
		// 客户端数据中的子文件既以重复键的形式出现，也以竖线/逗号/分号
		// 列表的形式出现。拆分仅限于 CHILD_FILE，因此普通的
		// Windows 路径不会被改动。
		values := []string{value}
		if strings.EqualFold(spec.Key, "ACTION_CHILD_FILE") {
			values = splitChildFiles(value)
		}
		for _, value := range values {
			if value != "" {
				refs = append(refs, parsedReference{Key: spec.Key, Relation: spec.Relation, Raw: value, Kind: spec.Kind})
			}
		}
	}
	return refs
}

func stripUTF8BOM(b []byte) []byte {
	return bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
}

func trimINIValue(value string) string {
	value = strings.TrimSpace(value)
	// 游戏数据在空白字符后使用分号注释。CHILD_FILE 列表中的分号
	// 只有在前面没有空格时，才会在进入此处之前先被处理。
	if i := strings.Index(value, " ;"); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	return strings.Trim(strings.TrimSpace(value), "\"'")
}

func splitChildFiles(value string) []string {
	f := func(r rune) bool { return r == '|' || r == ',' || r == ';' }
	parts := strings.FieldsFunc(value, f)
	for i := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(parts[i]), "\"'")
	}
	return parts
}

// resolveReference 遵循从 FxModelAdv 观察到的 action 文件行为：带显式
// 虚拟根前缀的路径保持绝对路径；其余路径一律相对引用文件展开。
// 这里有意保留输入的分隔符风格，以便报告能够暴露 '/' 与 '\\'
// 两种查找失败的区别。
func resolveReference(from, raw string, virtualRoots map[string]bool) string {
	raw = strings.TrimSpace(strings.Trim(raw, "\"'"))
	if raw == "" {
		return ""
	}
	sep := `\`
	if strings.Contains(raw, "/") && !strings.Contains(raw, `\`) {
		sep = "/"
	}
	normalized := strings.ReplaceAll(raw, `\`, "/")
	first := strings.ToLower(strings.SplitN(strings.TrimLeft(normalized, "/"), "/", 2)[0])
	if strings.HasPrefix(raw, `/`) || strings.HasPrefix(raw, `\`) || virtualRoots[first] {
		normalized = strings.TrimLeft(normalized, "/")
	} else {
		base := path.Dir(strings.ReplaceAll(from, `\`, "/"))
		normalized = path.Join(base, normalized)
	}
	normalized = path.Clean(normalized)
	if normalized == "." {
		return ""
	}
	if sep == `\` {
		return strings.ReplaceAll(normalized, "/", `\`)
	}
	return normalized
}
