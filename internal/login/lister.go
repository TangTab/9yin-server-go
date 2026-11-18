package login

import (
	"bufio"
	"fmt"
	gonet "net"
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
// 字段：`$svrlist` 固定头｜`#`+`\uXXXX`（大写十六进制 UTF-16 码元）转义的
// 服务器名｜`$`+通告地址｜游戏端口｜尾标（golden 恒 0，语义未证）。
// `#`/`$` 前缀语义未见官方文档，按对拍原样保留。
const (
	svrlistRequest  = "svrlist"
	svrlistMaxLine  = 256 // 请求行长上限（请求恒 7B，超限即线路异常）
	svrlistNameDemo = "本地测试" // golden 样本名
)

// ServerEntry 区服列表条目。当前 golden 只观测到单条目应答形态；
// 多条目的分隔方言未知，不做发明——需要列表时以新抓包扩展。
type ServerEntry struct {
	Name string // 显示名（源内 UTF-8，线路转 \uXXXX）
	Host string // 通告地址（客户端实际连接的游戏服 IP）
	Port int    // 通告端口
	Flag int    // 应答尾标，golden 恒 0（语义未证）
}

// EncodeServerList 生成 svrlist 应答行（含 CRLF）。纯函数，golden 对拍入口。
func EncodeServerList(e ServerEntry) []byte {
	var b strings.Builder
	b.WriteString("$svrlist")
	b.WriteString(" #")
	b.WriteString(escapeUnicode(e.Name))
	b.WriteString(" $")
	b.WriteString(e.Host)
	b.WriteByte(' ')
	b.WriteString(fmt.Sprintf("%d %d\r\n", e.Port, e.Flag))
	return []byte(b.String())
}

// escapeUnicode 把名字转成线路形态：非 ASCII 码元写成 \uXXXX（大写十六进制、
// UTF-16 码元，增补面拆代理对——golden 的 \u672C\u5730\u6D4B\u8BD5 形态），
// ASCII 可打印字符原样保留（golden 无 ASCII 样本，按通用惯例假设）。
func escapeUnicode(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x80 {
			b.WriteRune(r)
			continue
		}
		for _, u := range utf16.Encode([]rune{r}) {
			fmt.Fprintf(&b, "\\u%04X", u)
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
	// Entry 下发的区服条目。
	Entry ServerEntry
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
// 保持同款行为）、写应答、断连。
func (l *Lister) serveConn(conn gonet.Conn) {
	defer l.wg.Done()
	defer conn.Close()

	line, err := readLine(conn)
	if err != nil {
		l.logf("lister: 读请求失败: %v", err)
		return
	}
	req := strings.TrimRight(string(line), "\r\n")
	if req != svrlistRequest {
		l.logf("lister: 未知请求 %q（仍按方言应答）", req)
	}
	if _, err := conn.Write(EncodeServerList(l.Entry)); err != nil {
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
	return ServerEntry{Name: svrlistNameDemo, Host: host, Port: parsePort(port), Flag: 0}
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
