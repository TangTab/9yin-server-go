package net

import (
	"fmt"
	"io"
	"sync"
)

// Options 连接装配参数。Framer 与两侧 Cipher 均可替换——这是对原版
// 可插拔编码层的落地：EE+XOR（现代客户端）、明文+EE（对拍基线）、
// Prefix+Identity（进程间总线）都是同一 Connection 的不同装配。
type Options struct {
	MaxFrameSize int            // 默认 DefaultMaxFrameSize
	Framer       FramerFactory  // 默认 EscapeFramerFactory
	ReadCipher   Cipher         // 默认 XOR(InitialKey)
	WriteCipher  Cipher         // 默认 XOR(InitialKey)
}

// Connection 组合 Framer + 方向性 Cipher 的有状态连接。
// 读写方向键独立（登录协商后可能各自轮换到不同键）。
type Connection struct {
	stream  io.ReadWriteCloser
	framer  Framer
	closing sync.Once

	readMu     sync.Mutex
	readCipher Cipher

	writeMu    sync.Mutex
	writeCipher Cipher
}

// NewConnection 按选项装配连接。stream 非空即可，其余参数走默认。
func NewConnection(stream io.ReadWriteCloser, opt Options) (*Connection, error) {
	if stream == nil {
		return nil, fmt.Errorf("nil connection stream")
	}
	if opt.MaxFrameSize <= 0 {
		opt.MaxFrameSize = DefaultMaxFrameSize
	}
	if opt.Framer == nil {
		opt.Framer = EscapeFramerFactory
	}
	if opt.ReadCipher == nil {
		opt.ReadCipher = NewXorCipher(InitialKey)
	}
	if opt.WriteCipher == nil {
		opt.WriteCipher = NewXorCipher(InitialKey)
	}
	framer, err := opt.Framer(stream, opt.MaxFrameSize)
	if err != nil {
		return nil, err
	}
	return &Connection{
		stream:      stream,
		framer:      framer,
		readCipher:  opt.ReadCipher,
		writeCipher: opt.WriteCipher,
	}, nil
}

// ReadMsg 读取一帧并解码为明文。
func (c *Connection) ReadMsg() ([]byte, error) {
	c.readMu.Lock()
	wire, err := c.framer.ReadFrame()
	if err != nil {
		c.readMu.Unlock()
		return nil, err
	}
	plain := c.readCipher.Decode(wire)
	c.readMu.Unlock()
	return plain, nil
}

// WriteMsg 编码明文并写出一帧。
func (c *Connection) WriteMsg(plain []byte) error {
	c.writeMu.Lock()
	wire := c.writeCipher.Encode(plain)
	err := c.framer.WriteFrame(wire)
	c.writeMu.Unlock()
	return err
}

// SetReadKey 轮换读方向键（仅对支持键的 Cipher 生效，如 XorCipher）。
func (c *Connection) SetReadKey(key Key) {
	c.readMu.Lock()
	if ks, ok := c.readCipher.(keySetter); ok {
		ks.SetKey(key)
	}
	c.readMu.Unlock()
}

// SetWriteKey 轮换写方向键。
func (c *Connection) SetWriteKey(key Key) {
	c.writeMu.Lock()
	if ks, ok := c.writeCipher.(keySetter); ok {
		ks.SetKey(key)
	}
	c.writeMu.Unlock()
}

// Close 关闭底层流（幂等由底层保证）。
func (c *Connection) Close() error {
	var err error
	c.closing.Do(func() { err = c.stream.Close() })
	return err
}
