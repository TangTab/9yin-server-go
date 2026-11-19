package fssolomember

import (
	"encoding/binary"
)

// ManageInfo 官方 CManageInfo 镜像（@this+0x440；PublishManageInfo 以
// CLoadArchive 序 {String, WideStr, Int, WideStr, Int, Int} 装载——
// store_back_manage_info_t 档案）。
type ManageInfo struct {
	Name    string
	Title   string // WideStr
	Flag    int32
	Comment string // WideStr
	Order   int32
	State   int32
}

// decodeManageInfo PublishManageInfo 的 archive 装载序（元素序实证
// @0x1029FB4A-0x1029FC12；容器为官方 CLoadArchive 变长流，此处按
// 元素序 + 宽串定长方言解码，未知形态返回 nil 清闩）。
func decodeManageInfo(p []byte) *ManageInfo {
	off := 0
	readStr := func() (string, bool) {
		if off+4 > len(p) {
			return "", false
		}
		n := int(binary.LittleEndian.Uint32(p[off:]))
		off += 4
		if off+n > len(p) || n < 0 {
			return "", false
		}
		s := cstringAt(p[off:off+n], 0)
		off += n
		return s, true
	}
	readWide := func() (string, bool) {
		if off+4 > len(p) {
			return "", false
		}
		n := int(binary.LittleEndian.Uint32(p[off:]))
		off += 4
		if off+n*2 > len(p) || n < 0 {
			return "", false
		}
		s := utf16ZAt(p[off:off+n*2], 0, n)
		off += n * 2
		return s, true
	}
	readInt := func() (int32, bool) {
		if off+4 > len(p) {
			return 0, false
		}
		v := int32(binary.LittleEndian.Uint32(p[off:]))
		off += 4
		return v, true
	}
	m := &ManageInfo{}
	var ok bool
	if m.Name, ok = readStr(); !ok {
		return nil
	}
	if m.Title, ok = readWide(); !ok {
		return nil
	}
	if m.Flag, ok = readInt(); !ok {
		return nil
	}
	if m.Comment, ok = readWide(); !ok {
		return nil
	}
	if m.Order, ok = readInt(); !ok {
		return nil
	}
	if m.State, ok = readInt(); !ok {
		return nil
	}
	return m
}

// processStoreMsg 官方 ProcessStoreMsg（@0x102A0200；字节跳表 @0x102A0398
// 定案：0xF..0x12 chunk 族 + 0x1E..0x21 manage 族，0x13..0x1D 为 default 空槽）。
func (s *Server) processStoreMsg(id1, id2 uint32, payload []byte) {
	switch id2 {
	case 0xF: // PublishChunkInfo：存档块清单（"no scene chunks/scene id error"）
		s.publishChunkInfo(payload)
	case 0x10: // PublishLoadChunk：块加载（"chunks amount error"）
		s.publishLoadChunk(payload)
	case 0x11: // PublishSaveChunk：块保存应答（"save failed"）
		if len(payload) >= 4 {
			s.mu.Lock()
			s.chunks[le32(payload, 0)] = true
			s.mu.Unlock()
		}
	case 0x12: // PublishDeleteChunk：块删除
		if len(payload) >= 4 {
			s.mu.Lock()
			delete(s.chunks, le32(payload, 0))
			s.mu.Unlock()
		}
	case 0x1E: // PublishManageInfo：管理配置全量 → +0x444 闩（Loading 门）
		if decodeManageInfo(payload) == nil {
			s.opt.Logf("(SoloMember::PublishManageInfo)failed")
			s.mu.Lock()
			s.manageData = nil
			s.mu.Unlock()
			return
		}
		s.mu.Lock()
		s.manageData = decodeManageInfo(payload)
		s.mu.Unlock()
	case 0x1F, 0x20, 0x21: // PublishManageBase/Update/Remove
		s.opt.Logf("publish manage 0x%X (%d bytes)", id2, len(payload))
	}
}

// publishChunkInfo 存档块清单：{scene u32, n u32, chunk_id[n]}。
func (s *Server) publishChunkInfo(p []byte) {
	if len(p) < 8 {
		s.opt.Logf("(SoloMember::PublishChunkInfo)scene id error")
		return
	}
	n := int(le32(p, 4))
	if len(p) < 8+n*4 {
		s.opt.Logf("(SoloMember::PublishChunkInfo)no scene chunks")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < n; i++ {
		s.chunks[le32(p, 8+i*4)] = false
	}
}

// publishLoadChunk 块加载：{scene, n, {chunk_id, len, bytes}…}——
// 对象内核域（CSceneChunks 装载），此处记账并透传场景宿主。
func (s *Server) publishLoadChunk(p []byte) {
	if len(p) < 8 {
		s.opt.Logf("(SoloMember::PublishLoadChunk)scene id error")
		return
	}
	n := int(le32(p, 4))
	if n == 0 {
		s.opt.Logf("(SoloMember::PublishLoadChunk)no scene chunks")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < n; i++ {
		if 8+i*4+4 <= len(p) {
			s.chunks[le32(p, 8+i*4)] = true
		}
	}
}

// ChunkCount 存档块表规模（测试/呈现）。
func (s *Server) ChunkCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.chunks)
}
