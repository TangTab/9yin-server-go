package fsgame

import (
	"fmt"
	"math/rand"
	"strings"
)

// 五期：物品玩法域核心五模块（docs/official/FsGame全解与Go复刻.md §8
// 养成·装备与物品玩法）。官方源码树：commonmodule\（dropmodule/
// pickupmodule）+ playerbasemodule\（toolboxmodule）+ itembasemodule\
// （strengthen）。

// ---- DropModule ----

// DropModule 复刻官方 commonmodule\dropmodule（Init @0x632860 /
// LoadResource @0x632C00，玩法表 Init 序 214；符号 1511）——掉落引擎。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddIntCustomHook ("player",68=0x44,@0x639950) OnCustomNpcGetDrop
//	DeclareLuaExt ×7：nx_get_normal_dropitems@0x6318D0 /
//	    nx_reload_drop_resource@0x631860 / nx_get_special_npcdropn /
//	    nx_get_world_npcdropn / nx_get_special_DropIDdropn /
//	    nx_get_world_DropIDdropn / nx_clear_dropitem_Manage
//
// LoadResource @0x632C00 六表链（ini/Drop/）：
//
//	Drop_Item.xml（物品组：ItemTeamID → "ItemID,数量" 串）
//	Drop_Team.xml（一级组：FstTeamID → "ItemTeamID,数量,权重,?;…"）
//	World_Drop_Simple.xml（世界直落：PackageID → ItemID/ItemNum/Probability）
//	World_Drop_Complex.xml（世界组落：PackageID → "FstTeamID,次数,权重"）
//	Spec_Drop_Simple/Complex.xml（专属同构——任务掉落）
type DropModule struct {
	BaseModule

	// 六表（ItemTeamID/FstTeamID/PackageID → 行；直落表同包多行——
	// 每行独立概率门）。
	DropItem     map[string]map[string]string   // ItemTeamID → 行
	DropTeam     map[string]map[string]string   // FstTeamID → 行
	WorldSimple  map[string][]map[string]string // PackageID → 候选行集
	WorldComplex map[string]map[string]string
	SpecSimple   map[string][]map[string]string
	SpecComplex  map[string]map[string]string

	// rnd 掉落随机源（测试可换固定源）。
	rnd *rand.Rand
	// rolls 掉落流水（测试面）。
	rolls []string
}

// DropItemEntry 一次掉落产物。
type DropItemEntry struct {
	ItemID string
	Count  int32
}

// Name 模块名。
func (m *DropModule) Name() string { return "DropModule" }

// Init 注册期（对齐官方 0x632860）。
func (m *DropModule) Init(k *Kernel) error {
	m.DropItem = make(map[string]map[string]string)
	m.DropTeam = make(map[string]map[string]string)
	m.WorldSimple = make(map[string][]map[string]string)
	m.WorldComplex = make(map[string]map[string]string)
	m.SpecSimple = make(map[string][]map[string]string)
	m.SpecComplex = make(map[string]map[string]string)
	m.rnd = rand.New(rand.NewSource(unixNow()))
	b := k.Bus()
	b.OnCommand("player", 68, m.onCustomNpcGetDrop)
	return nil
}

// LoadResource 官方 @0x632C00：ini/Drop/ 六表。
func (m *DropModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: DropModule.LoadResource: ResourcePath 未设置")
	}
	// 掉落包表（World/Spec）以 PackageID 为主键（真档行 ID 为自增序号；
	// 直落表同包多行）。
	// 物品组/一级组以组键（ItemTeamID/FstTeamID 列）为主键。
	if err := m.loadTable(k, "ini/Drop/Drop_Item.xml", func(id string, row map[string]string) {
		m.DropItem[row["ItemTeamID"]] = row
	}); err != nil {
		return err
	}
	if err := m.loadTable(k, "ini/Drop/Drop_Team.xml", func(id string, row map[string]string) {
		m.DropTeam[row["FstTeamID"]] = row
	}); err != nil {
		return err
	}
	if err := m.loadTable(k, "ini/Drop/World_Drop_Simple.xml", func(id string, row map[string]string) {
		m.WorldSimple[row["PackageID"]] = append(m.WorldSimple[row["PackageID"]], row)
	}); err != nil {
		return err
	}
	if err := m.loadTable(k, "ini/Drop/World_Drop_Complex.xml", func(id string, row map[string]string) {
		m.WorldComplex[row["PackageID"]] = row
	}); err != nil {
		return err
	}
	if err := m.loadTable(k, "ini/Drop/Spec_Drop_Simple.xml", func(id string, row map[string]string) {
		m.SpecSimple[row["PackageID"]] = append(m.SpecSimple[row["PackageID"]], row)
	}); err != nil {
		return err
	}
	if err := m.loadTable(k, "ini/Drop/Spec_Drop_Complex.xml", func(id string, row map[string]string) {
		m.SpecComplex[row["PackageID"]] = row
	}); err != nil {
		return err
	}
	k.Tracef("DropModule: LoadResource 六表 %d/%d/%d/%d/%d/%d",
		len(m.DropItem), len(m.DropTeam), len(m.WorldSimple),
		len(m.WorldComplex), len(m.SpecSimple), len(m.SpecComplex))
	return nil
}

// loadTable 单表装载（行回调）。
func (m *DropModule) loadTable(k *Kernel, path string, put func(id string, row map[string]string)) error {
	data, err := readResource(k.ResourcePath, path)
	if err != nil {
		return fmt.Errorf("fsgame: DropModule::LoadResource %s: %w", path, err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	for id, row := range ruleByID(rows) {
		put(id, row)
	}
	return nil
}

// onCustomNpcGetDrop 官方 @0x639950（cust 68=0x44）：NPC 掉落表查询下发。
func (m *DropModule) onCustomNpcGetDrop(k *Kernel, self, from PersistID, args VarList) {
	entries := m.RollDrop(args.String(0), true)
	k.Tracef("DropModule: NpcGetDrop %s → %d 项", args.String(0), len(entries))
}

// RollDrop 掉落主入口（nx_get_world_DropIDdropn/nx_get_special_DropIDdropn
// 的 Go 等价）：PackageID 先查 Simple（直落概率门）再查 Complex
//（FstTeam 组展开，Drop_Team 权重 → Drop_Item 物品组）；world=false 走
// Spec 表族。
func (m *DropModule) RollDrop(packageID string, world bool) []DropItemEntry {
	var out []DropItemEntry
	simple, complex := m.WorldSimple, m.WorldComplex
	if !world {
		simple, complex = m.SpecSimple, m.SpecComplex
	}
	if rows := simple[packageID]; len(rows) > 0 {
		for _, row := range rows { // 同包多行，每行独立概率门
			if m.rnd.Float64() <= atof64(row["Probability"]) {
				out = append(out, DropItemEntry{row["ItemID"], atoi32(row["ItemNum"])})
			}
		}
		return out
	}
	if row := complex[packageID]; row != nil {
		for _, fst := range strings.Split(row["FstTeamID"], ";") {
			parts := strings.Split(strings.TrimSpace(fst), ",")
			if len(parts) < 2 {
				continue
			}
			teamID := parts[0]
			times := atoi32(parts[1])
			for i := int32(0); i < times; i++ {
				out = append(out, m.rollTeam(teamID)...)
			}
		}
	}
	m.rolls = append(m.rolls, fmt.Sprintf("%s(%v)→%d", packageID, world, len(out)))
	return out
}

// rollTeam 一级组掷取（Drop_Team 行 "ItemTeamID,数量,权重,?;…"——按权重
// 抽一组）→ 物品组展开（Drop_Item "ItemID,数量" 串）。
func (m *DropModule) rollTeam(fstTeamID string) []DropItemEntry {
	row := m.DropTeam[fstTeamID]
	if row == nil {
		return nil
	}
	type cand struct {
		itemTeam string
		count    int32
		weight   float64
	}
	var cands []cand
	var total float64
	for _, seg := range strings.Split(row["ItemTeamID"], ";") {
		parts := strings.Split(strings.TrimSpace(seg), ",")
		if len(parts) < 3 {
			continue
		}
		w := atof64(parts[2])
		cands = append(cands, cand{parts[0], atoi32(parts[1]), w})
		total += w
	}
	if len(cands) == 0 || total <= 0 {
		return nil
	}
	pick := m.rnd.Float64() * total
	var chosen cand
	for _, c := range cands {
		pick -= c.weight
		if pick <= 0 {
			chosen = c
			break
		}
	}
	if chosen.itemTeam == "" {
		chosen = cands[0]
	}
	var out []DropItemEntry
	if ir := m.DropItem[chosen.itemTeam]; ir != nil {
		for _, seg := range strings.Split(ir["ItemID"], ";") {
			parts := strings.Split(strings.TrimSpace(seg), ",")
			if len(parts) >= 1 && parts[0] != "" {
				n := chosen.count
				if len(parts) >= 2 {
					n = atoi32(parts[1])
				}
				out = append(out, DropItemEntry{parts[0], n})
			}
		}
	}
	return out
}

// NxGetNormalDropitems nx_get_normal_dropitems 的 Go 等价（普通怪掉落包）。
func (m *DropModule) NxGetNormalDropitems(packageID string) []DropItemEntry {
	return m.RollDrop(packageID, true)
}

// NxReloadDropResource nx_reload_drop_resource 的 Go 等价（清表缓存）。
func (m *DropModule) NxReloadDropResource(k *Kernel) error { return m.LoadResource(k) }

// Rolls 掉落流水（测试面）。
func (m *DropModule) Rolls() []string { return m.rolls }

// ---- PickUpModule ----

// PickUpModule 复刻官方 commonmodule\pickupmodule（Init @0x6BC4C0，玩法表
// Init 序 208；符号 248）——拾取与战利品分配。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("AttackNpc","OnSelect",@0x6BDFC0) / ("item","OnSelect",
//	    @0x6BE490 OnItemSelect) / ("GivingNpc","OnSelect",@0x6BE500)
//	AddEventCallback ("AttackNpc"/"CommonNpc","OnDestroy",@0x6C4A00
//	    OnNpcDestroy——尸体掉落箱物化)
//	AddEventCallback ("NpcDropBox","OnAfterAdd",@0x6BF470) / ("NpcDropBox",
//	    "OnRemove",@0x6BF5E0)
//	AddEventCallback ("player","OnLeaveScene",@0x6BC9E0)
//	AddIntCustomHook ("player",382,@0x6BEEE0 OnCloseView) / (380,
//	    @0x6BFBA0 OnCustomPickUpNpcDropItem) / (388,@0x6BFC30
//	    OnCustomCaptainAlloNpcDropItem 队长分配) / (387,@0x6BFF00
//	    OnCustomDiceResult 掷骰) / (383,@0x6BFEB0 OnCustomArrangeDropItem)
//	DeclareHeartBeat ×2：H_Dice_Wait@0x6C4800（掷骰等待）/ H_Turn_Protect@
//	    @0x6C4C30（轮流保护期）
//
// 归属面（AttackNpc schema）：open_player_rec（可拾取玩家表）/
// pickup_member_rec（组队成员表）；CanPick 判定串。
type PickUpModule struct {
	BaseModule

	// PickOut 拾取落包出口（官方：物品对象入玩家 ToolBox；Go 面 sync 挂接——
	// item 对象句柄由 ContainerModule 物化，此处出 (player, itemID, count)）。
	PickOut func(player PersistID, itemID string, count int32)

	// dropBoxes 掉落箱登记：NPC → 箱对象句柄。
	dropBoxes map[PersistID]PersistID
	// picked 拾取流水（测试面）。
	picked []string
}

// Name 模块名。
func (m *PickUpModule) Name() string { return "PickUpModule" }

// Init 注册期（对齐官方 0x6BC4C0）。
func (m *PickUpModule) Init(k *Kernel) error {
	m.dropBoxes = make(map[PersistID]PersistID)
	b := k.Bus()
	b.OnEvent("AttackNpc", "OnSelect", m.onNpcSelect)
	b.OnEvent("item", "OnSelect", m.onItemSelect)
	b.OnEvent("GivingNpc", "OnSelect", m.onPlayerSelect)
	b.OnEvent("AttackNpc", "OnDestroy", m.onNpcDestroy)
	b.OnEvent("CommonNpc", "OnDestroy", m.onNpcDestroy)
	b.OnEvent("NpcDropBox", "OnAfterAdd", m.onDropBoxAfterAdd)
	b.OnEvent("NpcDropBox", "OnRemove", m.onDropBoxRemove)
	b.OnEvent("player", "OnLeaveScene", m.onLeaveScene)
	b.OnCommand("player", 382, m.onCloseView)
	b.OnCommand("player", 380, m.onCustomPickUpNpcDropItem)
	b.OnCommand("player", 388, m.onCustomCaptainAlloNpcDropItem)
	b.OnCommand("player", 387, m.onCustomDiceResult)
	b.OnCommand("player", 383, m.onCustomArrangeDropItem)
	b.HeartBeat("PickUpModule::H_Dice_Wait", m.hbDiceWait)
	b.HeartBeat("PickUpModule::H_Turn_Protect", m.hbTurnProtect)
	return nil
}

// onNpcDestroy 官方 @0x6C4A00：NPC 死亡 → 掉落箱物化（NpcDropBox 容器 +
// open_player_rec 归属登记）。
func (m *PickUpModule) onNpcDestroy(k *Kernel, self, from PersistID, args VarList) {
	if m.dropBoxes[self].Valid() {
		return
	}
	id, err := k.CreateObject("NpcDropBox")
	if err != nil {
		return
	}
	m.dropBoxes[self] = id
	k.Tracef("PickUpModule: dropbox ← npc %v", self.ID())
}

// DropInto 掉落注入（DropModule 掷取产物 → NPC 掉落箱；官方经容器行）。
// Go 面直接记录待拾取列表。
func (m *PickUpModule) DropInto(k *Kernel, npc PersistID, entries []DropItemEntry) {
	if m.dropBoxes[npc] == NullPersistID {
		m.onNpcDestroy(k, npc, NullPersistID, nil)
	}
	box := m.dropBoxes[npc]
	if !box.Valid() {
		return
	}
	if o := k.GetGameObj(box); o != nil {
		if r := o.Record("QuestDropRecord"); r != nil {
			for _, e := range entries {
				r.AppendRow(VarList{e.ItemID, fmt.Sprintf("%d", e.Count)})
			}
		}
	}
	k.RunEvent("NpcDropBox", "OnAfterAdd", box, npc, nil)
}

// onDropBoxAfterAdd 官方 @0x6BF470：箱内新增（拾取权开放计时面）。
func (m *PickUpModule) onDropBoxAfterAdd(k *Kernel, self, from PersistID, args VarList) {}

// onDropBoxRemove 官方 @0x6BF5E0：箱空销毁。
func (m *PickUpModule) onDropBoxRemove(k *Kernel, self, from PersistID, args VarList) {
	for npc, box := range m.dropBoxes {
		if box == self {
			delete(m.dropBoxes, npc)
		}
	}
}

// CanPick 拾取权判定（open_player_rec 归属表——首杀者优先期；表空=全开放）。
func (m *PickUpModule) CanPick(k *Kernel, npc, player PersistID) bool {
	box := m.dropBoxes[npc]
	if !box.Valid() {
		return false
	}
	o := k.GetGameObj(box)
	if o == nil {
		return false
	}
	r := o.Record("QuestDropRecord")
	if r == nil || r.Rows() == 0 {
		return false
	}
	if or := k.GetGameObj(npc).Record("open_player_rec"); or != nil && or.Rows() > 0 {
		for i := 0; i < or.Rows(); i++ {
			if or.Row(i)[0] == Var(player) {
				return true
			}
		}
		return false
	}
	return true
}

// onCustomPickUpNpcDropItem 官方 @0x6BFBA0（cust 380=0x17C）：拾取
//（args: NPC 句柄, 物品串）→ CanPick 门 → 首条出箱 → PickOut（入 ToolBox）。
func (m *PickUpModule) onCustomPickUpNpcDropItem(k *Kernel, self, from PersistID, args VarList) {
	npc := args.Object(0)
	if !m.CanPick(k, npc, self) {
		return
	}
	box := m.dropBoxes[npc]
	o := k.GetGameObj(box)
	if o == nil {
		return
	}
	r := o.Record("QuestDropRecord")
	if r == nil || r.Rows() == 0 {
		return
	}
	row := r.Row(0)
	r.DeleteRow(0)
	if m.PickOut != nil {
		m.PickOut(self, row.String(0), atoi32(row.String(1)))
	}
	m.picked = append(m.picked, row.String(0))
	if r.Rows() == 0 { // 箱空销毁（OnRemove 面 + 登记清理）
		k.RunEvent("NpcDropBox", "OnRemove", box, npc, nil)
		k.RemoveObject(box)
		delete(m.dropBoxes, npc)
	}
}

// onCustomCaptainAlloNpcDropItem 官方 @0x6BFC30（cust 388）：队长分配
//（TeamModule 协作面）。
func (m *PickUpModule) onCustomCaptainAlloNpcDropItem(k *Kernel, self, from PersistID, args VarList) {
	m.picked = append(m.picked, "captain_allo")
}

// onCustomDiceResult 官方 @0x6BFF00（cust 387）：掷骰结果收令
//（H_Dice_Wait 等待期结束）。
func (m *PickUpModule) onCustomDiceResult(k *Kernel, self, from PersistID, args VarList) {
	m.picked = append(m.picked, fmt.Sprintf("dice:%d", args.Int(0)))
}

// onCustomArrangeDropItem 官方 @0x6BFEB0（cust 383）：整理掉落箱视图。
func (m *PickUpModule) onCustomArrangeDropItem(k *Kernel, self, from PersistID, args VarList) {}

// onCloseView 官方 @0x6BEEE0（cust 382）：关闭拾取视图。
func (m *PickUpModule) onCloseView(k *Kernel, self, from PersistID, args VarList) {}

// onNpcSelect 官方 @0x6BDFC0：NPC 选中（拾取视图开启）。
func (m *PickUpModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {}

// onItemSelect 官方 @0x6BE490：地上物品选中。
func (m *PickUpModule) onItemSelect(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerSelect 官方 @0x6BE500。
func (m *PickUpModule) onPlayerSelect(k *Kernel, self, from PersistID, args VarList) {}

// onLeaveScene 官方 @0x6BC9E0：离场清拾取面。
func (m *PickUpModule) onLeaveScene(k *Kernel, self, from PersistID, args VarList) {}

// hbDiceWait 官方 @0x6C4800：掷骰等待期。
func (m *PickUpModule) hbDiceWait(k *Kernel, self PersistID, beats int32) {}

// hbTurnProtect 官方 @0x6C4C30：轮流保护期。
func (m *PickUpModule) hbTurnProtect(k *Kernel, self PersistID, beats int32) {}

// DropBoxOf 掉落箱句柄（测试面）。
func (m *PickUpModule) DropBoxOf(npc PersistID) PersistID { return m.dropBoxes[npc] }

// Picked 拾取流水（测试面）。
func (m *PickUpModule) Picked() []string { return m.picked }

// ---- ToolBoxModule ----

// ToolBoxModule 复刻官方 playerbasemodule\toolboxmodule（Init @0x1633F0，
// 玩法表 Init 序 147；符号 183）——背包容器模块。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnRecover",@0x163900) / ("player","OnReady",
//	    @0x163C80)
//	AddEventCallback ("ToolBox","OnCreate",@0x163F80) / ("ToolBox",
//	    "OnAfterAdd",@0x163750)
//	AddEventCallback ("AddToolBox","OnCreate",@0x163FE0) / ("AddToolBox",
//	    "OnNoAdd",@0x164040)
//	AddIntCustomHook ("player",31,@0x164460 OnCustomUseItem) / (29,
//	    @0x164620 OnCustomUseItem_OnItem) / (914,@0x165500 OnCustomOpen_
//	    ExtGroove) / (917,@0x164DB0 OnCustomOpenLinkAuth)
//	AddIntCommandHook ("player",319,@0x165F70 OnToolBoxDecTicketRes) /
//	    (320,@0x166100 OnOpenGrooveDecTicketRes)
type ToolBoxModule struct {
	BaseModule

	// AddOut 物品入包出口（ContainerModule 物化后挂接；测试面）。
	AddOut func(player PersistID, itemID string, count int32)
	// items 背包镜像：玩家 → (物品ID → 数量)。
	items map[PersistID]map[string]int32
	// ops 命令流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *ToolBoxModule) Name() string { return "ToolBoxModule" }

// Init 注册期（对齐官方 0x1633F0）。
func (m *ToolBoxModule) Init(k *Kernel) error {
	m.items = make(map[PersistID]map[string]int32)
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onPlayerRecover)
	b.OnEvent("player", "OnReady", m.onPlayerReady)
	b.OnEvent("ToolBox", "OnCreate", m.onToolBoxCreate)
	b.OnEvent("ToolBox", "OnAfterAdd", m.onToolBoxAfterAdd)
	b.OnEvent("AddToolBox", "OnCreate", m.onAddToolBoxCreate)
	b.OnEvent("AddToolBox", "OnNoAdd", m.onAddToolBoxOnNoAdd)
	b.OnCommand("player", 31, m.onCustomUseItem)
	b.OnCommand("player", 29, m.onCustomUseItemOnItem)
	b.OnCommand("player", 914, m.onCustomOpenExtGroove)
	b.OnCommand("player", 917, m.onCustomOpenLinkAuth)
	b.OnIntCommand("player", 319, m.onToolBoxDecTicketRes)
	b.OnIntCommand("player", 320, m.onOpenGrooveDecTicketRes)
	return nil
}

// onPlayerRecover 官方 @0x163900：背包回载。
func (m *ToolBoxModule) onPlayerRecover(k *Kernel, self, from PersistID, args VarList) {
	if m.items[self] == nil {
		m.items[self] = make(map[string]int32)
	}
}

// onPlayerReady 官方 @0x163C80：背包容量/整理下发。
func (m *ToolBoxModule) onPlayerReady(k *Kernel, self, from PersistID, args VarList) {}

// onToolBoxCreate 官方 @0x163F80。
func (m *ToolBoxModule) onToolBoxCreate(k *Kernel, self, from PersistID, args VarList) {}

// onToolBoxAfterAdd 官方 @0x163750：物品入包（堆叠计数）。
func (m *ToolBoxModule) onToolBoxAfterAdd(k *Kernel, self, from PersistID, args VarList) {}

// onAddToolBoxCreate 官方 @0x163FE0：扩容包物化。
func (m *ToolBoxModule) onAddToolBoxCreate(k *Kernel, self, from PersistID, args VarList) {}

// onAddToolBoxOnNoAdd 官方 @0x164040：背包满。
func (m *ToolBoxModule) onAddToolBoxOnNoAdd(k *Kernel, self, from PersistID, args VarList) {}

// AddItem 入包（拾取/奖励/邮件附件统一落点；堆叠合并）。
func (m *ToolBoxModule) AddItem(k *Kernel, self PersistID, itemID string, count int32) {
	if m.items[self] == nil {
		m.items[self] = make(map[string]int32)
	}
	m.items[self][itemID] += count
	if m.AddOut != nil {
		m.AddOut(self, itemID, count)
	}
	k.Tracef("ToolBoxModule: +item %s×%d player=%v", itemID, count, self.ID())
}

// RemoveItem 出包（消耗；数量不足拒）。
func (m *ToolBoxModule) RemoveItem(k *Kernel, self PersistID, itemID string, count int32) bool {
	bag := m.items[self]
	if bag == nil || bag[itemID] < count {
		return false
	}
	bag[itemID] -= count
	if bag[itemID] == 0 {
		delete(bag, itemID)
	}
	return true
}

// CountOf 持有数（测试面）。
func (m *ToolBoxModule) CountOf(self PersistID, itemID string) int32 {
	return m.items[self][itemID]
}

// onCustomUseItem 官方 @0x164460（cust 31=0x1F）：使用道具
//（args: 物品ID[, 数量]）→ 消耗 + 使用效果出口。
func (m *ToolBoxModule) onCustomUseItem(k *Kernel, self, from PersistID, args VarList) {
	itemID := args.String(0)
	count := args.Int(1)
	if count <= 0 {
		count = 1
	}
	if !m.RemoveItem(k, self, itemID, count) {
		return
	}
	m.ops = append(m.ops, "use:"+itemID)
	k.Tracef("ToolBoxModule: use %s×%d", itemID, count)
}

// onCustomUseItemOnItem 官方 @0x164620（cust 29=0x1D）：对目标物使用道具。
func (m *ToolBoxModule) onCustomUseItemOnItem(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "use_on:"+args.String(0))
}

// onCustomOpenExtGroove 官方 @0x165500（cust 914=0x392）：扩展槽开启。
func (m *ToolBoxModule) onCustomOpenExtGroove(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "ext_groove")
}

// onCustomOpenLinkAuth 官方 @0x164DB0（cust 917=0x395）：链接授权。
func (m *ToolBoxModule) onCustomOpenLinkAuth(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "link_auth")
}

// onToolBoxDecTicketRes 官方 @0x165F70（命令 319=0x13F）：扣券回执。
func (m *ToolBoxModule) onToolBoxDecTicketRes(k *Kernel, self, from PersistID, args VarList) {}

// onOpenGrooveDecTicketRes 官方 @0x166100（命令 320=0x140）。
func (m *ToolBoxModule) onOpenGrooveDecTicketRes(k *Kernel, self, from PersistID, args VarList) {}

// Ops 命令流水（测试面）。
func (m *ToolBoxModule) Ops() []string { return m.ops }

// ---- EquipmentModule ----

// EquipmentModule 复刻官方 equipmentmodule（Init @0xDFA50，玩法表 Init 序
// 85；符号 1701）——装备穿戴。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnRecover",@0xE2B30) / ("player","OnReady",
//	    @0xE2DA0) / ("player","OnLeaveScene",@0xE2ED0)
//	AddEventCallback ("EquipBox","OnCreate",@0xE2F20) / ("OnAfterAdd",
//	    @0xE40F0——Energy/EquipType/EnergyValue/EnergyExpLv/EnergyLevel
//	    属性链 + LockActionSet + FashionState) / ("OnRemove",@0xE4410) /
//	    ("OnNoAdd",@0xE4690) / ("OnNoRemove",@0xE47B0)
//	AddEventCallback ("Equipment","OnLoad",@0xE48B0) / ("OnRecover",
//	    @0xE4A20) / ("EnergyShield","OnLoad",@0xE4B50) / ("OnRecover",
//	    @0xE4BB0)
//	AddIntCustomHook ("player",61,@0xE4DA0 OnCustomShoweEquipType 显形) /
//	    (233,@0xEACD0 OnCustomSkillShortcut) / (59,@0xED950
//	    OnCustomEnergyUpgrade 能量升级)
//	AddEventCallback ("CommonNpc","OnSelect",@0xED5F0)
//	AddIntCommandHook ("player",22,@0xED740 OnMenuSelect)
//	DeclareCritical ×4+：C_OnHardinessChanged@0xE2F80 / C_OnWeaponChanged@
//	    @0xE3540 / C_OnWeaponStateChanged@0xE34F0 / C_OnEquipEnergyCount@
//	    @0xE30E0
//	DeclareHeartBeat ×2：H_BeginDepletion_Energy（能量衰减泵）
type EquipmentModule struct {
	BaseModule

	// equipped 穿戴表：玩家 → EquipType → 装备句柄。
	equipped map[PersistID]map[string]PersistID
	// ops 命令流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *EquipmentModule) Name() string { return "EquipmentModule" }

// Init 注册期（对齐官方 0xDFA50）。
func (m *EquipmentModule) Init(k *Kernel) error {
	m.equipped = make(map[PersistID]map[string]PersistID)
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onPlayerRecover)
	b.OnEvent("player", "OnReady", m.onPlayerReady)
	b.OnEvent("player", "OnLeaveScene", m.onLeaveScene)
	b.OnEvent("EquipBox", "OnCreate", m.onEquipBoxCreate)
	b.OnEvent("EquipBox", "OnAfterAdd", m.onEquipBoxAfterAdd)
	b.OnEvent("EquipBox", "OnRemove", m.onEquipBoxRemove)
	b.OnEvent("EquipBox", "OnNoAdd", m.onEquipBoxNoAdd)
	b.OnEvent("EquipBox", "OnNoRemove", m.onEquipBoxNoRemove)
	b.OnEvent("Equipment", "OnLoad", m.onEquipmentLoad)
	b.OnEvent("Equipment", "OnRecover", m.onEquipmentRecover)
	b.OnEvent("EnergyShield", "OnLoad", m.onEquipmentLoad)
	b.OnEvent("EnergyShield", "OnRecover", m.onEquipmentRecover)
	b.OnCommand("player", 61, m.onCustomShoweEquipType)
	b.OnCommand("player", 233, m.onCustomSkillShortcut)
	b.OnCommand("player", 59, m.onCustomEnergyUpgrade)
	b.OnEvent("CommonNpc", "OnSelect", m.onCommonNpcSelect)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	b.Critical("player", "Hardiness", m.onHardinessChanged)
	b.Critical("player", "Weapon", m.onWeaponChanged)
	b.Critical("player", "WeaponState", m.onWeaponStateChanged)
	b.Critical("player", "EquipEnergyCount", m.onEquipEnergyCount)
	b.HeartBeat("EquipmentModule::H_BeginDepletion_Energy", m.hbBeginDepletionEnergy)
	return nil
}

// Equip 穿戴闭环（官方 EquipBox OnAfterAdd 面）：EquipType 栏位判定 →
// 同栏旧装卸下 → Energy/EquipType/EnergyValue 属性链登记 → 武器栏触发
// C_OnWeaponChanged（战斗面换武器重算）。
func (m *EquipmentModule) Equip(k *Kernel, self, item PersistID) bool {
	o := k.GetGameObj(self)
	io := k.GetGameObj(item)
	if o == nil || io == nil {
		return false
	}
	equipType := io.GetString("EquipType")
	if equipType == "" {
		return false
	}
	if m.equipped[self] == nil {
		m.equipped[self] = make(map[string]PersistID)
	}
	if old := m.equipped[self][equipType]; old.Valid() {
		m.Unequip(k, self, old)
	}
	m.equipped[self][equipType] = item
	// 官方属性链（OnAfterAdd 串定案）：Energy 激活 + 能量等级面。
	if _, ok := io.Class().Prop("Energy"); ok {
		_ = io.Set("Energy", int32(1))
	}
	if equipType == "Weapon" {
		// C_OnWeaponChanged 临界联动（真档 dump schema 无 Weapon object 属性，
		// 官方运行期有——写失败容忍，换武器面以栏位表为准）。
		_ = k.SetProp(self, "Weapon", item)
	}
	m.ops = append(m.ops, "equip:"+equipType)
	k.Tracef("EquipmentModule: equip %s player=%v", equipType, self.ID())
	return true
}

// Unequip 卸下（OnRemove 面）。
func (m *EquipmentModule) Unequip(k *Kernel, self, item PersistID) bool {
	io := k.GetGameObj(item)
	if io == nil {
		return false
	}
	equipType := io.GetString("EquipType")
	if m.equipped[self][equipType] != item {
		return false
	}
	delete(m.equipped[self], equipType)
	if equipType == "Weapon" {
		_ = k.SetProp(self, "Weapon", NullPersistID)
	}
	m.ops = append(m.ops, "unequip:"+equipType)
	return true
}

// EquippedAt 栏位装备（测试面）。
func (m *EquipmentModule) EquippedAt(self PersistID, equipType string) PersistID {
	return m.equipped[self][equipType]
}

// onPlayerRecover 官方 @0xE2B30：穿戴面回载（EquipBox 重建）。
func (m *EquipmentModule) onPlayerRecover(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerReady 官方 @0xE2DA0。
func (m *EquipmentModule) onPlayerReady(k *Kernel, self, from PersistID, args VarList) {}

// onLeaveScene 官方 @0xE2ED0。
func (m *EquipmentModule) onLeaveScene(k *Kernel, self, from PersistID, args VarList) {}

// onEquipBoxCreate 官方 @0xE2F20。
func (m *EquipmentModule) onEquipBoxCreate(k *Kernel, self, from PersistID, args VarList) {}

// onEquipBoxAfterAdd 官方 @0xE40F0：穿戴落地（FashionState/LockActionSet/
// Energy/EnergyValue/EnergyExpLv/EnergyLevel 面记）。
func (m *EquipmentModule) onEquipBoxAfterAdd(k *Kernel, self, from PersistID, args VarList) {}

// onEquipBoxRemove 官方 @0xE4410。
func (m *EquipmentModule) onEquipBoxRemove(k *Kernel, self, from PersistID, args VarList) {}

// onEquipBoxNoAdd 官方 @0xE4690：栏位冲突。
func (m *EquipmentModule) onEquipBoxNoAdd(k *Kernel, self, from PersistID, args VarList) {}

// onEquipBoxNoRemove 官方 @0xE47B0。
func (m *EquipmentModule) onEquipBoxNoRemove(k *Kernel, self, from PersistID, args VarList) {}

// onEquipmentLoad/onEquipmentRecover 官方 @0xE48B0/@0xE4A20：装备对象回载。
func (m *EquipmentModule) onEquipmentLoad(k *Kernel, self, from PersistID, args VarList)    {}
func (m *EquipmentModule) onEquipmentRecover(k *Kernel, self, from PersistID, args VarList) {}

// onCustomShoweEquipType 官方 @0xE4DA0（cust 61=0x3B）：显形装备切换。
func (m *EquipmentModule) onCustomShoweEquipType(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "show")
}

// onCustomSkillShortcut 官方 @0xEACD0（cust 233=0xE9）：装备技能快捷栏。
func (m *EquipmentModule) onCustomSkillShortcut(k *Kernel, self, from PersistID, args VarList) {
}

// onCustomEnergyUpgrade 官方 @0xED950（cust 59=0x3D）：装备能量升级。
func (m *EquipmentModule) onCustomEnergyUpgrade(k *Kernel, self, from PersistID, args VarList) {
}

// onCommonNpcSelect 官方 @0xED5F0：NPC 交互（修理/升级面）。
func (m *EquipmentModule) onCommonNpcSelect(k *Kernel, self, from PersistID, args VarList) {}

// onMenuSelect 官方 @0xED740（命令 22=0x16）。
func (m *EquipmentModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {}

// C_OnHardinessChanged 官方 @0xE2F80：装备硬度变更（战斗磨损面）。
func (m *EquipmentModule) onHardinessChanged(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnWeaponChanged 官方 @0xE3540：换武器（技能组/普攻映射切换——
// SkillModule 普攻表 Skill_Normal_Attack 协作面）。
func (m *EquipmentModule) onWeaponChanged(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnWeaponStateChanged 官方 @0xE34F0。
func (m *EquipmentModule) onWeaponStateChanged(k *Kernel, self PersistID, prop string, old Var) {
}

// C_OnEquipEnergyCount 官方 @0xE30E0：装备能量计数。
func (m *EquipmentModule) onEquipEnergyCount(k *Kernel, self PersistID, prop string, old Var) {}

// H_BeginDepletion_Energy 官方能量衰减泵。
func (m *EquipmentModule) hbBeginDepletionEnergy(k *Kernel, self PersistID, beats int32) {}

// Ops 命令流水（测试面）。
func (m *EquipmentModule) Ops() []string { return m.ops }

// ---- Strengthen ----

// Strengthen 复刻官方 itembasemodule\strengthen（Init @0xD4540 /
// LoadResource @0xD4860，玩法表 Init 序 91；符号 405）——装备强化。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnLeaveScene"/"OnContinue",@0xD4E80
//	    OnChangeScene——强化台清理)
//	AddIntCustomHook ("player",82,@0xD5440 OnCustomStrengthenClose) /
//	    (83,@0xD55E0 OnCustomStrengthenSelect 选装备——StrenthenNpc/
//	    EquipType/Energy/Special/ColorLevel/LockStatus/StarLevel 校验) /
//	    (84,@0xD5E60 OnCustomStrengthenExecute——CurrentTicket/
//	    CurrentBlackgold 消耗 → StarLevel+1) / (85,@0xD5CC0
//	    OnCustomStrengthenDrop 放入)
//	AddIntCommandHook ("player",22,@0xD5150 OnMenuSelect)
//	AddEventCallback ("CommonNpc","OnSelect",@0xD5020 OnNpcSelect——
//	    StrenthenNpc 强化台登记)
//	DeclareCritical ("…StarLevel")→C_OnStarLevelChanged@0xD8B90
//	DeclareLuaExt ("nx_strengthen",@0xD4300)
//	DeclareHeartBeat ×2：HB_RefreshCustomInfo@0xD6450 / DoContinueEnhance@
//	    @0xD6300（连续强化泵）
//
// LoadResource @0xD4860：ini/ItemEnhance/enhance_limit.xml
//（StartLevel=15 强化起始等级门）。
type Strengthen struct {
	BaseModule

	// EnhanceLimit 起始等级门（enhance_limit.xml StartLevel）。
	EnhanceLimit int32
	// 台面状态：玩家 → {Npc, Equip, Ticket, Blackgold}（官方 StrenthenNpc/
	// StrenthenEquip/CurrentTicket/CurrentBlackgold 属性面）。
	benches map[PersistID]*strengthenBench
	// sysMsgs 系统信息观测（sys_item_0067/0070、sys_depot_0016）。
	sysMsgs []string
	// ops 命令流水（测试面）。
	ops []string
}

type strengthenBench struct {
	Npc       PersistID
	Equip     PersistID
	Ticket    int32
	Blackgold int32
}

// Name 模块名。
func (m *Strengthen) Name() string { return "Strengthen" }

// Init 注册期（对齐官方 0xD4540）。
func (m *Strengthen) Init(k *Kernel) error {
	m.benches = make(map[PersistID]*strengthenBench)
	b := k.Bus()
	b.OnEvent("player", "OnLeaveScene", m.onChangeScene)
	b.OnEvent("player", "OnContinue", m.onChangeScene)
	b.OnCommand("player", 82, m.onCustomStrengthenClose)
	b.OnCommand("player", 83, m.onCustomStrengthenSelect)
	b.OnCommand("player", 84, m.onCustomStrengthenExecute)
	b.OnCommand("player", 85, m.onCustomStrengthenDrop)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.Critical("Equipment", "StarLevel", m.onStarLevelChanged)
	b.HeartBeat("Strengthen::HB_RefreshCustomInfo", m.hbRefreshCustomInfo)
	b.HeartBeat("Strengthen::DoContinueEnhance", m.doContinueEnhance)
	return nil
}

// LoadResource 官方 @0xD4860：ini/ItemEnhance/enhance_limit.xml。
func (m *Strengthen) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: Strengthen.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/ItemEnhance/enhance_limit.xml")
	if err != nil {
		return fmt.Errorf("fsgame: Strengthen::LoadResource enhance_limit.xml: %w", err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	m.EnhanceLimit = 1
	for id, row := range ruleByID(rows) {
		if id == "StartLevel" {
			m.EnhanceLimit = atoi32(row["Value"])
		}
	}
	k.Tracef("Strengthen: LoadResource enhance_limit StartLevel=%d", m.EnhanceLimit)
	return nil
}

// onNpcSelect 官方 @0xD5020：强化 NPC 选中（StrenthenNpc 台面登记）。
func (m *Strengthen) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {
	m.benches[from] = &strengthenBench{Npc: self}
}

// onCustomStrengthenSelect 官方 @0xD55E0（cust 83=0x53）：选装备上台
//（校验串定案：EquipType/Energy/Special/ColorLevel/LockStatus/StarLevel；
// LockStatus 锁定拒 sys_depot_0016）。
func (m *Strengthen) onCustomStrengthenSelect(k *Kernel, self, from PersistID, args VarList) {
	bench := m.benches[self]
	if bench == nil {
		return
	}
	equip := args.Object(0)
	eo := k.GetGameObj(equip)
	if eo == nil || eo.ClassName() != "Equipment" {
		m.sys("sys_item_0067")
		return
	}
	if eo.GetInt("LockStatus") != 0 {
		m.sys("sys_depot_0016")
		return
	}
	if eo.GetInt("Level") < m.EnhanceLimit {
		m.sys("sys_item_0070") // 等级门（enhance_limit StartLevel）
		return
	}
	bench.Equip = equip
	m.ops = append(m.ops, "select")
}

// onCustomStrengthenExecute 官方 @0xD5E60（cust 84=0x54）：执行强化
//（CurrentTicket/CurrentBlackgold 消耗 → StarLevel+1 →
// C_OnStarLevelChanged 属性重算联动）。
func (m *Strengthen) onCustomStrengthenExecute(k *Kernel, self, from PersistID, args VarList) {
	bench := m.benches[self]
	if bench == nil || !bench.Equip.Valid() {
		return
	}
	eo := k.GetGameObj(bench.Equip)
	if eo == nil {
		return
	}
	// 消耗（官方 CurrentTicket 金钱/CurrentBlackgold 黑金面；Go 面记账）。
	star := eo.GetInt("StarLevel")
	cost := (star + 1) * 100
	bench.Ticket += cost
	bench.Blackgold += star + 1
	_ = k.SetInt(bench.Equip, "StarLevel", star+1) // C_OnStarLevelChanged 临界
	m.ops = append(m.ops, fmt.Sprintf("exec→%d", star+1))
}

// onCustomStrengthenDrop 官方 @0xD5CC0（cust 85=0x55）：放入强化材料。
func (m *Strengthen) onCustomStrengthenDrop(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "drop")
}

// onCustomStrengthenClose 官方 @0xD5440（cust 82=0x52）：关闭强化台
//（StrenthenEquip/StrenthenNpc 清理）。
func (m *Strengthen) onCustomStrengthenClose(k *Kernel, self, from PersistID, args VarList) {
	delete(m.benches, self)
	m.ops = append(m.ops, "close")
}

// onMenuSelect 官方 @0xD5150（命令 22）。
func (m *Strengthen) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {}

// onChangeScene 官方 @0xD4E80：离场/继续（强化台清理）。
func (m *Strengthen) onChangeScene(k *Kernel, self, from PersistID, args VarList) {
	delete(m.benches, self)
}

// C_OnStarLevelChanged 官方 @0xD8B90：星级变更 → 装备属性重算
//（Datasource/TyPe 静态属性包面——ItemProCompute 协作域）。
func (m *Strengthen) onStarLevelChanged(k *Kernel, self PersistID, prop string, old Var) {
	k.Tracef("Strengthen: StarLevel %v→%v", old, k.GetGameObj(self).Get("StarLevel"))
}

// HB_RefreshCustomInfo 官方 @0xD6450。
func (m *Strengthen) hbRefreshCustomInfo(k *Kernel, self PersistID, beats int32) {}

// DoContinueEnhance 官方 @0xD6300：连续强化泵。
func (m *Strengthen) doContinueEnhance(k *Kernel, self PersistID, beats int32) {}

// BenchOf 台面状态（测试面）。
func (m *Strengthen) BenchOf(self PersistID) *strengthenBench { return m.benches[self] }

// NxStrengthen nx_strengthen Lua 面（GM 强化直达）。
func (m *Strengthen) NxStrengthen(k *Kernel, equip PersistID, star int32) {
	_ = k.SetInt(equip, "StarLevel", star)
}

func (m *Strengthen) sys(tpl string) { m.sysMsgs = append(m.sysMsgs, tpl) }

// SysMsgs 系统信息流水（测试面）。
func (m *Strengthen) SysMsgs() []string { return m.sysMsgs }

// Ops 命令流水（测试面）。
func (m *Strengthen) Ops() []string { return m.ops }

// atof64 字符串转 float64（概率面）。
func atof64(s string) float64 {
	var v float64
	fmt.Sscanf(strings.TrimSpace(s), "%g", &v)
	return v
}
