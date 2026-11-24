// Package fscollide 是官方 FsCollide.dll（BGserverCJ 2014 快照，带 PDB/全符号 map）
// 的 Go 复刻。原 DLL 为服务端场景碰撞模块（fscollide 目录：blockfile/blockarea/
// blockchunk/blockheight2/blockwalk/collidemanager/scenecollide3d/scenecollideflat/
// scenemap/terrainheight/terrainregion/terrainwalkable/terrainzone/walk* 系列），
// 以 FxModule_* 八导出挂入 Fs 框架，实体名 CollideManager。
//
// 本文件：CBlockFile 块文件容器（blockfile.cpp）。
//
// 容器格式（全由反汇编实证，golden 见 _test）：
//
//	char signature[sigLen]   // "FxTer\n\n\n"（8 字节；sig 由构造方传入）
//	loop:
//	  u32 size               // 数据区长度，有符号 >=0（负→"block size error"）
//	  u32 id                 // 四字符块类型 ARE0/HGT2/WLK1/CHK0…
//	  byte data[size]        // 由 id 对应 reader 解析
//	  u32 crc                // 读走但官方 Load 不校验（writer 写入）
//
// 未知 id 的块被 fseek 跳过（"block unknown" 仅打印）。reader->Read 直接消费
// FILE*，因此 data 长度校验发生在各 reader 内部（GetSize 与文件 size 比对）。
package fscollide

import (
	"encoding/binary"
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// Signature 官方地形碰撞文件签名。
const Signature = "FxTer\n\n\n"

// Block 是容器内一个块的原始内容。
type Block struct {
	ID   uint32 // 四字符小端（'ARE0' == 0x30455241）
	Data []byte
}

// LoadBlockFile 读取整个 FxTer 容器，返回按文件顺序的全部块。
// size/id/crc 任一读失败即报错（与官方 Load 的告警路径一致，但官方在
// 块 id 未知时跳过继续；这里由调用方按 ID 过滤）。
func LoadBlockFile(path string) ([]Block, error) {
	raw, err := res.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < len(Signature) || string(raw[:len(Signature)]) != Signature {
		return nil, fmt.Errorf("fscollide: %s: signature not match", path)
	}
	r := raw[len(Signature):]
	var blocks []Block
	for {
		if len(r) < 4 {
			break // EOF（正常出口：第一读 fread!=1 静默结束）
		}
		size := binary.LittleEndian.Uint32(r)
		if int32(size) < 0 {
			return nil, fmt.Errorf("fscollide: %s: block size error", path)
		}
		r = r[4:]
		if len(r) < 4 {
			return nil, fmt.Errorf("fscollide: %s: read block id failed", path)
		}
		id := binary.LittleEndian.Uint32(r)
		r = r[4:]
		if uint64(len(r)) < uint64(size)+4 {
			return nil, fmt.Errorf("fscollide: %s: read block size not match", path)
		}
		data := r[:size]
		r = r[size:]
		_ = binary.LittleEndian.Uint32(r) // crc：官方读走不校验
		r = r[4:]
		blocks = append(blocks, Block{ID: id, Data: data})
		if len(r) == 0 {
			break
		}
	}
	return blocks, nil
}

// 块类型四字符（小端读出的 u32）。
var (
	blockIDArea    = fourCC("ARE0") // 区域位图（zone_x_y.area）
	blockIDHeight2 = fourCC("HGT2") // 地形高度四叉树（zone_x_y.ground）
	blockIDWalk    = fourCC("WLK1") // 可行走/多层地板（walk/zone_x_y.walk）
	blockIDChunk   = fourCC("CHK0") // 编辑器 chunk 记录（服务端跳过）
)

func fourCC(s string) uint32 {
	return binary.LittleEndian.Uint32([]byte(s))
}
