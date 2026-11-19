package fssolomember

import (
	"strings"
)

// processPublicMsg 官方 ProcessPublicMsg（@0x1029CE90，id2 0..0xE 顺序；§六）。
// 载荷方言与集群 14 型同构（CPubSpace 镜像），公共镜像复用 fsmember.PubSpaces。
func (s *Server) processPublicMsg(id1, id2 uint32, payload []byte) {
	switch id2 {
	case 0x0: // PublicSpaces：空间声明清单 → +0x434 闩（@0x10299DD7 写点）
		names := splitZList(payload)
		if err := s.Pub.DeclareSpaces(names); err != nil {
			s.opt.Logf("(SoloMember::PublicSpaces)%v", err)
		}
		s.setPubReady()
	case 0x1: // PublicData：空间→数据对象建立 → 同闩（@0x1029A235 写点）
		space, data := splitZ2(payload)
		if err := s.Pub.EnsureData(space, data); err != nil {
			s.opt.Logf("(SoloMember::PublicData)%v", err)
		}
		s.setPubReady()
	case 0x2: // PublicAdd
		space, data := splitZ2(payload)
		if err := s.Pub.AddData(space, data); err != nil {
			s.opt.Logf("(SoloMember::PublicAdd)%v", err)
		}
	case 0x3: // PublicRemove
		space, data := splitZ2(payload)
		if err := s.Pub.RemoveData(space, data); err != nil {
			s.opt.Logf("(SoloMember::PublicRemove)%v", err)
		}
	case 0x4, 0x5: // PublicAddAttr/RemoveAttr
		parts := splitZ3(payload)
		var err error
		if id2 == 0x4 {
			err = s.Pub.SetAttr(parts[0], parts[1], parts[2], parts[3])
		} else {
			err = s.Pub.DelAttr(parts[0], parts[1], parts[2])
		}
		if err != nil {
			s.opt.Logf("(SoloMember::PublicAttr)%v", err)
		}
	case 0x6, 0x7: // PublicAddRec/RemoveRec
		parts := splitZAll(payload)
		get := func(i int) string {
			if i < len(parts) {
				return parts[i]
			}
			return ""
		}
		if id2 == 0x6 {
			var cols []string
			if len(parts) > 3 {
				cols = parts[3:]
			}
			if err := s.Pub.AddRec(get(0), get(1), get(2), cols); err != nil {
				s.opt.Logf("(SoloMember::PublicAddRec)%v", err)
			}
		} else if err := s.Pub.DelRec(get(0), get(1), get(2)); err != nil {
			s.opt.Logf("(SoloMember::PublicRemoveRec)%v", err)
		}
	case 0x8, 0x9, 0xA: // PublicRecAddRow/RemoveRow/Clear
		s.opt.Logf("public rec row message 0x%X (%d bytes)", id2, len(payload))
	case 0xB, 0xC: // PublicRecChange/RecSetRow
		s.opt.Logf("public rec change message 0x%X (%d bytes)", id2, len(payload))
	case 0xD: // PublicMessage：透传场景消息
		if len(payload) < 4 {
			return
		}
		if !s.sendToScene(le32(payload, 0), 0xD, payload[4:]) {
			s.opt.Logf("(SoloMember::PublicMessage)scene not active")
		}
	case 0xE: // PublicCommand（solo 独有）：透传场景命令
		if len(payload) < 4 {
			return
		}
		s.sendToScene(le32(payload, 0), 0xE, payload[4:])
	}
}

func (s *Server) setPubReady() {
	s.mu.Lock()
	s.pubReady = true
	s.mu.Unlock()
}

// splitZList 载荷 = NUL 分隔字符串清单（空间声明）。
func splitZList(p []byte) []string {
	var out []string
	for _, part := range strings.Split(string(p), "\x00") {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// splitZ2/parts 辅助：按 NUL 切前 2/3 段（不足补空串）。
func splitZ2(p []byte) (string, string) {
	parts := strings.SplitN(string(p), "\x00", 3)
	for len(parts) < 2 {
		parts = append(parts, "")
	}
	return parts[0], parts[1]
}

func splitZ3(p []byte) []string {
	parts := strings.SplitN(string(p), "\x00", 4)
	for len(parts) < 3 {
		parts = append(parts, "")
	}
	return parts[:3]
}

func splitZAll(p []byte) []string { return strings.Split(string(p), "\x00") }
