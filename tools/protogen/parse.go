package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// MsgDef 一条自定义消息定义。
type MsgDef struct {
	Name    string // lua 常量名（CLIENT_CUSTOMMSG_CHAT）
	Key     string // S2C 字符串消息名；C2S 为 ""
	ID      uint32 // C2S 数值 ID；S2C 为 0
	Desc    string // 注释简述（格式行以外的说明）
	Format  string // “格式：...”原文（字段权威描述）
	Fields  []byte // 尽力解析出的 TVarList 类型标签序列
	VarArgs bool   // 格式含可变参数（.../参数表）
}

var (
	defRe    = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+?)\s*$`)
	formatRe = regexp.MustCompile(`格式[：:]`)
	// 格式文本中的类型关键字 → TVarList 标签
	typeWords = []struct {
		word string
		tag  byte
	}{
		{"wstring", 7}, {"widestr", 7}, {"wide", 7},
		{"int64", 3},
		{"double", 5}, {"real", 5},
		{"float", 4},
		{"string", 6},
		{"object", 8}, {"obj", 8},
		{"unsigned", 2}, {"uint", 2}, {"int", 2}, {"byte", 2}, {"word", 2}, {"bool", 2},
	}
)

// toUTF8 编码自适应：合法 UTF-8 直通，否则 GBK 转码
// （testdata 快照为 UTF-8，NINEYIN_DATA 直读原版为 GBK）。
func toUTF8(raw []byte) ([]byte, error) {
	if utf8.Valid(raw) {
		return raw, nil
	}
	return simplifiedchinese.GBK.NewDecoder().Bytes(raw)
}

// parseDefineFile 解析一个 *_custom_define.lua。
// 结构约定：`--注释`（含“格式：”行）紧邻 `NAME = value` 定义行。
func parseDefineFile(path string) ([]MsgDef, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	utf8, err := toUTF8(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: GBK 转码失败: %w", path, err)
	}

	var defs []MsgDef
	var comments []string
	for _, line := range strings.Split(string(utf8), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			comments = append(comments, strings.TrimPrefix(trimmed, "--"))
			continue
		}
		m := defRe.FindStringSubmatch(trimmed)
		if m == nil {
			comments = comments[:0] // 非定义非注释行重置上下文
			continue
		}
		def := MsgDef{Name: m[1]}
		val := m[2]

		if key, err := strconv.Unquote(val); err == nil && strings.Contains(val, `"`) {
			def.Key = key // S2C 字符串名
		} else if n, err := strconv.ParseUint(val, 10, 32); err == nil {
			def.ID = uint32(n) // C2S 数值 ID
		} else {
			comments = comments[:0]
			continue // 非消息常量（如 TIPSTYPE_* 这类同文件杂项则仍会带 ID 0/空）
		}

		for _, c := range comments {
			if formatRe.MatchString(c) {
				def.Format = strings.TrimSpace(formatRe.ReplaceAllString(c, ""))
				def.Fields, def.VarArgs = parseFields(def.Format)
			} else if def.Desc == "" && !strings.HasPrefix(strings.TrimSpace(c), "--") {
				def.Desc = strings.TrimSpace(c)
			}
		}
		if def.Format != "" || def.Key != "" || def.ID != 0 {
			defs = append(defs, def)
		}
		comments = comments[:0]
	}
	return defs, nil
}

// parseFields 从格式文本尽力提取类型序列。
// 例："int msgid, int tipstype, string stringid, ...(参数表)"
//	→ [2,2,6], VarArgs=true
func parseFields(format string) (fields []byte, varArgs bool) {
	seg := strings.NewReplacer(",", " ", "，", " ", "(", " ", ")", " ", "（", " ", "）", " ").Replace(format)
	for _, tok := range strings.Fields(seg) {
		if strings.HasPrefix(tok, ".") || strings.Contains(tok, "…") {
			varArgs = true
			continue
		}
		lower := strings.ToLower(tok)
		for _, tw := range typeWords {
			if lower == tw.word {
				fields = append(fields, tw.tag)
				break
			}
		}
	}
	return fields, varArgs
}
