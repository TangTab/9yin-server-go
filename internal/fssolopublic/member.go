package fssolopublic

import (
	"encoding/binary"

	"github.com/lizhanyin/9yin-server-go/internal/fspublic"
)

// processMemberMsg 官方 ProcessMemberMsg（cmp 链：0/1/2）。
func (s *Server) processMemberMsg(index int, sub uint32, p []byte) {
	switch sub {
	case 0:
		s.memberRegister(index, p)
	case 1:
		s.memberData(index, p)
	case 2:
		s.memberMessage(index, p)
	default:
		s.logf("(SoloPublic::ProcessMemberMsg)unknown sub %d", sub)
	}
}

// memberRegister 官方 type0：len==8 断言；State==5 才应答（官方门
// [this+0x34]==5）；回包（定向 type0）= archive 空间清单。
func (s *Server) memberRegister(index int, p []byte) {
	if len(p) != 8 {
		s.logf("(SoloPublic::MemberRegister)len == sizeof(public_forw_register_t)=8，得 %d", len(p))
		return
	}
	if s.State() != StateOpened {
		return // 官方静默丢弃
	}
	body, err := encodeSpaceList(s.PubSpaces())
	if err != nil {
		s.logf("(SoloPublic::MemberRegister)send spaces failed: %v", err)
		return
	}
	if !s.inner.SendToMember(index, 0, 0, body) {
		s.logf("(SoloPublic::MemberRegister)send spaces failed")
	}
}

// memberData 官方 type1：len==0x74（solo 加宽；集群 0x29B）。回包
// （定向 type1）= 0x90 头 {u32 count@0, u32 index@4, char space[0x21]@8,
// wchar name[0x21]@0x29, char key[0x21]@0x6B, u32 1@0x8C} + PUB4
// blob(flag=0)。请求头：{u32 ?@0, u32 index@4, char space[0x21]@8,
// wchar prevName[0x21]@0x29, ...}。
func (s *Server) memberData(index int, p []byte) {
	if len(p) != sizeMemberDataReq {
		s.logf("(SoloPublic::MemberData)len == sizeof(public_forw_data_t)=0x74，得 %d", len(p))
		return
	}
	space := cstr(p[8 : 8+0x21])
	sp := s.FindPubSpace(space)
	if sp == nil {
		s.logf("(SoloPublic::MemberData)space not exists: %s", space)
		return
	}
	count := sp.Count()
	if int(u32at(p, 4)) != count {
		s.logf("(SoloPublic::MemberData)data amount changed")
		return
	}
	prev := fixedWide(p[0x29 : 0x29+0x42])
	next := -1
	if prev == "" {
		if count > 0 {
			next = 0
		}
	} else {
		idx := sp.Find(prev)
		if idx < 0 {
			s.logf("(SoloPublic::MemberData)prev name error: %s", prev)
			return
		}
		next = idx + 1
	}
	if next < 0 || next >= count {
		return // 本空间已到尾（member 侧换下一空间）
	}
	d := sp.All()[next]
	blob, err := d.StoreArchiveBlob(false)
	if err != nil {
		s.logf("(SoloPublic::MemberData)store data failed: %v", err)
		return
	}
	out := make([]byte, 0x90, 0x90+len(blob))
	binary.LittleEndian.PutUint32(out[0:], uint32(count))
	binary.LittleEndian.PutUint32(out[4:], uint32(next))
	putFixedStr(out[8:8+0x21], space)
	putFixedWide(out[0x29:0x29+0x42], d.Name)
	putFixedStr(out[0x6B:0x6B+0x21], d.Key)
	binary.LittleEndian.PutUint32(out[0x8C:], 1)
	out = append(out, blob...)
	if !s.inner.SendToMember(index, 0, 1, out) {
		s.logf("(SoloPublic::MemberData)send data failed")
	}
}

// memberMessage 官方 type2：{int msg_id@0, int argc@4} + get_vary_message
// → Callee->OnPublicMessage（非分派器，纯转发）。
func (s *Server) memberMessage(index int, p []byte) {
	if len(p) < 8 {
		s.logf("(SoloPublic::MemberMessage)len >= sizeof(public_forw_message_t)=8，得 %d", len(p))
		return
	}
	msgID := u32at(p, 0)
	argc := int(u32at(p, 4))
	arc := fspublic.NewLoadArchive(p[8:])
	args, err := fspublic.GetVaryMessage(arc, argc)
	if err != nil {
		s.logf("(SoloPublic::MemberMessage)load message failed: %v", err)
		return
	}
	s.callee.OnPublicMessage(nil, uint32(CommMemberMin+index), msgID, args)
}

// ---- public→member 推送（SyncPublic 族；与集群 FsPublic §6 同构）----

// 推送 op 常量（member 镜像侧 14 型对应）。
const (
	pushSpaceList  = 0x0 // Register 应答 / LoadSpaces 广播（archive 清单）
	pushDataSnap   = 0x1 // MemberData 应答（0x90 头 + blob）
	pushAddData    = 0x2
	pushRemoveData = 0x3
	pushAttr       = 0x4
	pushRemoveAttr = 0x5
	pushAddRecord  = 0x6
	pushRemoveRec  = 0x7
	pushAddRow     = 0x8
	pushRemoveRow  = 0x9
	pushClearRows  = 0xA
	pushSetGrid    = 0xB
	pushSetRow     = 0xC
	pushMessage    = 0xD // 定向 SendPublicMessage
)

// pushHeader 公共推送头（0x84B）：{char space[0x21]; wchar name[0x21]@0x21;
// char key[0x21]@0x63}。
func pushHeader(space, wname, key string) []byte {
	out := make([]byte, 0x84)
	putFixedStr(out[0:0x21], space)
	putFixedWide(out[0x21:0x63], wname)
	putFixedStr(out[0x63:0x84], key)
	return out
}

// SyncPublic 官方 SyncPublic(op, data, len)：门 State==5 →
// SendToAllMember(0, 0, op, …) 广播全体 member（含请求者）。
func (s *Server) SyncPublic(op uint32, data []byte) bool {
	if s.State() != StateOpened {
		return false
	}
	return s.inner.SendToAllMember(0, op, data)
}

// SendPublicMessage 官方 SoloPublic::SendPublicMessage(target, msgid,
// varlist)：定向 type0xD {int msg_id, int argc} + put_vary_message。
func (s *Server) SendPublicMessage(memberIndex int, msgID uint32, args []fspublic.Value) bool {
	arc := fspublic.NewStoreArchive(16 + len(args)*12)
	if err := arc.WriteInt(int32(msgID)); err != nil {
		return false
	}
	if err := arc.WriteInt(int32(len(args))); err != nil {
		return false
	}
	if err := fspublic.PutVaryMessage(arc, args); err != nil {
		return false
	}
	return s.inner.SendToMember(memberIndex, 0, pushMessage, arc.Bytes())
}

// syncAddData 0x2：CPubSpace::AddPubData 后的广播（头 0x84 + blob flag=0）。
func (s *Server) syncAddData(sp *fspublic.PubSpace, d *fspublic.PubData) {
	blob, err := d.StoreArchiveBlob(false)
	if err != nil {
		return
	}
	body := append(pushHeader(sp.Name, d.Name, d.Key), blob...)
	s.SyncPublic(pushAddData, body)
}

// syncRemoveData 0x3：头截 0x63（space+name，无 key）。
func (s *Server) syncRemoveData(space, wname string) {
	out := make([]byte, 0x63)
	putFixedStr(out[0:0x21], space)
	putFixedWide(out[0x21:0x63], wname)
	s.SyncPublic(pushRemoveData, out)
}
