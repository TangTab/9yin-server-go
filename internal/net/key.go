package net

import "encoding/binary"

// Key 是四字节循环 XOR 密钥（小端字节序，FxClient 表示形式）。
// 键值由配置或握手注入（对应原版 ini 的 MessageEncodeKey），不编译进代码。
type Key struct {
	bytes [4]byte
}

// NewKey 用小端 uint32 构造密钥。
func NewKey(value uint32) Key {
	var key Key
	binary.LittleEndian.PutUint32(key.bytes[:], value)
	return key
}

// InitialKeyValue 是九阴现代客户端首个登录 flight 上观测到的初始键。
const InitialKeyValue uint32 = 0xFA6D3BCC

// InitialKey 客户端首航默认键；登录协商后应轮换（SetReadKey/SetWriteKey）。
var InitialKey = NewKey(InitialKeyValue)

// Uint32 以客户端二进制的表示形式返回键值。
func (k Key) Uint32() uint32 { return binary.LittleEndian.Uint32(k.bytes[:]) }

// Apply 返回 src 的 XOR 变换副本（对称操作，编码解码同一函数）。
func (k Key) Apply(src []byte) []byte {
	dst := make([]byte, len(src))
	for i, b := range src {
		dst[i] = b ^ k.bytes[i&3]
	}
	return dst
}
