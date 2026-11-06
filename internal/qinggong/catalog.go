// Package qinggong 加载当前客户端的轻功定义，供服务端
// 校验并扣除 C2S USE_QINGGONG (216) 请求。
package qinggong

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Definition 是 QGDefine.ini 某个小节中与服务端相关的部分。
// Para 字段描述客户端的移动表现；Consume 是客户端请求该轻功 ID 时
// 服务端必须扣除的资源量。
type Definition struct {
	ID       string
	Type     int32
	UseType  int32
	Consume  int32
	BufferID string
}

type Catalog struct{ byID map[string]Definition }

func Load(path string) (*Catalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	catalog := &Catalog{byID: make(map[string]Definition)}
	var current *Definition
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 16<<10), 1<<20)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			id := strings.TrimSpace(line[1 : len(line)-1])
			if id == "" {
				return nil, fmt.Errorf("%s:%d: empty section", path, lineNumber)
			}
			entry := Definition{ID: id}
			catalog.byID[id] = entry
			current = &entry
			continue
		}
		if current == nil {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		entry := catalog.byID[current.ID]
		switch strings.TrimSpace(key) {
		case "Consume":
			amount, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
			if parseErr != nil || amount < 0 {
				return nil, fmt.Errorf("%s:%d: invalid Consume %q", path, lineNumber, value)
			}
			entry.Consume = int32(amount)
		case "Type":
			value, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
			if parseErr != nil {
				return nil, fmt.Errorf("%s:%d: invalid Type", path, lineNumber)
			}
			entry.Type = int32(value)
		case "UseType":
			value, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
			if parseErr != nil {
				return nil, fmt.Errorf("%s:%d: invalid UseType", path, lineNumber)
			}
			entry.UseType = int32(value)
		case "BufferID":
			entry.BufferID = strings.TrimSpace(value)
		}
		catalog.byID[current.ID] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(catalog.byID) == 0 {
		return nil, fmt.Errorf("%s: no qinggong definitions", path)
	}
	return catalog, nil
}

func (catalog *Catalog) Lookup(id string) (Definition, bool) {
	if catalog == nil {
		return Definition{}, false
	}
	entry, ok := catalog.byID[id]
	return entry, ok
}

func (catalog *Catalog) Count() int {
	if catalog == nil {
		return 0
	}
	return len(catalog.byID)
}
