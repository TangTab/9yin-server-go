package fssologlobal

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// defaultGameGlobalINI 是内置默认配置（官方 game_global.ini 方言，
// Outer/Echo 改为回环可用缺省）：未指定路径且当前目录没有
// game_global.ini 时释放到当前目录并采用（已存在的绝不覆盖）——
// 与 cmd/solo game_entry.ini / cmd/auction game_auction.ini 同一范式。
//
//go:embed game_global.ini
var defaultGameGlobalINI []byte

// GlobalConfig 是官方 game_global.ini（game_global.exe →
// FsSoloGlobal.dll 全局名字服务器）的投影。方言与
// auction.AuctionConfig 同族：`[节]` 头、`键=值`、`//`（及 `;`/`#`）
// 整行注释、[Module] 裸行模块列表、未知节/键忽略（前向兼容）。
//
// 缺省值来自官方 Lua\sologlobal\main.lua（ReadString 第三参）：
// Outer.Addr=127.0.0.1、Outer.Port=4003、Echo.Addr=127.0.0.1、
// Echo.Port=15012（官方快照 ini 本身把 Outer 写成 192.168.33.70:4005，
// 属部署值而非缺省）。
type GlobalConfig struct {
	MainEntity    string   // [System] MainEntity（呈现用）
	MainScript    string   // [System] MainScript（呈现用）
	Modules       []string // [Module] 裸行模块列表（呈现用）
	ScriptPath    string   // [Script] Path（呈现用）
	ConnectString string   // [Main] ConnectString（官方 OLEDB 方言；Go 侧仅呈现/告警）
	OuterAddr     string   // 空=官方缺省 127.0.0.1
	OuterPort     int      // 0=官方缺省 4003
	EchoAddr      string   // 空=官方缺省 127.0.0.1
	EchoPort      int      // 0=官方缺省 15012
}

// IsOLEDB 报告 ConnectString 是否为官方 OLEDB 方言
// （Provider= 前缀）——此时 Go 复刻无法直连，须 -mysql-dsn。
func (c GlobalConfig) IsOLEDB() bool {
	return strings.HasPrefix(strings.TrimSpace(c.ConnectString), "Provider=")
}

// ParseGameGlobalINI 解析官方 game_global.ini 文本。GBK 官方文件亦可
// 解析（关心的键均为 ASCII）。
func ParseGameGlobalINI(data []byte) (GlobalConfig, error) {
	var cfg GlobalConfig
	section := ""
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "//") ||
			strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			if strings.EqualFold(section, "Module") {
				// [Module] 节是官方方言的裸行模块列表。
				if name := strings.TrimSpace(line); name != "" {
					cfg.Modules = append(cfg.Modules, name)
				}
				continue
			}
			return cfg, fmt.Errorf("game_global ini 第 %d 行无 '='：%q", i+1, line)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if err := cfg.set(section, key, value); err != nil {
			return cfg, fmt.Errorf("game_global ini 第 %d 行 %s/%s: %w", i+1, section, key, err)
		}
	}
	if cfg.OuterPort == 0 {
		cfg.OuterPort = 4003 // 官方 main.lua ReadInteger 缺省
	} else if cfg.OuterPort < 0 || cfg.OuterPort > 65535 {
		return cfg, fmt.Errorf("game_global ini [Outer] Port 非法：%d", cfg.OuterPort)
	}
	if cfg.EchoPort == 0 {
		cfg.EchoPort = 15012 // 官方 main.lua 本地缺省
	} else if cfg.EchoPort < 0 || cfg.EchoPort > 65535 {
		return cfg, fmt.Errorf("game_global ini [Echo] Port 非法：%d", cfg.EchoPort)
	}
	if cfg.OuterAddr == "" {
		cfg.OuterAddr = "127.0.0.1"
	}
	if cfg.EchoAddr == "" {
		cfg.EchoAddr = "127.0.0.1"
	}
	return cfg, nil
}

func (c *GlobalConfig) set(section, key, value string) error {
	switch section + "/" + key {
	case "Outer/Port", "Echo/Port":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("须为整数，得 %q", value)
		}
		if section == "Outer" {
			c.OuterPort = n
		} else {
			c.EchoPort = n
		}
	case "System/MainEntity":
		c.MainEntity = value
	case "System/MainScript":
		c.MainScript = value
	case "Script/Path":
		c.ScriptPath = value
	case "Main/ConnectString":
		c.ConnectString = value
	case "Outer/Addr":
		c.OuterAddr = value
	case "Echo/Addr":
		c.EchoAddr = value
	default:
		// 未知键忽略（前向兼容）；Sleep/Reload 等官方方言键不接线。
	}
	return nil
}

// LoadGameGlobalINI 读取 path 的 game_global.ini；不存在时把内置默认
// 释放到该路径（已存在的绝不覆盖）并返回释放内容与 created=true。
// 与 auction.LoadGameAuctionINI 同范式。
func LoadGameGlobalINI(path string) (cfg GlobalConfig, data []byte, created bool, err error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		cfg, err = ParseGameGlobalINI(raw)
		return cfg, raw, false, err
	}
	if !os.IsNotExist(err) {
		return cfg, nil, false, err
	}
	if werr := os.WriteFile(path, defaultGameGlobalINI, 0o644); werr != nil {
		return cfg, nil, false, fmt.Errorf("释放内置默认 %s: %w", path, werr)
	}
	cfg, err = ParseGameGlobalINI(defaultGameGlobalINI)
	return cfg, defaultGameGlobalINI, true, err
}
