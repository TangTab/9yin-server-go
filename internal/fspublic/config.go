package fspublic

import (
	"strconv"
	"strings"
)

// Config 是 GamePublic.ini 方言（集群公共数据服版）。键名大小写不敏感，
// 缺省值对齐 Lua/GamePublic/main.lua：Inner 127.0.0.1:16003、
// 广播 127.255.255.255:15000、Echo 127.0.0.1:15003（固定值，无公式——
// 区别于 member 的 +MemberID 缺省）。
type Config struct {
	DistrictID     uint32
	ServerID       uint32
	PublicLogicDll string // [Main] PublicLogicDll（缺省 FsPubLogic.dll）

	InnerAddr string
	InnerPort int // 缺省 16003

	BroadAddr string // 缺省 127.255.255.255
	BroadPort int    // 缺省 15000

	EchoAddr string
	EchoPort int // 缺省 15003

	// 集群外连目标（官方由 CInterConnect Check*Connect 重连；Go 版显式
	// 注入，空=不外连，仅受面）。ident 见 wire.go 命名空间。
	WorldAddr string // world 服务地址
	StoreAddrs []string // store 服务地址（按序）
}

// DefaultConfig 官方缺省（main.lua 字面值）。
func DefaultConfig() Config {
	return Config{
		DistrictID:     1,
		ServerID:       1,
		PublicLogicDll: "FsPubLogic.dll",
		InnerAddr:      "127.0.0.1",
		InnerPort:      16003,
		BroadAddr:      "127.255.255.255",
		BroadPort:      15000,
		EchoAddr:       "127.0.0.1",
		EchoPort:       15003,
	}
}

// ParseGamePublicINI 解析 GamePublic.ini 文本（大写系集群方言）。
func ParseGamePublicINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
	section := ""
	for line := range eachLine(string(data)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		val := strings.TrimSpace(line[eq+1:])
		cfg.set(section, key, val)
	}
	return cfg, nil
}

func (c *Config) set(section, key, val string) {
	switch section {
	case "main":
		switch key {
		case "districtid":
			c.DistrictID = parseU32(val, c.DistrictID)
		case "serverid":
			c.ServerID = parseU32(val, c.ServerID)
		case "publiclogicdll":
			c.PublicLogicDll = val
		}
	case "inner":
		switch key {
		case "addr":
			c.InnerAddr = val
		case "port":
			c.InnerPort = parseInt(val, c.InnerPort)
		}
	case "broadcast":
		switch key {
		case "addr":
			c.BroadAddr = val
		case "port":
			c.BroadPort = parseInt(val, c.BroadPort)
		}
	case "echo":
		switch key {
		case "addr":
			c.EchoAddr = val
		case "port":
			c.EchoPort = parseInt(val, c.EchoPort)
		}
	}
	// 未知键忽略（前向兼容）
}

func eachLine(s string) func(func(string) bool) {
	return func(yield func(string) bool) {
		for len(s) > 0 {
			i := strings.IndexByte(s, '\n')
			var line string
			if i < 0 {
				line, s = s, ""
			} else {
				line, s = s[:i], s[i+1:]
			}
			if !yield(line) {
				return
			}
		}
	}
}

func parseU32(val string, def uint32) uint32 {
	v, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return def
	}
	return uint32(v)
}

func parseInt(val string, def int) int {
	v, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return v
}
