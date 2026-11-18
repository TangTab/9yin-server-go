package net

import "sync"

// Cipher 是可插拔编码层（对应原版 MessageVerifyDll 可插拔语义）：
// 帧格式（Framer）与编码（Cipher）正交组合，可按配置/握手协商替换。
type Cipher interface {
	Encode(plain []byte) []byte
	Decode(wire []byte) []byte
}

// keySetter 由支持键注入/轮换的 Cipher 实现（如 XorCipher）。
// Connection.SetReadKey/SetWriteKey 通过该接口下发音阶。
type keySetter interface{ SetKey(Key) }

// XorCipher 四字节循环 XOR 编码，内置读写锁支持运行中轮换键
// （登录 flight 协商出新键后热切换，无需重建连接）。
type XorCipher struct {
	mu  sync.RWMutex
	key Key
}

// NewXorCipher 以指定键创建 XOR 编码层。
func NewXorCipher(key Key) *XorCipher { return &XorCipher{key: key} }

// Encode 变换明文（返回副本）。
func (c *XorCipher) Encode(plain []byte) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.key.Apply(plain)
}

// Decode 与 Encode 对称。
func (c *XorCipher) Decode(wire []byte) []byte { return c.Encode(wire) }

// SetKey 轮换键。
func (c *XorCipher) SetKey(key Key) {
	c.mu.Lock()
	c.key = key
	c.mu.Unlock()
}

// IdentityCipher 明文直通（对应原版 CJ 分支 MessageEncodeKey 注释掉的
// 默认明文运行形态），保留在接口体系内以便配置切换与对拍。
type IdentityCipher struct{}

// Encode 原样返回（不复制，调用方不得修改返回的切片别名）。
func (IdentityCipher) Encode(plain []byte) []byte { return plain }

// Decode 原样返回。
func (IdentityCipher) Decode(wire []byte) []byte { return wire }
