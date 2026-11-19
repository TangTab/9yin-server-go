package fssololog

import (
	"fmt"
	"strconv"
	"strings"
)

// echoResponse 官方 echo_response：">" + 信息 + "\r"（SendEchoLine 再
// 叠 "\r\n" → 客户端每命令见双换行，官方行为）。
func echoResponse(info string) string { return ">" + info + "\r" }

// logEntityName 本实体反射名（main.lua nx_set_value("logger", logger)；
// solo_log/SoloLog 为等价别名）。
var logEntityAliases = map[string]bool{"solo_log": true, "logger": true, "SoloLog": true}

// echoDispatch 官方 ReceiveInput 分发（§6.2）+ main.lua 命令集（§6.4）：
//
//	args = parse_string(line)
//	空/首参非串 → on_input(原文) → undefined 兜底
//	cmd = lower(args[0])；quit/end 具名；getp/setp/… 反射族；
//	其余 → ">undefined command: <line>\r"
func (s *Server) echoDispatch(line string) (string, bool) {
	args := ParseLine(line)
	if len(args) == 0 {
		return echoResponse("undefined command: " + line), false
	}
	cmd, ok := args[0].(string)
	if !ok {
		return echoResponse("undefined command: " + line), false
	}
	switch strings.ToLower(cmd) {
	case "quit":
		// 官方：echo_response("server will shutdown...") → close()
		//（WriteLine + Stop + nx_quit）。
		go func() {
			s.WriteLine("server will shutdown...")
			s.Stop()
		}()
		return echoResponse("server will shutdown..."), false
	case "end":
		// 官方 CloseEcho()：仅断开本维护连接。
		return "", true
	case "getp":
		return s.cmdGetProperty(args), false
	case "setp":
		return s.cmdSetProperty(args), false
	case "getc":
		// 官方查全局类属性；Go 等价反射面落本实体类（包文档取舍）。
		return s.cmdGetProperty(args), false
	case "setc":
		return s.cmdSetProperty(args), false
	case "method":
		return s.cmdMethod(args)
	case "listp":
		return s.cmdListProperties(args), false
	case "listc":
		return echoResponse("SoloLog"), false
	case "listm":
		return echoResponse("Start Stop SetConsoleColor WriteLine SendEchoLine CloseEcho"), false
	case "listg":
		return echoResponse("logger"), false
	case "function":
		// 官方列全局函数表；Go 等价列内置命令集。
		return echoResponse("quit end getp setp getc setc method listp listc listm listg function debugger"), false
	case "debugger":
		// 官方调试器开关；Go 复刻无此面。
		return echoResponse("debugger: not supported in go replica"), false
	case "state":
		// Go 扩展（与 soloentry state 同型）：状态速查。
		return echoResponse(s.State()), false
	case "help":
		// Go 扩展：命令帮助。
		return echoResponse("commands: quit end getp setp getc setc method listp listc listm listg function debugger state help"), false
	default:
		return echoResponse("undefined command: " + line), false
	}
}

// logProperties 官方 7 Lua 属性（§3.2；State 只读）。
var logProperties = []struct{ Name, Kind string }{
	{"UseThread", "bool"},
	{"AutoExit", "bool"},
	{"EchoAddr", "string"},
	{"EchoPort", "int"},
	{"EveryDayAlone", "bool"},
	{"LogFileDir", "string"},
	{"State", "string"},
}

func (s *Server) cmdListProperties(args []any) string {
	if len(args) >= 2 && !isLogEntity(args[1]) {
		return echoResponse("entity not found: " + anyText(args[1]))
	}
	var b strings.Builder
	for i, p := range logProperties {
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%s(%s)", p.Name, p.Kind)
	}
	return echoResponse(b.String())
}

func (s *Server) cmdGetProperty(args []any) string {
	if len(args) < 3 || !isLogEntity(args[1]) {
		return echoResponse("usage: getp solo_log <property>")
	}
	name, _ := args[2].(string)
	v, ok := s.propertyGet(name)
	if !ok {
		return echoResponse("property not found: " + name)
	}
	return echoResponse(v)
}

func (s *Server) cmdSetProperty(args []any) string {
	if len(args) < 4 || !isLogEntity(args[1]) {
		return echoResponse("usage: setp solo_log <property> <value>")
	}
	name, _ := args[2].(string)
	if err := s.propertySet(name, anyText(args[3])); err != nil {
		return echoResponse(err.Error())
	}
	return echoResponse("ok")
}

func (s *Server) cmdMethod(args []any) (string, bool) {
	if len(args) < 3 || !isLogEntity(args[1]) {
		return echoResponse("usage: method solo_log <method> [args...]"), false
	}
	method, _ := args[2].(string)
	rest := args[3:]
	text := strings.TrimSpace(strings.Join(textArgs(rest), " "))
	switch strings.ToLower(method) {
	case "writeline":
		if text == "" {
			return echoResponse("assert: info must be non-empty"), false
		}
		s.WriteLine(text)
		return echoResponse("ok"), false
	case "sendecholine":
		if text == "" {
			return echoResponse("assert: info must be non-empty"), false
		}
		s.SendEchoLine(echoResponse(text))
		return echoResponse("ok"), false
	case "setconsolecolor":
		s.SetConsoleColor(text) // 非控制台 no-op
		return echoResponse("ok"), false
	case "closeecho":
		s.CloseEcho()
		return "", true // 断开本连接
	case "start":
		return echoResponse("ok"), false // 已在运行（单实例互斥下不可重入）
	case "stop":
		go func() {
			s.WriteLine("server will shutdown...")
			s.Stop()
		}()
		return echoResponse("server will shutdown..."), false
	default:
		return echoResponse("method not found: " + method), false
	}
}

// propertyGet/propertySet 本实体属性面（m_nState 等在锁内取）。
func (s *Server) propertyGet(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "usethread":
		return strconv.FormatBool(s.opt.UseThread), true
	case "autoexit":
		s.mu.Lock()
		defer s.mu.Unlock()
		return strconv.FormatBool(s.opt.AutoExit), true
	case "echoaddr":
		return s.opt.EchoAddr, true
	case "echoport":
		return strconv.Itoa(s.opt.EchoPort), true
	case "everydayalone":
		s.mu.Lock()
		defer s.mu.Unlock()
		return strconv.FormatBool(s.opt.EveryDayAlone), true
	case "logfiledir":
		return s.opt.LogFileDir, true
	case "state":
		return s.State(), true
	default:
		return "", false
	}
}

func (s *Server) propertySet(name, value string) error {
	switch strings.ToLower(name) {
	case "usethread":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return errBadValue(value)
		}
		s.opt.UseThread = b // Go no-op 面（包文档取舍）
		return nil
	case "autoexit":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return errBadValue(value)
		}
		s.mu.Lock()
		s.opt.AutoExit = b
		s.mu.Unlock()
		return nil
	case "echoaddr":
		s.mu.Lock()
		s.opt.EchoAddr = value
		s.mu.Unlock()
		return nil
	case "echoport":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 || n > 65535 {
			return errBadValue(value)
		}
		s.mu.Lock()
		s.opt.EchoPort = n
		s.mu.Unlock()
		return nil
	case "everydayalone":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return errBadValue(value)
		}
		s.mu.Lock()
		s.opt.EveryDayAlone = b
		s.mu.Unlock()
		return nil
	case "logfiledir":
		s.mu.Lock()
		s.opt.LogFileDir = value
		s.mu.Unlock()
		return nil
	case "state":
		return fmt.Errorf("property is read-only: State")
	default:
		return fmt.Errorf("property not found: %s", name)
	}
}

func errBadValue(v string) error { return fmt.Errorf("bad value: %s", v) }

func isLogEntity(arg any) bool {
	name, ok := arg.(string)
	return ok && logEntityAliases[name]
}

func anyText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func textArgs(args []any) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, anyText(a))
	}
	return out
}
