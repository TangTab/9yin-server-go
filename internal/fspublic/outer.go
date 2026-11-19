package fspublic

import (
	"fmt"
	gonet "net"
	"strconv"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
)

// Outer 复刻 CInterConnect 主动外连面（public 侧 CheckWorld/StoreConnect
// 的重连路径）：world/store 由 UDP 广播发现后幂等补连。member 不外连
//（member 主动连 public）。
type Outer struct {
	ownIdent uint32

	mu    sync.Mutex
	links map[uint32]*fscomm.MultiSock
}

// NewOuter 创建发面（ownIdent=1）。
func NewOuter(ownIdent uint32) *Outer {
	return &Outer{ownIdent: ownIdent, links: make(map[uint32]*fscomm.MultiSock)}
}

// AddLink 登记一路外连（ident → addr）。
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
		OnRead: func(data []byte) {},
	})
	o.mu.Lock()
	if old, ok := o.links[ident]; ok {
		old.Stop()
	}
	o.links[ident] = sock
	o.mu.Unlock()
}

// Start/Stop 全部外连启停。
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

// Send 发 {src=1, dst=0, type}+payload 到 ident 对应外连。
func (o *Outer) Send(ident, typ uint32, payload []byte) error {
	o.mu.Lock()
	sock, ok := o.links[ident]
	o.mu.Unlock()
	if !ok || !sock.Connected() {
		return errNotConnected(ident)
	}
	return sock.Send(encodeWire(o.ownIdent, 0, typ, payload))
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
