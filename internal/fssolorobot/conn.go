package fssolorobot

import (
	"net"
	"sync"
)

// ConnState 是官方连接级状态常量（GetConnectState 跳转表
// 0x1005ca4c 定案；connect_t+0x04）。
type ConnState int

// 官方八态。数值与名字串（0x100c9b34..0x100c9b7c）一一对应。
const (
	ConnUnknown ConnState = iota
	ConnConnected
	ConnLogging
	ConnCreating
	ConnChoosing
	ConnInGame
	ConnFailed
	ConnClosed
)

// String 官方 GetConnectState 的呈现名。
func (s ConnState) String() string {
	switch s {
	case ConnConnected:
		return "connected"
	case ConnLogging:
		return "logging"
	case ConnCreating:
		return "creating"
	case ConnChoosing:
		return "choosing"
	case ConnInGame:
		return "ingame"
	case ConnFailed:
		return "failed"
	case ConnClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// recvBufSize 官方 connect_t+0x24 收包缓冲 0x1000。
const recvBufSize = 0x1000

// connect 复刻官方 connect_t（分配 0x1030；字段见
// docs/official/FsSoloRobot全解与Go复刻.md §三）。
type connect struct {
	index       int      // +0x00 机器人序号（账号 = AccountPrefix+index）
	state       ConnState // +0x04
	waitSeconds int      // +0x08 每秒累加；ingame 时为移动倒计时
	ident1      uint32   // +0x10 PERSISTID.ident1（entry_scene 写入）
	ident2      uint32   // +0x14 PERSISTID.ident2
	x, y, z     float32  // +0x18/+0x1c/+0x20 当前坐标

	// Go 侧设施（官方经 CReactor 槽位与 IOCP 承载）。
	nc      net.Conn
	peer    string
	writeMu sync.Mutex

	// 官方 OnReceive 解转义状态机（+0x24 缓冲 / +0x1024 长度 /
	// +0x1028 上一字节）。
	recv     [recvBufSize]byte
	recvLen  int
	lastByte byte
}

// feed 官方 SoloRobot::OnReceive@0x1005ecf0 逐字节状态机：
//
//	b==EE 且 last==EE → 帧完成：去掉终止符首 EE 后分发（len-1>0 才分发），
//	  复位 len/last；
//	b==00 且 last==EE → 丢弃（EE 00 → 字面 EE）；
//	其余缓冲（len < 0x1000 才入），last=b。
func (c *connect) feed(data []byte, onFrame func(payload []byte)) {
	for _, b := range data {
		if b == 0xEE && c.lastByte == 0xEE {
			if n := c.recvLen - 1; n > 0 {
				payload := make([]byte, n)
				copy(payload, c.recv[:n])
				onFrame(payload)
			}
			c.recvLen = 0
			c.lastByte = 0
			continue
		}
		skip := b == 0x00 && c.lastByte == 0xEE
		if !skip {
			if c.recvLen < recvBufSize {
				c.recv[c.recvLen] = b
				c.recvLen++
			}
		}
		c.lastByte = b
	}
}

// writeFrame 官方 SendToServer 尾段：转义 + EE EE 终止后整写。
func (c *connect) writeFrame(payload []byte) error {
	wire := escapeFrame(payload)
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err := c.nc.Write(wire)
	return err
}
