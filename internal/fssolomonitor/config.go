package fssolomonitor

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// defaultGameMonitorINI 是内置默认配置（官方 game_monitor.ini 方言，
// ConnectString 置空、Outer/Echo 用回环缺省）：未指定路径且当前目录
// 没有 game_monitor.ini 时释放并采用（已存在绝不覆盖）——与
// fssologlobal/fssolostore 同一范式。
//
//go:embed game_monitor.ini
var defaultGameMonitorINI []byte

// defaultMonitorLimitINI 是官方 Res\MonitorLimit.ini 快照原文
// （2014-03-22，含 [System]/[SysItemLimit]/[RoleItemLimit]/
// [RoleCapitalLimit] 四节）。cmd 侧 -limit-ini 未命中时释放。
//
//go:embed monitor_limit.ini
var defaultMonitorLimitINI []byte

// Config 是官方 game_monitor.ini（game_monitor.exe → FsSoloMonitor.dll）
// 的投影。缺省值：Outer=127.0.0.1:4008（官方 ini 实配；lua 读值缺省
// 4003 系 FsSoloGlobal 复制残留）、Echo=127.0.0.1:15012（lua 本地
// 缺省——官方 ini 无 [Echo] 节）。
type Config struct {
	MainEntity    string   // [System] MainEntity（呈现用）
	MainScript    string   // [System] MainScript（呈现用）
	Modules       []string // [Module] 裸行模块列表（呈现用）
	ScriptPath    string   // [Script] Path（呈现用）
	ServerID      int      // [System] ServerId（缺省 0；AppName=monitor_%d）
	ConnectString string   // [Main] ConnectString（官方 OLEDB 方言；仅呈现）
	OuterAddr     string
	OuterPort     int
	EchoAddr      string // 空=缺省 127.0.0.1
	EchoPort      int    // 0=缺省 15012
}

// AppName 官方 "monitor_%d"（ServerId）。
func (c Config) AppName() string {
	return "monitor_" + strconv.Itoa(c.ServerID)
}

// IsOLEDB 报告 ConnectString 是否为官方 OLEDB 方言
// （Provider= 前缀）——Go 复刻无法直连，须 MySQL DSN。
func (c Config) IsOLEDB() bool {
	return strings.HasPrefix(strings.TrimSpace(c.ConnectString), "Provider=")
}

// ParseGameMonitorINI 解析官方 game_monitor.ini 文本（分词统一走
// fskit.ParseItems＝官方 CIniFile Build 方言；[Module] 裸行模块列表
// 按官方 key-only 条目语义收集；关心的键均 ASCII，GBK 原文件亦可解析）。
func ParseGameMonitorINI(data []byte) (Config, error) {
	var cfg Config
	if err := fskit.ParseItems(data, func(sec, key, value string) error {
		// [Module] 节是官方方言的裸行模块列表（key-only 条目；
		// '#' 行官方 Build 不吃，漏成 key-only 条目，这里挡掉）。
		if strings.EqualFold(sec, "Module") && value == "" && !strings.HasPrefix(key, "#") {
			cfg.Modules = append(cfg.Modules, key)
			return nil
		}
		return cfg.set(sec, key, value)
	}); err != nil {
		return cfg, fmt.Errorf("game_monitor ini %w", err)
	}
	if cfg.OuterPort == 0 {
		cfg.OuterPort = 4008 // 官方 game_monitor.ini 实配
	} else if cfg.OuterPort < 0 || cfg.OuterPort > 65535 {
		return cfg, fmt.Errorf("game_monitor ini [Outer] Port 非法：%d", cfg.OuterPort)
	}
	if cfg.EchoPort == 0 {
		cfg.EchoPort = 15012 // 官方 lua 本地缺省
	} else if cfg.EchoPort < 0 || cfg.EchoPort > 65535 {
		return cfg, fmt.Errorf("game_monitor ini [Echo] Port 非法：%d", cfg.EchoPort)
	}
	if cfg.OuterAddr == "" {
		cfg.OuterAddr = "127.0.0.1"
	}
	if cfg.EchoAddr == "" {
		cfg.EchoAddr = "127.0.0.1"
	}
	return cfg, nil
}

func (c *Config) set(section, key, value string) error {
	switch section + "/" + key {
	case "Outer/Port", "Echo/Port", "System/ServerId":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("须为整数，得 %q", value)
		}
		switch section {
		case "Outer":
			c.OuterPort = n
		case "Echo":
			c.EchoPort = n
		default:
			c.ServerID = n
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

// LoadGameMonitorINI 读取 path；不存在时释放内置默认（已存在绝不
// 覆盖）并返回 created=true。与 fssologlobal.LoadGameGlobalINI 同范式。
func LoadGameMonitorINI(path string) (cfg Config, data []byte, created bool, err error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		cfg, err = ParseGameMonitorINI(raw)
		return cfg, raw, false, err
	}
	if !os.IsNotExist(err) {
		return cfg, nil, false, err
	}
	if werr := os.WriteFile(path, defaultGameMonitorINI, 0o644); werr != nil {
		return cfg, nil, false, fmt.Errorf("释放内置默认 %s: %w", path, werr)
	}
	cfg, err = ParseGameMonitorINI(defaultGameMonitorINI)
	return cfg, defaultGameMonitorINI, true, err
}

// LimitConfig 是官方 Res\MonitorLimit.ini → LoadSystemConfig@0x100a5a80
// 的字段映射（docs §8.1）。注意官方失配行为：代码从 [system]（小写）
// 读 Livepoint/DiffCapital/ExchangeAmount/RoleNum/OnlineRoleNum 五键，
// 而官方 ini 把它们写在大写 [System] 节且 CReadIni 大小写敏感 →
// 五键恒取缺省；本结构保持同一语义（见 ParseMonitorLimitINI）。
type LimitConfig struct {
	CheckTimerInterval  int   // [System]，缺省 1200（下发两表，按秒消费）
	FaculityLimit       int   // [System]，缺省 900000
	LivepointLimit      int   // [system].FaculityLimit（键串复用，官方笔误），缺省 1000
	DiffCapitalLimit    int64 // [system]，缺省 15000000
	ExchangeAmountLimit int   // [system]，缺省 0 → RoleData.SetTradeCounterLimit
	RoleNumLimit        int   // [system]，缺省 10 → LoginDetail.SetAmountLimit
	OnlineRoleNumLimit  int   // [system]，缺省 3 → LoginDetail.SetOnlineAmountLimit

	SysItemLimits     map[string]int // [SysItemLimit] → CreateItem 表每物品阈值
	RoleItemLimits    map[string]int // [RoleItemLimit] → RoleData 表每物品阈值
	RoleCapitalLimits [8]int64       // [RoleCapitalLimit] CapitalTypeN
}

// DefaultLimitConfig 官方 LoadSystemConfig 各 ReadInteger 缺省值。
func DefaultLimitConfig() LimitConfig {
	return LimitConfig{
		CheckTimerInterval:  1200,
		FaculityLimit:       900000,
		LivepointLimit:      1000,
		DiffCapitalLimit:    15000000,
		ExchangeAmountLimit: 0,
		RoleNumLimit:        10,
		OnlineRoleNumLimit:  3,
		SysItemLimits:       map[string]int{},
		RoleItemLimits:      map[string]int{},
	}
}

// ParseMonitorLimitINI 按官方 CReadIni 语义解析 MonitorLimit.ini：
// 节名/键名均大小写敏感（官方哈希表如此——[System] 与 [system] 是
// 两个节），值 atoi 容错。返回的 LimitConfig 已含缺省值。
func ParseMonitorLimitINI(data []byte) (LimitConfig, error) {
	cfg := DefaultLimitConfig()
	sections := parseCaseSensitiveINI(data)
	readInt := func(section, key string, def int) int {
		if v, ok := sections[section][key]; ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				return n
			}
			// atoi 容错：非数字前缀取前导数字，否则 0（C 语义近似）。
			return cAtoi(v)
		}
		return def
	}
	cfg.CheckTimerInterval = readInt("System", "CheckTimerInterval", cfg.CheckTimerInterval)
	cfg.FaculityLimit = readInt("System", "FaculityLimit", cfg.FaculityLimit)
	// 官方笔误：LivepointLimit 实际读 [system].FaculityLimit（键串复用）。
	cfg.LivepointLimit = readInt("system", "FaculityLimit", cfg.LivepointLimit)
	cfg.DiffCapitalLimit = int64(readInt("system", "DiffCapitalLimit", int(cfg.DiffCapitalLimit)))
	cfg.ExchangeAmountLimit = readInt("system", "ExchangeAmountLimit", cfg.ExchangeAmountLimit)
	cfg.RoleNumLimit = readInt("system", "RoleNumLimit", cfg.RoleNumLimit)
	cfg.OnlineRoleNumLimit = readInt("system", "OnlineRoleNumLimit", cfg.OnlineRoleNumLimit)
	for _, kv := range []struct {
		section string
		dst     map[string]int
	}{
		{"SysItemLimit", cfg.SysItemLimits},
		{"RoleItemLimit", cfg.RoleItemLimits},
	} {
		for k, v := range sections[kv.section] {
			kv.dst[k] = cAtoi(v)
		}
	}
	for k, v := range sections["RoleCapitalLimit"] {
		if !strings.HasPrefix(k, "CapitalType") {
			continue
		}
		// 官方 GetCapitalType=取键末字符 atoi（"CapitalType12"→2）。
		t := lastCharAtoi(k)
		if t < 0 || t >= len(cfg.RoleCapitalLimits) {
			continue
		}
		cfg.RoleCapitalLimits[t] = int64(cAtoi(v))
	}
	return cfg, nil
}

// LoadMonitorLimitINI 读取 path 的 MonitorLimit.ini；不存在时释放内置
// 官方快照（已存在绝不覆盖）。搜索序（官方 LoadSystemConfig 固定
// ..\Res\MonitorLimit.ini，Go 侧放宽）：path 存在即用。
func LoadMonitorLimitINI(path string) (cfg LimitConfig, data []byte, created bool, err error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		cfg, err = ParseMonitorLimitINI(raw)
		return cfg, raw, false, err
	}
	if !os.IsNotExist(err) {
		return cfg, nil, false, err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			return cfg, nil, false, fmt.Errorf("释放内置默认 %s: %w", path, mkErr)
		}
	}
	if werr := os.WriteFile(path, defaultMonitorLimitINI, 0o644); werr != nil {
		return cfg, nil, false, fmt.Errorf("释放内置默认 %s: %w", path, werr)
	}
	cfg, err = ParseMonitorLimitINI(defaultMonitorLimitINI)
	return cfg, defaultMonitorLimitINI, true, err
}

// parseCaseSensitiveINI 通用 INI 解析（分词统一走 fskit.ParseItems＝
// 官方 CIniFile Build 方言，节/键保留原文大小写——官方 CReadIni 行为，
// [System] 与 [system] 是两个节）。'#' 行官方 Build 不吃、漏成 key-only
// 条目，这里挡掉；裸行同样入表为 key→""（无消费者，等价旧忽略口径）。
func parseCaseSensitiveINI(b []byte) map[string]map[string]string {
	out := map[string]map[string]string{}
	_ = fskit.ParseItems(b, func(sec, k, v string) error {
		if strings.HasPrefix(k, "#") {
			return nil // 注释行不入表（官方方言）
		}
		if out[sec] == nil {
			out[sec] = map[string]string{}
		}
		out[sec][k] = v
		return nil
	})
	return out
}

// cAtoi 复刻 C atoi：跳过前导空白，可选正负号，取最长数字前缀，
// 无数字返回 0。
func cAtoi(s string) int {
	s = strings.TrimSpace(s)
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	j := i
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	if j == i {
		return 0
	}
	n, err := strconv.Atoi(s[:j])
	if err != nil {
		return 0
	}
	return n
}

// lastCharAtoi 官方 GetCapitalType@0x100a5d00：空串→-1，否则取末字符
// atoi（非数字→0）。
func lastCharAtoi(s string) int {
	if s == "" {
		return -1
	}
	return cAtoi(s[len(s)-1:])
}
