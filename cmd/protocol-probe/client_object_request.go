package main

import "encoding/binary"

// clientObjectRequest 是已观察到的固定 C2S 0x07 参考包。
// 客户端在点击场景内的对象时发送此包。其所属的请求类型
// 在现代客户端中尚未得到命名，因此这里只暴露已被证实的身份
// 字段，而不是伪装成对话包。
type clientObjectRequest struct {
	Sequence uint32
	ObjectID uint32
	OwnerID  uint32
	FuncID   int32
}

func parseClientObjectRequest(message []byte) (clientObjectRequest, bool) {
	// 抓包恒为 33 字节。身份字段起始于偏移 21 和 25；
	// 接受更长的兼容包，但绝不解析不完整的包。
	if len(message) < 29 || message[0] != 0x07 {
		return clientObjectRequest{}, false
	}
	request := clientObjectRequest{
		Sequence: binary.LittleEndian.Uint32(message[17:]),
		ObjectID: binary.LittleEndian.Uint32(message[21:]),
		OwnerID:  binary.LittleEndian.Uint32(message[25:]),
		FuncID:   int32(binary.LittleEndian.Uint32(message[29:])),
	}
	if request.ObjectID == 0 || request.OwnerID == 0 {
		return clientObjectRequest{}, false
	}
	return request, true
}
