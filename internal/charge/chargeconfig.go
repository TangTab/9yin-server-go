// chargeconfig.go 官方 GameCharge.ini 解析（FsLoader 启动脚本读入，
// main.lua 消费；节构 [System]/[Module]/[Script]/[Main]/[District]/
// [Center]/[Outer]/[Echo]）。Go 复刻消费 [Main] 与 [Outer]；数据库节
// 由 Backend 实现方接线（MySQL 后端用 [District] ConnString DSN 或
// 独立配置，Oracle 串本身不复用）。
package charge

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LoadOrCreateGameChargeINI 读取 path 配置；不存在时写入内置默认 def
// 并返回（created=true）。已存在的绝不覆盖（同 login 家族口径）。
func LoadOrCreateGameChargeINI(path string, def []byte) (data []byte, created bool, err error) {
	if b, err := os.ReadFile(path); err == nil {
		return b, false, nil
	} else if !os.IsNotExist(err) {
		return nil, false, err
	}
	if err := os.WriteFile(path, def, 0o644); err != nil {
		return nil, false, err
	}
	return def, true, nil
}

// ParseGameChargeINI 解析官方 GameCharge.ini 文本（方言同
// ParseGameListerINI：`[节]`/`键=值`/`//`、`;`、`#` 注释；未知节键忽略）。
// 键名大小写不敏感（官方 ini 读写组件同特性）。
func ParseGameChargeINI(data []byte) (Config, error) {
	cfg := DefaultConfig()
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
			if strings.EqualFold(section, "Module") || strings.EqualFold(section, "System") ||
				strings.EqualFold(section, "Script") || strings.EqualFold(section, "Resource") {
				continue
			}
			return cfg, fmt.Errorf("gamecharge ini 第 %d 行无 '='：%q", i+1, line)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if err := cfg.set(section, key, value); err != nil {
			return cfg, fmt.Errorf("gamecharge ini 第 %d 行 %s/%s: %w", i+1, section, key, err)
		}
	}
	if cfg.Addr == "" {
		return cfg, fmt.Errorf("gamecharge ini 缺 [Outer] Addr")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return cfg, fmt.Errorf("gamecharge ini [Outer] Port 非法：%d", cfg.Port)
	}
	return cfg, nil
}

func (c *Config) set(section, key, value string) error {
	sec := strings.ToLower(section)
	key = strings.ToLower(key)
	inMain := sec == "" || sec == "main"
	switch {
	case inMain && key == "gameid":
		return setInt(&c.GameID, value, "GameID")
	case inMain && key == "serveridoffset":
		return setInt(&c.ServerIDOffset, value, "ServerIDOffset")
	case inMain && key == "pointminutes":
		return setInt(&c.PointMinutes, value, "PointMinutes")
	case inMain && key == "autodumpminutes":
		return setInt(&c.AutoDumpMin, value, "AutoDumpMinutes")
	case inMain && key == "awaketime":
		return setInt(&c.AwakeSec, value, "AwakeTime")
	case inMain && key == "flusherrormax":
		return setInt(&c.FlushErrorMax, value, "FlushErrorMax")
	case inMain && key == "usershaketime":
		return setInt(&c.UserShakeSec, value, "UserShakeTime")
	case inMain && key == "maxclient":
		return setInt(&c.MaxClient, value, "MaxClient")
	case inMain && (key == "flushlog" || key == "consolelog" || key == "filelog"):
		c.FlushLog = truthy(value)
		return nil
	case sec == "outer" && key == "addr":
		c.Addr = value
		return nil
	case sec == "outer" && key == "port":
		return setInt(&c.Port, value, "Port")
	}
	return nil // 未知键忽略（前向兼容）
}

func setInt(dst *int, value, name string) error {
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%s 须为整数：%q", name, value)
	}
	*dst = n
	return nil
}

// truthy 官方 nx_boolean 字符串口径：非空且非 0/false 即真。
func truthy(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v != "" && v != "0" && v != "false"
}
