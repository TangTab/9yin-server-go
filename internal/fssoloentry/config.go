package fssoloentry

import (
	"fmt"
	"strconv"
	"strings"
)

// Config 是 game_entry.ini 的 solo 方言（lua\soloentry\main.lua 读键面）。
// 与网关版 GameEntry.ini 同族但缺省不同：InnerPort 缺省 0（DLL 内动态
// 分配并回写字段，Lua 随后打印实际端口）；MaxConnects 键名。
type Config struct {
	IDC              uint32
	DistrictID       uint32
	ServerID         uint32
	EntryID          uint16
	MaxConnects      int
	MessageEncodeKey string
	MessageVerifyDll string

	OuterAddr string
	OuterPort int

	InnerAddr string
	InnerPort int // 缺省 0：动态分配

	BroadAddr string
	BroadPort int // 缺省 15000

	EchoAddr string
	EchoPort int // 缺省 15200+EntryID

	License string // 缺省 <OuterAddr>.lic
}

// DefaultConfig 官方缺省（soloentry main.lua 字面值）。
func DefaultConfig() Config {
	return Config{
		IDC: 1, DistrictID: 1, ServerID: 1, EntryID: 0,
		MaxConnects: 1024,
		OuterAddr:   "127.0.0.1", OuterPort: 2001,
		InnerAddr: "127.0.0.1", InnerPort: 0,
		BroadAddr: "127.255.255.255", BroadPort: 15000,
		EchoAddr: "127.0.0.1", EchoPort: 15200,
	}
}

// ParseSoloINI 解析 game_entry.ini 文本（键名大小写不敏感；[System]/
// [Module]/[Script] 节静默跳过——壳层与装载器事务，见 internal/fsloader）。
func ParseSoloINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
	section := ""
	for _, line := range strings.Split(string(data), "\n") {
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
		if err := cfg.set(section, key, val); err != nil {
			return cfg, err
		}
	}
	if cfg.EchoPort == 15200 {
		cfg.EchoPort += int(cfg.EntryID)
	}
	if cfg.License == "" {
		cfg.License = cfg.OuterAddr + ".lic"
	}
	return cfg, nil
}

func (c *Config) set(section, key, val string) error {
	switch section {
	case "main":
		switch key {
		case "idc":
			return setU32(&c.IDC, val)
		case "districtid":
			return setU32(&c.DistrictID, val)
		case "serverid":
			return setU32(&c.ServerID, val)
		case "entryid":
			if err := setU16(&c.EntryID, val); err != nil {
				return err
			}
			if c.EchoPort == 15200 {
				c.EchoPort = 15200 + int(c.EntryID)
			}
			return nil
		case "maxconnects", "maxplayer":
			return setInt(&c.MaxConnects, val)
		case "messageencodekey":
			c.MessageEncodeKey = val
		case "messageverifydll":
			c.MessageVerifyDll = val
		}
	case "outer":
		switch key {
		case "addr":
			c.OuterAddr = val
		case "port":
			return setInt(&c.OuterPort, val)
		}
	case "inner":
		switch key {
		case "addr":
			c.InnerAddr = val
		case "port":
			return setInt(&c.InnerPort, val)
		}
	case "broadcast":
		switch key {
		case "addr":
			c.BroadAddr = val
		case "port":
			return setInt(&c.BroadPort, val)
		}
	case "echo":
		switch key {
		case "addr":
			c.EchoAddr = val
		case "port":
			return setInt(&c.EchoPort, val)
		}
	}
	return nil
}

func setInt(dst *int, val string) error {
	v, err := strconv.Atoi(val)
	if err != nil {
		return fmt.Errorf("非法整数 %q: %w", val, err)
	}
	*dst = v
	return nil
}

func setU32(dst *uint32, val string) error {
	v, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return fmt.Errorf("非法 u32 %q: %w", val, err)
	}
	*dst = uint32(v)
	return nil
}

func setU16(dst *uint16, val string) error {
	v, err := strconv.ParseUint(val, 10, 16)
	if err != nil {
		return fmt.Errorf("非法 u16 %q: %w", val, err)
	}
	*dst = uint16(v)
	return nil
}
