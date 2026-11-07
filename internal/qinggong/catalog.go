// Package qinggong 加载当前客户端的轻功定义，供服务端
// 校验并扣除 C2S USE_QINGGONG (216) 请求。
package qinggong

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Definition 是 QGDefine.ini 某个小节中与服务端相关的部分。
// Para 字段描述客户端的移动表现；Consume 是客户端请求该轻功 ID 时
// 服务端必须扣除的资源量。ExistBufID 是能力门槛（qinggong_4 凌空踏虚/
// qinggong_5 扶摇步法指向 buf_DHadd_2rd/3rd）：未挂该 buff 的使用被拒，
// 已挂则免 Consume（native-menu 反编译实证的免费分支）；旧表缺该键
// 为空串=无门槛，按现状放行。
type Definition struct {
	ID         string
	Type       int32
	UseType    int32
	Consume    int32
	BufferID   string
	ExistBufID string
}

type Catalog struct{ byID map[string]Definition }

// Load 直接从文件系统读取；资源经由 MPQ 资源仓的调用方
// 应先取字节再走 LoadFromReader。
func Load(path string) (*Catalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return LoadFromReader(file, path)
}

// LoadFromReader 从任意来源解析（source 仅用于报错定位）。
func LoadFromReader(r io.Reader, source string) (*Catalog, error) {
	catalog := &Catalog{byID: make(map[string]Definition)}
	var current *Definition
	scanner := bufio.NewScanner(r)
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
				return nil, fmt.Errorf("%s:%d: empty section", source, lineNumber)
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
				return nil, fmt.Errorf("%s:%d: invalid Consume %q", source, lineNumber, value)
			}
			entry.Consume = int32(amount)
		case "Type":
			value, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
			if parseErr != nil {
				return nil, fmt.Errorf("%s:%d: invalid Type", source, lineNumber)
			}
			entry.Type = int32(value)
		case "UseType":
			value, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
			if parseErr != nil {
				return nil, fmt.Errorf("%s:%d: invalid UseType", source, lineNumber)
			}
			entry.UseType = int32(value)
		case "BufferID":
			entry.BufferID = strings.TrimSpace(value)
		case "ExistBufID":
			entry.ExistBufID = strings.TrimSpace(value)
		}
		catalog.byID[current.ID] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(catalog.byID) == 0 {
		return nil, fmt.Errorf("%s: no qinggong definitions", source)
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
