package fssolopublic

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// defaultGamePublicINI 是内置默认配置（官方 game_public.ini 方言）：
// 未指定路径且当前目录没有 game_public.ini 时释放并采用（已存在的
// 绝不覆盖）——与 cmd/global game_global.ini 同范式。
//
//go:embed game_public.ini
var defaultGamePublicINI []byte

// Config 是官方 game_public.ini（game_public.exe → FsSoloPublic.dll）
// 的投影。缺省值来自官方 lua\solopublic\main.lua（ReadString/
// ReadInteger 第三参）：Inner 127.0.0.1:0（端口 0=系统分配，Start 后
// GetListenerPort 回填）、Broadcast 127.255.255.255:15000、Echo
// 127.0.0.1:15003、SaveMinutes 20（实体 ctor 缺省 10）。
type Config struct {
	MainEntity     string   // [System] MainEntity（呈现用，官方 SoloPublic）
	MainScript     string   // [System] MainScript（呈现用）
	Modules        []string // [Module] 裸行模块列表（呈现用）
	ScriptPath     string   // [Script] Path（呈现用）
	DistrictID     uint32   // [Main] DistrictID（缺省 1）
	ServerID       uint32   // [Main] ServerID（缺省 1；快照实配 11）
	PublicLogicDll string   // [Main] PublicLogicDll（Go 版仅呈现）
	SaveMinutes    int      // [Main] SaveMinutes（缺省 20；0=不周期存盘）
	InnerAddr      string   // [Inner] Addr（缺省 127.0.0.1）
	InnerPort      int      // [Inner] Port（缺省 0=系统分配回填）
	BroadAddr      string   // [Broadcast] Addr（缺省 127.255.255.255）
	BroadPort      int      // [Broadcast] Port（缺省 15000）
	EchoAddr       string   // [Echo] Addr（缺省 127.0.0.1）
	EchoPort       int      // [Echo] Port（缺省 15003）
}

// ParseGamePublicINI 解析官方 game_public.ini 文本（GBK 文件亦可：
// 关心的键均为 ASCII）。
func ParseGamePublicINI(data []byte) (Config, error) {
	var cfg Config
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
				if name := strings.TrimSpace(line); name != "" {
					cfg.Modules = append(cfg.Modules, name)
				}
				continue
			}
			return cfg, fmt.Errorf("game_public ini 第 %d 行无 '='：%q", i+1, line)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if err := cfg.set(section, key, value); err != nil {
			return cfg, fmt.Errorf("game_public ini 第 %d 行 %s/%s: %w", i+1, section, key, err)
		}
	}
	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) set(section, key, value string) error {
	switch section + "/" + key {
	case "System/MainEntity":
		c.MainEntity = value
	case "System/MainScript":
		c.MainScript = value
	case "Script/Path":
		c.ScriptPath = value
	case "Main/DistrictID", "Main/ServerID", "Main/SaveMinutes":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("须为整数，得 %q", value)
		}
		switch key {
		case "DistrictID":
			c.DistrictID = uint32(n)
		case "ServerID":
			c.ServerID = uint32(n)
		default:
			c.SaveMinutes = n
		}
	case "Main/PublicLogicDll":
		c.PublicLogicDll = value
	case "Inner/Addr":
		c.InnerAddr = value
	case "Inner/Port", "Broadcast/Port", "Echo/Port":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("须为整数，得 %q", value)
		}
		if n < 0 || n > 65535 {
			return fmt.Errorf("端口非法：%d", n)
		}
		switch section {
		case "Inner":
			c.InnerPort = n
		case "Broadcast":
			c.BroadPort = n
		default:
			c.EchoPort = n
		}
	case "Broadcast/Addr":
		c.BroadAddr = value
	case "Echo/Addr":
		c.EchoAddr = value
	default:
		// 未知键忽略（前向兼容）；Sleep/Reload 等官方方言键不接线。
	}
	return nil
}

// applyDefaults 补官方 main.lua 本地缺省。
func (c *Config) applyDefaults() {
	if c.DistrictID == 0 {
		c.DistrictID = 1
	}
	if c.ServerID == 0 {
		c.ServerID = 1
	}
	if c.SaveMinutes == 0 {
		c.SaveMinutes = 20
	}
	if c.InnerAddr == "" {
		c.InnerAddr = "127.0.0.1"
	}
	if c.BroadAddr == "" {
		c.BroadAddr = "127.255.255.255"
	}
	if c.BroadPort == 0 {
		c.BroadPort = 15000
	}
	if c.EchoAddr == "" {
		c.EchoAddr = "127.0.0.1"
	}
	if c.EchoPort == 0 {
		c.EchoPort = 15003
	}
}

// LoadGamePublicINI 读取 path 的 game_public.ini；不存在时释放内置默认
// （已存在的绝不覆盖）并返回 created=true。
func LoadGamePublicINI(path string) (cfg Config, data []byte, created bool, err error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		cfg, err = ParseGamePublicINI(raw)
		return cfg, raw, false, err
	}
	if !os.IsNotExist(err) {
		return cfg, nil, false, err
	}
	if werr := os.WriteFile(path, defaultGamePublicINI, 0o644); werr != nil {
		return cfg, nil, false, fmt.Errorf("释放内置默认 %s: %w", path, werr)
	}
	cfg, err = ParseGamePublicINI(defaultGamePublicINI)
	return cfg, defaultGamePublicINI, true, err
}

// AppName 官方 main.lua：`"public_" .. nx_string(server_id)`。
func AppName(serverID uint32) string { return fmt.Sprintf("public_%d", serverID) }
