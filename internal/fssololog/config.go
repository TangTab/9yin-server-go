package fssololog

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// defaultGameLogINI 是内置默认配置（官方 game_log.ini 方言；[Echo]
// 节为 main.lua 实读键的回环缺省展开）：未指定路径且当前目录没有
// game_log.ini 时释放到当前目录并采用（已存在的绝不覆盖）——与
// cmd/solo game_entry.ini / cmd/global game_global.ini 同一范式。
//
//go:embed game_log.ini
var defaultGameLogINI []byte

// LogConfig 是官方 game_log.ini（game_log.exe → FsSoloLog.dll 日志
// 汇聚服务器）的投影。方言与 GlobalConfig 同族：`[节]` 头、`键=值`、
// `//`（及 `;`/`#`）整行注释、[Module] 裸行模块列表、未知节/键忽略
// （前向兼容）。官方快照 ini 的 [Main] 为空节——部署期才填。
type LogConfig struct {
	MainEntity string   // [System] MainEntity（呈现用；缺省 SoloLog）
	MainScript string   // [System] MainScript（呈现用；缺省 main）
	Modules    []string // [Module] 裸行模块列表（呈现用）
	ScriptPath string   // [Script] Path（呈现用）
	Sleep      int      // [System] Sleep（呈现用；壳方言键不接线）
	EchoAddr   string   // 空=官方缺省 127.0.0.1
	EchoPort   int      // 0=官方缺省 15011
}

// ParseGameLogINI 解析官方 game_log.ini 文本（分词统一走
// fskit.ParseItems＝官方 CIniFile Build 方言；[Module] 裸行模块列表
// 按官方 key-only 条目语义收集）。GBK 官方文件亦可解析（关心的键均
// 为 ASCII）。
func ParseGameLogINI(data []byte) (LogConfig, error) {
	var cfg LogConfig
	if err := fskit.ParseItems(data, func(sec, key, value string) error {
		// [Module] 节是官方方言的裸行模块列表（key-only 条目）。
		if strings.EqualFold(sec, "Module") && value == "" {
			cfg.Modules = append(cfg.Modules, key)
			return nil
		}
		return cfg.set(sec, key, value)
	}); err != nil {
		return cfg, fmt.Errorf("game_log ini %w", err)
	}
	if cfg.EchoPort == 0 {
		cfg.EchoPort = 15011 // 官方 main.lua ReadInteger 缺省
	} else if cfg.EchoPort < 0 || cfg.EchoPort > 65535 {
		return cfg, fmt.Errorf("game_log ini [Echo] Port 非法：%d", cfg.EchoPort)
	}
	if cfg.EchoAddr == "" {
		cfg.EchoAddr = "127.0.0.1"
	}
	if cfg.MainEntity == "" {
		cfg.MainEntity = "SoloLog"
	}
	if cfg.MainScript == "" {
		cfg.MainScript = "main"
	}
	return cfg, nil
}

func (c *LogConfig) set(section, key, value string) error {
	switch section + "/" + key {
	case "System/MainEntity":
		c.MainEntity = value
	case "System/MainScript":
		c.MainScript = value
	case "System/Sleep":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("须为整数，得 %q", value)
		}
		c.Sleep = n
	case "Script/Path":
		c.ScriptPath = value
	case "Echo/Addr":
		c.EchoAddr = value
	case "Echo/Port":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("须为整数，得 %q", value)
		}
		c.EchoPort = n
	default:
		// 未知键忽略（前向兼容）；Reload 等官方方言键不接线。
	}
	return nil
}

// LoadGameLogINI 读取 path 的 game_log.ini；不存在时把内置默认释放
// 到该路径（已存在的绝不覆盖）并返回释放内容与 created=true。
// 与 fssologlobal.LoadGameGlobalINI 同范式。
func LoadGameLogINI(path string) (cfg LogConfig, data []byte, created bool, err error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		cfg, err = ParseGameLogINI(raw)
		return cfg, raw, false, err
	}
	if !os.IsNotExist(err) {
		return cfg, nil, false, err
	}
	if werr := os.WriteFile(path, defaultGameLogINI, 0o644); werr != nil {
		return cfg, nil, false, fmt.Errorf("释放内置默认 %s: %w", path, werr)
	}
	cfg, err = ParseGameLogINI(defaultGameLogINI)
	return cfg, defaultGameLogINI, true, err
}
