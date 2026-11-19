package fssolomember

import (
	"strconv"
	"strings"
)

// Config 是 game_member.ini 方言（小写系 solo 场景服版；§一）。
// 缺省公式对齐 Lua/solomember/main.lua：Inner 口 0（官方临时口回填，
// 集群公式为 16300+MemberID）、Broadcast 127.255.255.255:15000、
// Echo 15300+MemberID。
type Config struct {
	DistrictID      uint32
	ServerID        uint32
	MemberID        uint32
	BindCore        int // [Main]（<0 不绑定）
	DebugHalt       bool
	CollideRootPath string

	LogicModules []string // [Logic] Module0..N（fsgame.dll 玩法层）

	InnerAddr string
	InnerPort int // 0 = 官方 CreateListener 临时口回填语义

	BroadAddr string // 缺省 127.255.255.255
	BroadPort int    // 缺省 15000

	EchoAddr string // 缺省 127.0.0.1
	EchoPort int    // 缺省 15300+MemberID
}

// DefaultConfig 官方 ctor/main.lua 缺省（§二/§三）。
func DefaultConfig() Config {
	return Config{
		DistrictID: 1, ServerID: 1, MemberID: 0,
		BindCore:  -1,
		DebugHalt: true,
		InnerAddr: "127.0.0.1", InnerPort: 0,
		BroadAddr: "127.255.255.255", BroadPort: 15000,
		EchoAddr: "127.0.0.1", EchoPort: 15300,
	}
}

// ParseGameMemberINI 解析 game_member.ini 文本（solo 方言；注释行 `//` 与
// 分号均容忍——快照 ini 以 `//` 注释）。
func ParseGameMemberINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
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
			continue // [Module] 裸行由装载器承担，此处不收
		}
		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		val := strings.TrimSpace(line[eq+1:])
		cfg.set(section, key, val)
	}
	if cfg.EchoPort == 15300 {
		cfg.EchoPort += int(cfg.MemberID)
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
		case "memberid":
			c.MemberID = parseU32(val, c.MemberID)
		case "bindcore":
			if n, err := strconv.Atoi(val); err == nil {
				c.BindCore = n
			}
		case "debughalt":
			c.DebugHalt = !strings.EqualFold(val, "false")
		case "colliderootpath":
			c.CollideRootPath = val
		}
	case "logic":
		if key != "modules" && strings.HasPrefix(key, "module") && val != "" {
			c.LogicModules = append(c.LogicModules, val)
		}
	case "inner":
		switch key {
		case "addr":
			c.InnerAddr = val
		case "port":
			if n, err := strconv.Atoi(val); err == nil {
				c.InnerPort = n
			}
		}
	case "broadcast":
		switch key {
		case "addr":
			c.BroadAddr = val
		case "port":
			if n, err := strconv.Atoi(val); err == nil {
				c.BroadPort = n
			}
		}
	case "echo":
		switch key {
		case "addr":
			c.EchoAddr = val
		case "port":
			if n, err := strconv.Atoi(val); err == nil {
				c.EchoPort = n
			}
		}
	}
}

func parseU32(val string, fallback uint32) uint32 {
	n, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return fallback
	}
	return uint32(n)
}
