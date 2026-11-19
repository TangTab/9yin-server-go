# -*- coding: utf-8 -*-
"""生成 docs/official/FsGame全解与Go复刻.md —— 框架层全解 + 241 模块玩法表。"""
import json, collections

idx = json.load(open("modules_index.json"))
regs = json.load(open("regs_calls.json"))
cmd_map = json.load(open("cmd_map.json"))

# ---- 分类: 域 -> (tier, [classes]) ----
# tier: 框架/基础 = 先复刻面；其余 = 玩法
DOMAIN = [
 ("框架·模块系统与配置", "框架", ["LogicClassModule","StaticDataQueryModule","StaticInstanceModule","SystemConfigModule","LuaExtModule","AsynCtrlModule","FlowModule","DelayAction","RequestModule","NotifyModule","OpenPromptModule","MsgBoxModule","ChannelModule","VirtualRecModule","SwitchManager","PubModule"]),
 ("框架·运维与后台", "框架", ["GMModule","LogModule","BackStageModule","ManageModule","DumpRoleModule","TestRobotModule","GmccModule","WebExMsg","WebExLock","WebExchange","WebExBox","SceneObjectSeedEditor","WordCheckModule"]),
 ("玩家基础", "基础", ["PlayerBaseModule","CreateRoleModule","LevelModule","LocationModule","MotionModule","SetPathModule","PathFindingModule","DeadModule","ReliveModule","RestoreModule","RevertPlayerModule","PlayerChangeModule","RoleExteriorModule","SafeModeModule","FreshmanModule","OffLineCompensateModule","ObjPropRefreshModule","DoSelectModule","EnergyModule"]),
 ("场景基础", "基础", ["SceneBaseModule","MapModule","MapMagModule","AreaObjModule","CloneSceneModule","WarFog","NpcResourceModule","VisibleNpcModule","CommonNpcModule","CreateMonsterModule"]),
 ("物品底座", "基础", ["ContainerBaseModule","AddContainerModule","ModifyPackModule","ItemBaseModule","ItemControlModule","ItemValidTimeModule","ItemBindModule","ItemLimited","ItemProCompute","ItemNotify","CapitalModule","ChargeModule","CoolDownModule"]),
 ("AI·世界模拟底座", "基础", ["AIModule","AIMotionModule","AICharacterModule","AISkillControlModule","AIOnCommandMotion","TempStopAI","HateModule","WarnModule","AIChatWithOther"]),
 ("事件系统底座", "基础", ["EventSpringModule","ConditionModule","ConditionExecutor","BehaviorExecutor","RewardExecutor","SwitchExecutor","TaskExecutor","NoticeExecutor","TriggerBaseModule","TriggerProcesserModule","TriggerDataModule","TriggerItemEvent","TriggerBufferEvent","TriggerAreaEvent","TriggerNpcEvent","TriggerSceneEvent","TriggerPlayerEvent","ProcessorComparisionBehavior","ProcessorConditionBehavior","ProcessorObjectBehavior","ProcessorPropertyBehavior","ProcessorRecordBehavior","ProcessorAIBehavior","ProcessorCommonBehavior","ProcessorCloneSceneBehavior"]),
 ("战斗", "玩法", ["SkillModule","SkillUpgradeModule","SkillUseLimitModule","SkillBreakManageModule","SkillEffectNpcModule","BufferModule","FightModule","FightRelationModule","FightStats","DamageCalculateModule","CSkillDataQuery","SkillEventManager","PKModule","CSideStepModule","MultiSpurt","Floating","QteModule"]),
 ("养成·装备与物品玩法", "玩法", ["EquipmentModule","FashionModule","ArmorChangeModule","TransformJobModule","UpgradeRuleModule","CInscriptionModule","Enchase","EnchaseCompound","SlotItem","Strengthen","InheritEnhance","ItemAppendModule","ItemTreasure","CompoundItem","ToolItemModule","ToolItemFunc","ToolBoxModule","ItemLaveModule","DropCapitalItemModule","DropModule","PickUpModule","FightBag","ShortCutModule","DepotModule"]),
 ("养成·职业与生活", "玩法", ["JobFormulaModule","JobPropModule","LifeJobModule","FormulaResearch"]),
 ("养成·宠物与骑乘", "玩法", ["PetLogicModule","PetMagician","PetOrcSoulSpecial","PetOnSkillBehavior","PetOnItemBehavior","CPetMacStrengthen","RideModule","NormalRideModule","MultiRideModule"]),
 ("社交", "玩法", ["ChatModule","ChatRoomModule","FriendModule","TeamModule","SpecialTeamModule","ExchangeModule","PostModule","LetterModule","LookModule","GuildModule","GuildStorageModule","MenuModule","OptionTalkModule","TriggerTalkModule"]),
 ("任务与成就", "玩法", ["QuestModule","QuestCriteriaModule","AchievementModule","AwardModule","GuideModule","EverydayGuideModule","MysticModule","CollectModule"]),
 ("活动与竞技", "玩法", ["ActivityModule","CampaignStatusModule","ChallengeModule","ArenaModule","ArenaRankModule","ArenaSceneModule","CBattleWarModule","CBattleWarStatisticsModule","BattleWarSceneModule","CBattleWarGamePlayModule","PVPTaskModule","PVPTaskQuery","PVPActivityModule","ScapegoatModule","BladeRunnerModule" if False else "CBladeRunnerModule","GunManModule","StageModule","GroupSceneModule","FieldCloneModule"]),
 ("经济", "玩法", ["ShopModule","AuctionHouseModule","GiftModule","VipModule","IncomeReduceModule"]),
 ("能量井（帮会资源战）", "玩法", ["EnergyWellModule","EnergyFillModule","EWCarrierProduceModule","EnergyWellModule","EnergyFillModule","EWCarrierProduceModule","EnergyWellGuildWar","EnergyWellGuildWarScene","BattleCarrierModule"]),
 ("NPC 玩法载体", "玩法", ["HiddenNpcModule","FormationNpcModule","ConvoyNpcModule","SummonNpcModule","FunctionNpcModule","BannerNpcModule","FlyNpcModule","GivingNpcModule","MultiSkillNpcModule","OrganNpcModule","SwitchDoorModule","SkillNpcModule","TechnologyNpcModule","NpcChangeModule","NpcSwitchLocateModule","MonsterAwardsModule","MonsterOwnershipModule"]),
 ("世界规则与状态", "玩法", ["ForceModule","ForceStateModule","CampModule","TitleModule","ReputeModule","HonorValueModule","WorldRankModule","ExileModule","UnenthrallModule","PunisherModule","UnderWaterModule","AirportModule","NewsModule","MovieModule"]),
 ("其余/未归类", "玩法", []),
]
cls2domain = {}
cls2tier = {}
for dom, tier, clss in DOMAIN:
    for c in clss:
        cls2domain[c] = dom
        cls2tier[c] = tier

bycls = {m["class"]: m for m in idx}
# Init 顺序 = 逆构造序: creators RVA 降序（GM 最先构造/最后 Init）
order = sorted(idx, key=lambda m: (-int(m["creator_ctor"], 16)))
for i, m in enumerate(order):
    m["init_seq"] = i + 1

# 分域输出
def fmt_cmds(m):
    cs = [c for c in m["cmds"] if c >= 3]
    if not cs: return ""
    if len(cs) <= 4: return " ".join("0x%x" % c for c in cs)
    return "%d个(%s…%s)" % (len(cs), "0x%x" % cs[0], "0x%x" % cs[-1])

tables = []
seen = set()
for dom, tier, clss in DOMAIN:
    rows = [m for m in order if cls2domain.get(m["class"]) == dom]
    if not rows: continue
    seen.update(m["class"] for m in rows)
    tables.append((dom, tier, rows))
un = [m for m in order if m["class"] not in seen]
if un:
    tables.append(("未归类（核对清单）", "玩法", un))

out = []
for dom, tier, rows in tables:
    out.append("#### %s（%s，%d 个）\n" % (dom, "先复刻面" if tier != "玩法" else "玩法域", len(rows)))
    out.append("| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |")
    out.append("|---|------|-----|------------|------|--------|------|------|------|-----|------------|")
    for m in rows:
        out.append("| %d | `%s` | %s | 0x%s | 0x%s | %d | %d | %d | %d | %d | %s |" % (
            m["init_seq"], m["class"], (m["obj"] or "?").replace(".obj",""),
            m["creator_ctor"], m["init"] or "-", m["init_seq"], m["syms"],
            m["ncmd"], m["nhb"], m["nlua"], fmt_cmds(m)))
    out.append("")
open("gameplay_table.md", "w", encoding="utf-8").write("\n".join(out))
print("domains:", len(tables), " rows:", sum(len(r) for _,_,r in tables))
if un: print("UN:", [m["class"] for m in un])
