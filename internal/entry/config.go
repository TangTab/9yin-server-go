package entry

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// Config 是 GameEntry.ini 方言（网关版）。与 solo 版 game_entry.ini
// 同族：键名大小写不敏感，缺省值对齐 Lua/GameEntry/main.lua。
type Config struct {
	IDC              uint32
	DistrictID       uint32
	ServerID         uint32
	EntryID          uint16
	MaxPlayer        int  // 网关版键名（solo 版为 MaxConnects）
	SockThreads      int  // 官方保留项（Go 版无操作）
	MessageEncodeKey string
	MessageVerifyDll string

	OuterAddr string
	OuterPort int

	InnerAddr string
	InnerPort int // 缺省 16200+EntryID

	BroadAddr string
	BroadPort int // 缺省 15000

	EchoAddr string
	EchoPort int // 缺省 15200+EntryID

	License string // 缺省 <OuterAddr>.lic
}

// DefaultConfig 官方缺省（Lua 主脚本缺省值）。
func DefaultConfig() Config {
	return Config{
		IDC: 1, DistrictID: 1, ServerID: 1, EntryID: 0,
		MaxPlayer: 1024, SockThreads: 1,
		OuterAddr: "127.0.0.1", OuterPort: 2001,
		InnerAddr: "127.0.0.1", InnerPort: 16200,
		BroadAddr: "127.255.255.255", BroadPort: 15000,
		EchoAddr: "127.0.0.1", EchoPort: 15200,
	}
}

// ParseGameEntryINI 解析 GameEntry.ini 文本（分词统一走 fskit.ParseItems
// ＝官方 CIniFile Build 方言；`#` 与裸行按 key-only 条目忽略）。
func ParseGameEntryINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
	if err := fskit.ParseItems(data, func(sec, key, val string) error {
		return cfg.set(strings.ToLower(sec), strings.ToLower(key), val)
	}); err != nil {
		return cfg, err
	}
	if cfg.InnerPort == 16200 {
		cfg.InnerPort += int(cfg.EntryID)
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
			v, err := parseUint(val)
			if err != nil {
				return err
			}
			if v > 0xFFFF {
				return fmt.Errorf("EntryID 超界: %d", v)
			}
			c.EntryID = uint16(v)
			return nil
		case "maxplayer", "maxconnects":
			v, err := parseInt(val)
			if err != nil {
				return err
			}
			c.MaxPlayer = v
			return nil
		case "sockthreads":
			v, err := parseInt(val)
			if err != nil {
				return err
			}
			c.SockThreads = v
			return nil
		case "messageencodekey":
			c.MessageEncodeKey = val
			return nil
		case "messageverifydll":
			c.MessageVerifyDll = val
			return nil
		}
	case "outer":
		switch key {
		case "addr":
			c.OuterAddr = val
			return nil
		case "port":
			return setInt(&c.OuterPort, val)
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

func parseUint(val string) (uint64, error) { return strconv.ParseUint(val, 10, 64) }
func parseInt(val string) (int, error)     { return strconv.Atoi(val) }

func setU32(dst *uint32, val string) error {
	v, err := parseUint(val)
	if err != nil {
		return err
	}
	*dst = uint32(v)
	return nil
}

func setInt(dst *int, val string) error {
	v, err := parseInt(val)
	if err != nil {
		return err
	}
	*dst = v
	return nil
}
