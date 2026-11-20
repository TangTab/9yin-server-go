package login

import (
	"fmt"
	"os"
	"strconv"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// GameListerConfig 是官方 Fs 框架 lister 家族 INI 配置的投影，兼容两个
// 变体（BGserverCJ 双体系，九阴复刻以小写系为权威）：
//
//	game_lister.ini （小写系，FsSoloLister.dll/九阴 solo）：[Main]
//	                 IDC/DistrictID/ListerID/MaxConnects，可带
//	                 ClientVersion + [Outer]/[Inner]/[Broadcast]
//	GameLister.ini  （大写系，FsLister.dll/黑金 Fs）：同构，
//	                 MaxPlayer 代替 MaxConnects，另有 SockThreads
//
// 线路相关字段（Outer.Addr/Outer.Port/ClientVersion）由调用方接线；
// IDC/DistrictID/ListerID/MaxPlayer/SockThreads 当前仅解析呈现（单服 Go
// 部署无对应物，集群形态时再接）。
type GameListerConfig struct {
	IDC           int
	DistrictID    int
	ListerID      int
	MaxPlayer     int // game_lister.ini 变体键名 MaxConnects，语义相同
	SockThreads   int
	ClientVersion int // [Main] 可选项；>0 即官方版本门（Lister.MinClientVersion）
	OuterAddr     string
	OuterPort     int
	InnerAddr     string
	BroadcastAddr string
	BroadcastPort int
}

// ParseGameListerINI 解析官方 GameLister.ini / game_lister.ini 文本。
// 方言（分词统一走 fskit.ParseItems＝官方 CIniFile Build 方言）：
// `[节]` 头、`键=值`、`//`/`;` 整行注释、`#` 与裸行按 key-only 条目
// 忽略（[Module] 模块名清单由此呈现）、注释掉的键（如
// `//Addr=127.0.0.1`）视为缺省；未知节/键忽略（前向兼容）。
// 必填 [Outer] Addr/Port，端口合法范围 1..65535。
func ParseGameListerINI(data []byte) (GameListerConfig, error) {
	var cfg GameListerConfig
	if err := fskit.ParseItems(data, cfg.set); err != nil {
		return cfg, fmt.Errorf("game_lister ini %w", err)
	}
	if cfg.OuterAddr == "" {
		return cfg, fmt.Errorf("game_lister ini 缺 [Outer] Addr")
	}
	if cfg.OuterPort < 1 || cfg.OuterPort > 65535 {
		return cfg, fmt.Errorf("game_lister ini [Outer] Port 非法：%d", cfg.OuterPort)
	}
	return cfg, nil
}

// LoadOrCreateGameListerINI 读取 path 的配置文件；不存在时把内置默认
// def 写到 path 并返回（created=true）。用于 exe 内置配置的首次释放：
// 已存在的文件绝不覆盖，用户改动得以保留。
func LoadOrCreateGameListerINI(path string, def []byte) (data []byte, created bool, err error) {
	return loadOrCreateINI(path, def)
}

// loadOrCreateINI 是 lister/entry 两族 INI 共用的装载-释放逻辑。
func loadOrCreateINI(path string, def []byte) (data []byte, created bool, err error) {
	data, err = os.ReadFile(path)
	if err == nil {
		return data, false, nil
	}
	if !os.IsNotExist(err) {
		return nil, false, err
	}
	if err := os.WriteFile(path, def, 0o644); err != nil {
		return nil, false, err
	}
	return def, true, nil
}

// set 把一个键值写进配置；整数键解析失败即报错（不静默置零）。
func (c *GameListerConfig) set(section, key, value string) error {
	switch section + "/" + key {
	case "Main/IDC", "Main/DistrictID", "Main/ListerID", "Main/MaxPlayer",
		"Main/MaxConnects", "Main/SockThreads", "Main/ClientVersion",
		"Outer/Port", "Broadcast/Port":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("须为整数，得 %q", value)
		}
		switch section + "/" + key {
		case "Main/IDC":
			c.IDC = n
		case "Main/DistrictID":
			c.DistrictID = n
		case "Main/ListerID":
			c.ListerID = n
		case "Main/MaxPlayer", "Main/MaxConnects":
			c.MaxPlayer = n
		case "Main/SockThreads":
			c.SockThreads = n
		case "Main/ClientVersion":
			c.ClientVersion = n
		case "Outer/Port":
			c.OuterPort = n
		case "Broadcast/Port":
			c.BroadcastPort = n
		}
	case "Outer/Addr":
		c.OuterAddr = value
	case "Inner/Addr":
		c.InnerAddr = value
	case "Broadcast/Addr":
		c.BroadcastAddr = value
	}
	return nil
}
