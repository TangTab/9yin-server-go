package sharedata

// idres.go 键值行档（text/chineses 下的 .idres：key=value 行）。
//
// 口径与构建器 tools/datagen/npc/text.go 对齐：UTF-8 BOM / GBK 自适应；
// 行首 // 与 ; 是注释；行内多个 '=' 时首个为界；键值都裁首尾空白；重复键
// 首见胜出并计数。

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// ReadIDres 单档键值表（键 → 值）。返回条目数与本档内的重复键数。
func ReadIDres(path string) (map[string]string, int, error) {
	raw, err := res.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	data, err := DecodeText(raw)
	if err != nil {
		return nil, 0, fmt.Errorf("sharedata: 解码 %s: %w", path, err)
	}
	entries := map[string]string{}
	conflicts := 0
	MergeIDresLines(data, entries, &conflicts)
	return entries, conflicts, nil
}

// MergeIDresLines 把一份键值行档的内容并入 into（首见胜出，重复键计入
// conflicts）。entries 为 nil 时原样返回。
func MergeIDresLines(data []byte, into map[string]string, conflicts *int) {
	if into == nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			continue
		}
		if _, exists := into[key]; exists {
			if conflicts != nil {
				*conflicts++
			}
			continue
		}
		into[key] = strings.TrimSpace(line[eq+1:])
	}
}

// ReadIDresDir 目录下全部 .idres（按文件名排序）并入一张键值表。
func ReadIDresDir(dir string) (map[string]string, int, int, error) {
	entries, err := res.ReadDir(dir)
	if err != nil {
		return nil, 0, 0, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".idres") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	into := map[string]string{}
	conflicts, files := 0, 0
	for _, name := range names {
		part, dup, err := ReadIDres(filepath.Join(dir, name))
		if err != nil {
			return nil, files, conflicts, err
		}
		files++
		conflicts += dup
		for key, value := range part {
			if _, exists := into[key]; exists {
				conflicts++
				continue
			}
			into[key] = value
		}
	}
	return into, files, conflicts, nil
}
