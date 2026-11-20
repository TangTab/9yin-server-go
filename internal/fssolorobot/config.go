package fssolorobot

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// defaultGameRobotINI 是内置默认配置（官方 game_robot.ini 方言，
// Outer 指回环网关缺省）：未指定路径且当前目录没有 game_robot.ini
// 时释放到当前目录并采用（已存在的绝不覆盖）——与 cmd/global
// game_global.ini 同一范式。
//
//go:embed game_robot.ini
var defaultGameRobotINI []byte

// Config 是官方 game_robot.ini（game_robot.exe → FsSoloRobot.dll
// 压测机器人客户端）的投影。方言与 fssologlobal.GlobalConfig 同族：
// `[节]` 头、`键=值`、`//`（及 `;`/`#`）整行注释、[Module] 裸行模块
// 列表、未知节/键忽略（前向兼容）。
//
// 缺省值来自官方 Lua\solorobot\main.lua（ReadInteger/ReadString 第三参）：
// MaxPlayer=1024、ConnectConcurrent=10、ConnectSpeed=50、MoveSeconds=5、
// AccountPrefix=PasswordPrefix="robot"、Outer=127.0.0.1:2001、
// Echo=127.0.0.1:15009（MessageEncodeKey 为官方注释掉的死配置，读后
// 从不使用，Go 版同样忽略）。
type Config struct {
	MainEntity         string   // [System] MainEntity（呈现用）
	MainScript         string   // [System] MainScript（呈现用）
	Modules            []string // [Module] 裸行模块列表（呈现用）
	ScriptPath         string   // [Script] Path（呈现用）
	MaxConnects        int      // [Main] MaxPlayer
	ConnectConcurrent  int      // [Main] ConnectConcurrent
	ConnectSpeed       int      // [Main] ConnectSpeed
	MoveSeconds        int      // [Main] MoveSeconds
	AccountPrefix      string   // [Main] AccountPrefix
	PasswordPrefix     string   // [Main] PasswordPrefix
	MessageEncodeKey   string   // [Main] MessageEncodeKey（官方死配置；呈现用）
	OuterAddr          string   // [Outer] Addr
	OuterPort          int      // [Outer] Port
	EchoAddr           string   // [Echo] Addr（空串=官方缺省；"off"=不开）
	EchoPort           int      // [Echo] Port
}

// ParseGameRobotINI 解析官方 game_robot.ini 文本（分词统一走
// fskit.ParseItems＝官方 CIniFile Build 方言；[Module] 裸行模块列表
// 按官方 key-only 条目语义收集，'#' 行滤掉）。GBK 官方文件亦可解析
// （关心的键均为 ASCII）。
func ParseGameRobotINI(data []byte) (Config, error) {
	cfg := Config{
		MainEntity:        "SoloRobot",
		MainScript:        "main",
		MaxConnects:       1024,
		ConnectConcurrent: 10,
		ConnectSpeed:      50,
		MoveSeconds:       5,
		AccountPrefix:     "robot",
		PasswordPrefix:    "robot",
		OuterAddr:         "127.0.0.1",
		OuterPort:         2001,
		EchoAddr:          "127.0.0.1",
		EchoPort:          15009,
	}
	_ = fskit.ParseItems(data, func(sec, key, val string) error {
		switch sec {
		case "System":
			switch strings.ToLower(key) {
			case "mainentity":
				cfg.MainEntity = val
			case "mainscript":
				cfg.MainScript = val
			}
		case "Module":
			if val == "" && !strings.HasPrefix(key, "#") { // key-only 裸行模块名
				cfg.Modules = append(cfg.Modules, strings.ToLower(key))
			}
		case "Script":
			if strings.EqualFold(key, "path") {
				cfg.ScriptPath = val
			}
		case "Main":
			switch strings.ToLower(key) {
			case "maxplayer":
				cfg.MaxConnects = atoiDefault(val, cfg.MaxConnects)
			case "connectconcurrent":
				cfg.ConnectConcurrent = atoiDefault(val, cfg.ConnectConcurrent)
			case "connectspeed":
				cfg.ConnectSpeed = atoiDefault(val, cfg.ConnectSpeed)
			case "moveseconds":
				cfg.MoveSeconds = atoiDefault(val, cfg.MoveSeconds)
			case "accountprefix":
				cfg.AccountPrefix = val
			case "passwordprefix":
				cfg.PasswordPrefix = val
			case "messageencodekey":
				cfg.MessageEncodeKey = val
			}
		case "Outer":
			switch strings.ToLower(key) {
			case "addr":
				cfg.OuterAddr = val
			case "port":
				cfg.OuterPort = atoiDefault(val, cfg.OuterPort)
			}
		case "Echo":
			switch strings.ToLower(key) {
			case "addr":
				cfg.EchoAddr = val
			case "port":
				cfg.EchoPort = atoiDefault(val, cfg.EchoPort)
			}
		}
		return nil
	})
	return cfg, nil
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

// LoadOrReleaseDefaultINI 装载配置：显式路径必须存在；缺省路径当前
// 目录 game_robot.ini，不存在则释放内置默认（已存在的绝不覆盖）。
// 返回（数据, 是否刚释放）。
func LoadOrReleaseDefaultINI(path string) ([]byte, bool, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, false, err
		}
		return data, false, nil
	}
	if _, err := os.Stat("game_robot.ini"); err == nil {
		data, err := os.ReadFile("game_robot.ini")
		if err != nil {
			return nil, false, err
		}
		return data, false, nil
	}
	if err := os.WriteFile("game_robot.ini", defaultGameRobotINI, 0o644); err != nil {
		return nil, false, fmt.Errorf("释放默认 game_robot.ini: %w", err)
	}
	return defaultGameRobotINI, true, nil
}
