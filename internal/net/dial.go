package net

import (
	"fmt"
	gonet "net"
	"time"
)

// DefaultDialTimeout 客户端拨号超时。
const DefaultDialTimeout = 5 * time.Second

// Dial 以客户端身份拨号并按选项装配 Connection（与服务端对称的
// 线路组装：EE+XOR、明文对拍基线等都由同一份 Options 表达）。
func Dial(addr string, opt Options) (*Connection, error) {
	stream, err := gonet.DialTimeout("tcp", addr, DefaultDialTimeout)
	if err != nil {
		return nil, fmt.Errorf("net dial %s: %w", addr, err)
	}
	return NewConnection(stream, opt)
}
