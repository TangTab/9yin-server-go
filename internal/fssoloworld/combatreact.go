package fssoloworld

// combatreact.go 命中时受击者与攻击者收到的附加帧：受击动作 / 受击特效、
// 连击计数（m24 10）与 MaxHits、击杀簇里的 m24 13。
//
// 判据分两层：
//
//	要不要发  招式表：判定帧的硬直触发器不带受击者动画（产物 Stagger=1）时
//	          受击反应由服务端下发；触发器自带动画（cqgf01 的 hurt_f_2_s /
//	          hurt_f_1_s / hurt_f_1_l、飞镖）则客户端按招式表自播，官服不发
//	          任何受击帧——女号包 cqgf01 22 次、普攻 65 次、飞镖 3 次命中零
//	          受击帧（docs/official/群战包解码定版与受控机制勘误.md §三）。
//	发什么    服务端专有的技能事件包（逐级属性 EventProcess，客户端资源里
//	          没有这张表）：按抓包复原，见 hitSpecs。

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// hitSpec 一次有效命中在 m24 17 之外还要发的帧（按技能），外加施放期间的
// 自身效果。
type hitSpec struct {
	effect    string        // 每击挂到受击者身上的特效（m24 40）
	firstOnly bool          // 受击动作 m18 hurt_half 只在首次交手发；缺省每击都发
	deathMark bool          // 击杀时另发 m24 13
	hurtSent  bool          // 受击动作由技能自己的帧序先行下发：击杀簇不再补发 m18
	selfBuff  *selfBuffSpec // 施放期间挂在玩家身上的自身 buff
	control   *hitControl   // 命中附带的受控（combatcontrol.go）
}

// selfBuffSpec 施放期间挂在玩家身上的自身 buff（技能事件包）。
type selfBuffSpec struct {
	id    string   // buff ConfigID
	props []string // 随 buff 置 1 的属性位（buff 的属性包；收招清回 0）
}

// hitSpecs 抓包复原的技能事件包（女号包 197~232s / 345~400s / 721~815s）：
//
//	CS_jh_cqgf04 / CS_jh_cqgf06           每击 body_hit_3_self；受击动作仅首次
//	                                      交手（357.896 首击 m18+c40，
//	                                      384.503 / 388.619 只 c40）
//	skill_musejuben_4                     推—拉：施放期间自身霸体 buf_fixed_red
//	                                      （CantBeSkillLocked / CantBeSkillMustLocked），
//	                                      命中挂击晕与减速 buff，第 2 拍起每击
//	                                      击退（combatcontrol.go）
//	skill_musejuben_4 / skill_musejuben_7  击杀带 m24 13（757.405 / 758.416 /
//	CS_jh_hyjf07                          820.649 / 挑战包 483.077）；两包其余
//	                                      26 次击杀都不带
//
//	CS_jh_hyjf08                          闪击通道技：每波的受击动作是两值的
//	                                      m18[目标,"hurt_f_1_l_half"]，由波次
//	                                      引擎在结算帧之前下发（skillflash.go）
//
// 不在表里的技能：Stagger 拍每击 m18 hurt_half（群战包常规武学的多数形态：
// yydf03/04、lyfh07、xhmdl08、jgjf05、txd08、lzs03、ywt07），无特效。
var hitSpecs = map[string]hitSpec{
	"CS_jh_cqgf04": {effect: "body_hit_3_self", firstOnly: true},
	"CS_jh_cqgf06": {effect: "body_hit_3_self", firstOnly: true},
	"skill_musejuben_4": {deathMark: true,
		selfBuff: &selfBuffSpec{id: "buf_fixed_red", props: []string{"CantBeSkillLocked", "CantBeSkillMustLocked"}},
		control:  &hitControl{buffs: []string{"buf_skill_musejuben_41", "buf_skill_musejuben_42"}, knock: true}},
	"skill_musejuben_7": {deathMark: true},
	"CS_jh_hyjf07":      {deathMark: true},
	"CS_jh_hyjf08":      {hurtSent: true},
}

func hitSpecOf(skill string) hitSpec {
	if spec, ok := hitSpecs[skill]; ok {
		return spec
	}
	return hitSpec{}
}

// comboWindow 连击计数窗口：两击间隔不超过它则累计。女号包全包 190 次相邻
// m24[自己,10,n]：延续的最长间隔 1.925s，重置的最短间隔 2.037s。
const comboWindow = 2 * time.Second

// comboHitLocked 记一次有效命中并累计连击（超窗先归零），返回连击数是否
// 刷新了最高纪录。调用方持 combatState.mu。
func comboHitLocked(live *playerLive, now time.Time) (record bool) {
	if live.comboAt.IsZero() || now.Sub(live.comboAt) > comboWindow {
		live.combo = 0
	}
	live.combo++
	live.comboAt = now
	if live.combo > live.maxHits {
		live.maxHits, record = live.combo, true
	}
	return record
}

// comboMissLocked 未命中（闪避）打断连击：此后首次命中重新从 1 计（挑战包
// 442.392：… 3 4 5 → 被闪避 → 1 2）。调用方持 combatState.mu。
func comboMissLocked(live *playerLive) {
	live.combo, live.comboAt = 0, time.Time{}
}

// comboFrames 连击帧：第 2 击起 m24[自己,10,连击数]；record（连击数刷新
// 最高纪录）时另推 MaxHits——女号包首击即推 MaxHits=1，其后 1→24 每次刷新
// 才推，平纪录不推。调用方持 combatState.mu（live 的锁）。
func (k *frameSink) comboFrames(live *playerLive, record bool) {
	if live.combo >= 2 {
		k.custom(24, k.playerObject(), entergen.CustomInt(10), entergen.CustomInt(int32(live.combo)))
	}
	if record {
		k.priv("MaxHits", entergen.Int32Value(int32(live.maxHits)))
	}
}

// stripNpcBuffs 摘掉 NPC 身上的全部 buff，返回逐个 m169 摘帧。官服击杀簇：
// 致死一击的 m24 17 之后、m24 29 之前，受击者身上的 buff 逐个摘除
// （358.469 摘 buf_CS_jh_cqgf01；757.405 摘 musejuben_42 与 buf_CS_wd_tjq08）。
// 到期事件随后落到空列表上，不再发帧。
func (s *Server) stripNpcBuffs(f *npcFighter) [][]byte {
	f.mu.Lock()
	buffs := f.buffs
	f.buffs = nil
	f.mu.Unlock()
	var frames [][]byte
	for _, buff := range buffs {
		if frame := s.buffFrame(false, npcHandle(f), buff); frame != nil {
			frames = append(frames, frame)
		}
	}
	return frames
}

// ---- 施放期间的自身 buff ----

// activeSelfBuff 施放期间在身的自身 buff（收招时摘）。combatState.mu 护。
type activeSelfBuff struct {
	buff  combatBuff
	slot  int
	props []string
}

// bufferSlotBase 玩家动态 buff 的首个槽位：抓包里玩家身上的首个 buff 落
// BufferInfo3（buf_fixed_red 与 BuffInParry 皆然）。
const bufferSlotBase = 3

// grantSelfBuffLocked 施放期间的自身 buff：属性位置 1 → buff 槽串
// BufferInfoN → m169 挂载（抓包 731.152：SP → CantBeSkillLocked=1 →
// CantBeSkillMustLocked=1 → BufferInfo3="21EA,350000F3,102A,1,C3934ED4F1D,0"
// → m169 buf_fixed_red）。调用方持 combatState.mu。
func (s *Server) grantSelfBuffLocked(b *frameBuilder, st *combatState, pl *Player, spec selfBuffSpec) {
	lua := s.combatLua()
	if lua.skills == nil {
		return
	}
	def, err := lua.skills.Buff(spec.id, 1)
	if err != nil {
		return
	}
	life := time.Duration(def.LifeTimeMS) * time.Millisecond
	if life <= 0 {
		life = 20 * time.Second
	}
	buff := combatBuff{id: def.ID, level: 1, category: def.Category, lua: def.LuaScript,
		caster: playerHandle(pl), expireAt: combatNow().Add(life)}
	slot := bufferSlotBase + len(st.live.buffs)
	st.live.buffs = append(st.live.buffs, buff)
	st.selfBuff = &activeSelfBuff{buff: buff, slot: slot, props: spec.props}
	for _, name := range spec.props {
		b.prop(name, entergen.ByteValue(1))
	}
	b.frames = append(b.frames, s.buffSlotFrame(pl, slot, buff, def.StaticData))
	b.frames = append(b.frames, s.buffFrame(true, playerHandle(pl), buff))
}

// releaseSelfBuffLocked 收招时摘掉施放期间的自身 buff：m169 摘 → 槽串清空
// → 属性位清回 0（抓包 738.115：CurSkillID="" → m169 摘 buf_fixed_red →
// BufferInfo3="" → CantBeSkillLocked=0 → CantBeSkillMustLocked=0）。调用方
// 持 combatState.mu；没有在身的自身 buff 返回 nil。
func (s *Server) releaseSelfBuffLocked(pl *Player, st *combatState) [][]byte {
	active := st.selfBuff
	if active == nil {
		return nil
	}
	st.selfBuff = nil
	buffs := &st.live.buffs
	for i := range *buffs {
		if (*buffs)[i].id == active.buff.id {
			*buffs = append((*buffs)[:i], (*buffs)[i+1:]...)
			break
		}
	}
	b := newFrameBuilder(pl)
	b.frames = append(b.frames, s.buffFrame(false, playerHandle(pl), active.buff))
	b.frames = append(b.frames, s.buffSlotFrame(pl, active.slot, combatBuff{}, 0))
	for _, name := range active.props {
		b.prop(name, entergen.ByteValue(0))
	}
	return b.frames
}

// buffSlotFrame 玩家 buff 槽串帧 BufferInfoN：在身时为
// "静态包ID,目标对象,own,等级,到期(1601 纪元毫秒),0"（各段大写十六进制），
// 摘除后为空串。
func (s *Server) buffSlotFrame(pl *Player, slot int, buff combatBuff, staticData int) []byte {
	text := ""
	ident := pl.wireIdent().orFixed()
	if buff.id != "" {
		text = fmt.Sprintf("%X,%X,%X,%d,%X,0", staticData, ident.objectID, ident.ownerID, buff.level, filetimeMS(buff.expireAt))
	}
	frame, err := entergen.ObjectPropertyFrame(ident.objectID, ident.ownerID, 0, []entergen.IndexedProperty{
		entergen.NamedProperty("BufferInfo"+strconv.Itoa(slot), entergen.StringValue(text))})
	if err != nil {
		return nil
	}
	return frame
}
