package fssolostore

import (
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// Config 是官方 game_store.ini 的字段投影（main.lua main() 全量读取序）。
type Config struct {
	// [Main]
	DistrictID int // DistrictID（缺省 1）
	ServerID   int // ServerID（缺省 1；快照实配 11）
	StoreID    int // StoreID（缺省 0；组内存储编号）
	UseMysql   bool

	// [Inner] 内网受面（Port 0 = 系统分配，官方 InnerPort==0 缺省；
	// 注释里的 16100+store_id 未启用）
	InnerAddr string
	InnerPort int

	// [Broadcast] UDP 广播自宣（缺省 127.255.255.255:15000）
	BroadAddr string
	BroadPort int

	// [Echo] 维护口（缺省 15100+store_id）
	EchoAddr string
	EchoPort int

	// [Wake] 唤醒口（官方 DB 线程唤醒回路；Go 复刻 goroutine 通道等价，
	// 仅保留配置位与回显）
	WakeAddr string
	WakePort int

	// [Account]/[Role]/[Log]/[Public]/[Manage] 五库连接串（官方
	// SQLOLEDB 方言；Go 侧经 Backend 注入，仅作呈现/日志）
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

	// AppName 官方 "store_{server_id}_{store_id}"
	AppName string

	LogDBPerformance bool // LogDBPerformance（缺省关）
}

// DefaultConfig 官方缺省 + 快照 game_store.ini 实配值。
func DefaultConfig() Config {
	return Config{
		DistrictID:   1,
		ServerID:     1,
		StoreID:      0,
		UseMysql:     true,
		InnerAddr:    "127.0.0.1",
		InnerPort:    0,
		BroadAddr:    "127.255.255.255",
		BroadPort:    15000,
		EchoAddr:     "127.0.0.1",
		EchoPort:     15100,
		WakeAddr:     "127.0.0.1",
		WakePort:     0,
		AccountPools: 1,
		RolePools:    1,
		LogPools:     1,
		PublicPools:  1,
		ManagePools:  1,
	}
}

// FillDefaults 补空位（AppName 派生 + Echo 端口基址）。
func (c *Config) FillDefaults() {
	d := DefaultConfig()
	if c.DistrictID == 0 {
		c.DistrictID = d.DistrictID
	}
	if c.ServerID == 0 {
		c.ServerID = d.ServerID
	}
	if c.InnerAddr == "" {
		c.InnerAddr = d.InnerAddr
	}
	if c.BroadAddr == "" {
		c.BroadAddr = d.BroadAddr
	}
	if c.BroadPort == 0 {
		c.BroadPort = d.BroadPort
	}
	if c.EchoAddr == "" {
		c.EchoAddr = d.EchoAddr
	}
	if c.EchoPort == 0 {
		c.EchoPort = 15100 + c.StoreID
	}
	if c.WakeAddr == "" {
		c.WakeAddr = d.WakeAddr
	}
	if c.AccountPools <= 0 {
		c.AccountPools = 1
	}
	if c.RolePools <= 0 {
		c.RolePools = 1
	}
	if c.LogPools <= 0 {
		c.LogPools = 1
	}
	if c.PublicPools <= 0 {
		c.PublicPools = 1
	}
	if c.ManagePools <= 0 {
		c.ManagePools = 1
	}
	if c.AppName == "" {
		c.AppName = "store_" + itoa(c.ServerID) + "_" + itoa(c.StoreID)
	}
}

// appRecordFrame 官方 CAppRecord::SetAppInfo 行（TimerCheck 每 5 拍，
// `<info state="%s" database="%s" db_err="%d" tick="%d" />`，
// Port_SafeSprintf @0x100C50D7 逐字）。
func appRecordFrame(state string, database string, dbErr int, tick uint32) string {
	return "<info state=\"" + state + "\" database=\"" + database +
		"\" db_err=\"" + itoa(dbErr) + "\" tick=\"" + utoa(uint64(tick)) + "\" />"
}

// ParseGameStoreINI 解析 game_store.ini 文本（solo 方言；分词统一走
// fskit.ParseItems＝官方 CIniFile Build 方言，注释行 `//` 与分号均容忍
// ——快照 ini 以 `//` 注释；节键归一小写交 set；[Module] 裸行由装载器
// 承担，此处不收）。
func ParseGameStoreINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
	_ = fskit.ParseItems(data, func(sec, key, val string) error {
		cfg.set(strings.ToLower(sec), strings.ToLower(key), val)
		return nil
	})
	if cfg.EchoPort == 15100 {
		cfg.EchoPort += cfg.StoreID
	}
	return cfg, nil
}

func (c *Config) set(section, key, val string) {
	i := func(dst *int) {
		if n, err := strconv.Atoi(val); err == nil {
			*dst = n
		}
	}
	b := func(dst *bool) {
		switch strings.ToLower(val) {
		case "true", "1", "yes":
			*dst = true
		case "false", "0", "no", "":
			*dst = false
		}
	}
	switch section {
	case "main":
		switch key {
		case "districtid":
			i(&c.DistrictID)
		case "serverid":
			i(&c.ServerID)
		case "storeid":
			i(&c.StoreID)
		case "usemysql":
			b(&c.UseMysql)
		case "logdbperformance":
			b(&c.LogDBPerformance)
		}
	case "inner":
		switch key {
		case "addr":
			c.InnerAddr = val
		case "port":
			i(&c.InnerPort)
		}
	case "broadcast":
		switch key {
		case "addr":
			c.BroadAddr = val
		case "port":
			i(&c.BroadPort)
		}
	case "echo":
		switch key {
		case "addr":
			c.EchoAddr = val
		case "port":
			i(&c.EchoPort)
		}
	case "wake":
		switch key {
		case "addr":
			c.WakeAddr = val
		case "port":
			i(&c.WakePort)
		}
	case "account":
		switch key {
		case "connectstring":
			c.AccountBase = val
		case "connectpool":
			i(&c.AccountPools)
		}
	case "role":
		switch key {
		case "connectstring":
			c.RoleBase = val
		case "connectpool":
			i(&c.RolePools)
		}
	case "log":
		switch key {
		case "connectstring":
			c.LogBase = val
		case "connectpool":
			i(&c.LogPools)
		}
	case "public":
		switch key {
		case "connectstring":
			c.PublicBase = val
		case "connectpool":
			i(&c.PublicPools)
		}
	case "manage":
		switch key {
		case "connectstring":
			c.ManageBase = val
		case "connectpool":
			i(&c.ManagePools)
		}
	}
}

// appRecordFile 官方快照文件名（AppName + 日期）。
func appRecordFile(appName string, now time.Time) string {
	return appName + "-" + now.Format("20060102") + ".rec"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func utoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
