package fssoloentry

import (
	"encoding/binary"
	"fmt"
)

// solo ProcessInnerMsg 15 型（FsSoloEntry.dll 0x100A1FE0 跳表直读，
// soloentry.cpp 断言串逐型钉死）：
//
//	 0 init_connect      ==8   {flag,window}：flag≠0→开(事件1)/=0→停(2)；
//	                          window 越出 license 时间窗 → 身份域清零
//	 1 break_client      ==8   {ident,serial} 踢线
//	 2 reset_client_inuse ==8  {ident,serial} 槽位回收（连接已断后确认）
//	 3 set_client_online ==0x10 {ident,serial,member,scene}（第 4 域是
//	                          scene——ClientOnline 写 client+0x18）
//	 4 send_to_multi     >=4   {n}+n×{ident,serial}+payload；
//	                          断言 0<n<0x400；ident 前缀须为本组
//	 5 send_to_scene     >=4   {scene}+payload → 场景过滤扇出
//	 6 send_to_client    >=8   {ident,serial}+payload
//	 7 online_send       ——   payload（仅在线）
//	 8 broadcast         ——   payload（全量）
//	 9 shake_client      ==8   {ident,serial}：在线回 1B{0x06}；
//	                          失联回报 world type 2
//	10 channel_add       ==0xC {ident,serial,channel}
//	11 channel_remove    ==0xC 同上
//	12 clear_channel     ==4   {channel}
//	13 send_to_channel   >=4   {channel}+payload
//	14 close             ——   EntryStateMachine(3)
const (
	SoloForwInitConnect = 0
	SoloForwBreakClient = 1
	SoloForwResetInuse  = 2
	SoloForwSetOnline   = 3
	SoloForwSendMulti   = 4
	SoloForwSendScene   = 5
	SoloForwSendClient  = 6
	SoloForwOnlineSend  = 7
	SoloForwBroadcast   = 8
	SoloForwShakeClient = 9
	SoloForwChannelAdd  = 10
	SoloForwChannelRm   = 11
	SoloForwClearChan   = 12
	SoloForwSendChannel = 13
	SoloForwClose       = 14
)

// ProcessInnerMsg 复刻 SoloEntry::ProcessInnerMsg。返回错误仅用于日志
//（官方断言路径）。
func (s *Server) ProcessInnerMsg(src, typ uint32, p []byte) error {
	if !SoloSourceCheck(src) {
		// 官方 trace "(ProcessInnerMsg)message source error! comm:%d"
		return fmt.Errorf("(SoloEntry::ProcessInnerMsg)message source error! comm:%d", src)
	}
	switch typ {
	case SoloForwInitConnect: // ==8
		if len(p) != 8 {
			return fmt.Errorf("len == sizeof(entry_forw_init_connect_t)")
		}
		flag := le32(p, 0)
		if flag != 0 {
			s.EntryStateMachine(1)
		} else {
			s.EntryStateMachine(2)
		}
		s.checkWorldWindow(le32(p, 4))
		return nil

	case SoloForwBreakClient: // ==8
		if len(p) != 8 {
			return fmt.Errorf("len == sizeof(entry_forw_break_client_t)")
		}
		if s.down != nil {
			s.down.ClientClose(le32(p, 0), le32(p, 4))
		}
		return nil

	case SoloForwResetInuse: // ==8
		if len(p) != 8 {
			return fmt.Errorf("len == sizeof(entry_forw_reset_client_inuse_t)")
		}
		s.resetInuse(le32(p, 0), le32(p, 4))
		return nil

	case SoloForwSetOnline: // ==0x10
		if len(p) != 0x10 {
			return fmt.Errorf("len == sizeof(entry_forw_set_client_online_t)")
		}
		ident, serial := le32(p, 0), le32(p, 4)
		if err := s.clients.SetOnline(ident, serial, int32(le32(p, 8)), le32(p, 12)); err != nil {
			return err
		}
		s.mu.Lock()
		s.sceneOf[ident] = le32(p, 12)
		s.mu.Unlock()
		return nil

	case SoloForwSendMulti: // >=4
		if len(p) < 4 {
			return fmt.Errorf("len >= sizeof(entry_forw_send_to_multi_t)")
		}
		return s.multiSend(p)

	case SoloForwSendScene: // >=4
		if len(p) < 4 {
			return fmt.Errorf("len >= sizeof(entry_forw_send_to_scene_t)")
		}
		scene := le32(p, 0)
		s.mu.Lock()
		targets := make([][2]uint32, 0, 8)
		for _, idx := range s.clients.AllMembers() {
			st := s.clients.Client(idx)
			if s.sceneOf[st.Ident] == scene {
				targets = append(targets, [2]uint32{st.Ident, st.Serial})
			}
		}
		s.mu.Unlock()
		if s.down != nil {
			for _, t := range targets {
				_ = s.down.ClientSend(t[0], t[1], p[4:])
			}
		}
		return nil

	case SoloForwSendClient: // >=8
		if len(p) < 8 {
			return fmt.Errorf("len >= sizeof(entry_forw_send_to_client_t)")
		}
		if s.down != nil {
			return s.down.ClientSend(le32(p, 0), le32(p, 4), p[8:])
		}
		return nil

	case SoloForwOnlineSend:
		if s.down != nil {
			s.down.SendOnline(p)
		}
		return nil

	case SoloForwBroadcast:
		if s.down != nil {
			s.down.SendAll(p)
		}
		return nil

	case SoloForwShakeClient: // ==8
		if len(p) != 8 {
			return fmt.Errorf("len == sizeof(entry_forw_shake_client_t)")
		}
		if s.down != nil {
			s.down.Shake(le32(p, 0), le32(p, 4))
		}
		return nil

	case SoloForwChannelAdd: // ==0xC
		if len(p) != 0xC {
			return fmt.Errorf("len == sizeof(entry_forw_channel_add_t)")
		}
		return s.clients.ChannelAdd(le32(p, 0), le32(p, 4), le32(p, 8))

	case SoloForwChannelRm: // ==0xC
		if len(p) != 0xC {
			return fmt.Errorf("len == sizeof(entry_forw_channel_remove_t)")
		}
		return s.clients.ChannelRemove(le32(p, 0), le32(p, 4), le32(p, 8))

	case SoloForwClearChan: // ==4
		if len(p) != 4 {
			return fmt.Errorf("len == sizeof(entry_forw_clear_channel_t)")
		}
		s.clients.ClearChannel(le32(p, 0))
		return nil

	case SoloForwSendChannel: // >=4
		if len(p) < 4 {
			return fmt.Errorf("len >= sizeof(entry_forw_send_to_channel_t)")
		}
		if s.down != nil {
			s.down.SendChannel(le32(p, 0), p[4:])
		}
		return nil

	case SoloForwClose:
		s.EntryStateMachine(3)
		return nil

	default:
		return fmt.Errorf("(SoloEntry::ProcessInnerMsg)entry msg type error")
	}
}

// checkWorldWindow 官方世界窗联动（init_connect 第 2 域）：license 带
// 有效时间窗（T1>T0）且世界窗值越界 → 身份域 8×u32 清零（撤销授权，
// TimerCheck 随即停发广播并告警）。
func (s *Server) checkWorldWindow(window uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.windowT1 <= s.windowT0 {
		return
	}
	if window > s.windowT1 || window < s.windowT0 {
		s.fields = [8]uint32{}
	}
}

// multiSend 复刻 COuterHook::MultiSend：{u32 n}+n×{ident,serial}+
// payload；断言 0<n<0x400 与 len>头长；ident>>16 须为本组前缀。
func (s *Server) multiSend(p []byte) error {
	n := int(le32(p, 0))
	if n <= 0 || n >= 0x400 {
		return fmt.Errorf("(count > 0) && (count < 1024)")
	}
	if len(p) <= 4+n*8 {
		return fmt.Errorf("(COuterHook::MultiSend)message too short")
	}
	payload := p[4+n*8:]
	s.mu.Lock()
	prefix := uint32(s.opt.EntryID)
	s.mu.Unlock()
	for i := 0; i < n; i++ {
		ident := le32(p, 4+i*8)
		if (ident>>16)&0xFFFF != prefix {
			continue
		}
		if s.down != nil {
			_ = s.down.ClientSend(ident, le32(p, 8+i*8), payload)
		}
	}
	return nil
}

// resetInuse 复刻 COuterHook::ResetInuse：槽位回收（官方含连接残留
// 强断与 connect 计数递减，Go 版由 Gateway 关闭路径等价承载）。
func (s *Server) resetInuse(ident, serial uint32) {
	st := s.clients.Lookup(ident, serial)
	if st == nil {
		s.logf("(COuterHook::ResetInuse)client not exists")
		return
	}
	if s.down != nil {
		s.down.ClientClose(ident, serial)
	}
	s.mu.Lock()
	delete(s.sceneOf, ident)
	s.mu.Unlock()
}

func le32(p []byte, off int) uint32 { return binary.LittleEndian.Uint32(p[off:]) }
