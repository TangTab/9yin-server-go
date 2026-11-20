// Package transport 实现游戏连接使用的字节流成帧。
// 它刻意不包含任何登录或游戏消息语义。
package transport

import "encoding/binary"

const InitialKeyValue uint32 = 0xFA6D3BCC

// Key 是不可变的四字节循环 XOR 密钥。将其字节保持私有
// 可防止调用者意外通过共享切片修改
// 活动连接的密钥。
type Key struct {
	bytes [4]byte
}

// NewKey 用 FxClient 使用的小端值构造密钥。
func NewKey(value uint32) Key {
	var key Key
	binary.LittleEndian.PutUint32(key.bytes[:], value)
	return key
}

// InitialKey 是在现代客户端首个登录 flight 上观测到的密钥。
var InitialKey = NewKey(InitialKeyValue)

// Uint32 以客户端二进制使用的表示形式返回密钥。
func (k Key) Uint32() uint32 {
	return binary.LittleEndian.Uint32(k.bytes[:])
}

// Apply 返回 src 变换后的副本。它绝不会别名引用或修改 src。
// XOR 是对称的，因此同一操作既可编码也可解码帧体。
func (k Key) Apply(src []byte) []byte {
	dst := make([]byte, len(src))
	for i, b := range src {
		dst[i] = b ^ k.bytes[i&3]
	}
	return dst
}
