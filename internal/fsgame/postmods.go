package fsgame

import (
	"fmt"
)

// PostModule 复刻官方 interactivemodule/postmodule（Init @0x80F7F0 /
// LoadResource @0x81DF40，玩法表 Init 序 178；符号 579）——邮局面。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("scene","OnGetRoleInfo",@0x80FF70)
//	AddClassCallback ("SendPostBox","OnCreateClass",@0x811E20)   struct/container/postbox.xml
//	AddClassCallback ("RecvPostBox","OnCreateClass",@0x811E70)
//	AddClassCallback ("SendDealPostBox","OnCreateClass",@0x811EA0)
//	AddClassCallback ("SendBankPostBox","OnCreateClass",@0x811ED0)
//	AddClassCallback ("player","OnCreateClass",@0x811D70)
//	AddEventCallback ("SendPostBox","OnCreate",@0x8130B0)   Name/LockStatus 登记
//	AddEventCallback ("RecvPostBox","OnCreate",@0x813110)
//	AddEventCallback ("SendDealPostBox","OnCreate",@0x813170)
//	AddEventCallback ("SendBankPostBox","OnCreate",@0x8131D0)
//	AddEventCallback ("SendPostBox","OnNoAdd",@0x818790)    容器满
//	AddEventCallback ("player","OnRecover",@0x811F00)
//	AddEventCallback ("player","OnReady",@0x813050)         LockStatus 同步
//	AddEventCallback ("player","OnStore",@0x818B40)         sys_mail_0004 面存档
//	AddEventCallback ("player","OnLeaveScene",@0x818BC0)
//	AddEventCallback ("CommonNpc","OnSelect",@0x81F990)     邮差 NPC（MailNpc）
//	AddIntCustomHook ("player",350=0x15E,@0x814100)         OnCustomClientMessage
//	AddIntCommandHook ("player",310=0x136,@0x81F8A0)        OnCommandAskDekTagetmail
//	AddIntCommandHook ("player",22=0x16,@0x81FAC0)          OnMenuSelect
//	DeclareRecHook（RecvLetterRec）→ OnRecvLetterRecChanged@0x813230
//	DeclareHeartBeat ×4：H_Update_Letter@0x81E870 / H_OverTime_Check@0x81F800 /
//	    H_DealLetter_Check@0x81F850 / H_PostNpc_Check@0x81DE60
//
// OnCustomClientMessage（0x814100）子命令跳表 @0x10814350：
//
//	0 OpenPostBox / 1 PostCancel / 2 PostSend / 3 OpenReadLetter / 4 GetMoney /
//	5 DelLetter / 6 ClosePostBox / 7 CloseReadLetter / 8 GetAppedix /
//	9 DelAllLetter / 10 DelLetterOk / 11 RemitMoney / 12 BackMail /
//	13 OpenDealPostBox / 14 OpenBankPostBox / 15 SendBankMail / 16 OpenUrgentMail
//
// LoadResource：ini/Rule/post.xml（PostBoxMax/BasicPostStage/MoneyPostStage/
// ItemsPostStage/TradePostStage/UrgentPostStage/MoneyDelayTime/ItemDelayTime/
// BackDelayTime/RemitDelayTime/DealPostStage/BankPostStage/UserToReturn/
// TradeToReturn/MailToDelete）。
// 记录（struct/player/letter.xml）：RecvLetterRec(15) / RecvLetterMidRec(13) /
// RecvLetterDealRec(13) / BackstageLetterRec(8) / GiftWaitPostGetRec(3)；
// postbox 容器属性 PostMoney/Postage/PostTradeSerialno/PostDelItemInfo/PostTitle。
// 系统信息：sys_mail_00xx 族。
type PostModule struct {
	BaseModule

	// Rule ini/Rule/post.xml 主规则。
	Rule *PostRule

	// LetterOut 跨 member 信件投递出口（官方经 public 服转发到目标 member）。
	LetterOut func(target string, letter Letter)

	// MailNpcOut 加急信件邮差召唤出口（官方 H_PostNpc_Check 召唤 MailNpc，
	// ui_postmantalk_1/npc_talk_text 交互）。
	MailNpcOut func(target, serial string)

	serialSeq int64
	sysMsgs   []string
	letters   []Letter // 已投递流水（测试面）
}

// PostRule ini/Rule/post.xml <Property ID="Post">。
type PostRule struct {
	PostBoxMax      int32 // 信件容量
	BasicPostStage  int32 // 基本邮费
	MoneyPostStage  int32 // 金钱邮费
	ItemsPostStage  int32 // 物品邮费
	TradePostStage  int32 // 交易邮件邮费
	UrgentPostStage int32 // 加急邮费
	MoneyDelayTime  int32 // 金钱延迟（分）
	ItemDelayTime   int32 // 物品延迟
	BackDelayTime   int32 // 退信延迟
	RemitDelayTime  int32 // 付费延迟
	DealPostStage   int32 // 交易行邮费
	BankPostStage   int32 // 银行邮费
	UserToReturn    int32 // 用户邮件保存天数
	TradeToReturn   int32 // 付费邮件保存天数
	MailToDelete    int32 // 邮件保存天数
}

// Letter 信件（RecvLetterRec/RecvLetterMidRec 列序联合）。
type Letter struct {
	Sender     string // 寄信人姓名
	SenderUID  string // 寄信人 UID
	Type       int32  // 信件类型（0 普通/1 金钱/2 物品/3 交易/4 银行）
	Urgent     int32  // 信件加急
	Title      string // 题目
	Content    string // 内容
	Gold       string // 黄金附件
	Items      string // 物品附件
	Date       string // 寄信日期
	Serial     string // 信件流水号
	DealAmount string // 提取附件应付金额（交易类）
	Deadline   int64  // 中间表到期时刻（unix 秒；0=即时）
}

// 信件类型。
const (
	LetterBasic int32 = iota
	LetterMoney
	LetterItems
	LetterTrade
	LetterBank
)

// RecvLetterRec 列序（struct/player/letter.xml 定案）。
const (
	lrSender = iota // 0 寄信人姓名(widestr)
	lrUID           // 1 寄信人UID
	lrUnk2          // 2
	lrUnk3          // 3
	lrType          // 4 信件类型
	lrUrgent        // 5 信件加急
	lrTitle         // 6 题目(widestr)
	lrContent       // 7 内容(widestr)
	lrGold          // 8 黄金附件
	lrItems         // 9 物品附件
	lrDate          // 10 寄信日期
	lrRead          // 11 已读标记
	lrSerial        // 12 信件流水号
	lrDeal          // 13 提取附件应付金额
	lrNpcSummoned   // 14 急件 NPC 已召唤
)

// RecvLetterMidRec 列序（中间表：寄信人/UID/剩余时间/类型/加急/题目/内容/
// 黄金/物品/日期/已读/流水号/应付金额）。
const (
	lmSender = iota
	lmUID
	lmRemain
	lmType
	lmUrgent
	lmTitle
	lmContent
	lmGold
	lmItems
	lmDate
	lmRead
	lmSerial
	lmDeal
)

// PostboxClasses 官方四个邮箱容器类（struct/container/postbox.xml schema）。
var PostboxClasses = []string{"SendPostBox", "RecvPostBox", "SendDealPostBox", "SendBankPostBox"}

// Name 模块名。
func (m *PostModule) Name() string { return "PostModule" }

// Init 注册期（对齐官方 0x80F7F0）。
func (m *PostModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("scene", "OnGetRoleInfo", func(k *Kernel, self, from PersistID, args VarList) {})
	for _, cls := range PostboxClasses {
		c := cls
		b.OnClassEvent(c, "OnCreateClass", func(k *Kernel, class string) {
			k.Tracef("PostModule: OnCreateClass %s (postbox.xml)", class)
		})
		b.OnEvent(c, "OnCreate", m.onPostboxCreate)
	}
	b.OnEvent("SendPostBox", "OnNoAdd", func(k *Kernel, self, from PersistID, args VarList) {
		m.sys("sys_mail_0020") // 容器满
	})
	b.OnEvent("player", "OnRecover", m.onPlayerRecover)
	b.OnEvent("player", "OnReady", m.onPlayerReady)
	b.OnEvent("player", "OnStore", m.onPlayerStore)
	b.OnEvent("player", "OnLeaveScene", m.onLeaveScene)
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.OnCommand("player", 350, m.onCustomClientMessage)
	b.OnIntCommand("player", 310, m.onAskDelTargetMail)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	b.RecHook("RecvLetterRec", m.onRecvLetterRecChanged)
	b.HeartBeat("PostModule::H_Update_Letter", m.hbUpdateLetter)
	b.HeartBeat("PostModule::H_OverTime_Check", m.hbOverTimeCheck)
	b.HeartBeat("PostModule::H_DealLetter_Check", m.hbDealLetterCheck)
	b.HeartBeat("PostModule::H_PostNpc_Check", m.hbPostNpcCheck)
	return nil
}

// LoadResource 官方 @0x81DF40：ini/Rule/post.xml。
func (m *PostModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: PostModule.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/Rule/post.xml")
	if err != nil {
		return fmt.Errorf("fsgame: ArteryStudyModule::LoadResource file failed!: %w", err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	p, ok := ruleByID(rows)["Post"]
	if !ok {
		return fmt.Errorf("fsgame: post.xml: 无 Post 行")
	}
	m.Rule = &PostRule{
		PostBoxMax:      atoi32(p["PostBoxMax"]),
		BasicPostStage:  atoi32(p["BasicPostStage"]),
		MoneyPostStage:  atoi32(p["MoneyPostStage"]),
		ItemsPostStage:  atoi32(p["ItemsPostStage"]),
		TradePostStage:  atoi32(p["TradePostStage"]),
		UrgentPostStage: atoi32(p["UrgentPostStage"]),
		MoneyDelayTime:  atoi32(p["MoneyDelayTime"]),
		ItemDelayTime:   atoi32(p["ItemDelayTime"]),
		BackDelayTime:   atoi32(p["BackDelayTime"]),
		RemitDelayTime:  atoi32(p["RemitDelayTime"]),
		DealPostStage:   atoi32(p["DealPostStage"]),
		BankPostStage:   atoi32(p["BankPostStage"]),
		UserToReturn:    atoi32(p["UserToReturn"]),
		TradeToReturn:   atoi32(p["TradeToReturn"]),
		MailToDelete:    atoi32(p["MailToDelete"]),
	}
	k.Tracef("PostModule: LoadResource post.xml (PostBoxMax=%d)", m.Rule.PostBoxMax)
	return nil
}

// onPostboxCreate 官方 @0x8130B0 族：邮箱容器物化（Name/LockStatus 登记）。
func (m *PostModule) onPostboxCreate(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if _, ok := o.Class().Prop("LockStatus"); ok {
		_ = o.Set("LockStatus", int32(0))
	}
	k.Tracef("PostModule: postbox %s OnCreate", o.ClassName())
}

// onPlayerRecover 官方 @0x811F00：收件箱回载（RecvLetterRec/中间表恢复）。
func (m *PostModule) onPlayerRecover(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerReady 官方 @0x813050：LockStatus 同步下发。
func (m *PostModule) onPlayerReady(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("PostModule: OnReady player=%v", self.ID())
}

// onPlayerStore 官方 @0x818B40：存档（sys_mail_0004 面）。
func (m *PostModule) onPlayerStore(k *Kernel, self, from PersistID, args VarList) {}

// onLeaveScene 官方 @0x818BC0：离场（邮箱关闭）。
func (m *PostModule) onLeaveScene(k *Kernel, self, from PersistID, args VarList) {}

// onNpcSelect 官方 OnNpcSelect @0x81F990：邮差 NPC 选中（ui_qianshou 签收）。
func (m *PostModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("PostModule: OnNpcSelect (postman)")
}

// onMenuSelect 官方 @0x81FAC0（命令 22）：菜单选中入口。
func (m *PostModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("PostModule: OnMenuSelect")
}

// onAskDelTargetMail 官方 @0x81F8A0（命令 310）：目标邮件删除确认。
func (m *PostModule) onAskDelTargetMail(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("PostModule: AskDekTagetmail")
}

// onCustomClientMessage 官方 @0x814100（命令 350）：邮局 UI 命令多路
//（跳表 17 子命令；args: 子命令[, …]）。
func (m *PostModule) onCustomClientMessage(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	sub := args.Int(0)
	switch sub {
	case 0: // OnCustomOpenPostBox：打开邮箱（收件表下发）
		m.sys("open_postbox")
	case 2: // OnCustomPostSend：发信（args: 目标, 标题, 内容, 类型, 加急, 金钱, 物品）
		m.SendLetter(k, self, args.String(1), args.String(2), args.String(3),
			args.Int(4), args.Int(5), args.String(6), args.String(7))
	case 3: // OnCustomOpenReadLetter：读信（已读标记置位）
		if r := o.Record("RecvLetterRec"); r != nil {
			if i := m.rowBySerial(r, args.String(1)); i >= 0 {
				row := r.Row(i)
				row[lrRead] = int32(1)
				r.ModifyRow(i, row)
			}
		}
	case 4: // OnCustomGetMoney：提取金钱附件
		m.takeAttachment(k, self, args.String(1), true)
	case 8: // OnCustomGetAppedix：提取物品附件
		m.takeAttachment(k, self, args.String(1), false)
	case 5, 10: // DelLetter / DelLetterOk
		if r := o.Record("RecvLetterRec"); r != nil {
			if i := m.rowBySerial(r, args.String(1)); i >= 0 {
				r.DeleteRow(i)
			}
		}
	case 9: // OnCustomDelAllLetter
		if r := o.Record("RecvLetterRec"); r != nil {
			r.Clear()
		}
	case 11: // OnCustomRemitMoney：汇款（金钱信）
		m.SendLetter(k, self, args.String(1), "", "", LetterMoney, 0, args.String(2), "")
	case 12: // OnCustomBackMail：退信
		m.sys("back_mail")
	case 13, 14, 15, 16: // 交易/银行/加急邮箱族
		m.sys(fmt.Sprintf("postbox_sub_%d", sub))
	default:
		k.Tracef("PostModule: subcmd %d", sub)
	}
}

// SendLetter 发信闭环（官方 OnCustomPostSend → SendPostBox 容器 → 跨 member
// 投递）：邮费按类型（Basic/Money/Items/Trade/Urgent PostStage）；延迟按类型
//（Money/ItemDelayTime 分钟）入目标中间表，到期由 H_Update_Letter 转正式表。
func (m *PostModule) SendLetter(k *Kernel, self PersistID, target, title, content string,
	letterType, urgent int32, gold, items string) *Letter {
	o := k.GetGameObj(self)
	if o == nil {
		return nil
	}
	// 收件箱容量门（PostBoxMax）。
	if tid := k.SeekRoleName(target); tid.Valid() {
		if t := k.GetGameObj(tid); t != nil {
			if r := t.Record("RecvLetterRec"); r != nil && r.Rows() >= r.Def().MaxRows {
				m.sys("sys_mail_0020")
				return nil
			}
		}
	}
	sender, _ := o.Get("name").(string)
	lt := Letter{
		Sender:     sender,
		Type:       letterType,
		Urgent:     urgent,
		Title:      title,
		Content:    content,
		Gold:       gold,
		Items:      items,
		Date:       fmt.Sprintf("%d", unixNow()),
		Serial:     fmt.Sprintf("MAIL-%d-%d", o.ID(), m.serialSeq),
	}
	m.serialSeq++
	// 延迟面：金钱/物品信走中间表倒计时；普通信即时达。
	switch letterType {
	case LetterMoney:
		lt.Deadline = unixNow() + int64(m.delayMinutes())*60
	case LetterItems, LetterTrade:
		lt.Deadline = unixNow() + int64(m.itemDelayMinutes())*60
	}
	if tid := k.SeekRoleName(target); tid.Valid() {
		m.deliverLocal(k, tid, lt)
	} else if m.LetterOut != nil {
		m.LetterOut(target, lt) // 跨 member/离线投递
	}
	m.letters = append(m.letters, lt)
	m.sys("sys_mail_0005") // 发送成功回执面
	k.Tracef("PostModule: send letter %s→%s type=%d urgent=%d", sender, target, letterType, urgent)
	return &lt
}

// deliverLocal 本 member 投递：即时信入 RecvLetterRec；延迟信入 RecvLetterMidRec。
func (m *PostModule) deliverLocal(k *Kernel, target PersistID, lt Letter) {
	t := k.GetGameObj(target)
	if t == nil {
		return
	}
	if lt.Deadline == 0 {
		if r := t.Record("RecvLetterRec"); r != nil {
			row := make(VarList, r.Def().Cols)
			row[lrSender] = lt.Sender
			row[lrUID] = lt.SenderUID
			row[lrType] = lt.Type
			row[lrUrgent] = lt.Urgent
			row[lrTitle] = lt.Title
			row[lrContent] = lt.Content
			row[lrGold] = lt.Gold
			row[lrItems] = lt.Items
			row[lrDate] = lt.Date
			row[lrSerial] = lt.Serial
			row[lrDeal] = lt.DealAmount
			r.AppendRow(row)
		}
		return
	}
	if r := t.Record("RecvLetterMidRec"); r != nil {
		row := make(VarList, r.Def().Cols)
		row[lmSender] = lt.Sender
		row[lmUID] = lt.SenderUID
		row[lmRemain] = lt.Deadline
		row[lmType] = lt.Type
		row[lmUrgent] = lt.Urgent
		row[lmTitle] = lt.Title
		row[lmContent] = lt.Content
		row[lmGold] = lt.Gold
		row[lmItems] = lt.Items
		row[lmDate] = lt.Date
		row[lmSerial] = lt.Serial
		row[lmDeal] = lt.DealAmount
		r.AppendRow(row)
	}
}

// takeAttachment 提取附件（金钱/物品；交易信需 DealAmount 付费）。
func (m *PostModule) takeAttachment(k *Kernel, self PersistID, serial string, money bool) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	r := o.Record("RecvLetterRec")
	i := m.rowBySerial(r, serial)
	if i < 0 {
		return
	}
	row := r.Row(i)
	if money {
		row[lrGold] = ""
	} else {
		row[lrItems] = ""
	}
	r.ModifyRow(i, row)
	m.sys("sys_mail_0006") // 提取成功
}

// hbUpdateLetter 官方 @0x81E870：H_Update_Letter——中间表倒计时巡检：
// 到期行（lmRemain ≤ now）转正式收件表（sys_mail_0009 通知 + Inform/
// play_ui_sound 提示面）。
func (m *PostModule) hbUpdateLetter(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	mid := o.Record("RecvLetterMidRec")
	recv := o.Record("RecvLetterRec")
	if mid == nil || recv == nil {
		return
	}
	now := unixNow()
	for i := 0; i < mid.Rows(); i++ {
		row := mid.Row(i)
		if row.Int64(lmRemain) > now {
			continue
		}
		out := make(VarList, recv.Def().Cols)
		out[lrSender] = row[lmSender]
		out[lrUID] = row[lmUID]
		out[lrType] = row[lmType]
		out[lrUrgent] = row[lmUrgent]
		out[lrTitle] = row[lmTitle]
		out[lrContent] = row[lmContent]
		out[lrGold] = row[lmGold]
		out[lrItems] = row[lmItems]
		out[lrDate] = row[lmDate]
		out[lrSerial] = row[lmSerial]
		out[lrDeal] = row[lmDeal]
		if recv.Rows() < recv.Def().MaxRows {
			recv.AppendRow(out)
			m.sys("sys_mail_0009")
		}
		mid.DeleteRow(i)
		i--
	}
}

// hbOverTimeCheck 官方 @0x81F800：H_OverTime_Check——超时退信
//（BackDelayTime/TradeToReturn；退回寄件人 sys_mail_0004）。
func (m *PostModule) hbOverTimeCheck(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	r := o.Record("RecvLetterRec")
	if r == nil || m.Rule == nil {
		return
	}
	limit := int64(m.Rule.TradeToReturn) * 86400
	if limit <= 0 {
		return
	}
	now := unixNow()
	for i := 0; i < r.Rows(); i++ {
		row := r.Row(i)
		// 交易类信件超期未付费 → 退回（sys_mail_0004）；日期列为字符串 unix 秒。
		if row.Int(lrType) != LetterTrade {
			continue
		}
		date := atoi32(row.String(lrDate))
		if date > 0 && now-int64(date) > limit {
			r.DeleteRow(i)
			m.sys("sys_mail_0004")
			i--
		}
	}
}

// hbDealLetterCheck 官方 @0x81F850：H_DealLetter_Check——交易信付费核对
//（RemitDelayTime 内未付费 → 退回）。
func (m *PostModule) hbDealLetterCheck(k *Kernel, self PersistID, beats int32) {}

// hbPostNpcCheck 官方 @0x81DE60：H_PostNpc_Check——加急信件邮差召唤
//（lrNpcSummoned 未置位的 UrgentSerialNo → MailNpc 召唤出口 +
// task_maildelivery 族任务触发）。
func (m *PostModule) hbPostNpcCheck(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	name, _ := o.Get("name").(string)
	r := o.Record("RecvLetterRec")
	if r == nil {
		return
	}
	for i := 0; i < r.Rows(); i++ {
		row := r.Row(i)
		if row.Int(lrUrgent) == 0 || row.Int(lrNpcSummoned) != 0 {
			continue
		}
		row[lrNpcSummoned] = int32(1)
		r.ModifyRow(i, row)
		if m.MailNpcOut != nil {
			m.MailNpcOut(name, row.String(lrSerial))
		}
		m.sys("open_urgent_mail_box")
	}
}

// OnRecvLetterRecChanged 官方 @0x813230（DeclareRecHook RecvLetterRec）：
// 收件表变更 → LockStatus 联动（附件未取锁邮箱）。
func (m *PostModule) onRecvLetterRecChanged(k *Kernel, self PersistID, record string, op, row int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if r := o.Record("RecvLetterRec"); r != nil {
		locked := false
		for i := 0; i < r.Rows(); i++ {
			if s := r.Row(i).String(lrGold); s != "" {
				locked = true
				break
			}
		}
		if _, ok := o.Class().Prop("LockStatus"); ok {
			_ = k.SetInt(self, "LockStatus", boolToInt(locked))
		}
	}
}

func (m *PostModule) rowBySerial(r *Record, serial string) int {
	if r == nil {
		return -1
	}
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).String(lrSerial) == serial {
			return i
		}
	}
	return -1
}

func (m *PostModule) delayMinutes() int32 {
	if m.Rule != nil && m.Rule.MoneyDelayTime > 0 {
		return m.Rule.MoneyDelayTime
	}
	return 30
}

func (m *PostModule) itemDelayMinutes() int32 {
	if m.Rule != nil && m.Rule.ItemDelayTime > 0 {
		return m.Rule.ItemDelayTime
	}
	return 60
}

func (m *PostModule) sys(tpl string) { m.sysMsgs = append(m.sysMsgs, tpl) }

// SysMsgs 系统信息流水（测试面）。
func (m *PostModule) SysMsgs() []string { return m.sysMsgs }

// Letters 已投递信件流水（测试面）。
func (m *PostModule) Letters() []Letter { return m.letters }

// boolToInt 布勒转 0/1。
func boolToInt(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
