package login

import (
	"fmt"
	"strconv"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// GameEntryConfig 是官方 game_entry.ini（小写系，game_entry.exe →
// FsSoloEntry.dll/九阴 solo 接入服）的投影。方言与 GameListerConfig
// 同族：`[节]` 头、`键=值`、`//`（及 `;`/`#`）整行注释、注释掉的键视为
// 缺省、未知节/键忽略（前向兼容）、[Module] 裸行模块列表。
//
// 线路相关字段（Outer.Addr/Outer.Port/ClientVersion/MessageEncodeKey）
// 由调用方接线；IDC/DistrictID/ServerID/EntryID/MaxConnects 及
// Inner/Broadcast/Echo/MessageVerifyDll 当前仅解析呈现（单进程 solo
// 部署无对应物，多进程拆分里程碑再接）。
type GameEntryConfig struct {
	MainEntity  string
	MainScript  string
	IDC         int
	DistrictID  int
	ServerID    int
	EntryID     int
	MaxConnects int
	// ClientVersion 是 Go 扩展键（官方由 verify DLL 虚接口供给期望版本）：
	// op02 帧 [0x01:0x05] u32 版本号须与之相等，不符回官方方言
	// {0x03, u32 0x4E85} 并断连（BGserverCJ CheckMessageIllegal 版本门
	// 反汇编实证；golden 观测值 333）。0=不校验。
	ClientVersion int
	// MessageEncodeKey 十六进制串（官方注释样本 A5375CAE），经
	// ConvertEncodeKey 转 u32 传输键；空=沿用内置现代客户端键。
	MessageEncodeKey string
	// MessageVerifyDll 官方 fsverify.dll 插件位（消息校验/槽解密挂点），
	// 本实现不接线，仅呈现。
	MessageVerifyDll string
	OuterAddr        string
	OuterPort        int
	InnerAddr        string
	InnerPort        int
	BroadcastAddr    string
	BroadcastPort    int
	EchoAddr         string
	EchoPort         int
}

// ParseGameEntryINI 解析官方 game_entry.ini 文本（分词统一走
// fskit.ParseItems＝官方 CIniFile Build 方言：`[节]`/`键=值`、`//`/`;`
// 注释、`#` 与裸行按 key-only 条目忽略、节键保留原文大小写交给 set）。
// 必填 [Outer] Addr/Port，端口合法范围 1..65535；其余键缺省零值。
func ParseGameEntryINI(data []byte) (GameEntryConfig, error) {
	var cfg GameEntryConfig
	if err := fskit.ParseItems(data, cfg.set); err != nil {
		return cfg, fmt.Errorf("game_entry ini %w", err)
	}
	if cfg.OuterAddr == "" {
		return cfg, fmt.Errorf("game_entry ini 缺 [Outer] Addr")
	}
	if cfg.OuterPort < 1 || cfg.OuterPort > 65535 {
		return cfg, fmt.Errorf("game_entry ini [Outer] Port 非法：%d", cfg.OuterPort)
	}
	return cfg, nil
}

// ConvertEncodeKey 把 ini MessageEncodeKey 十六进制串转成 u32 传输键
// （官方 SoloEntry::ConvertEncodeKey 反汇编实证：逐字符移位累加 nibble，
// '0'-'9'/'A'-'F'；Go 实现兼容小写并限 1..8 位，空串返回 0=未设置）。
func ConvertEncodeKey(value string) (uint32, error) {
	if value == "" {
		return 0, nil
	}
	if len(value) > 8 {
		return 0, fmt.Errorf("MessageEncodeKey 超长：%q", value)
	}
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("MessageEncodeKey 须为十六进制：%q", value)
	}
	return uint32(n), nil
}

// LoadOrCreateGameEntryINI 读取 path 的配置文件；不存在时把内置默认
// def 写到 path 并返回（created=true）。已存在的文件绝不覆盖。
func LoadOrCreateGameEntryINI(path string, def []byte) (data []byte, created bool, err error) {
	return loadOrCreateINI(path, def)
}

func (c *GameEntryConfig) set(section, key, value string) error {
	switch section + "/" + key {
	case "Main/IDC", "Main/DistrictID", "Main/ServerID", "Main/EntryID",
		"Main/MaxConnects", "Main/ClientVersion",
		"Outer/Port", "Inner/Port", "Broadcast/Port", "Echo/Port":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("须为整数，得 %q", value)
		}
		switch section + "/" + key {
		case "Main/IDC":
			c.IDC = n
		case "Main/DistrictID":
			c.DistrictID = n
		case "Main/ServerID":
			c.ServerID = n
		case "Main/EntryID":
			c.EntryID = n
		case "Main/MaxConnects":
			c.MaxConnects = n
		case "Main/ClientVersion":
			c.ClientVersion = n
		case "Outer/Port":
			c.OuterPort = n
		case "Inner/Port":
			c.InnerPort = n
		case "Broadcast/Port":
			c.BroadcastPort = n
		case "Echo/Port":
			c.EchoPort = n
		}
	case "System/MainEntity":
		c.MainEntity = value
	case "System/MainScript":
		c.MainScript = value
	case "Main/MessageEncodeKey":
		c.MessageEncodeKey = value
	case "Main/MessageVerifyDll":
		c.MessageVerifyDll = value
	case "Outer/Addr":
		c.OuterAddr = value
	case "Inner/Addr":
		c.InnerAddr = value
	case "Broadcast/Addr":
		c.BroadcastAddr = value
	case "Echo/Addr":
		c.EchoAddr = value
	}
	return nil
}
