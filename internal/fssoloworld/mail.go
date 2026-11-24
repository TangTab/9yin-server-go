package fssoloworld

// mail.go GM 邮件链（2433 线 mail_records.go/Java LoginFlow applySendMail+
// handleReadLetter/handleGetAppendix 移植）：
//
//	邮箱=记录表 RecvLetterRec(319) 0x11 行（entergen/mailrecords.go 编码）；
//	登录 ClientReady 尾/切图 0x0C 重入簇整表重放，GM 在线直推单行；
//	行变更方言=0x12 删行 + 0x11 尾插（服务端切片与客户端行序 1:1）；
//	C2S 353 读信（serial）/359 领附件（serial——入包 0x19+0x18 先行，
//	MailFlowTest 帧序钉死）/350/357/358/363 零动作簿记；
//	离线目标 store 往返：合成 clientID 段 name-only 0x02 查档（回执
//	RST1+ACC1 尾段）→ 补信 → 0x04 回写，重启不丢。

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// C2S 邮件族自定义消息号（lua64-protocol-dict.md form_mail.lua 直读口径；
// Java/2433 按字面 msgId 处理，MailFlowTest 真机验证）。
const (
	clientCustomPostOpenBox      int32 = 350 // 开邮箱（列表=记录行绑定，零动作）
	clientCustomPostCancel       int32 = 351 // 取消写信
	clientCustomPostSend         int32 = 352 // 玩家发信（未实现，簿记）
	clientCustomPostRead         int32 = 353 // 读信（参数=serial）
	clientCustomPostSelect       int32 = 357 // 勾选
	clientCustomPostCloseReadBox int32 = 358 // 关读信窗
	clientCustomPostGetAppendix  int32 = 359 // 领附件（参数=serial）
	clientCustomPostSelectSend   int32 = 363 // 选发信附件
)

// 邮箱/文案限值（Java applySendMail 校验同口径）。
const (
	mailBoxCap       = 50
	mailTitleLimit   = 64
	mailContentLimit = 1024
)

// errBagPageFull 入包分类页满（gm item/邮件领附件共用哨兵）。
var errBagPageFull = errors.New("bag page full")

// gmMailSeq GM 邮件序列号计数（Java GM_MAIL_SEQ 同源：gm<毫秒>-<序号>）。
var gmMailSeq atomic.Uint64

// ---- 离线投递簿记（合成 clientID 段；world 进程生命周期） ----

const offlineSynthClientBase uint32 = 0x7E00_0000

var (
	offlineMailMu       sync.Mutex
	offlineMailByClient = map[uint32]*offlineMailCtx{}
	offlineMailCounter  atomic.Uint32
)

type offlineMailCtx struct {
	serial uint32 // 0x02 请求随行 serial（回写 0x04 原样带回）
	name   string // 目标角色名（name-only 查档键）
	letter rolestate.Letter
}

// NewGMLetter 构造一封 GM 系统邮件（Type 101 系统页签；金/银官方数字
// 窄串方言恒 "0"——多币种邮件后续扩展）。
func NewGMLetter(title, content, appendix string) rolestate.Letter {
	if title == "" {
		title = "系统邮件"
	}
	return rolestate.Letter{
		Serial:   fmt.Sprintf("gm%d-%d", time.Now().UnixMilli(), gmMailSeq.Add(1)),
		Type:     entergen.LetterSystemPost,
		Sender:   "GM",
		Title:    title,
		Content:  content,
		Gold:     "0",
		Silver:   "0",
		Appendix: appendix,
		DateUnix: time.Now().Unix(),
	}
}

// pushLetterRows 整表重放（登录 ClientReady 尾/切图重入簇/GM 直推后）。
// 空表零帧。返回下发帧数。
func (s *Server) pushLetterRows(pl *Player) int {
	if !pl.haveRoleState || len(pl.roleState.Letters) == 0 {
		return 0
	}
	ident := pl.wireIdent()
	frames, err := entergen.LetterRowFrames(ident.objectID, ident.ownerID, pl.roleState.Letters)
	if err != nil {
		s.trace("letter frames failed client=%d: %v", pl.Conn1, err)
		return 0
	}
	for _, frame := range frames {
		s.sendToClient(pl, frame)
	}
	return len(frames)
}

// deliverLetterOnline 在线投递：入档 + 单行 0x11 直推 + 存档（邮箱满拒）。
func (s *Server) deliverLetterOnline(pl *Player, letter rolestate.Letter) error {
	if len(pl.roleState.Letters) >= mailBoxCap {
		return fmt.Errorf("mailbox full (%d)", mailBoxCap)
	}
	pl.roleState.Letters = append(pl.roleState.Letters, letter)
	ident := pl.wireIdent()
	frame, err := entergen.LetterRowFrame(ident.objectID, ident.ownerID, letter)
	if err != nil {
		s.trace("letter row frame failed client=%d: %v", pl.Conn1, err)
	} else {
		s.sendToClient(pl, frame)
	}
	s.sendRoleSave(pl)
	return nil
}

// dispatchOfflineMail 离线投递第一步：合成 clientID 段登记 + name-only
// 0x02 查档（store LoadRole account 空串=按名匹配，回执带 ACC1 真实账户）。
// 回执走 storeLoadEnter 的 FindByConnect 落空分支 → offlineMailReply。
func (s *Server) dispatchOfflineMail(name string, letter rolestate.Letter) bool {
	id := offlineSynthClientBase + offlineMailCounter.Add(1)
	ctx := &offlineMailCtx{serial: 0, name: name, letter: letter}
	offlineMailMu.Lock()
	offlineMailByClient[id] = ctx
	offlineMailMu.Unlock()
	payload := buildLoadRoleStore(id, ctx.serial, uint32(s.cfg.ServerID), "", name)
	if !s.inner.SendToStore(0, 0, 0x02, payload) {
		offlineMailMu.Lock()
		delete(offlineMailByClient, id)
		offlineMailMu.Unlock()
		return false
	}
	return true
}

// offlineMailReply 合成 clientID 的 0x02 回执：补信入 RST1 → 0x04 回写。
// rest=回执 para 宽串起的尾段（parseRST1/parseACC1 扫描定位）。
func (s *Server) offlineMailReply(clientID uint32, rest []byte) {
	offlineMailMu.Lock()
	ctx := offlineMailByClient[clientID]
	delete(offlineMailByClient, clientID)
	offlineMailMu.Unlock()
	if ctx == nil {
		return
	}
	raw := parseRST1(rest)
	if raw == nil {
		s.trace("offline mail: role %q has no RST1 (old profile), letter dropped", ctx.name)
		return
	}
	state, err := rolestate.UnmarshalRoleState(*raw)
	if err != nil {
		s.trace("offline mail: role %q state unmarshal failed: %v", ctx.name, err)
		return
	}
	if len(state.Letters) >= mailBoxCap {
		s.trace("offline mail: role %q mailbox full, letter dropped", ctx.name)
		return
	}
	state.Letters = append(state.Letters, ctx.letter)
	rst1, err := rolestate.MarshalRoleState(state)
	if err != nil {
		s.trace("offline mail: role %q marshal failed: %v", ctx.name, err)
		return
	}
	account := parseACC1(rest)
	payload := buildSaveStore(clientID, ctx.serial, account, ctx.name,
		state.Position, []byte(rst1))
	if s.inner.SendToStore(0, 0, 0x04, payload) {
		s.trace("offline mail: role %q letter %q stored (account=%q)", ctx.name, ctx.letter.Serial, account)
	}
}

// parseACC1 回执尾段解 'ACC1' 账户串（magic 方言同 parseRST1）。
func parseACC1(rest []byte) string {
	magic := uint32(0x31434341) // 'ACC1'
	for off := 0; off+8 <= len(rest); off++ {
		if binaryLE(rest[off:]) != magic {
			continue
		}
		ln := int(binaryLE(rest[off+4:]))
		if ln <= 0 || off+8+ln > len(rest) {
			return ""
		}
		return string(rest[off+8 : off+8+ln])
	}
	return ""
}

func binaryLE(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// ---- C2S 处理（dispatchCustom case 接线） ----

// handlePostTrace 零动作邮件族（开箱/勾选/关窗/写信族）：列表由客户端
// data_binder 绑记录行，服务端无应答——trace 降噪合并。
func (s *Server) handlePostTrace(pl *Player, id int32) {
	s.trace("post msg %d client=%d (no server action)", id, pl.Conn1)
}

// handleReadLetter C2S 353：按 serial 置已读 → 0x12 删行 + 0x11 尾插
// （行移尾保持服务端切片与客户端行序 1:1，Java reorderLetterRow 同款）。
func (s *Server) handleReadLetter(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomPostRead) {
		return
	}
	serial, ok := customStringAt(message, 1)
	if !ok {
		return
	}
	idx := findLetter(pl.roleState.Letters, serial)
	if idx < 0 {
		s.trace("read letter: serial %q not found client=%d", serial, pl.Conn1)
		return
	}
	letter := pl.roleState.Letters[idx]
	if letter.Read {
		return // 幂等
	}
	letter.Read = true
	s.reorderLetterRow(pl, idx, letter)
	s.sendRoleSave(pl)
}

// handleGetAppendix C2S 359：解析附件 XML → 入包（0x19+0x18 先行）→
// 0x12 删行 + 0x11 重推（附件清空+置已读）→ 存档。重复领取幂等零帧。
func (s *Server) handleGetAppendix(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomPostGetAppendix) {
		return
	}
	serial, ok := customStringAt(message, 1)
	if !ok {
		return
	}
	idx := findLetter(pl.roleState.Letters, serial)
	if idx < 0 {
		s.trace("get appendix: serial %q not found client=%d", serial, pl.Conn1)
		return
	}
	letter := pl.roleState.Letters[idx]
	if letter.Appendix == "" {
		if !letter.Read { // 已领过但未读：补置已读
			letter.Read = true
			s.reorderLetterRow(pl, idx, letter)
			s.sendRoleSave(pl)
		}
		return
	}
	configID, amount := parseAppendixXML(letter.Appendix)
	if configID == "" {
		s.trace("get appendix: serial %q malformed xml client=%d", serial, pl.Conn1)
		return
	}
	if amount <= 0 {
		amount = 1
	}
	// 帧序钉死：入包 0x19+0x18 → 0x12 删行 → 0x11 重推（MailFlowTest）。
	if _, push, err := s.grantBagItem(pl, configID, amount, 1); err != nil {
		s.trace("get appendix: grant %q x%d failed client=%d: %v", configID, amount, pl.Conn1, err)
		return
	} else {
		for _, frame := range push {
			s.sendToClient(pl, frame)
		}
	}
	letter.Appendix = ""
	letter.Read = true
	s.reorderLetterRow(pl, idx, letter)
	s.sendRoleSave(pl)
}

// reorderLetterRow 行移尾：切片删除+尾插，先 0x12（原行号）后 0x11。
func (s *Server) reorderLetterRow(pl *Player, idx int, letter rolestate.Letter) {
	st := &pl.roleState
	st.Letters = append(st.Letters[:idx], st.Letters[idx+1:]...)
	st.Letters = append(st.Letters, letter)
	ident := pl.wireIdent()
	s.sendToClient(pl, entergen.LetterDelRowFrame(ident.objectID, ident.ownerID, idx))
	if frame, err := entergen.LetterRowFrame(ident.objectID, ident.ownerID, letter); err == nil {
		s.sendToClient(pl, frame)
	} else {
		s.trace("letter row frame failed client=%d: %v", pl.Conn1, err)
	}
}

// grantBagItem 入包：FreeBagSlot/NextItemSeq 分配 + BagItems 落档 +
// 0x19/0x18 成对帧（gm item 与邮件领附件共用；满页 errBagPageFull）。
// 模板行数据随目录入档（物品目录可用时）：ItemType/MaxAmount 与装备行
// 渲染属性——使用链（itemuse.go）与行帧都取自这里。
func (s *Server) grantBagItem(pl *Player, configID string, amount, viewID int) (rolestate.BagItem, [][]byte, error) {
	slot := pl.roleState.FreeBagSlot(viewID)
	if slot == 0 {
		return rolestate.BagItem{}, nil, errBagPageFull
	}
	item := rolestate.BagItem{
		Slot:     slot,
		ConfigID: configID,
		Amount:   amount,
		ViewID:   viewID,
		Seq:      pl.roleState.NextItemSeq(),
	}
	if entry, ok := s.itemCatalogOf().Lookup(configID); ok {
		item.ItemType, item.MaxAmount, item.EquipType = entry.ItemType, entry.MaxAmount, entry.EquipType
		item.ColorLevel, item.Photo = entry.ColorLevel, entry.Photo
	}
	push, err := entergen.BagItemPushFrames(item)
	if err != nil {
		return rolestate.BagItem{}, nil, err
	}
	pl.roleState.BagItems = append(pl.roleState.BagItems, item)
	return item, push, nil
}

// ---- 解析 helpers ----

// customStringAt TVarList 第 i 值取串（Type 6；越界/非串 false）。
func customStringAt(message entergen.CustomInbound, i int) (string, bool) {
	if i >= len(message.Values) || message.Values[i].Type != 6 {
		return "", false
	}
	return message.Values[i].Text, true
}

// findLetter 按 serial 定位（O(n)，邮箱 ≤50）。
func findLetter(letters []rolestate.Letter, serial string) int {
	for i := range letters {
		if letters[i].Serial == serial {
			return i
		}
	}
	return -1
}

// parseAppendixXML 单物件附件 XML 取 Config/Amount（AppendixXML 对偶；
// 属性序不定，双锚独立扫描）。
func parseAppendixXML(xml string) (string, int) {
	configID := scanAttr(xml, "Config")
	amount := 1
	if v := scanAttr(xml, "Amount"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			amount = n
		}
	}
	return configID, amount
}

func scanAttr(s, key string) string {
	marker := key + `="`
	i := strings.Index(s, marker)
	if i < 0 {
		return ""
	}
	rest := s[i+len(marker):]
	if j := strings.IndexByte(rest, '"'); j >= 0 {
		return rest[:j]
	}
	return ""
}
