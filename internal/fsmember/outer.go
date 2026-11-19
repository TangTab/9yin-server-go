package fsmember

import (
	"fmt"
	gonet "net"
	"strconv"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
)

// Outer 复刻 CInterConnect 发面：按 ident 寻路的主动外连池。
// 每路一个 fscomm.MultiSock（自动重连）；连上即发 0xFFFF 注册帧
// 声明自身 ident（与 world 连入 entry 时的行为同族）。
type Outer struct {
	ownIdent uint32

	mu    sync.Mutex
	links map[uint32]*fscomm.MultiSock
}

// NewOuter 创建发面；ownIdent 为本 member 内网标识（0x18+MemberID）。
func NewOuter(ownIdent uint32) *Outer {
	return &Outer{ownIdent: ownIdent, links: make(map[uint32]*fscomm.MultiSock)}
}

// AddLink 登记一路外连（ident → addr）。重复登记替换旧路。
func (o *Outer) AddLink(ident uint32, addr string, onState func(ident uint32, connected bool)) {
	host, port, err := splitHostPort(addr)
	if err != nil {
		return
	}
	selfIdent := o.ownIdent
	var sock *fscomm.MultiSock
	sock = fscomm.NewMultiSock(fscomm.MultiSockOptions{
		Address:    host,
		Port:       port,
		AutoRetry:  true,
		RetryDelay: 3 * time.Second,
		OutBufLen:  1024 * 1024,
	}, fscomm.MultiSockHandler{
		OnConnected: func() {
			// 注册帧：{src=ownIdent, dst=0, type=0xFFFF}
			_ = sock.Send(encodeWire(selfIdent, 0, IdentRegister, nil))
			if onState != nil {
				onState(ident, true)
			}
		},
		OnClose: func(err error) {
			if onState != nil {
				onState(ident, false)
			}
		},
		OnRead: func(data []byte) {}, // 发面：对端回包走受面（同一 ident 双向）
	})
	o.mu.Lock()
	if old, ok := o.links[ident]; ok {
		old.Stop()
	}
	o.links[ident] = sock
	o.mu.Unlock()
}

// Start 启动全部外连。
func (o *Outer) Start() {
	o.mu.Lock()
	links := make([]*fscomm.MultiSock, 0, len(o.links))
	for _, l := range o.links {
		links = append(links, l)
	}
	o.mu.Unlock()
	for _, l := range links {
		l.Start()
	}
}

// Stop 停止全部外连。
func (o *Outer) Stop() {
	o.mu.Lock()
	links := make([]*fscomm.MultiSock, 0, len(o.links))
	for _, l := range o.links {
		links = append(links, l)
	}
	o.links = make(map[uint32]*fscomm.MultiSock)
	o.mu.Unlock()
	for _, l := range links {
		l.Stop()
	}
}

// Send 发 {src,dst,type}+payload 到 ident 对应外连。
func (o *Outer) Send(ident, typ uint32, payload []byte) error {
	o.mu.Lock()
	sock, ok := o.links[ident]
	o.mu.Unlock()
	if !ok || !sock.Connected() {
		return errNotConnected(ident)
	}
	return sock.Send(encodeWire(o.ownIdent, ident, typ, payload))
}

// Connected 指定 ident 是否在网。
func (o *Outer) Connected(ident uint32) bool {
	o.mu.Lock()
	sock, ok := o.links[ident]
	o.mu.Unlock()
	return ok && sock.Connected()
}

func errNotConnected(ident uint32) error {
	return fmt.Errorf("(CInterConnect::SendMsg)not connected: comm_id %d", ident)
}

func splitHostPort(addr string) (string, int, error) {
	host, portStr, err := gonet.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}
