package login

import (
	"bufio"
	"fmt"
	gonet "net"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
)

// svrlist 方言（现代客户端的区服列表分发，独立于 EE+XOR 游戏线路的
// 明文 ASCII 行协议）。golden 钉死于 testdata/captures/svrlist-{request,
// response}.bin（旧分支 loopback-lister 对官服系方言的实测转录）：
//
//	请求：客户端连上后发一行 "svrlist\n"，读到 \r 或 \n 即完整请求
//	应答："$svrlist #\u672C\u5730\u6D4B\u8BD5 $127.0.0.1 19061 0\r\n"
//	      应答写完即断连（一连接一请求）
//
// 编码规格已由官方 FsSoloLister.dll 反汇编终局背书（2026-11-18，
// BGserverCJ\Debug 全符号快照，text_encode_msg@0x1005af80）：应答 =
// TVarList 文本编码——`$`+窄串（服务器名宽串 `#`+`#` 前缀逐 UTF-16 码元转
// 义）、整数十进制，元素间以单个空格分隔；多服 = 同一行顺序追加，每服
// 4 值（名/通告地址/端口/busy）连排，组间无额外分隔符。尾标 = 繁忙度
// busy（官方公式见 BusyLevel），golden 恒 0 = 空载。
const (
	svrlistRequest  = "svrlist"
	svrlistMaxLine  = 256 // 请求行长上限（官方 client_t 行缓冲同宽）
	svrlistNameDemo = "本地测试" // golden 样本名

	// svrlistVersionError 是官方版本门（ClientVersion>0）不达标时的
	// 应答行（TVarList{AddString("version_error")} 编码 + CRLF）。
	svrlistVersionError = "$version_error\r\n"
)

// ServerEntry 区服列表条目（官方 server_t 单线路槽的通告投影）。
type ServerEntry struct {
	Name string // 显示名（源内 UTF-8，线路按官方规则转义）
	Host string // 通告地址（客户端实际连接的游戏服 IP）
	Port int    // 通告端口
	Busy int    // 繁忙度尾标（官方公式 BusyLevel；golden 恒 0=空载）
}

// BusyLevel 官方繁忙度公式（ReceiveCommand@0x100634xx 实证）：
// 推荐值>10 直接采用；否则按 cur/max 满载五分位取整（cur*5/max）；
// max<=0 时为 0。
func BusyLevel(cur, max, recommend int) int {
	if recommend > 10 {
		return recommend
	}
	if max > 0 {
		return cur * 5 / max
	}
	return 0
}

// EncodeServerList 生成 svrlist 应答行（含 CRLF）。纯函数，golden 对拍
// 入口。多服按官方方言连排：`$svrlist #名 $ip 端口 busy #名2 $ip2 …`。
func EncodeServerList(entries ...ServerEntry) []byte {
	parts := make([]string, 0, 1+4*len(entries))
	parts = append(parts, "$svrlist")
	for _, e := range entries {
		parts = append(parts,
			"#"+escapeUnicode(e.Name),
			"$"+e.Host,
			fmt.Sprintf("%d", e.Port),
			fmt.Sprintf("%d", e.Busy),
		)
	}
	return []byte(strings.Join(parts, " ") + "\r\n")
}

// escapeUnicode 把名字转成线路形态，逐 UTF-16 码元按官方 text_encode_msg
// 规则转义（FsSoloLister.dll 宽串分支实证）：
//   - 可打印（0x20<u<0x7f）且非 '%'：低字节原样；'\' 追加双写
//   - u>=0x100：\u + 4 位大写十六进制（增补面拆代理对——golden 形态）
//   - 其余（空格/控制符/'%'/0x80~0xFF）：\x + 2 位大写十六进制
func escapeUnicode(s string) string {
	var b strings.Builder
	for _, u := range utf16.Encode([]rune(s)) {
		switch {
		case u == '%':
			fmt.Fprintf(&b, "\\x%02X", u)
		case u > 0x20 && u < 0x7f:
			b.WriteByte(byte(u))
			if u == '\\' {
				b.WriteByte('\\')
			}
		case u >= 0x100:
			fmt.Fprintf(&b, "\\u%04X", u)
		default:
			fmt.Fprintf(&b, "\\x%02X", u)
		}
	}
	return b.String()
}

// Lister 是区服列表分发服务：accept → 读一行请求 → 回一条列表 → 断连。
// 对应原版 GameLister 的 MassSock 职责（solo 形态收在 login 包，
// cmd/lister 是它的独立进程载体）。
type Lister struct {
	// Addr 监听地址；":0" 由测试取随机端口。
	Addr string
	// Entry 下发的首条区服条目（必填）。
	Entry ServerEntry
	// Extra 追加条目，按官方多服方言在同一应答行连排（可空）。
	Extra []ServerEntry
	// MinClientVersion 官方版本门（game_lister.ini ClientVersion）。
	// >0 时请求须为 "svrlist <ver>" 且 ver 达标才回列表，不达标回
	// $version_error；前缀/长度不符则静默不理（官方 return false 同款）。
	// 0（默认）不设门，任意行按 golden 行为应答。
	MinClientVersion int
	// Logf 可选日志钩子（连接/未知请求）。
	Logf func(format string, args ...any)

	mu       sync.Mutex
	listener gonet.Listener
	closed   bool
	wg       sync.WaitGroup
}

// Listen 绑定端口并返回实际地址。
func (l *Lister) Listen() (gonet.Addr, error) {
	ln, err := gonet.Listen("tcp", l.Addr)
	if err != nil {
		return nil, fmt.Errorf("lister listen %s: %w", l.Addr, err)
	}
	l.mu.Lock()
	l.listener = ln
	l.mu.Unlock()
	return ln.Addr(), nil
}

// Serve 阻塞 accept 循环，直到 Close 或 listener 出错。
func (l *Lister) Serve() error {
	for {
		l.mu.Lock()
		ln, closed := l.listener, l.closed
		l.mu.Unlock()
		if ln == nil || closed {
			return nil
		}
		conn, err := ln.Accept()
		if err != nil {
			l.mu.Lock()
			closed := l.closed
			l.mu.Unlock()
			if closed {
				return nil
			}
			return fmt.Errorf("lister accept: %w", err)
		}
		l.wg.Add(1)
		go l.serveConn(conn)
	}
}

// serveConn 一连接一请求：读行（旧实现不校验请求内容、有求必应，
// 保持同款行为；MinClientVersion>0 时启用官方版本门）、写应答、断连。
func (l *Lister) serveConn(conn gonet.Conn) {
	defer l.wg.Done()
	defer conn.Close()

	line, err := readLine(conn)
	if err != nil {
		l.logf("lister: 读请求失败: %v", err)
		return
	}
	req := strings.TrimRight(string(line), "\r\n")

	if l.MinClientVersion > 0 {
		// 官方 ReceiveCommand 版本门方言：行长>=8、前 7 字节 "svrlist"、
		// atoi(第 9 字符起) >= ClientVersion；前缀/长度不符静默不理，
		// 仅版本不达标回 $version_error。
		if len(req) < 8 || req[:7] != svrlistRequest {
			l.logf("lister: 版本门静默丢弃 %q", req)
			return
		}
		ver, _ := strconv.Atoi(req[8:])
		if ver < l.MinClientVersion {
			if _, err := conn.Write([]byte(svrlistVersionError)); err != nil {
				l.logf("lister: 写 version_error 失败: %v", err)
			}
			return
		}
	} else if req != svrlistRequest {
		l.logf("lister: 未知请求 %q（仍按方言应答）", req)
	}

	entries := make([]ServerEntry, 0, 1+len(l.Extra))
	entries = append(entries, l.Entry)
	entries = append(entries, l.Extra...)
	if _, err := conn.Write(EncodeServerList(entries...)); err != nil {
		l.logf("lister: 写应答失败: %v", err)
	}
}

// readLine 读到首个 \r 或 \n 为止（svrlist 请求定界方言），超限报错。
func readLine(conn gonet.Conn) ([]byte, error) {
	br := bufio.NewReader(conn)
	line := make([]byte, 0, 32)
	for {
		b, err := br.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == '\r' || b == '\n' {
			return line, nil
		}
		line = append(line, b)
		if len(line) > svrlistMaxLine {
			return nil, fmt.Errorf("lister request line over %d bytes", svrlistMaxLine)
		}
	}
}

// Close 关闭 listener 并等待在途会话退出（幂等）。
func (l *Lister) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	var err error
	if l.listener != nil {
		err = l.listener.Close()
	}
	l.mu.Unlock()
	l.wg.Wait()
	return err
}

func (l *Lister) logf(format string, args ...any) {
	if l.Logf != nil {
		l.Logf(format, args...)
	}
}

// DefaultListerEntry 由游戏服监听地址推导 svrlist 通告条目：
// 通告地址 = 游戏服监听 host（通配时回退 127.0.0.1），端口 = 游戏服端口。
func DefaultListerEntry(gameAddr string) ServerEntry {
	host, port, err := gonet.SplitHostPort(gameAddr)
	if err != nil || isWildcardHost(host) {
		host = "127.0.0.1"
	}
	if err != nil {
		port = "9600"
	}
	return ServerEntry{Name: svrlistNameDemo, Host: host, Port: parsePort(port), Busy: 0}
}

// ResolveEntryHost 通告地址留空或通配时，回退到监听地址的 host
// （仍通配则 127.0.0.1）；其余字段原样保留。
func ResolveEntryHost(e ServerEntry, listenAddr string) ServerEntry {
	if e.Host != "" && !isWildcardHost(e.Host) {
		return e
	}
	host, _, err := gonet.SplitHostPort(listenAddr)
	if err != nil || isWildcardHost(host) {
		host = "127.0.0.1"
	}
	e.Host = host
	return e
}

func isWildcardHost(host string) bool {
	return host == "" || host == "0.0.0.0" || host == "::" || host == "*"
}

func parsePort(s string) int {
	var p int
	fmt.Sscanf(s, "%d", &p)
	return p
}
