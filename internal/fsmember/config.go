package fsmember

import (
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// Config 是 GameMember.ini 方言（集群场景服版）。键名大小写不敏感，
// 缺省值对齐 Lua/GameMember/main.lua 的公式：Inner 16300+MemberID、
// Echo 15300+MemberID、广播 127.255.255.255:15000。
type Config struct {
	DistrictID       uint32
	ServerID         uint32
	MemberID         uint32
	CollideRootPath  string
	LogicModules     []string // [Logic] Module0..N（空 Path=当前目录）
	ScenesMemberFile string   // 静态场景分配表（Res\scenes_member.txt 方言）

	InnerAddr string
	InnerPort int // 缺省 16300+MemberID

	BroadAddr string // 缺省 127.255.255.255
	BroadPort int    // 缺省 15000

	EchoAddr string
	EchoPort int // 缺省 15300+MemberID

	// 集群外连目标（官方由 CInterConnect 配置/服务发现；Go 版显式注入，
	// 空=不外连，仅受面）。ident 见 wire.go 命名空间。
	WorldAddr  string // ident 0
	PublicAddr string // ident 1
	StoreAddrs []string // ident 4..7（按序）
	EntryAddrs []string // ident 8..0x17（按序）
}

// DefaultConfig 官方缺省（main.lua 公式）。
func DefaultConfig() Config {
	return Config{
		DistrictID: 1, ServerID: 1, MemberID: 0,
		InnerAddr: "127.0.0.1", InnerPort: 16300,
		BroadAddr: "127.255.255.255", BroadPort: 15000,
		EchoAddr: "127.0.0.1", EchoPort: 15300,
	}
}

// ParseGameMemberINI 解析 GameMember.ini 文本（大写系集群方言；分词统一
// 走 fskit.ParseItems＝官方 CIniFile Build 方言，节键归一小写交 set）。
func ParseGameMemberINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
	if err := fskit.ParseItems(data, func(sec, key, val string) error {
		return cfg.set(strings.ToLower(sec), strings.ToLower(key), val)
	}); err != nil {
		return cfg, err
	}
	if cfg.InnerPort == 16300 {
		cfg.InnerPort += int(cfg.MemberID)
	}
	if cfg.EchoPort == 15300 {
		cfg.EchoPort += int(cfg.MemberID)
	}
	return cfg, nil
}

func (c *Config) set(section, key, val string) error {
	switch section {
	case "main":
		switch key {
		case "districtid":
			return setU32(&c.DistrictID, val)
		case "serverid":
			return setU32(&c.ServerID, val)
		case "memberid":
			return setU32(&c.MemberID, val)
		case "colliderootpath":
			c.CollideRootPath = val
			return nil
		}
	case "logic":
		switch {
		case key == "modules":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			_ = n // 模块名按 Module0..N 收集，数量仅作参考（官方同宽容）
		case strings.HasPrefix(key, "module"):
			if val != "" {
				c.LogicModules = append(c.LogicModules, val)
			}
		}
	case "inner":
		switch key {
		case "addr":
			c.InnerAddr = val
			return nil
		case "port":
			return setInt(&c.InnerPort, val)
		}
	case "broadcast":
		switch key {
		case "addr":
			c.BroadAddr = val
			return nil
		case "port":
			return setInt(&c.BroadPort, val)
		}
	case "echo":
		switch key {
		case "addr":
			c.EchoAddr = val
			return nil
		case "port":
			return setInt(&c.EchoPort, val)
		}
	}
	return nil // 未知键忽略（前向兼容）
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

func setU32(dst *uint32, val string) error {
	v, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return err
	}
	*dst = uint32(v)
	return nil
}

func setInt(dst *int, val string) error {
	v, err := strconv.Atoi(val)
	if err != nil {
		return err
	}
	*dst = v
	return nil
}

// ParseScenesMemberText 解析 Res\scenes_member.txt（静态场景分配表）：
// 「Member=<id>: <场景id CSV>」→ memberID → 场景 id 列表。
func ParseScenesMemberText(data []byte) map[uint32][]uint32 {
	out := map[uint32][]uint32{}
	for line := range eachLine(string(data)) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Member=") {
			continue
		}
		body := line[len("Member="):]
		colon := strings.IndexByte(body, ':')
		if colon < 0 {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimSpace(body[:colon]), 10, 32)
		if err != nil {
			continue
		}
		for _, part := range strings.Split(body[colon+1:], ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			sid, err := strconv.ParseUint(part, 10, 32)
			if err != nil {
				continue
			}
			out[uint32(id)] = append(out[uint32(id)], uint32(sid))
		}
	}
	return out
}

// MemberScenesFor 返回本 member 的静态场景 id 列表（无则空）。
func MemberScenesFor(data []byte, memberID uint32) []uint32 {
	return ParseScenesMemberText(data)[memberID]
}
