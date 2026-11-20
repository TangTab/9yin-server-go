package fsgame

// ChatModule 复刻官方 interactivemodule/chatmodule
// （Init @0x8514F0，玩法表 Init 序 184；符号 436）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnRecover",@0x8519C0)  chatpage_rec 初始化
//	                    （ui_zonghe/ui_zhandou/ui_siliao/ui_guild/ui_fujin 五默认页）
//	AddEventCallback ("player","OnReady",@0x857870)    OnPlayerReady
//	AddEventCallback ("player","OnContinue",@0x857760)
//	AddEventCallback ("player","OnStore",@0x858440)    OnPlayOut（离场摘除 chat 通道）
//	AddIntCustomHook ("player",0x01,@0x852190)         OnCustomChat 聊天页命令多路
//	AddIntCommandHook ("player",914=0x392,@0x853350)   OnChatWhisperDissScene 跨场景私聊
//	AddIntCommandHook ("player",938=0x3AA,@0x8537A0)   OnChatWhisperDissSceneResult
//	AddCommandHook   ("player","chat",…)               聊天发送字符串命令
//	DeclareHeartBeat ("ChatModule::H_Silence",@0x852D00) 禁言倒计时
//
// OnCustomChat（0x852190）子命令分发表（跳表 @0x1086B810 同型）：
//
//	100 OnNewPage / 101 OnPagePos / 102 OnDeletePage / 99 OnChangePageName /
//	98 OnPageComposeState / 95 OnPageSize / 96 OnPageTransparentValue /
//	97 OnPageFixedState / 106 OnPageChannelChange / 121 CanChat / 22 CleanWords
//
// chatpage_rec（struct/player/chatpage.xml，22 列 ×10 行）：
// 0 索引/1 名称/2 战斗页/3 组合/4 锁定/5 透明度/6 x/7 y/8 宽/9 高/
// 10..21 频道开关（附近/密聊/喊话/区域/世界/队伍/团队/工会/战场/招募/系统/战斗）。
// 系统信息模板：sys_item_0083（禁言）、sys_item_0055（频繁）、BG_131001（解禁）。
type ChatModule struct {
	BaseModule

	// ChatOut 聊天下发出口（官方经 Custom 通道按频道广播；sync 层挂接）。
	ChatOut func(channel int32, from, content string)
	// WhisperOut 跨场景私聊出口（官方 914 经 public 服转发到目标 member）。
	WhisperOut func(from, to, content string)

	// initPages 已初始化 chatpage 的玩家（OnRecover 幂等门）。
	initPages map[PersistID]bool
	// sysMsgs 系统信息观测（官方走 BG_systeminfo 模板下发）。
	sysMsgs []string
	// sentChats 喊话流水（CanChat 放行记录；测试面）。
	sentChats []chatSend
}

type chatSend struct {
	Channel int32
	From    string
	Content string
}

// 聊天频道号（chatpage_rec 10..21 列序）。
const (
	ChannelNearby int32 = iota // 附近
	ChannelWhisper             // 密聊
	ChannelShout               // 喊话
	ChannelZone                // 区域
	ChannelWorld               // 世界
	ChannelTeam                // 队伍
	ChannelRaid                // 团队
	ChannelGuild               // 工会
	ChannelBattle              // 战场
	ChannelRecruit             // 招募
	ChannelSystem              // 系统
	ChannelFight               // 战斗
)

// chatDefaultPages 官方 OnRecover @0x8519C0 的五默认页（页名 → 战斗页标志）。
var chatDefaultPages = []struct {
	Name     string
	IsBattle int32
	Channels []int32
}{
	{"ui_zonghe", 0, []int32{ChannelNearby, ChannelShout, ChannelZone, ChannelWorld, ChannelTeam, ChannelRaid, ChannelGuild, ChannelBattle, ChannelRecruit, ChannelSystem}},
	{"ui_zhandou", 1, []int32{ChannelFight}},
	{"ui_siliao", 0, []int32{ChannelWhisper}},
	{"ui_guild", 0, []int32{ChannelGuild}},
	{"ui_fujin", 0, []int32{ChannelNearby}},
}

// Name 模块名。
func (m *ChatModule) Name() string { return "ChatModule" }

// Init 注册期（对齐官方 0x8514F0）。
func (m *ChatModule) Init(k *Kernel) error {
	m.initPages = make(map[PersistID]bool)
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnReady", m.onPlayerReady)
	b.OnEvent("player", "OnContinue", func(k *Kernel, self, from PersistID, args VarList) {})
	b.OnEvent("player", "OnStore", m.onPlayOut)
	b.OnCommand("player", 0x01, m.onCustomChat)
	b.OnIntCommand("player", 914, m.onWhisperDissScene)
	b.OnIntCommand("player", 938, m.onWhisperDissSceneResult)
	b.OnStringCommand("player", "chat", m.cmdChat)
	b.HeartBeat("ChatModule::H_Silence", m.hbSilence)
	return nil
}

// onRecover 官方 @0x8519C0：chatpage_rec 初始化（SilenceTick 清零 + 五默认页）。
func (m *ChatModule) onRecover(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil || m.initPages[self] {
		return
	}
	m.initPages[self] = true
	if r := o.Record("chatpage_rec"); r != nil {
		for i, p := range chatDefaultPages {
			row := make(VarList, 22)
			row[0] = int32(i)               // 表单索引
			row[1] = p.Name                 // 表单名称
			row[2] = p.IsBattle             // 是否战斗表单
			row[3] = int32(1)               // 组合状态
			row[4] = int32(0)               // 解锁
			row[5] = int32(100)             // 透明度
			row[6] = float32(0)             // x
			row[7] = float32(0)             // y
			row[8] = int32(300)             // 宽
			row[9] = int32(200)             // 高
			for _, ch := range p.Channels { // 频道开关
				row[10+int(ch)] = int32(1)
			}
			r.AppendRow(row)
		}
	}
	k.Tracef("ChatModule: OnRecover player=%v chatpage×%d", self.ID(), len(chatDefaultPages))
}

// onPlayerReady 官方 @0x857870：OnPlayerReady（登记 chat 通道身份 Name/Camp/Race）。
func (m *ChatModule) onPlayerReady(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("ChatModule: OnPlayerReady player=%v", self.ID())
}

// onPlayOut 官方 @0x858440：OnPlayOut（离场：Name/ConfigID/Camp/Race 摘除 chat 通道）。
func (m *ChatModule) onPlayOut(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("ChatModule: OnPlayOut player=%v", self.ID())
}

// onCustomChat 官方 @0x852190（命令 1）：聊天页命令多路（args[0] 子命令）。
func (m *ChatModule) onCustomChat(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	sub := args.Int(0)
	r := o.Record("chatpage_rec")
	switch sub {
	case 100: // OnNewPage：新页（客户端名/索引）
		if r != nil {
			row := make(VarList, 22)
			row[0] = args.Int(1)
			row[1] = args.String(2)
			r.AppendRow(row)
		}
	case 101: // OnPagePos：位置（索引,x,y）
		if r != nil {
			if i := m.pageRow(r, args.Int(1)); i >= 0 {
				row := r.Row(i)
				row[6] = args.Float(2)
				row[7] = args.Float(3)
				r.ModifyRow(i, row)
			}
		}
	case 102: // OnDeletePage
		if r != nil {
			if i := m.pageRow(r, args.Int(1)); i >= 0 {
				r.DeleteRow(i)
			}
		}
	case 99: // OnChangePageName
		if r != nil {
			if i := m.pageRow(r, args.Int(1)); i >= 0 {
				row := r.Row(i)
				row[1] = args.String(2)
				r.ModifyRow(i, row)
			}
		}
	case 98: // OnPageComposeState（1 组合 0 分解）
		if r != nil {
			if i := m.pageRow(r, args.Int(1)); i >= 0 {
				row := r.Row(i)
				row[3] = args.Int(2)
				r.ModifyRow(i, row)
			}
		}
	case 95: // OnPageSize（宽,高）
		if r != nil {
			if i := m.pageRow(r, args.Int(1)); i >= 0 {
				row := r.Row(i)
				row[8] = args.Int(2)
				row[9] = args.Int(3)
				r.ModifyRow(i, row)
			}
		}
	case 96: // OnPageTransparentValue
		if r != nil {
			if i := m.pageRow(r, args.Int(1)); i >= 0 {
				row := r.Row(i)
				row[5] = args.Int(2)
				r.ModifyRow(i, row)
			}
		}
	case 97: // OnPageFixedState（锁定）
		if r != nil {
			if i := m.pageRow(r, args.Int(1)); i >= 0 {
				row := r.Row(i)
				row[4] = args.Int(2)
				r.ModifyRow(i, row)
			}
		}
	case 106: // OnPageChannelChange（频道列开关）
		if r != nil {
			if i := m.pageRow(r, args.Int(1)); i >= 0 {
				row := r.Row(i)
				if ch := args.Int(2); ch >= 0 && ch < 12 {
					row[10+int(ch)] = args.Int(3)
					r.ModifyRow(i, row)
				}
			}
		}
	case 121: // CanChat：禁言校验（0 可聊 / 禁言剩余秒）
		_ = m.CanChat(k, self)
	case 22: // CleanWords：清空喊话记录（PlayerLastContent）
	default:
		k.Tracef("ChatModule: chatpage subcmd %d", sub)
	}
}

// pageRow 按表单索引定位 chatpage_rec 行。
func (m *ChatModule) pageRow(r *Record, idx int32) int {
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).Int(0) == idx {
			return i
		}
	}
	return -1
}

// cmdChat 官方字符串命令 "chat"：聊天发送入口（CanChat 门 + 广播）。
// args：频道号, 内容（密聊再带目标名）。
func (m *ChatModule) cmdChat(k *Kernel, self, from PersistID, args VarList) {
	if !m.CanChat(k, self) {
		return
	}
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	ch, content := args.Int(0), args.String(1)
	name, _ := o.Get("name").(string)
	if ch == ChannelWhisper {
		target := args.String(2)
		m.whisper(k, self, name, target, content)
		return
	}
	m.sentChats = append(m.sentChats, chatSend{Channel: ch, From: name, Content: content})
	if m.ChatOut != nil {
		m.ChatOut(ch, name, content)
	}
	k.Tracef("ChatModule: chat ch=%d from=%s", ch, name)
}

// whisper 密聊发送（在线直投 / 离线跨场景走 914 转发面）。
func (m *ChatModule) whisper(k *Kernel, self PersistID, from, to, content string) {
	if tid := k.SeekRoleName(to); tid.Valid() {
		if m.ChatOut != nil {
			m.ChatOut(ChannelWhisper, from, content)
		}
		return
	}
	// 目标不在本 member：跨场景私聊（官方 914 OnChatWhisperDissScene 经 public 服）。
	m.onWhisper(k, self, from, to, content)
}

// onWhisperDissScene 官方 @0x853350（命令 914）：跨场景私聊——查目标黑名单
// （对方 blacklist_rec 含发送者则拒发 sys_friend_0018 语义）后经 public 转发。
func (m *ChatModule) onWhisperDissScene(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	name, _ := o.Get("name").(string)
	m.onWhisper(k, self, name, args.String(0), args.String(1))
}

func (m *ChatModule) onWhisper(k *Kernel, self PersistID, from, to, content string) {
	if tid := k.SeekRoleName(to); tid.Valid() {
		if t := k.GetGameObj(tid); t != nil {
			if bl := t.Record("blacklist_rec"); bl != nil && rowByName(bl, from) >= 0 {
				m.sysMsgs = append(m.sysMsgs, "sys_friend_0018")
				return
			}
		}
	}
	if m.WhisperOut != nil {
		m.WhisperOut(from, to, content)
	}
	k.Tracef("ChatModule: whisper(diss-scene) %s→%s", from, to)
}

// onWhisperDissSceneResult 官方 @0x8537A0（命令 938）：跨场景私聊回执。
func (m *ChatModule) onWhisperDissSceneResult(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("ChatModule: whisper result %v", args.Int(0))
}

// CanChat 禁言/频率校验（官方 CanChat 子命令 121 + SilenceTick 面）。
// SilenceTick 属性（player schema）为禁言剩余拍数；>0 时拒绝并下发
// sys_item_0083（禁言X分钟模板）。
func (m *ChatModule) CanChat(k *Kernel, self PersistID) bool {
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	if o.GetInt("SilenceTick") > 0 {
		m.sysMsgs = append(m.sysMsgs, "sys_item_0083")
		return false
	}
	return true
}

// SetSilence 禁言（官方 GM/Gmcc 面：写 SilenceTick；分钟 → 拍数）。
func (m *ChatModule) SetSilence(k *Kernel, self PersistID, beats int32) {
	if o := k.GetGameObj(self); o != nil {
		_ = k.SetInt(self, "SilenceTick", beats)
		m.sysMsgs = append(m.sysMsgs, "sys_item_0083")
	}
}

// hbSilence 官方 @0x852D00：H_Silence 禁言倒计时——逐拍递减 SilenceTick，
// 归零时按 BG_131001 模板公告解除（Name/Camp/Race 身份面）。
func (m *ChatModule) hbSilence(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	tick := o.GetInt("SilenceTick")
	if tick <= 0 {
		return
	}
	if tick <= 1 {
		_ = k.SetInt(self, "SilenceTick", 0)
		m.sysMsgs = append(m.sysMsgs, "BG_131001")
		k.Tracef("ChatModule: silence end player=%v", self.ID())
		return
	}
	_ = k.SetInt(self, "SilenceTick", tick-1)
}

// SysMsgs 系统信息流水（测试面）。
func (m *ChatModule) SysMsgs() []string { return m.sysMsgs }

// SentChats 喊话流水（测试面）。
func (m *ChatModule) SentChats() []chatSend { return m.sentChats }

// rowByName 按 widestr 首列查记录行（friend/blacklist/unfamiliar/enemy 共用面）。
func rowByName(r *Record, name string) int {
	if r == nil {
		return -1
	}
	for i := 0; i < r.Rows(); i++ {
		if s, ok := r.Row(i)[0].(string); ok && s == name {
			return i
		}
	}
	return -1
}
