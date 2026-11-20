package fsgame

import (
	"bytes"
	"fmt"
)

// bytesReader 小工具（kernel.LoadStruct 入参用）。
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

// Module 复刻 ILogicModule 五虚槽的 Go 面（dtor 由 GC 承担）：
// Init（注册期）→ BeforeLaunch → LoadResource（静态配置）→ 运行 → Shut。
type Module interface {
	// Name 模块名（IModuleCreator::Name，官方返回类名字符串）。
	Name() string
	// Init 注册期（官方 ?Init@Xxx@@UAE_NPAVIKernel@@@Z）。
	Init(k *Kernel) error
	// BeforeLaunch 预启动（官方默认实现恒真，可覆盖）。
	BeforeLaunch(k *Kernel) error
	// LoadResource 静态配置装载（ini/*.xml 经 PreloadConfig）。
	LoadResource(k *Kernel) error
	// Shut 关停。
	Shut(k *Kernel) error
}

// BaseModule 默认实现（官方 ILogicModule 默认槽：BeforeLaunch/LoadResource 恒真）。
type BaseModule struct{}

// BeforeLaunch 默认恒真（官方 0x3A3590 默认实现）。
func (BaseModule) BeforeLaunch(*Kernel) error { return nil }

// LoadResource 默认恒真（官方 0x3A3600 默认实现）。
func (BaseModule) LoadResource(*Kernel) error { return nil }

// Shut 默认恒真。
func (BaseModule) Shut(*Kernel) error { return nil }

// Creator 复刻 IModuleCreator：Name/Create/Destroy。
type Creator struct {
	NameStr string
	Create  func() Module
}

// Registry 模块注册链（官方 g_pModuleCreator 头插链；Go 面按序登记，
// Assemble 以登记序执行 Init——即官方「构造逆序 = Init 序」的等价表达：
// 玩法表 §8 的 Init 序即本表推荐登记序：LogicClass → StaticDataQuery → …）。
type Registry struct {
	creators []Creator
}

// Register 登记创建器（重复名报错——官方链内同名模块互斥）。
func (r *Registry) Register(c Creator) error {
	for _, e := range r.creators {
		if e.NameStr == c.NameStr {
			return fmt.Errorf("fsgame: 模块 %s 重复注册", c.NameStr)
		}
	}
	r.creators = append(r.creators, c)
	return nil
}

// Assemble 全量装配：创建 → Init → BeforeLaunch → LoadResource → Shut 回调登记。
// 返回内核（模块表已注入，GetLogicModule 可用）。
func (r *Registry) Assemble(k *Kernel) ([]Module, error) {
	var mods []Module
	for _, c := range r.creators {
		m := c.Create()
		if err := m.Init(k); err != nil {
			return mods, fmt.Errorf("fsgame: %s.Init: %w", c.NameStr, err)
		}
		k.modules[c.NameStr] = m
		mods = append(mods, m)
	}
	for _, m := range mods {
		if err := m.BeforeLaunch(k); err != nil {
			return mods, fmt.Errorf("fsgame: %s.BeforeLaunch: %w", m.Name(), err)
		}
	}
	for _, m := range mods {
		if err := m.LoadResource(k); err != nil {
			return mods, fmt.Errorf("fsgame: %s.LoadResource: %w", m.Name(), err)
		}
	}
	return mods, nil
}

// ShutAll 逆序关停。
func ShutAll(mods []Module, k *Kernel) {
	for i := len(mods) - 1; i >= 0; i-- {
		_ = mods[i].Shut(k)
	}
}

// RegisterFirstBatch 登记第一批（登录可玩最小闭环）十模块。
// 顺序对齐官方 Init 序（玩法表 §8：容器/物品先于 PlayerBase，
// PlayerBase(94) → CreateRole(133)/Motion(134) → Level … Request(180)）。
func RegisterFirstBatch(r *Registry) error {
	seq := []Creator{
		{"ContainerBaseModule", func() Module { return &ContainerBaseModule{} }},
		{"ItemBaseModule", func() Module { return &ItemBaseModule{} }},
		{"CapitalModule", func() Module { return &CapitalModule{} }},
		{"SceneBaseModule", func() Module { return &SceneBaseModule{} }},
		{"LocationModule", func() Module { return &LocationModule{} }},
		{"PlayerBaseModule", func() Module { return &PlayerBaseModule{} }},
		{"CreateRoleModule", func() Module { return &CreateRoleModule{} }},
		{"MotionModule", func() Module { return &MotionModule{} }},
		{"LevelModule", func() Module { return &LevelModule{} }},
		{"RequestModule", func() Module { return &RequestModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterSecondBatch 登记第二批（社交底座）：Gmcc/Post/Team/Friend/Chat/Log
// 六模块（docs/official/FsGame全解与Go复刻.md §7 后续批次定案）。
// 顺序对齐官方 Init 序（玩法表 §8）：Gmcc(13) → Post(178) → Team(182) →
// Friend(183) → Chat(184) → Log(240)。
func RegisterSecondBatch(r *Registry) error {
	seq := []Creator{
		{"GmccModule", func() Module { return &GmccModule{} }},
		{"PostModule", func() Module { return &PostModule{} }},
		{"TeamModule", func() Module { return &TeamModule{} }},
		{"FriendModule", func() Module { return &FriendModule{} }},
		{"ChatModule", func() Module { return &ChatModule{} }},
		{"LogModule", func() Module { return &LogModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterThirdBatch 登记第三批（战斗域整族）：DamageCalculate/Fight/
// CoolDown/Buffer/Skill 五模块（§10 建议批次）。
// 顺序对齐官方 Init 序（玩法表 §8）：DamageCalculate(57) → Fight(63) →
// CoolDown(64) → Buffer(65) → Skill(68)。
func RegisterThirdBatch(r *Registry) error {
	seq := []Creator{
		{"DamageCalculateModule", func() Module { return &DamageCalculateModule{} }},
		{"FightModule", func() Module { return &FightModule{} }},
		{"CoolDownModule", func() Module { return &CoolDownModule{} }},
		{"BufferModule", func() Module { return &BufferModule{} }},
		{"SkillModule", func() Module { return &SkillModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterFourthBatch 登记第四批（任务域）：QuestCriteria/Quest/Achievement
// 三模块（引擎先行——QuestCriteria 依赖 QuestModule 协作，官方同域相邻装载）。
// 顺序对齐官方 Init 序（玩法表 §8）：Quest(79) → QuestCriteria(80) →
// Achievement(81)。
func RegisterFourthBatch(r *Registry) error {
	seq := []Creator{
		{"QuestModule", func() Module { return &QuestModule{} }},
		{"QuestCriteriaModule", func() Module { return &QuestCriteriaModule{} }},
		{"AchievementModule", func() Module { return &AchievementModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterFifthBatch 登记第五批（物品玩法域核心）：Equipment/Strengthen/
// ToolBox/PickUp/Drop 五模块。
// 顺序对齐官方 Init 序（玩法表 §8）：Equipment(85) → Strengthen(91) →
// ToolBox(147) → PickUp(208) → Drop(214)。
func RegisterFifthBatch(r *Registry) error {
	seq := []Creator{
		{"EquipmentModule", func() Module { return &EquipmentModule{} }},
		{"Strengthen", func() Module { return &Strengthen{} }},
		{"ToolBoxModule", func() Module { return &ToolBoxModule{} }},
		{"PickUpModule", func() Module { return &PickUpModule{} }},
		{"DropModule", func() Module { return &DropModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterSixthBatch 登记第六批（经济域）：Shop/Gift/Vip/IncomeReduce/
// AuctionHouse 五模块。
// 顺序对齐官方 Init 序（玩法表 §8）：AuctionHouse(71) → Shop(73) →
// IncomeReduce(118) → Vip(123) → Gift(130)。
func RegisterSixthBatch(r *Registry) error {
	seq := []Creator{
		{"AuctionHouseModule", func() Module { return &AuctionHouseModule{} }},
		{"ShopModule", func() Module { return &ShopModule{} }},
		{"IncomeReduceModule", func() Module { return &IncomeReduceModule{} }},
		{"VipModule", func() Module { return &VipModule{} }},
		{"GiftModule", func() Module { return &GiftModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterSeventhBatch 登记第七批（场景域核心）：WarFog/MapMag/AreaObj/
// Map/CloneScene 五模块。
// 顺序对齐官方 Init 序（玩法表 §8）：WarFog(12) → Map(111) → AreaObj(112)
// → MapMag(113) → CloneScene(115)。
func RegisterSeventhBatch(r *Registry) error {
	seq := []Creator{
		{"WarFog", func() Module { return &WarFog{} }},
		{"MapModule", func() Module { return &MapModule{} }},
		{"AreaObjModule", func() Module { return &AreaObjModule{} }},
		{"MapMagModule", func() Module { return &MapMagModule{} }},
		{"CloneSceneModule", func() Module { return &CloneSceneModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterEighthBatch 登记第八批（NPC 载体域底座）：GivingNpc/FunctionNpc/
// SummonNpc/CommonNpc/VisibleNpc/NpcResource 六模块。
// 顺序对齐官方 Init 序（玩法表 §8）：GivingNpc(163) → FunctionNpc(166) →
// SummonNpc(167) → CommonNpc(169) → VisibleNpc(171) → NpcResource(225)。
func RegisterEighthBatch(r *Registry) error {
	seq := []Creator{
		{"GivingNpcModule", func() Module { return &GivingNpcModule{} }},
		{"FunctionNpcModule", func() Module { return &FunctionNpcModule{} }},
		{"SummonNpcModule", func() Module { return &SummonNpcModule{} }},
		{"CommonNpcModule", func() Module { return &CommonNpcModule{} }},
		{"VisibleNpcModule", func() Module { return &VisibleNpcModule{} }},
		{"NpcResourceModule", func() Module { return &NpcResourceModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterNinthBatch 登记第九批（事件系统核心）：EventSpring/
// TriggerProcesser/Reward/Switch/Task 五模块。
// 顺序对齐官方 Init 序（玩法表 §8）：EventSpring(109) → TaskExecutor(104)
// → SwitchExecutor(105) → RewardExecutor(106) → TriggerProcesser(102)
// ——按号序：TriggerProcesser(102) → TaskExecutor(104) → SwitchExecutor(105)
// → RewardExecutor(106) → EventSpring(109)。
func RegisterNinthBatch(r *Registry) error {
	seq := []Creator{
		{"TriggerProcesserModule", func() Module { return &TriggerProcesserModule{} }},
		{"TaskExecutor", func() Module { return &TaskExecutor{} }},
		{"SwitchExecutor", func() Module { return &SwitchExecutor{} }},
		{"RewardExecutor", func() Module { return &RewardExecutor{} }},
		{"EventSpringModule", func() Module { return &EventSpringModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterTenthBatch 登记第十批（AI·世界模拟底座）九模块。
// 顺序对齐官方 Init 序（玩法表 §8）：AICharacter(39) → AISkillControl(40)
// → AIOnCommandMotion(41) → WarnModule(42) → HateModule(43) →
// AIChatWithOther(44) → AIMotionModule(45) → AIModule(46) → TempStopAI(48)。
func RegisterTenthBatch(r *Registry) error {
	seq := []Creator{
		{"AICharacterModule", func() Module { return &AICharacterModule{} }},
		{"AISkillControlModule", func() Module { return &AISkillControlModule{} }},
		{"AIOnCommandMotion", func() Module { return &AIOnCommandMotion{} }},
		{"WarnModule", func() Module { return &WarnModule{} }},
		{"HateModule", func() Module { return &HateModule{} }},
		{"AIChatWithOther", func() Module { return &AIChatWithOther{} }},
		{"AIMotionModule", func() Module { return &AIMotionModule{} }},
		{"AIModule", func() Module { return &AIModule{} }},
		{"TempStopAI", func() Module { return &TempStopAI{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterEleventhBatch 登记第十一批（玩家基础余量）十四模块。
// 顺序对齐官方 Init 序（玩法表 §8）：Energy(52) → RevertPlayer(122) →
// SafeMode(129) → RoleExterior(132) → PlayerChange(135) → Restore(137) →
// Relive(141) → Dead(143) → Freshman(144) → ObjPropRefresh(192) →
// OffLineCompensate(195) → SetPath(206) → PathFinding(212) → DoSelect(215)。
func RegisterEleventhBatch(r *Registry) error {
	seq := []Creator{
		{"EnergyModule", func() Module { return &EnergyModule{} }},
		{"RevertPlayerModule", func() Module { return &RevertPlayerModule{} }},
		{"SafeModeModule", func() Module { return &SafeModeModule{} }},
		{"RoleExteriorModule", func() Module { return &RoleExteriorModule{} }},
		{"PlayerChangeModule", func() Module { return &PlayerChangeModule{} }},
		{"RestoreModule", func() Module { return &RestoreModule{} }},
		{"ReliveModule", func() Module { return &ReliveModule{} }},
		{"DeadModule", func() Module { return &DeadModule{} }},
		{"FreshmanModule", func() Module { return &FreshmanModule{} }},
		{"ObjPropRefreshModule", func() Module { return &ObjPropRefreshModule{} }},
		{"OffLineCompensateModule", func() Module { return &OffLineCompensateModule{} }},
		{"SetPathModule", func() Module { return &SetPathModule{} }},
		{"PathFindingModule", func() Module { return &PathFindingModule{} }},
		{"DoSelectModule", func() Module { return &DoSelectModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterTwelfthBatch 登记第十二批（物品底座余量）九模块。
// 顺序对齐官方 Init 序（玩法表 §8）：Charge(70) → ItemNotify(86) →
// ItemProCompute(95) → ItemLimited(96) → ItemBind(97) → ItemValidTime(100)
// → ItemControl(197) → ModifyPack(230) → AddContainer(237)。
func RegisterTwelfthBatch(r *Registry) error {
	seq := []Creator{
		{"ChargeModule", func() Module { return &ChargeModule{} }},
		{"ItemNotify", func() Module { return &ItemNotify{} }},
		{"ItemProCompute", func() Module { return &ItemProCompute{} }},
		{"ItemLimited", func() Module { return &ItemLimited{} }},
		{"ItemBindModule", func() Module { return &ItemBindModule{} }},
		{"ItemValidTimeModule", func() Module { return &ItemValidTimeModule{} }},
		{"ItemControlModule", func() Module { return &ItemControlModule{} }},
		{"ModifyPackModule", func() Module { return &ModifyPackModule{} }},
		{"AddContainerModule", func() Module { return &AddContainerModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterThirteenthBatch 登记第十三批（装备与物品玩法余量）十九模块。
// 顺序对齐官方 Init 序（玩法表 §8）：CInscription(51) → UpgradeRule(54) →
// Fashion(83) → ItemLave(84) → ToolItemFunc(87) → ItemTreasure(88) →
// ItemAppend(89) → InheritEnhance(90) → SlotItem(92) → EnchaseCompound(93)
// → CompoundItem(94) → DropCapitalItem(98) → ToolItem(99) → ArmorChange(126)
// → TransformJob(136) → Depot(145) → ShortCut(146) → FightBag(196) →
// Enchase(198)。
func RegisterThirteenthBatch(r *Registry) error {
	seq := []Creator{
		{"CInscriptionModule", func() Module { return &CInscriptionModule{} }},
		{"UpgradeRuleModule", func() Module { return &UpgradeRuleModule{} }},
		{"FashionModule", func() Module { return &FashionModule{} }},
		{"ItemLaveModule", func() Module { return &ItemLaveModule{} }},
		{"ToolItemFunc", func() Module { return &ToolItemFunc{} }},
		{"ItemTreasure", func() Module { return &ItemTreasure{} }},
		{"ItemAppendModule", func() Module { return &ItemAppendModule{} }},
		{"InheritEnhance", func() Module { return &InheritEnhance{} }},
		{"SlotItem", func() Module { return &SlotItem{} }},
		{"EnchaseCompound", func() Module { return &EnchaseCompound{} }},
		{"CompoundItem", func() Module { return &CompoundItem{} }},
		{"DropCapitalItemModule", func() Module { return &DropCapitalItemModule{} }},
		{"ToolItemModule", func() Module { return &ToolItemModule{} }},
		{"ArmorChangeModule", func() Module { return &ArmorChangeModule{} }},
		{"TransformJobModule", func() Module { return &TransformJobModule{} }},
		{"DepotModule", func() Module { return &DepotModule{} }},
		{"ShortCutModule", func() Module { return &ShortCutModule{} }},
		{"FightBag", func() Module { return &FightBag{} }},
		{"Enchase", func() Module { return &Enchase{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RegisterFourteenthBatch 登记第十四批（世界规则与状态）十四模块。
// 顺序对齐官方 Init 序（玩法表 §8）：HonorValue(119) → Airport(120) →
// Punisher(127) → UnderWater(138) → Camp(139) → Repute(140) → Title(142) →
// WorldRank(174) → Exile(193) → News(199) → ForceState(201) → Force(202) →
// Movie(207) → Unenthrall(213)。
func RegisterFourteenthBatch(r *Registry) error {
	seq := []Creator{
		{"HonorValueModule", func() Module { return &HonorValueModule{} }},
		{"AirportModule", func() Module { return &AirportModule{} }},
		{"PunisherModule", func() Module { return &PunisherModule{} }},
		{"UnderWaterModule", func() Module { return &UnderWaterModule{} }},
		{"CampModule", func() Module { return &CampModule{} }},
		{"ReputeModule", func() Module { return &ReputeModule{} }},
		{"TitleModule", func() Module { return &TitleModule{} }},
		{"WorldRankModule", func() Module { return &WorldRankModule{} }},
		{"ExileModule", func() Module { return &ExileModule{} }},
		{"NewsModule", func() Module { return &NewsModule{} }},
		{"ForceStateModule", func() Module { return &ForceStateModule{} }},
		{"ForceModule", func() Module { return &ForceModule{} }},
		{"MovieModule", func() Module { return &MovieModule{} }},
		{"UnenthrallModule", func() Module { return &UnenthrallModule{} }},
	}
	for _, c := range seq {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}
