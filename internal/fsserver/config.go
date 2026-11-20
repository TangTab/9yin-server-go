package fsserver

import (
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// Config GameWorld.ini 官方方言（main.lua 读取面 + [Main]/[Inner]/
// [Broadcast]/[Echo]/[Charge]/[Gmcc]/[Stat]/[ManageServer] 节）。
type Config struct {
	DistrictID uint32
	ServerID   uint32
	ServerName string
	MaxPlayers int

	InnerAddr string
	InnerPort int // main.lua 默认 16001
	BroadAddr string
	BroadPort int // 默认 15000
	EchoAddr  string
	EchoPort  int // 默认 15001

	ChargeInuse bool
	ChargeAddr  string
	ChargePort  int
	GmccInuse   bool
	GmccAddr    string
	GmccPort    int
	StatInuse   bool
	StatAddr    string
	StatPort    int
	ManageInuse bool
	ManageAddr  string
	ManagePort  int
}

// DefaultConfig main.lua 内置缺省。
func DefaultConfig() Config {
	return Config{
		DistrictID: 1,
		ServerID:   1,
		ServerName: "",
		MaxPlayers: 8192,
		InnerAddr:  "127.0.0.1",
		InnerPort:  16001,
		BroadAddr:  "127.255.255.255",
		BroadPort:  15000,
		EchoAddr:   "127.0.0.1",
		EchoPort:   15001,
	}
}

// ParseGameWorldINI 官方 INI 方言（分词统一走 fskit.ParseItems＝
// fskit.IniFile/官方 CIniFile Build 同方言；节键归一小写分发）。
func ParseGameWorldINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
	_ = fskit.ParseItems(data, func(sec, key, val string) error {
		switch strings.ToLower(sec) + "." + strings.ToLower(key) {
		case "main.districtid":
			cfg.DistrictID = parseU32(val, cfg.DistrictID)
		case "main.serverid":
			cfg.ServerID = parseU32(val, cfg.ServerID)
		case "main.servername":
			cfg.ServerName = val
		case "main.maxplayers":
			cfg.MaxPlayers = int(parseI(val, int64(cfg.MaxPlayers)))
		case "inner.addr":
			cfg.InnerAddr = val
		case "inner.port":
			cfg.InnerPort = int(parseI(val, int64(cfg.InnerPort)))
		case "broadcast.addr":
			cfg.BroadAddr = val
		case "broadcast.port":
			cfg.BroadPort = int(parseI(val, int64(cfg.BroadPort)))
		case "echo.addr":
			cfg.EchoAddr = val
		case "echo.port":
			cfg.EchoPort = int(parseI(val, int64(cfg.EchoPort)))
		case "charge.inuse":
			cfg.ChargeInuse = parseBool(val)
		case "charge.addr":
			cfg.ChargeAddr = val
		case "charge.port":
			cfg.ChargePort = int(parseI(val, 0))
		case "gmcc.inuse":
			cfg.GmccInuse = parseBool(val)
		case "gmcc.addr":
			cfg.GmccAddr = val
		case "gmcc.port":
			cfg.GmccPort = int(parseI(val, 0))
		case "stat.inuse":
			cfg.StatInuse = parseBool(val)
		case "stat.addr":
			cfg.StatAddr = val
		case "stat.port":
			cfg.StatPort = int(parseI(val, 0))
		case "manageserver.inuse":
			cfg.ManageInuse = parseBool(val)
		case "manageserver.addr":
			cfg.ManageAddr = val
		case "manageserver.port":
			cfg.ManagePort = int(parseI(val, 0))
		}
		return nil
	})
	return cfg, nil
}

func parseU32(s string, def uint32) uint32 {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return def
	}
	return uint32(n)
}

func parseI(s string, def int64) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return def
	}
	return n
}

// parseBool nx_boolean 方言：非空且非 "false"/"0" 为真。
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s != "" && s != "false" && s != "0"
}
