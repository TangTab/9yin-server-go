package auction

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// defaultGameAuctionINI 是内置默认配置（官方 game_auction.ini 方言）：
// 未指定路径且当前目录没有 game_auction.ini 时释放到当前目录并采用
// （已存在的绝不覆盖）——与 cmd/solo game_entry.ini / cmd/lister
// game_lister.ini 同一范式。
//
//go:embed game_auction.ini
var defaultGameAuctionINI []byte

// AuctionConfig 是官方 game_auction.ini（game_auction.exe →
// FsSoloAuction.dll 拍卖行数据服务器）的投影。方言与 GameEntryConfig
// 同族：`[节]` 头、`键=值`、`//`（及 `;`/`#`）整行注释、注释掉的键视为
// 缺省、未知节/键忽略（前向兼容）、[Module] 裸行模块列表、
// [Account]/[Role]/[Log]/[Public] ConnectString 兼容呈现。
//
// 接线：DistrictID/ServerID + Inner.Addr/Port + ResourcePath（Go 扩展，
// ItemType.xml 装载根）；SaveMinutes/SaveBatchSize 解析呈现
// （Store PUB4 持久化里程碑接线）；其余为官方方言兼容。
type AuctionConfig struct {
	MainEntity      string
	MainScript      string
	DistrictID      int
	ServerID        int
	AuctionLogicDll string // 官方逻辑 DLL 名（本实现内置等价，仅呈现）
	SaveMinutes     int    // 定时存盘间隔分钟（官方 main.lua 缺省 20）
	SaveBatchSize   int    // 每轮存盘批量（0=未配置）
	InnerAddr       string // 空=侦听全部地址
	InnerPort       int    // 官方缺省 16003
	BroadcastAddr   string
	BroadcastPort   int
	EchoAddr        string
	EchoPort        int
	// ResourcePath 是 Go 扩展键 [Resource] Path：资源根目录，
	// ItemTypeXML() 由此推导；空=Res。
	ResourcePath string
}

// ItemTypeXML 推导 ItemType.xml 全路径（官方 LoadResource 取
// kernel GetResourcePath() + "ini/trade/AuctionHouse/ItemType.xml"）。
func (c *AuctionConfig) ItemTypeXML() string {
	root := c.ResourcePath
	if root == "" {
		root = "Res"
	}
	return root + string(os.PathSeparator) + "ini" + string(os.PathSeparator) +
		"Trade" + string(os.PathSeparator) + "AuctionHouse" + string(os.PathSeparator) + "ItemType.xml"
}

// ParseGameAuctionINI 解析官方 game_auction.ini 文本。
// 必填 [Main] DistrictID/ServerID；[Inner] Port 官方文件不写
// （端口来自 main.lua 本地缺省 16003），缺省回落同值，
// 显式给值时须 1..65535；GBK 官方文件亦可解析（关心的键均为 ASCII）。
func ParseGameAuctionINI(data []byte) (AuctionConfig, error) {
	var cfg AuctionConfig
	section := ""
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "//") ||
			strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			// [Module] 节是官方方言的裸行模块列表，跳过。
			if strings.EqualFold(section, "Module") {
				continue
			}
			return cfg, fmt.Errorf("game_auction ini 第 %d 行无 '='：%q", i+1, line)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if err := cfg.set(section, key, value); err != nil {
			return cfg, fmt.Errorf("game_auction ini 第 %d 行 %s/%s: %w", i+1, section, key, err)
		}
	}
	if cfg.DistrictID == 0 {
		return cfg, fmt.Errorf("game_auction ini 缺 [Main] DistrictID")
	}
	if cfg.ServerID == 0 {
		return cfg, fmt.Errorf("game_auction ini 缺 [Main] ServerID")
	}
	switch {
	case cfg.InnerPort == 0:
		cfg.InnerPort = 16003 // 官方 main.lua 缺省
	case cfg.InnerPort < 0 || cfg.InnerPort > 65535:
		return cfg, fmt.Errorf("game_auction ini [Inner] Port 非法：%d", cfg.InnerPort)
	}
	return cfg, nil
}

func (c *AuctionConfig) set(section, key, value string) error {
	switch section + "/" + key {
	case "Main/DistrictID", "Main/ServerID", "Main/SaveMinutes",
		"Main/SaveBatchSize", "Inner/Port", "Broadcast/Port", "Echo/Port":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("须为整数，得 %q", value)
		}
		switch section + "/" + key {
		case "Main/DistrictID":
			c.DistrictID = n
		case "Main/ServerID":
			c.ServerID = n
		case "Main/SaveMinutes":
			c.SaveMinutes = n
		case "Main/SaveBatchSize":
			c.SaveBatchSize = n
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
	case "Main/AuctionLogicDll":
		c.AuctionLogicDll = value
	case "Inner/Addr":
		c.InnerAddr = value
	case "Broadcast/Addr":
		c.BroadcastAddr = value
	case "Echo/Addr":
		c.EchoAddr = value
	case "Resource/Path":
		c.ResourcePath = value
	default:
		// 未知键忽略（前向兼容）；[Account]/[Role]/[Log]/[Public]
		// ConnectString 等官方遗留键不接线、不报错。
	}
	return nil
}

// LoadGameAuctionINI 读取 path 的 game_auction.ini；不存在时把内置默认
// 释放到该路径（已存在的绝不覆盖）并返回释放内容与 created=true。
// 与 login.LoadOrCreateGameEntryINI 同范式。
func LoadGameAuctionINI(path string) (cfg AuctionConfig, data []byte, created bool, err error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		cfg, err = ParseGameAuctionINI(raw)
		return cfg, raw, false, err
	}
	if !os.IsNotExist(err) {
		return cfg, nil, false, err
	}
	if werr := os.WriteFile(path, defaultGameAuctionINI, 0o644); werr != nil {
		return cfg, nil, false, fmt.Errorf("释放内置默认 %s: %w", path, werr)
	}
	cfg, err = ParseGameAuctionINI(defaultGameAuctionINI)
	return cfg, defaultGameAuctionINI, true, err
}
