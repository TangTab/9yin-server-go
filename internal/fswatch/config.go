package fswatch

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DefaultGameWatchINI 内置默认配置（官方 GameWatch.ini 方言，
// Outer/Echo 改为回环可用缺省）：cmd/watch 未指定路径且当前目录没有
// GameWatch.ini 时释放到当前目录并采用（已存在的绝不覆盖）。
//
//go:embed game_watch.ini
var DefaultGameWatchINI []byte

// ServerEntry 官方 server_data_t（AddServer(key, val, "") 的 name/path 双槽，
// ReceiveServers 交替回推 name(+0)/path(+0x30)）。
type ServerEntry struct {
	Name string // [Servers] key
	Path string // [Servers] val（启动命令行）
}

// AccessEntry 官方 access.ini 节（CQueryDb::LoadFromFile 三键
// app_name/type/connect_string，rdata 0x100f9764/0x100f975c/0x100f9748）。
type AccessEntry struct {
	Section       string
	AppName       string
	Type          string
	ConnectString string
}

// Config 是 GameWatch.ini 的受面子集（main.lua 全量读取面）。
type Config struct {
	Sleep           int // [System] 循环休眠毫秒（<0 不休眠）；官方 1
	MonitorPassword string
	QueryPassword   string
	UpdatePassword  string
	LockSystem      bool
	MaxClient       int
	OuterAddr       string
	OuterPort       int
	EchoAddr        string
	EchoPort        int
	ClientAddresses []string          // [Address] AddressN 白名单
	Servers         []ServerEntry     // [Servers]
	ExePaths        map[string]string // [ExePath]
	ResPaths        map[string]string // [ResPath]
	Access          []AccessEntry     // access.ini（可选）
}

// DefaultConfig 官方 GameWatch.ini 方言缺省（Outer 回环可用、Echo 官方缺省
// 127.0.0.1:15006）。
func DefaultConfig() *Config {
	return &Config{
		Sleep:     1,
		MaxClient: 16,
		OuterAddr: "127.0.0.1",
		OuterPort: 6000,
		EchoAddr:  "127.0.0.1",
		EchoPort:  15006,
		ExePaths:  map[string]string{},
		ResPaths:  map[string]string{},
	}
}

// iniSections 把官方 ini（'//' 行注释、[节] 键=值）解析成有序节表。
// 官方 IniDocument 允许无键节的列表项（[Module] 下裸 DLL 名），此处一并
// 保留为空键项。
func iniSections(path string) (map[string][][2]string, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	secs := map[string][][2]string{}
	var order []string
	cur := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			cur = strings.TrimSpace(line[1 : len(line)-1])
			if _, ok := secs[cur]; !ok {
				order = append(order, cur)
			}
			continue
		}
		if i := strings.IndexByte(line, '='); i >= 0 {
			secs[cur] = append(secs[cur], [2]string{strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])})
		} else {
			secs[cur] = append(secs[cur], [2]string{"", line})
		}
	}
	return secs, order, sc.Err()
}

func (c *Config) apply(secs map[string][][2]string) error {
	get := func(sec, key, def string) string {
		for _, kv := range secs[sec] {
			if kv[0] != "" && strings.EqualFold(kv[0], key) {
				return kv[1]
			}
		}
		return def
	}
	getInt := func(sec, key string, def int) int {
		if v := get(sec, key, ""); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		return def
	}
	c.Sleep = getInt("System", "Sleep", c.Sleep)
	c.MaxClient = getInt("Main", "MaxClient", c.MaxClient)
	c.MonitorPassword = get("Main", "MonitorPassword", c.MonitorPassword)
	c.QueryPassword = get("Main", "QueryPassword", c.QueryPassword)
	c.UpdatePassword = get("Main", "UpdatePassword", c.UpdatePassword)
	if v := strings.ToLower(get("Main", "LockSystem", "")); v == "1" || v == "true" {
		c.LockSystem = true
	}
	c.OuterAddr = get("Outer", "Addr", c.OuterAddr)
	c.OuterPort = getInt("Outer", "Port", c.OuterPort)
	c.EchoAddr = get("Echo", "Addr", c.EchoAddr)
	c.EchoPort = getInt("Echo", "Port", c.EchoPort)

	for _, kv := range secs["Servers"] {
		if kv[0] != "" && kv[1] != "" {
			c.Servers = append(c.Servers, ServerEntry{Name: kv[0], Path: kv[1]})
		}
	}
	for _, kv := range secs["ExePath"] {
		if kv[0] != "" && kv[1] != "" {
			c.ExePaths[kv[0]] = kv[1]
		}
	}
	for _, kv := range secs["ResPath"] {
		if kv[0] != "" && kv[1] != "" {
			c.ResPaths[kv[0]] = kv[1]
		}
	}
	num := getInt("Address", "AddressNum", 0)
	for i := 0; i < num; i++ {
		if v := get("Address", fmt.Sprintf("Address%d", i), ""); v != "" {
			c.ClientAddresses = append(c.ClientAddresses, v)
		}
	}
	return nil
}

// LoadConfig 读 GameWatch.ini；accessIni 非空时一并装载（官方
// game_watch:LoadAccessConfig("access.ini")，文件缺省不报错——空表）。
func LoadConfig(path, accessIni string) (*Config, error) {
	c := DefaultConfig()
	secs, _, err := iniSections(path)
	if err != nil {
		return nil, fmt.Errorf("fswatch: read %s: %w", path, err)
	}
	if err := c.apply(secs); err != nil {
		return nil, err
	}
	if accessIni != "" {
		if err := c.LoadAccessConfig(accessIni); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// LoadAccessConfig 官方 CQueryDb::LoadFromFile：逐节取 app_name/type/
// connect_string 三键，缺节键跳过（InnerAddAccess 失败仅日志）。
func (c *Config) LoadAccessConfig(path string) error {
	secs, order, err := iniSections(path)
	if err != nil {
		return fmt.Errorf("fswatch: read %s: %w", path, err)
	}
	for _, sec := range order {
		get := func(key string) string {
			for _, kv := range secs[sec] {
				if kv[0] != "" && strings.EqualFold(kv[0], key) {
					return kv[1]
				}
			}
			return ""
		}
		e := AccessEntry{Section: sec, AppName: get("app_name"), Type: get("type"), ConnectString: get("connect_string")}
		if e.AppName == "" && e.Type == "" && e.ConnectString == "" {
			continue
		}
		c.Access = append(c.Access, e)
	}
	return nil
}

// DestPath 官方 GameWatch::GetDestPath(0x6edf0)：[ExePath] 键优先，
// 其次 [ResPath]，都没有 → ok=false（update 分支回 "no dest path"）。
func (c *Config) DestPath(server string) (string, bool) {
	if v, ok := c.ExePaths[server]; ok {
		return v, true
	}
	if v, ok := c.ResPaths[server]; ok {
		return v, true
	}
	return "", false
}
