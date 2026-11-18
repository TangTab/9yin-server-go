package fskit

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const MainConsoleTypeName = "MainConsole"

// ColorAttr 官方 MainConsole::SetColor（0x100563c0）的 14 色表：
// stricmp 匹配名 → Win32 控制台属性值。
var ColorAttr = map[string]int{
	"Red":         4,
	"Blue":        1,
	"Green":       2,
	"Yellow":      6,
	"Purple":      5,
	"Cyan":        3,
	"LightRed":    0x0C,
	"LightBlue":   9,
	"LightGreen":  0x0A,
	"LightYellow": 0x0E,
	"LightPurple": 0x0D,
	"LightCyan":   0x0B,
	"Gray":        7,
	"White":       0x0F,
}

// attrToANSI 把 Win32 attr 折算成 ANSI SGR（前景位 0..F），供 Go 终端等价输出。
func attrToANSI(attr int) string {
	fg := attr & 0x07
	bright := attr&0x08 != 0
	code := 30 + fg
	if fg == 7 && bright { // White (0xF)
		return "\x1b[97m"
	}
	if bright {
		return fmt.Sprintf("\x1b[%d;1m", code)
	}
	return fmt.Sprintf("\x1b[%dm", code)
}

// MainConsole 复刻官方 fskit/mainconsole.cpp 实体。
type MainConsole struct {
	color string
	Out   io.Writer // 官方 printf → stdout；可注入便于测试
	ANSI  bool      // true 时 SetColor 附带 ANSI 转义（Go 化的 SetConsoleTextAttribute 等价）
}

func NewMainConsole() *MainConsole {
	return &MainConsole{Out: os.Stdout}
}

// Init 官方 0x100561f0：SetThreadExecutionState(ES_CONTINUOUS|SYSTEM_REQUIRED|
// DISPLAY_REQUIRED) + SetConsoleCtrlHandler(忽略 C/BREAK/CLOSE/LOGOFF/SHUTDOWN)。
// Go 侧：防休眠与控制事件忽略属宿主进程职责，此处保持空实现并记录语义。
func (m *MainConsole) Init() {}

// WriteLine 官方 0x10056330：printf("%s\r\n", line) → true（CRLF 定案）。
func (m *MainConsole) WriteLine(line string) bool {
	fmt.Fprintf(m.Out, "%s\r\n", line)
	return true
}

// SetColor 官方 0x100563c0：命色 → SetConsoleTextAttribute + 存名；
// 未知名 → 仅存名（不改色）。
func (m *MainConsole) SetColor(color string) {
	if attr, ok := ColorAttr[strings.ToLower(color)]; ok {
		if m.ANSI {
			fmt.Fprint(m.Out, attrToANSI(attr))
		}
		m.color = color
		return
	}
	m.color = color
}

// GetColor 官方 0x100566b0：返回最近一次 SetColor 的名字原样。
func (m *MainConsole) GetColor() string { return m.color }
