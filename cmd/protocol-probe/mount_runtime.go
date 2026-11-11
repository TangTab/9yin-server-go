package main

// 坐骑运行域（menu2 mount.go :351-789 函数级对齐 + Java LoginFlow
// applyMountRide/handleRideToggle/handleNewRide 语义蓝本，2026-11-11
// 按 9yin_server1/build/9yin-game-native-menu.exe 符号/行号还原重写）：
//
//   - USEITEM script=Mount（handleMountItemUse）：已拥有 → 直接上马；
//     未拥有 → 挂 pending + S2C 240 激活提示（客户端确认后回 266 sub=1）。
//   - C2S 266 custom_new_ride(sub,...)（handleNewRideCustom）：
//     sub=1 USE_ITEM——pending 命中目录且道具在包 → 消耗 1 个 + 补拥有
//     （ValidTime=0 永久口径）+ 整推 NewRideRec，激活不自动骑（menu2
//     "new ride activated (no auto-mount)"）；sub=2 SELECT_RIDE——切换
//     选中 + 整推 + 选中即骑（menu2 :632 尾接 handleRideCustom）。
//   - C2S 243 custom_ride(int)（handleRideCustom）：引擎骑乘键——骑乘中
//     下马；步行按选中上马（无选中且仅拥有一匹时选中它，menu2
//     soleOwnedMount 同口径），无坐骑拒绝。
//   - 211 出招链骑乘技能闸（mountSkillGate）：骑乘技能白名单 = 全目录
//     RideSkillID 合集；dismount 技能转下马；步行发骑乘技能忽略；
//     骑乘中非白名单技能忽略；白名单放行走官服受理簇（mount_skills.go）。
//     riding_start（骑术精通，starter 五件套）= 召唤技：步行按选中上马
//     （2026-11-12 真机 23:19 会话实证客户端主技能栏会连发它的 211）。
//   - S2C 回执：239 上马 [pack, pack]（双 int 序基双覆盖，真机勘误）/
//     240 激活提示 [1, pack, static] / 242 下马 []（均 0x1E int 外壳；
//     menu2 的 on_custom_new_ride 字符串外壳已删——客户端只认 int msgid）。
//
// 骑乘态不跨登录：重登恒步行（Mount 空串），拥有集/选中位跨登录存档
//（role_mounts.snapshot，mount_records.go 同链）。

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// C2S 坐骑族消息号（menu2 main.go 分发同值；243=引擎骑乘键，
// 266=ride_util 子令）。
const (
	clientCustomRide     int32 = 243 // CLIENT_CUSTOMMSG_RIDE
	clientCustomNewRide  int32 = 266 // CLIENT_CUSTOMMSG_NEW_RIDE
	serverCustomRideOn   int32 = 239 // 上马回执（menu2 startMount :445）
	serverCustomRideAsk  int32 = 240 // 激活确认提示（handleMountItemUse :669）
	serverCustomRideOff  int32 = 242 // 下马回执（stopMount :504 双发）

	// ridingStartSkillID 骑术精通（starter 五件套 21515，skill_view.go
	// 同名行）：主技能栏召唤技——客户端步行连发它的 211（真机 23:19 会话
	// 50+ 次），转 startSelectedMount。
	ridingStartSkillID = "riding_start"
)

// mountSession 坐骑运行域会话上下文（itemSession 同款式参数包；
// store/catalog/inventory 可 nil/空降级——纯内存口径，帧链照走）。
type mountSession struct {
	link      sceneMessageConnection
	player    *playerActor
	store     *mountDBStore
	catalog   *mountCatalog
	inventory *inventoryStore
	ctx       context.Context
	roleID    role.RoleID
	remote    string
	now       func() time.Time
}

// ---- playerActor 会话助手（锁纪律：一律经 p.mu） ----

// restoreMounts 登录装载拥有集镜像（role_mounts 快照；重登步行口径——
// 骑乘态不装载）。
func (p *playerActor) restoreMounts(mounts []mountEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mounts = mounts
}

// mountsSnapshot 拥有集快照（读侧拷贝）。
func (p *playerActor) mountsSnapshot() []mountEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshot := make([]mountEntry, len(p.mounts))
	copy(snapshot, p.mounts)
	return snapshot
}

// resetMountRows 0x0C 重入后 NewRideRec 行记账复位（客户端已无旧行可删，
// 登录链 pushMounts 同口径整表重放）。
func (p *playerActor) resetMountRows() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mountRowsSent = 0
}

// ridingMountConfig 骑乘中坐骑 configId（空=步行）。
func (p *playerActor) ridingMountConfig() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ridingConfig
}

// ridingMount 骑乘中判定。
func (p *playerActor) ridingMount() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.riding
}

// ---- 帧族 ----

// publishOwnedMounts 拥有集整推（清旧 NewRideRec 行 → 重推行 → 骑乘态帧；
// menu2 publishOwnedMountRecords :611/:627 同位，行数记账走
// player.mountRowsSent）。
func (m *mountSession) publishOwnedMounts() error {
	m.player.mu.Lock()
	oldRows := m.player.mountRowsSent
	mounts := make([]mountEntry, len(m.player.mounts))
	copy(mounts, m.player.mounts)
	m.player.mu.Unlock()
	rows, err := pushMountFrames(m.link, mounts, m.catalog, oldRows)
	if err != nil {
		return err
	}
	m.player.mu.Lock()
	m.player.mountRowsSent = rows
	m.player.mu.Unlock()
	return nil
}

// startMount 上马：骑乘态属性组（Mount=model 非空）+ S2C 239 回执。
// 239 双 int [包id, 包id]（2026-11-12 真机勘误）：引擎处理器 GetInt 取
// 值的序基（0/1-based）无法从反汇编单独定案——双覆盖两种序基都取到包
// id，多余值引擎不读。旧版另发的 on_custom_new_ride 窄串命名帧是 menu2
// 发明方言（客户端 OnCustom 只认 int msgid，串首值=type error 逐帧拒，
// trace.log 实证 23:48:22.606）——已删。骑乘速度 buf（SpeedQuality/
// RideBuffer，menu2 activateBuff :419）暂不接——Java 校准点 5：默认速度
// 跟人走，真机需要时二期补。
func (m *mountSession) startMount(def *mountDefinition) error {
	frame, err := mountRideStateFrame(def.model, def.packID, def.mountType)
	if err != nil {
		return err
	}
	if err := m.link.WriteFrame(frame); err != nil {
		return err
	}
	if frame, err = serverCustomIntMessageWithOpcode(0x1E, serverCustomRideOn,
		customInt(int32(def.packID)), customInt(int32(def.packID))); err == nil {
		err = m.link.WriteFrame(frame)
	}
	if err != nil {
		return err
	}
	m.player.mu.Lock()
	m.player.riding = true
	m.player.ridingConfig = def.configID
	m.player.mu.Unlock()
	log.Printf("%s: mounted ride config=%s static=%d model=%s type=%d speed=%.1f",
		m.remote, def.configID, def.packID, def.model, def.mountType, def.speed)
	return nil
}

// stopMount 下马（menu2 :478-512）：骑乘态属性组（Mount 空串+双 0）+
// S2C 242 回执。拥有/选中保留（骑乘键可再上）。
func (m *mountSession) stopMount() error {
	config := m.player.ridingMountConfig()
	frame, err := mountRideStateFrame("", 0, 0)
	if err != nil {
		return err
	}
	if err := m.link.WriteFrame(frame); err != nil {
		return err
	}
	if frame, err = serverCustomIntMessageWithOpcode(0x1E, serverCustomRideOff); err == nil {
		err = m.link.WriteFrame(frame)
	}
	if err != nil {
		return err
	}
	m.player.mu.Lock()
	riding := m.player.riding
	m.player.riding = false
	m.player.ridingConfig = ""
	pack := mountCurrentPack(m.player.mounts)
	m.player.mu.Unlock()
	if !riding {
		return nil
	}
	log.Printf("%s: dismounted ride config=%s static=%d", m.remote, config, pack)
	return nil
}

// persistMounts 拥有集变更落库（失败仅记日志——内存/帧已生效）。
func (m *mountSession) persistMounts(reason string) {
	if m.store == nil || m.roleID == 0 {
		return
	}
	if err := m.store.Save(m.roleID, m.player.mountsSnapshot()); err != nil {
		log.Printf("%s: persist mounts after %s: %v", m.remote, reason, err)
	}
}

// ---- USEITEM script=Mount 分流（item_use.go 挂接） ----

// handleMountItemUse 使用坐骑道具（menu2 :640-670）：已拥有 → 直接上马
//（不消耗道具，GM/已激活口径）；未拥有 → 挂 pending + S2C 240 激活提示，
// 等客户端确认回 266 sub=1 再消耗入档。
func handleMountItemUse(h *itemSession, item bagRow, def *mountDefinition) {
	m := h.mount
	if m == nil {
		log.Printf("%s: mount item use: mount session unavailable config=%s", h.remote, item.configID)
		return
	}
	log.Printf("%s: mount item use: config=%s static=%d model=%s type=%d",
		h.remote, def.configID, def.packID, def.model, def.mountType)
	h.player.mu.Lock()
	owned := ownsMountPack(h.player.mounts, def.packID)
	h.player.mu.Unlock()
	if owned {
		if err := m.startMount(def); err != nil {
			log.Printf("%s: mount item use start: %v", h.remote, err)
		}
		return
	}
	h.player.mu.Lock()
	h.player.pendingMount = def.configID
	h.player.mu.Unlock()
	if frame, err := serverCustomIntMessageWithOpcode(0x1E, serverCustomRideAsk,
		customInt(1), customInt(int32(def.packID)), customInt(int32(def.packID))); err == nil {
		if err := h.link.WriteFrame(frame); err != nil {
			log.Printf("%s: ride activate prompt: %v", h.remote, err)
		}
	} else {
		log.Printf("%s: encode ride activate prompt: %v", h.remote, err)
	}
	log.Printf("%s: ride activate prompt config=%s static=%d valid=%d",
		h.remote, def.configID, def.packID, 0)
}

// ---- C2S 243 引擎骑乘键 ----

// handleRideCustom 243 custom_ride(int)：toggle——骑乘中下马；步行按
// 选中上马（无选中且仅拥有一匹时选中它；无坐骑拒绝，menu2 :524-546）。
func handleRideCustom(m *mountSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomRide {
		return false, nil
	}
	if m == nil || m.player == nil {
		return true, nil
	}
	m.player.mu.Lock()
	riding := m.player.riding
	mounts := make([]mountEntry, len(m.player.mounts))
	copy(mounts, m.player.mounts)
	m.player.mu.Unlock()
	if riding {
		return true, m.stopMount()
	}
	return true, m.startSelectedMount()
}

// startSelectedMount 按选中上马（handleRideCustom 步行分支与
// riding_start 召唤技共用）：无选中且仅拥有一匹时选中它
//（soleOwnedMount，menu2 :524-546）；无坐骑/目录未收录拒绝，零帧。
// 返回已处理错误（拒绝时 nil，日志已记）。
func (m *mountSession) startSelectedMount() error {
	m.player.mu.Lock()
	mounts := make([]mountEntry, len(m.player.mounts))
	copy(mounts, m.player.mounts)
	m.player.mu.Unlock()
	selected := mountCurrentPack(mounts)
	soleOwned := selected == 0 && len(mounts) == 1
	if selected == 0 && !soleOwned {
		log.Printf("%s: ride rejected: no mount selected/owned", m.remote)
		return nil
	}
	current := selected
	if soleOwned {
		current = mounts[0].PackID // soleOwnedMount：唯一拥有自动选中
	}
	def := m.catalog.defByPack(current)
	if def == nil {
		log.Printf("%s: ride rejected: pack %d not in mount catalog", m.remote, current)
		return nil
	}
	if soleOwned {
		// 唯一拥有自动选中：落档并整推（选中位跨登录保持）。
		m.player.mu.Lock()
		m.player.mounts = withMountSelected(m.player.mounts, current)
		m.player.mu.Unlock()
		m.persistMounts("ride select")
		if err := m.publishOwnedMounts(); err != nil {
			log.Printf("%s: publish mounts after ride select: %v", m.remote, err)
		}
	}
	return m.startMount(def)
}

// ---- C2S 266 ride_util 子令 ----

// handleNewRideCustom 266 custom_new_ride(sub,...)：sub=1 USE_ITEM /
// sub=2 SELECT_RIDE。参数形状引擎侧不可见——tolerant 按「tag6/7 串=
// config / tag2 int=包 id」命中，不命中落 shape probe INFO（Java 同款
// 钓真机口径）。
func handleNewRideCustom(m *mountSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomNewRide {
		return false, nil
	}
	if m == nil || m.player == nil {
		return true, nil
	}
	sub := int32(0)
	if len(custom.Values) > 1 {
		if value, ok := custom.Values[1].exactInt32(); ok {
			sub = value
		}
	}
	def := newRideMountArgDef(m.catalog, custom)
	if def == nil {
		argTag, argText := byte(255), ""
		if len(custom.Values) > 2 {
			argTag = custom.Values[2].Type
			argText = custom.Values[2].Text
		}
		log.Printf("%s: new ride sub=%d arg not resolved to a mount (tag=%d text=%q) shape probe",
			m.remote, sub, argTag, argText)
		return true, nil
	}
	switch sub {
	case 1:
		newRideUseItem(m, def)
	case 2:
		newRideSelect(m, def)
	default:
		log.Printf("%s: new ride sub=%d unhandled config=%s", m.remote, sub, def.configID)
	}
	return true, nil
}

// newRideMountArgDef 266 第三值解析：tag6/7 串按 config、tag2 int 按包。
func newRideMountArgDef(catalog *mountCatalog, custom clientCustomMessage) *mountDefinition {
	if len(custom.Values) < 3 {
		return nil
	}
	arg := custom.Values[2]
	switch arg.Type {
	case 6, 7:
		return catalog.defByConfig(strings.TrimSpace(arg.Text))
	case 2:
		if pack, ok := arg.exactInt32(); ok && pack > 0 {
			return catalog.defByPack(uint32(pack))
		}
	}
	return nil
}

// newRideUseItem sub=1 USE_ITEM（menu2 :569-618）：pending 命中 + 道具
// 在包 → 消耗 1 个 + 补拥有（永久）+ 落档 + 整推；激活不自动骑。
// pending 与目标不一致（道具中途被用/丢）只清挂起，不动拥有。
func newRideUseItem(m *mountSession, def *mountDefinition) {
	m.player.mu.Lock()
	pending := m.player.pendingMount
	m.player.pendingMount = ""
	owned := ownsMountPack(m.player.mounts, def.packID)
	m.player.mu.Unlock()
	if pending != def.configID {
		log.Printf("%s: new ride sub=1 but no pending mount (pending=%q config=%s)", m.remote, pending, def.configID)
		return
	}
	if owned {
		// 已拥有再使用：不消耗（GM 口径），清挂起即止。
		return
	}
	row, inBag := findBagMountRow(m.player, def.configID)
	if !inBag {
		log.Printf("%s: new ride sub=1 mount item gone config=%s", m.remote, def.configID)
		return
	}
	m.player.mu.Lock()
	m.player.mounts = withOwnedMount(m.player.mounts, def.packID, def.configID)
	m.player.mu.Unlock()
	if !m.consumeMountItem(row) {
		// 消耗失败（DB 拒绝/行漂移）：回滚拥有，防白送。
		m.player.mu.Lock()
		m.player.mounts = removeMountPack(m.player.mounts, def.packID)
		m.player.mu.Unlock()
		log.Printf("%s: new ride sub=1 consume failed config=%s", m.remote, def.configID)
		return
	}
	m.persistMounts("new ride use item")
	if err := m.publishOwnedMounts(); err != nil {
		log.Printf("%s: publish mounts after new ride: %v", m.remote, err)
	}
	log.Printf("%s: new ride activated (no auto-mount) config=%s static=%d", m.remote, def.configID, def.packID)
}

// consumeMountItem 消耗坐骑道具 1 个（DB+内存+帧对一体，
// consumeBagItemCount 同链；menu2 "persist bag after mount consume" :603）。
func (m *mountSession) consumeMountItem(row bagRow) bool {
	session := &itemSession{
		link: m.link, player: m.player, store: m.inventory,
		ctx: m.ctx, roleID: m.roleID, remote: m.remote, now: m.now,
	}
	return session.consumeBagItemCount(row, 1)
}

// newRideSelect sub=2 SELECT_RIDE（menu2 :626-632）：切换选中 + 落档 +
// 整推 + 选中即骑（尾接 handleRideCustom 同语义——步行直接上选中坐骑）。
func newRideSelect(m *mountSession, def *mountDefinition) {
	m.player.mu.Lock()
	owned := ownsMountPack(m.player.mounts, def.packID)
	m.player.mu.Unlock()
	if !owned {
		// SELECT_RIDE 只切换已拥有坐骑（菜单2 "new ride selected" 前置）；
		// 未拥有的走 USE_ITEM 链入档。
		log.Printf("%s: new ride select rejected: pack %d not owned", m.remote, def.packID)
		return
	}
	m.player.mu.Lock()
	m.player.mounts = withMountSelected(m.player.mounts, def.packID)
	riding := m.player.riding
	m.player.mu.Unlock()
	m.persistMounts("new ride select")
	if err := m.publishOwnedMounts(); err != nil {
		log.Printf("%s: publish mounts after new ride select: %v", m.remote, err)
	}
	log.Printf("%s: new ride selected static=%d", m.remote, def.packID)
	if riding {
		// 骑乘中切换：换骑（先下后上，属性组以新坐骑收尾）。
		if err := m.stopMount(); err != nil {
			log.Printf("%s: dismount before switch: %v", m.remote, err)
		}
	}
	if err := m.startMount(def); err != nil {
		log.Printf("%s: new ride select start: %v", m.remote, err)
	}
}

// removeMountPack 拥有集剔除（消耗失败回滚用；纯函数）。
func removeMountPack(mounts []mountEntry, packID uint32) []mountEntry {
	next := make([]mountEntry, 0, len(mounts))
	for _, entry := range mounts {
		if entry.PackID != packID {
			next = append(next, entry)
		}
	}
	return next
}

// findBagMountRow 背包找坐骑道具行（首个同 config 行；0x19 帧寻址用）。
func findBagMountRow(player *playerActor, configID string) (bagRow, bool) {
	player.mu.Lock()
	defer player.mu.Unlock()
	for _, row := range player.bag {
		if row.configID == configID && row.amount > 0 {
			return row, true
		}
	}
	return bagRow{}, false
}

// ---- 211 出招链骑乘技能闸（skill_combat.go 挂接） ----

// mountSkillGateResult 骑乘技能闸三态。
type mountSkillGateResult int

const (
	mountSkillPass     mountSkillGateResult = iota // 放行（普通或白名单骑乘技能）
	mountSkillDismount                             // 下马技能：转下马、不出招
	mountSkillReject                               // 拒绝（步行发骑乘技/骑乘发非白名单技）
)

// mountSkillGate 出招前骑乘闸（menu2 handleRideSkillCustom :695-738）：
//   - 步行发骑乘技能 → 拒绝（"ride skill while not riding (ignored)"）；
//   - 骑乘中发非白名单技能 → 拒绝（"ride skill not in mount skills"）；
//   - 骑乘中发 dismount 技能 → 下马不出招；
//   - 其余放行（放行后由 emitRideSkillCast 发官服受理簇，mount_skills.go）。
func mountSkillGate(m *mountSession, skillID string) mountSkillGateResult {
	if m == nil || m.player == nil || skillID == "" {
		return mountSkillPass
	}
	isRideSkill := m.catalog.hasRideSkill(skillID)
	m.player.mu.Lock()
	riding := m.player.riding
	m.player.mu.Unlock()
	if !riding && isRideSkill {
		log.Printf("%s: ride skill %q while not riding (ignored)", m.remote, skillID)
		return mountSkillReject
	}
	if riding && !isRideSkill {
		log.Printf("%s: ride skill %q not in mount skills (ignored)", m.remote, skillID)
		return mountSkillReject
	}
	if riding && isMountDismountSkill(skillID) {
		if err := m.stopMount(); err != nil {
			log.Printf("%s: dismount via ride skill: %v", m.remote, err)
		}
		return mountSkillDismount
	}
	return mountSkillPass
}

// isMountDismountSkill 下马技能判定（RideSkillID 命名族
// riding_dismount / riding_dismount_diao，menu2 isDismountSkill 同义）。
func isMountDismountSkill(skillID string) bool {
	return strings.Contains(strings.ToLower(skillID), "dismount")
}
