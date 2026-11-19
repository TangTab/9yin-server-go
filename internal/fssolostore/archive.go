package fssolostore

import (
	"encoding/binary"
	"unicode/utf16"
)

// encodeManageInfo member 端 decodeManageInfo（fssolomember/store.go，
// 官方 CLoadArchive 装载序 {String, WideStr, Int, WideStr, Int, Int}）的
// 逆编码——store 回执 0x1E payload。
func encodeManageInfo(e ManageEntry) []byte {
	var b []byte
	b = appendArchStr(b, e.Name)
	b = appendArchWide(b, e.Title)
	b = appendArchInt(b, e.Type)
	b = appendArchWide(b, e.Comment)
	b = appendArchInt(b, e.Order)
	b = appendArchInt(b, e.State)
	return b
}

func appendArchStr(b []byte, s string) []byte {
	raw := append([]byte(s), 0) // 官方串域含 NUL 终止
	b = appendU32(b, uint32(len(raw)))
	return append(b, raw...)
}

func appendArchWide(b []byte, s string) []byte {
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	b = appendU32(b, uint32(len(u)))
	for _, c := range u {
		b = appendU16(b, uint16(c))
	}
	return b
}

func appendArchInt(b []byte, v int32) []byte { return appendU32(b, uint32(v)) }

func appendU32(b []byte, v uint32) []byte {
	var t [4]byte
	binary.LittleEndian.PutUint32(t[:], v)
	return append(b, t[:]...)
}

func appendU16(b []byte, v uint16) []byte {
	var t [2]byte
	binary.LittleEndian.PutUint16(t[:], v)
	return append(b, t[:]...)
}

// buildChunkInfoFrame 回执 0x0F（member publishChunkInfo 读法）：
// {scene u32, n u32, chunk_id[n]}。
func buildChunkInfoFrame(scene uint32, ids []uint32) []byte {
	b := make([]byte, 8+len(ids)*4)
	putLe32(b, 0, scene)
	putLe32(b, 4, uint32(len(ids)))
	for i, id := range ids {
		putLe32(b, 8+i*4, id)
	}
	return b
}

// buildLoadChunkFrame 回执 0x10（member publishLoadChunk 读法 + 官方
// CSceneChunks 块装载 {id, len, bytes} 流）：
// {scene u32, n u32, 逐块 {id u32, len u32, bytes}}。缺失块以 len=0
// 占位（官方空块语义）。
func buildLoadChunkFrame(scene uint32, data map[uint32][]byte, order []uint32) []byte {
	sizes := make([]int, len(order))
	total := 8
	for i, id := range order {
		sizes[i] = len(data[id])
		total += 8 + sizes[i]
	}
	b := make([]byte, total)
	putLe32(b, 0, scene)
	putLe32(b, 4, uint32(len(order)))
	off := 8
	for i, id := range order {
		putLe32(b, off, id)
		putLe32(b, off+4, uint32(sizes[i]))
		copy(b[off+8:], data[id])
		off += 8 + sizes[i]
	}
	return b
}
