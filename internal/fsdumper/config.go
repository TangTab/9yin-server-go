package fsdumper

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config 对应官方 GameDumper.ini（FsLoader 装载，main.lua 消费）。
// 段名/键名与官方一一对应；output←manage 回退 quirk 保留在
// AccessDSN("output") 中（main.lua get_db_config 三处实证）。
type Config struct {
	LogicDLL    string // [Main] DumpLogicDll
	UseMysql    bool   // [Main] UseMysql（官方缺省 true）
	OuterAddr   string // [Outer] Addr
	OuterPort   int    // [Outer] Port（0=不开外口）
	EchoAddr    string // [Echo] Addr（缺省 127.0.0.1）
	EchoPort    int    // [Echo] Port（main.lua 缺省 15009）
	ResourceDir string // 装载方注入（官方 [Resource] Path=..\Res）
	RoleDSN     string // [Role] ConnectString
	PublicDSN   string // [Public] ConnectString
	ManageDSN   string // [Manage] ConnectString
	OutputDSN   string // [Output]（官方读 [Manage] 段；空回退 Role）
}

// DefaultConfig 官方缺省值形态。
func DefaultConfig() *Config {
	return &Config{
		UseMysql: true,
		EchoAddr: "127.0.0.1",
		EchoPort: 15009,
	}
}

// LoadConfig 解析 GameDumper.ini（GBK/UTF-8 兼读；官方 ini 为 GBK）。
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fsdumper: 读配置 %s: %w", path, err)
	}
	cfg := DefaultConfig()
	sec := ""
	for _, line := range strings.Split(decodeGBK(raw), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "\r"))
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sec = strings.ToLower(line[1 : len(line)-1])
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch sec + "." + strings.ToLower(k) {
		case "main.dumplogicdll":
			cfg.LogicDLL = v
		case "main.usemysql":
			cfg.UseMysql = parseBoolDefault(v, true)
		case "outer.addr":
			cfg.OuterAddr = v
		case "outer.port":
			cfg.OuterPort = atoiDefault(v, 0)
		case "echo.addr":
			cfg.EchoAddr = v
		case "echo.port":
			cfg.EchoPort = atoiDefault(v, 15009)
		case "resource.path":
			cfg.ResourceDir = v
		case "role.connectstring":
			cfg.RoleDSN = v
		case "public.connectstring":
			cfg.PublicDSN = v
		case "manage.connectstring":
			cfg.ManageDSN = v
		case "output.connectstring":
			cfg.OutputDSN = v
		}
	}
	return cfg, nil
}

// AccessDSN 返回指定库名（role/public/manage/output）的连接串。
// 官方回退序（main.lua）：public→role、manage→role、output←manage（quirk）
// 再→role。
func (c *Config) AccessDSN(name string) string {
	switch name {
	case "role":
		return c.RoleDSN
	case "public":
		if c.PublicDSN != "" {
			return c.PublicDSN
		}
		return c.RoleDSN
	case "manage":
		if c.ManageDSN != "" {
			return c.ManageDSN
		}
		return c.RoleDSN
	case "output":
		if c.OutputDSN != "" {
			return c.OutputDSN
		}
		if c.ManageDSN != "" {
			return c.ManageDSN
		}
		return c.RoleDSN
	}
	return ""
}

func parseBoolDefault(s string, def bool) bool {
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	}
	return def
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// decodeGBK GBK 内容按 GBK 读（含 BOM 的 UTF-8 直接去 BOM）。
// 官方 ini 均为 GBK（NLS 注释），UTF-8 回退保证可读。
func decodeGBK(b []byte) string {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return string(b[3:])
	}
	if utf8Only(b) {
		return string(b)
	}
	if out, err := gbkDecode(b); err == nil {
		return out
	}
	return string(b)
}

func utf8Only(b []byte) bool {
	for i := 0; i < len(b); {
		if b[i] < 0x80 {
			i++
			continue
		}
		n := 0
		switch {
		case b[i]&0xE0 == 0xC0:
			n = 2
		case b[i]&0xF0 == 0xE0:
			n = 3
		case b[i]&0xF8 == 0xF0:
			n = 4
		default:
			return false
		}
		if i+n > len(b) {
			return false
		}
		i += n
	}
	return true
}
