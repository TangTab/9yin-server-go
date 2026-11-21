package fssoloworld

import (
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// Config 对齐 game_world.ini + SoloWorld 属性缺省（§二 ctor 定案）。
type Config struct {
	// [Main]
	DistrictID int
	ServerID   int
	ServerName string

	// [Inner]（Port==0 → 系统分配后回填）
	InnerAddr string
	InnerPort int

	// [Broadcast]
	BroadAddr string
	BroadPort int

	// [Echo]（缺省 127.0.0.1:15001；Addr 为空不启）
	EchoAddr string
	EchoPort int

	// [Charge]/[Gmcc]/[ManageServer]/[Monitor]/[DumpServer]
	ChargeInuse  bool
	ChargeAddr   string
	ChargePort   int
	GmccInuse    bool
	GmccAddr     string
	GmccPort     int
	ManageInuse  bool
	ManageAddr   string
	ManagePort   int
	MonitorInuse bool
	MonitorAddr  string
	MonitorPort  int
	DumpInuse    bool
	DumpAddr     string
	DumpPort     int

	// 属性缺省（ctor）
	MaxPlayers    int
	SaveMinutes   int
	FlushMinutes  int
	AccountRoles  int
	RoleNameLimit int
	AutoOpen      bool
	PlayerQueue   bool
	VipQueue      bool
	QuitOnClosed  bool
}

// DefaultConfig ctor（0x100af750）缺省值定案。
func DefaultConfig() Config {
	return Config{
		DistrictID:    1,
		ServerID:      1,
		ServerName:    "",
		InnerAddr:     "127.0.0.1",
		InnerPort:     0,
		BroadAddr:     "127.255.255.255",
		BroadPort:     15000,
		EchoAddr:      "127.0.0.1",
		EchoPort:      15001,
		MaxPlayers:    8192, // +0x68 = 0x2000
		SaveMinutes:   10,   // +0x70 = 0xA
		FlushMinutes:  0,
		AccountRoles:  5, // main.lua 显式 5（ctor 缺省 1）
		RoleNameLimit: 16,
		AutoOpen:      true,
		PlayerQueue:   true,
		VipQueue:      true,
		QuitOnClosed:  false,
	}
}

// AppName 官方 `"world_" .. server_id`。
func (c Config) AppName() string { return "world_" + strconv.Itoa(c.ServerID) }

// ParseGameWorldINI 解析 game_world.ini（分词统一走 fskit.ParseItems＝
// 官方 CIniFile Build 方言：节/键=值）。与家族 ParseGame*INI 同模式。
func ParseGameWorldINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
	assign := func(sec, k, v string) {
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		b := func() bool { return v == "1" || strings.EqualFold(v, "true") }
		n := func() int {
			i, _ := strconv.Atoi(v)
			return i
		}
		switch sec {
		case "main":
			switch k {
			case "districtid":
				cfg.DistrictID = n()
			case "serverid":
				cfg.ServerID = n()
			case "servername":
				cfg.ServerName = v
			case "maxplayers":
				cfg.MaxPlayers = n()
			case "saveminutes":
				cfg.SaveMinutes = n()
			case "flushminutes":
				cfg.FlushMinutes = n()
			case "rolenamesizelimit":
				cfg.RoleNameLimit = n()
			case "playerqueue":
				cfg.PlayerQueue = b()
			case "vipqueue":
				cfg.VipQueue = b()
			}
		case "inner":
			switch k {
			case "addr":
				cfg.InnerAddr = v
			case "port":
				cfg.InnerPort = n()
			}
		case "broadcast":
			switch k {
			case "addr":
				cfg.BroadAddr = v
			case "port":
				cfg.BroadPort = n()
			}
		case "echo":
			switch k {
			case "addr":
				cfg.EchoAddr = v
			case "port":
				cfg.EchoPort = n()
			}
		case "charge":
			switch k {
			case "inuse":
				cfg.ChargeInuse = b()
			case "addr":
				cfg.ChargeAddr = v
			case "port":
				cfg.ChargePort = n()
			}
		case "gmcc":
			switch k {
			case "inuse":
				cfg.GmccInuse = b()
			case "addr":
				cfg.GmccAddr = v
			case "port":
				cfg.GmccPort = n()
			}
		case "manageserver":
			switch k {
			case "inuse":
				cfg.ManageInuse = b()
			case "addr":
				cfg.ManageAddr = v
			case "port":
				cfg.ManagePort = n()
			}
		case "monitor":
			switch k {
			case "inuse":
				cfg.MonitorInuse = b()
			case "addr":
				cfg.MonitorAddr = v
			case "port":
				cfg.MonitorPort = n()
			}
		case "dumpserver":
			switch k {
			case "inuse":
				cfg.DumpInuse = b()
			case "addr":
				cfg.DumpAddr = v
			case "port":
				cfg.DumpPort = n()
			}
		}
	}
	_ = fskit.ParseItems(data, func(sec, k, v string) error {
		assign(strings.ToLower(sec), k, v)
		return nil
	})
	return cfg, nil
}
