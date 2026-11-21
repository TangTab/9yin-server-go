package fsunite

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloauction"
	"github.com/lizhanyin/9yin-server-go/internal/fssololog"
	"github.com/lizhanyin/9yin-server-go/internal/fssolomember"
	"github.com/lizhanyin/9yin-server-go/internal/fssolopublic"
	"github.com/lizhanyin/9yin-server-go/internal/fssolostore"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloworld"
)

// Config 是 game_unite.ini 方言的投影（solounite/main.lua main() 全量
// 读取序）。缺省值 = main.lua ReadString/ReadInteger 第三参字面值。
type Config struct {
	// [System]/[Module]/[Script]/[Resource]（呈现层；MainEntity=SoloWorld）
	MainEntity   string
	MainScript   string
	Modules      []string
	ScriptPath   string
	ScriptReload bool
	ResourcePath string

	// [Main]
	DistrictID        int
	ServerID          int
	ServerName        string
	PublicLogicDll    string // fspublogic.dll（呈现）
	AuctionLogicDll   string // fsauctionlogic.dll（呈现）
	MaxPlayers        int    // 缺省 8192
	CollideRootPath   string
	UseMysql          bool
	LogDBPerformance  bool
	MessageEncodeKey  string // hex（空=明文）
	MessageVerifyDll  string // fsverify.dll（呈现）
	DebugHalt         bool   // 缺省 true
	FlushMinutes      int
	RoleNameSizeLimit int // 缺省 16

	// [Inner]（合一形态 main.lua 恒置 InnerPort=0 动态分配）
	InnerAddr string
	InnerPort int

	// [Broadcast]（七实体共享 UDP 口）
	BroadAddr string
	BroadPort int

	// [Echo] → world 维护面（缺省 127.0.0.1:15001）
	EchoAddr string
	EchoPort int

	// [Outer] → entry 外网受面（缺省 127.0.0.1:2001；license=
	// <OuterAddr>.lic）
	OuterAddr string
	OuterPort int

	// [Charge]/[Gmcc]/[ManageServer]/[Monitor]/[DumpServer] → world
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

	// [Account]/[Role]/[Log]/[Public]/[Manage] 五库连接串（SQLOLEDB
	// 方言；Go 侧仅呈现，后端 = 内存 store）
	AccountBase  string
	AccountPools int
	RoleBase     string
	RolePools    int
	LogBase      string
	LogPools     int
	PublicBase   string
	PublicPools  int
	ManageBase   string
	ManagePools  int

	// [Logic]（fsgame.dll/SnailEditor.dll；呈现层）
	LogicPath    string
	LogicModules []string

	// testXxxEcho 测试口位覆盖（ini 无此键；同包测试注入——实体 ctor
	// 缺省口 15003/15100/15200/16001 与并行包测试进程冲突时改写）。
	testPublicEcho   int
	testStoreEcho    int
	testEntryEcho    int
	testAuctionEcho  int
	testAuctionInner int    // 测试 auction 内口覆写（官方缺省 16003）
	testLogMutex     string // 测试互斥名覆写（与在跑服务互斥体解耦）
}

// DefaultConfig 官方 game_unite.ini + main.lua 缺省定案。
func DefaultConfig() Config {
	return Config{
		MainEntity:        "SoloWorld",
		MainScript:        "main",
		ScriptReload:      true,
		DistrictID:        1,
		ServerID:          1,
		MaxPlayers:        8192,
		DebugHalt:         true,
		RoleNameSizeLimit: 16,
		InnerAddr:         "127.0.0.1",
		BroadAddr:         "127.255.255.255",
		BroadPort:         15000,
		EchoAddr:          "127.0.0.1",
		EchoPort:          15001,
		OuterAddr:         "127.0.0.1",
		OuterPort:         2001,
		AccountPools:      0,
		RolePools:         0,
		LogPools:          0,
		PublicPools:       0,
		ManagePools:       0,
	}
}

// ParseGameUniteINI 解析 game_unite.ini 文本（分词统一走 fskit.ParseItems
// ＝官方 CIniFile Build 方言：`[节]` + `键=值`；`//` 与 `;` 注释；
// [Module] 裸行按 key-only 条目收集）。
// MySQLDSN 存储后端 DSN 解析（公共方法）：环境变量 NINEYIN_MYSQL_DSN
// 优先（env.server），缺省回落 game_unite.ini [Role] ConnectString
// （官方 SQLOLEDB 方言，经 fssolostore.ParseConnectStringDSN 转换）；
// 两者皆空返回 ""（内存库，重启即丢）。
func (c Config) MySQLDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("NINEYIN_MYSQL_DSN")); dsn != "" {
		return dsn
	}
	return fssolostore.ParseConnectStringDSN(c.RoleBase) // RoleBase=[Role] ConnectString
}

func ParseGameUniteINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
	b := func(v string) bool { return v == "1" || strings.EqualFold(v, "true") }
	n := func(v string) int { i, _ := strconv.Atoi(v); return i }
	if err := fskit.ParseItems(data, func(sec, key, val string) error {
		sec = strings.ToLower(sec)
		// [Module] 裸行模块列表（key-only 条目；'#' 行滤掉）。
		if val == "" && sec == "module" && !strings.HasPrefix(key, "#") {
			cfg.Modules = append(cfg.Modules, key)
			return nil
		}
		return cfg.set(sec, strings.ToLower(key), val, b, n)
	}); err != nil {
		return cfg, fmt.Errorf("game_unite ini %w", err)
	}
	return cfg, nil
}

func (c *Config) set(sec, key, val string, b func(string) bool, n func(string) int) error {
	switch sec + "/" + key {
	// [System]/[Script]/[Resource]
	case "system/mainentity":
		c.MainEntity = val
	case "system/mainscript":
		c.MainScript = val
	case "script/path":
		c.ScriptPath = val
	case "script/reload":
		c.ScriptReload = b(val)
	case "resource/path":
		c.ResourcePath = val

	// [Main]
	case "main/districtid":
		c.DistrictID = n(val)
	case "main/serverid":
		c.ServerID = n(val)
	case "main/servername":
		c.ServerName = val
	case "main/publiclogicdll":
		c.PublicLogicDll = val
	case "main/auctionlogicdll":
		c.AuctionLogicDll = val
	case "main/maxplayers":
		c.MaxPlayers = n(val)
	case "main/colliderootpath":
		c.CollideRootPath = val
	case "main/usemysql":
		c.UseMysql = b(val)
	case "main/logdbperformance":
		c.LogDBPerformance = b(val)
	case "main/messageencodekey":
		c.MessageEncodeKey = val
	case "main/messageverifydll":
		c.MessageVerifyDll = val
	case "main/debughalt":
		c.DebugHalt = b(val)
	case "main/flushminutes":
		c.FlushMinutes = n(val)
	case "main/rolenamesizelimit":
		c.RoleNameSizeLimit = n(val)

	// [Inner]/[Broadcast]/[Echo]/[Outer]
	case "inner/addr":
		c.InnerAddr = val
	case "inner/port":
		c.InnerPort = n(val)
	case "broadcast/addr":
		c.BroadAddr = val
	case "broadcast/port":
		c.BroadPort = n(val)
	case "echo/addr":
		c.EchoAddr = val
	case "echo/port":
		c.EchoPort = n(val)
	case "outer/addr":
		c.OuterAddr = val
	case "outer/port":
		c.OuterPort = n(val)

	// world 外连四开关
	case "charge/inuse":
		c.ChargeInuse = b(val)
	case "charge/addr":
		c.ChargeAddr = val
	case "charge/port":
		c.ChargePort = n(val)
	case "gmcc/inuse":
		c.GmccInuse = b(val)
	case "gmcc/addr":
		c.GmccAddr = val
	case "gmcc/port":
		c.GmccPort = n(val)
	case "manageserver/inuse":
		c.ManageInuse = b(val)
	case "manageserver/addr":
		c.ManageAddr = val
	case "manageserver/port":
		c.ManagePort = n(val)
	case "monitor/inuse":
		c.MonitorInuse = b(val)
	case "monitor/addr":
		c.MonitorAddr = val
	case "monitor/port":
		c.MonitorPort = n(val)
	case "dumpserver/inuse":
		c.DumpInuse = b(val)
	case "dumpserver/addr":
		c.DumpAddr = val
	case "dumpserver/port":
		c.DumpPort = n(val)

	// 五库
	case "account/connectstring":
		c.AccountBase = val
	case "account/connectpool":
		c.AccountPools = n(val)
	case "role/connectstring":
		c.RoleBase = val
	case "role/connectpool":
		c.RolePools = n(val)
	case "log/connectstring":
		c.LogBase = val
	case "log/connectpool":
		c.LogPools = n(val)
	case "public/connectstring":
		c.PublicBase = val
	case "public/connectpool":
		c.PublicPools = n(val)
	case "manage/connectstring":
		c.ManageBase = val
	case "manage/connectpool":
		c.ManagePools = n(val)

	// [Logic]
	case "logic/path":
		c.LogicPath = val
	case "logic/modules":
		// 计数键：Module0..ModuleN 枚举计数（main.lua for i = 0, modules）
		if cnt := n(val); cnt > 0 && len(c.LogicModules) == 0 {
			c.LogicModules = make([]string, 0, cnt+1)
		}
	default:
		if sec == "logic" && strings.HasPrefix(key, "module") && val != "" {
			c.LogicModules = append(c.LogicModules, val)
		}
	}
	return nil
}

// WorldConfig 投影 fssoloworld.Config（main.lua world 属性块 256-281）。
// AppName 覆盖（"unite_<sid>"）由装配面 Options 注入。
func (c Config) WorldConfig() fssoloworld.Config {
	cfg := fssoloworld.DefaultConfig()
	cfg.DistrictID = c.DistrictID
	cfg.ServerID = c.ServerID
	cfg.ServerName = c.ServerName
	cfg.MaxPlayers = c.MaxPlayers
	cfg.FlushMinutes = c.FlushMinutes
	cfg.AccountRoles = 5 // main.lua 显式
	cfg.RoleNameLimit = c.RoleNameSizeLimit
	cfg.InnerAddr = c.InnerAddr
	cfg.InnerPort = 0 // 合一形态 main.lua 恒 0（动态分配）
	cfg.BroadAddr = c.BroadAddr
	cfg.BroadPort = c.BroadPort
	cfg.EchoAddr = c.EchoAddr
	cfg.EchoPort = c.EchoPort
	cfg.ChargeInuse, cfg.ChargeAddr, cfg.ChargePort = c.ChargeInuse, c.ChargeAddr, c.ChargePort
	cfg.GmccInuse, cfg.GmccAddr, cfg.GmccPort = c.GmccInuse, c.GmccAddr, c.GmccPort
	cfg.ManageInuse, cfg.ManageAddr, cfg.ManagePort = c.ManageInuse, c.ManageAddr, c.ManagePort
	cfg.MonitorInuse, cfg.MonitorAddr, cfg.MonitorPort = c.MonitorInuse, c.MonitorAddr, c.MonitorPort
	cfg.DumpInuse, cfg.DumpAddr, cfg.DumpPort = c.DumpInuse, c.DumpAddr, c.DumpPort
	return cfg
}

// StoreConfig 投影 fssolostore.Config（main.lua store 属性块 300-325；
// echo/wake 为实体 ctor 缺省——unite 的 main.lua 不设置）。
func (c Config) StoreConfig() fssolostore.Config {
	cfg := fssolostore.DefaultConfig()
	cfg.DistrictID = c.DistrictID
	cfg.ServerID = c.ServerID
	cfg.StoreID = 0
	cfg.UseMysql = c.UseMysql
	cfg.InnerAddr = c.InnerAddr
	cfg.InnerPort = 0
	cfg.BroadAddr = c.BroadAddr
	cfg.BroadPort = c.BroadPort
	cfg.WakeAddr = "127.0.0.1"
	cfg.WakePort = 0
	cfg.AccountBase, cfg.AccountPools = c.AccountBase, c.AccountPools
	cfg.RoleBase, cfg.RolePools = c.RoleBase, c.RolePools
	cfg.LogBase, cfg.LogPools = c.LogBase, c.LogPools
	cfg.PublicBase, cfg.PublicPools = c.PublicBase, c.PublicPools
	cfg.ManageBase, cfg.ManagePools = c.ManageBase, c.ManagePools
	cfg.AppName = fmt.Sprintf("store_%d_0", c.ServerID)
	cfg.LogDBPerformance = c.LogDBPerformance
	if c.testStoreEcho != 0 {
		cfg.EchoPort = c.testStoreEcho
	}
	return cfg
}

// PublicConfig 投影 fssolopublic.Config（main.lua public 属性块
// 283-298；SaveMinutes/Echo 为实体缺省——unite 不设置）。
func (c Config) PublicConfig() fssolopublic.Config {
	var cfg fssolopublic.Config
	cfg.DistrictID = uint32(c.DistrictID)
	cfg.ServerID = uint32(c.ServerID)
	cfg.PublicLogicDll = c.PublicLogicDll
	cfg.InnerAddr = c.InnerAddr
	cfg.InnerPort = 0
	cfg.BroadAddr = c.BroadAddr
	cfg.BroadPort = c.BroadPort
	// fssolopublic.Config 零值缺省在 applyDefaults 侧（15003 等）；
	// 显式补齐与家族一致的 ctor 面。
	cfg.EchoAddr = "127.0.0.1"
	cfg.EchoPort = 15003
	cfg.SaveMinutes = 20
	if c.testPublicEcho != 0 {
		cfg.EchoPort = c.testPublicEcho
	}
	return cfg
}

// MemberConfig 投影 fssolomember.Config（main.lua member 属性块
// 368-391；echo 为实体缺省 15300）。
func (c Config) MemberConfig() fssolomember.Config {
	cfg := fssolomember.DefaultConfig()
	cfg.DistrictID = uint32(c.DistrictID)
	cfg.ServerID = uint32(c.ServerID)
	cfg.MemberID = 0
	cfg.DebugHalt = c.DebugHalt
	cfg.CollideRootPath = c.CollideRootPath
	cfg.LogicModules = append([]string(nil), c.LogicModules...)
	cfg.InnerAddr = c.InnerAddr
	cfg.InnerPort = 0
	cfg.BroadAddr = c.BroadAddr
	cfg.BroadPort = c.BroadPort
	return cfg
}

// AuctionOptions 投影 fssoloauction.Options（main.lua auction 属性块
// 329-337；echo/inner 口为实体缺省 16001/16003——unite 不设置）。
func (c Config) AuctionOptions() fssoloauction.Options {
	echoPort := 16001
	if c.testAuctionEcho != 0 {
		echoPort = c.testAuctionEcho
	}
	return fssoloauction.Options{
		AppName:       fmt.Sprintf("auction_%d", c.ServerID),
		DistrictID:    uint32(c.DistrictID),
		ServerID:      uint32(c.ServerID),
		InnerAddr:     c.InnerAddr,
		InnerPort:     c.testAuctionInner, // 0=官方缺省 16003
		EchoAddr:      "127.0.0.1",
		EchoPort:      echoPort,
		BroadAddr:     c.BroadAddr,
		BroadPort:     c.BroadPort,
		SaveMinutes:   20,
		SaveBatchSize: 100,
		ExitForceSave: true,
	}
}

// LoggerOptions 投影 fssololog.Options（main.lua logger 属性块 35-39；
// AutoExit 关闭——见包文档偏差清单）。
func (c Config) LoggerOptions() fssololog.Options {
	return fssololog.Options{
		AppName:       "SoloLog",
		LogFileDir:    `log\`,
		EveryDayAlone: false,          // main.lua 显式 false（每日一档关闭）
		AutoExit:      false,          // 偏差：无 LogShareMem 生产者，看门狗必杀 → 关
		MutexName:     c.testLogMutex, // 空=官方单实例互斥名
	}
}

// AppName 官方 `"unite_" .. server_id`。
func (c Config) AppName() string { return "unite_" + strconv.Itoa(c.ServerID) }

// LicensePath 官方 outer_addr .. ".lic"。
func (c Config) LicensePath() string { return c.OuterAddr + ".lic" }
