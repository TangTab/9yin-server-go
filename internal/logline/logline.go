// Package logline 给进程 stdout/stderr 的每一行加「时间 服务名」前缀。
//
// 仓库里的日志是散落的 fmt.Printf/Println（没有统一日志器），这里不改
// 调用点：Install 把 os.Stdout/os.Stderr 换成管道写端，转发协程按行
// 切分、加前缀后写回真实 stdout/stderr。控制台窗口、bat 重定向、面板
// 托管落盘（logs/<服务>-<日期>.log）看到的是同一份带前缀的行：
//
//	2006-01-02 15:04:05.000 <server> <原行>
//
// 约定：
//   - Install 在 flag.Parse 之后、首条输出之前调用（flag 自身的用法/
//     报错不经前缀，ExitOnError 的退出也不欠排空）；
//   - 此后退出一律走 Exit / defer Flush——os.Exit 与 main 返回都不给
//     转发协程机会，直接退出会丢最后几行（含致命错误）；
//   - 空行保持空行；无换行收尾的半行在进程收尾时补发；
//   - 不要把本进程的 os.Stdout/os.Stderr 原样交给子进程（exec.Cmd 的
//     Stdout/Stderr 留空即如此）：子进程的输出会经本进程转发、被标成
//     本进程的来源。
package logline

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// timeLayout 前缀时间：本地时间，毫秒。
const timeLayout = "2006-01-02 15:04:05.000"

// drainTimeout 退出前排空的最长等待。正常为微秒级；兜底真实目的地
// 写阻塞（如控制台处于选择态）——到点照常退出，不再卡住进程。
const drainTimeout = 2 * time.Second

// mark 实体标签在管道流内的标记，置于行首、转发协程据此换该行的来源
// （正文不会出现 NUL）：
//
//	\x00<server>\x00   Tagged：这一行来源为 <server>
//	\x00=<server>\x00  SetServer：此后的缺省来源为 <server>
//
// 标记按写入顺序排在流里：先写的行用旧来源，后写的行用新来源。
const mark byte = 0

var cur atomic.Pointer[sink]

type sink struct {
	mu  sync.Mutex // 两路转发的写出互斥（防半行交错）
	out *stream    // stdout
	err *stream    // stderr

	flushOnce sync.Once
}

// stream 一路转发：管道读端 → 按行加前缀 → dst。
type stream struct {
	sk  *sink
	r   *os.File // 管道读端
	w   *os.File // 管道写端（= os.Stdout / os.Stderr；Flush 时关闭）
	dst io.Writer

	def   string // 当前行区间适用的进程缺省来源（SetServer 标记改写）
	rest  []byte // 未完结的行（尚无换行）
	name  string // rest 所属行的实体来源（Tagged 标记；空＝def）
	batch []byte
	done  chan struct{}
}

// Install 给本进程 stdout/stderr 接上行前缀转发，来源标签 server。
// 幂等（首次生效）；管道建立失败则保持原样输出，并在 stderr 提示。
func Install(server string) {
	if cur.Load() != nil {
		return
	}
	s, err := newSink(server, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logline: 行前缀转发未启用：%v\n", err)
		return
	}
	attach(s)
	go s.out.run()
	go s.err.run()
}

// attach 换流并登记。标准库 log 在包初始化时就缓存了 os.Stderr，这里
// 一并改接到管道，并去掉它自带的日期（时间由行前缀给）——否则这类行
// （dbconn 的缺省日志等）没有服务名、又是另一套时间格式。
func attach(s *sink) {
	// 先换流再登记：登记之后 Tagged 才会往管道写标记。
	os.Stdout, os.Stderr = s.out.w, s.err.w
	log.SetFlags(0)
	log.SetOutput(os.Stderr)
	cur.Store(s)
}

// SetServer 改进程缺省来源标签——启动早期来源未定时先 Install 再改
// （如 member 序号要等配置装载后才知道）。未 Install 时无操作。
func SetServer(server string) {
	if cur.Load() != nil {
		m := []byte{mark, '='}
		os.Stdout.Write(append(append(m, server...), mark))
		os.Stderr.Write(append(append(m, server...), mark))
	}
}

// Tagged 返回带来源标签的 Logf：已 Install 时，这条输出的来源标签换成
// server（合一进程内据此区分各实体）；未 Install 时原样转给 next（单测
// 的捕获 Logf 不受影响）。next 为空即丢弃。
func Tagged(server string, next func(format string, args ...any)) func(format string, args ...any) {
	return func(format string, args ...any) {
		s := cur.Load()
		if s == nil {
			if next != nil {
				next(format, args...)
			}
			return
		}
		msg := fmt.Sprintf(format, args...)
		var b strings.Builder
		for len(msg) > 0 {
			line := msg
			if i := strings.IndexByte(msg, '\n'); i >= 0 {
				line, msg = msg[:i], msg[i+1:]
			} else {
				msg = ""
			}
			b.WriteByte(mark)
			b.WriteString(server)
			b.WriteByte(mark)
			b.WriteString(line)
			b.WriteByte('\n')
		}
		os.Stdout.WriteString(b.String()) // 整条一次写出，不与他人的写交错
	}
}

// Flush 关闭转发入口并等两路排空（至多 drainTimeout），幂等。进程收尾
// 用（main 返回处 defer）；此后再写 stdout/stderr 即丢弃。
func Flush() {
	s := cur.Load()
	if s == nil {
		return
	}
	s.flushOnce.Do(func() {
		s.out.w.Close()
		s.err.w.Close()
		for _, st := range []*stream{s.out, s.err} {
			select {
			case <-st.done:
			case <-time.After(drainTimeout):
			}
		}
	})
}

// Exit 排空转发队列后以 code 退出——已 Install 的进程里替代 os.Exit。
func Exit(code int) {
	Flush()
	os.Exit(code)
}

func newSink(server string, dstOut, dstErr io.Writer) (*sink, error) {
	or, ow, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	er, ew, err := os.Pipe()
	if err != nil {
		or.Close()
		ow.Close()
		return nil, err
	}
	s := &sink{}
	s.out = &stream{sk: s, r: or, w: ow, dst: dstOut, def: server, done: make(chan struct{})}
	s.err = &stream{sk: s, r: er, w: ew, dst: dstErr, def: server, done: make(chan struct{})}
	return s, nil
}

func (p *stream) run() {
	defer close(p.done)
	buf := make([]byte, 32*1024)
	for {
		n, err := p.r.Read(buf)
		if n > 0 {
			p.consume(buf[:n])
		}
		if err != nil { // EOF：写端已关（Flush）且余量读尽
			break
		}
	}
	if len(p.rest) > 0 {
		p.emitLine(p.rest) // 收尾补发无换行的末行
		p.write()
	}
}

// consume 吃一段字节：遇 '\n' 出一行；遇标记换来源。
func (p *stream) consume(data []byte) {
	p.rest = append(p.rest, data...)
	for {
		nl := bytes.IndexByte(p.rest, '\n')
		m := bytes.IndexByte(p.rest, mark)
		if m >= 0 && (nl < 0 || m < nl) {
			j := bytes.IndexByte(p.rest[m+1:], mark)
			if j < 0 {
				break // 标记未完整：等后续字节
			}
			if m > 0 {
				p.emitLine(p.rest[:m]) // 行中标记＝强制断行（上一行没以换行收尾）
			}
			name := string(p.rest[m+1 : m+1+j])
			p.rest = p.rest[m+j+2:]
			if strings.HasPrefix(name, "=") {
				p.def, name = name[1:], "" // SetServer：此后的缺省来源
			}
			p.name = name
			continue
		}
		if nl < 0 {
			break
		}
		p.emitLine(p.rest[:nl])
		p.rest = p.rest[nl+1:]
		p.name = "" // 标记只管一行
	}
	p.write()
}

// emitLine 往当前批次追加一行（带前缀）。
func (p *stream) emitLine(line []byte) {
	if len(bytes.TrimSuffix(line, []byte{'\r'})) == 0 {
		p.batch = append(p.batch, line...) // 空行保持空行
		p.batch = append(p.batch, '\n')
		return
	}
	server := p.name
	if server == "" {
		server = p.def
	}
	line = trimSelfTag(line, server)
	p.batch = append(p.batch, time.Now().Format(timeLayout)...)
	p.batch = append(p.batch, ' ')
	p.batch = append(p.batch, server...)
	if len(line) > 0 {
		p.batch = append(p.batch, ' ')
		p.batch = append(p.batch, line...)
	}
	p.batch = append(p.batch, '\n')
}

func (p *stream) write() {
	if len(p.batch) == 0 {
		return
	}
	p.sk.mu.Lock()
	p.dst.Write(p.batch)
	p.sk.mu.Unlock()
	p.batch = p.batch[:0]
}

// trimSelfTag 行首已是同名来源标记（「[monitor] …」「(world)…」——各
// 进程/实体历史上自带的来源前缀）时去掉，免得「monitor [monitor] …」。
func trimSelfTag(line []byte, server string) []byte {
	if len(line) < 2 || (line[0] != '[' && line[0] != '(') {
		return line
	}
	closing := byte(']')
	if line[0] == '(' {
		closing = ')'
	}
	i := bytes.IndexByte(line, closing)
	if i < 0 || string(line[1:i]) != server {
		return line
	}
	line = line[i+1:]
	return bytes.TrimPrefix(line, []byte{' '})
}
