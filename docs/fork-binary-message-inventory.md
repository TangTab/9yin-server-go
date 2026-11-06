# fork 二进制消息清单与功能盘点

> 对象：`F:\Game\Online\9yin\9yin_server1\build\9yin-game-native-menu.exe`
> （`github.com/local/9yin-go-server` fork，commit `fb21001f` 2026-09-14，go1.26.5，符号未剥离）
>
> 方法：`go tool nm` 提符号 + `go tool objdump` 行号级反汇编。
> msgId 映射三重交叉验证：分发器 switch 守卫 CMP / handler 入口自校验
> `values[0].i32 == msgId` / `handledActivityCustomID` 排除表。
> 复现脚本：`C:\Users\lzy\AppData\Local\Temp\fork_re\{extract_switch,handler_ids}.py`。

## 一、C2S custom 消息分发映射（0x0A 信封）

| msgId | handler | fork 源文件 | 功能 |
|---|---|---|---|
| 1 | handleNearbyChatCustom | chat_player.go | 附近聊天（自校验 `==1`） |
| 30 | handleMoveItemCustom | equip_wear.go | 移动物品/穿脱装备 |
| 31 | handleUseItemCustom | equip_wear.go | 使用物品 |
| 34 | handleDeleteItemCustom | equip_wear.go | 丢弃物品 |
| 36 | handleArrangeItemCustom | equip_wear.go | 整理背包 |
| 54 | handleBlockStateCustom | active_parry.go | 手动格挡状态 |
| 70 | handleShopBuyCustom | shop_buy.go | 商店购买 |
| 71 | handleShopSellCustom | shop_buy.go | 商店出售 |
| 107 | handleCustomizingSave | selection.go | 外观自定义保存 |
| 150 | handleJingMaiCultivateCustom | jingmai_progress.go | 经脉修炼 |
| 161 | handleJingMaiActiveCustom | jingmai_progress.go | 经脉激活 |
| 180 | handleFwzCardCustom | fwz_card.go | 风物志卡牌（子令 3/6/7/8/11/12/13） |
| 195 | handleKeyBindCustom | selection.go | 键位方案上报 |
| 200/201 | handleShortcutCustom | shortcut_records.go | 快捷键设置/删除 |
| 211 | handleSkillCustom | skill_combat.go | 使用技能 |
| 212 | handleCastDashLanding | qinggong_gj.go | 续招/冲锋落地（RESUME_SKILL） |
| 215/1006 | handlePlayerProgressCustom | player_progress.go | 内功进度/活动内功 |
| 216 | handleQingGongCustom | main.go | 使用轻功 |
| 217 | handleActiveQingGongCustom | main.go | 激活轻功 |
| 218 | handleActiveParryCustom | active_parry.go | 激活招架 |
| 240 | handleSitcrossCustom | shortcut_records.go | 打坐 |
| 243 | handleRideCustom | mount.go | 骑乘开关 |
| 244+961 | handleRideSkillCustom | mount.go | 骑乘技能（双 case） |
| 249 | handleWindHoldPowerEnd | qinggong_gj.go | 御风蓄力结束 |
| 266 | handleNewRideCustom | mount.go | 新骑乘子令（1=USE_ITEM/2=SELECT_RIDE） |
| 292 | handleChangeWarningCustom | active_parry.go | 切换预警 |
| 295 | handleSetTitleCustom | title_system.go | 设置称号 |
| 431 | handleSelectTargetCustom | selection.go | 选择目标 |
| 770 | handleJyzStepQuery | jyz_step.go | 九阴志步骤查询 |
| 825 | handleCheckSwitchQuery | switch_sync.go | 系统开关同步查询 |
| 861 | handleQuenchCastCustom | weapon_quench.go | 武器淬火 |
| 864 | handleLifeJobCustom | life_job.go | 生活职业 |
| 914 | （内联 main.go:2626） | — | 招式结束异步位置（ASY_POS_IN_ZHAOSHI_END） |
| 924 | handledActivityCustomID 门控 | main.go | 活动消息（非已知 id 走活动内功路由） |
| 933/934/935/938 | handleWeaponRecastCustom | weapon_recast.go | 重铸 preview/lock/unlock/confirm |
| 972 | handleFreshManChoice | freshman_guide.go | 新手选择 |
| 973 | handleGuideAddSchoolCustom | school.go | 新手指南-加入门派 |
| 997 | handleSchoolCustom | school.go | 门派操作 |

引擎层 C2S 帧（同在 `main.handle` 分发）：0x02 登录、0x04、0x07 对象请求
（objectRequest，33B）、0x09、0x18 ping、msg9 登录握手、msg15 移动上报、
msg298 刷新 NPC 任务特效、msg924 MSG_DELAY 心跳。

## 二、选择目标链路（selection.go:102-129 已还原）

```go
// handleSelectTargetCustom: 校验 values[0]==i32(431) && values[1].tag==8(raw16)
// packed := values[1].int64; objId := uint32(packed); owner := uint32(packed>>32)
if objId == 0 {
    scene.clearSelectedTarget()   // → writePlayerTargetLocked(0,0)
    updateSkillRangeHints(...)    // 刷新技能射程可用性
    return
}
scene.selectSceneObject(objId, owner)   // 查实体表，combat/非combat 分流
if scene.selectedCombatTargetTransform() != nil {
    updateSkillRangeHints(...)          // 有战斗目标 → 刷新射程提示
}
```

`writePlayerTargetLocked`（scene_lifecycle.go:1090-1094）回两帧 22B：
`0x10 | obj=0x11000001 | owner=0x7a5c0b | cnt=1 | idx 0x19e(414 LastObject) / 0x19f(415 AutoSelectObject) | packed(obj,owner)`
——与 49840 抓包逐字节一致。玩家对象 0x11000001、owner 0x7a5c0b。

## 三、功能盘点（fork 95 源文件，按子系统）

**本仓库已有**（不列）；下列为 fork 独有：

1. **持久化**：mysql_repository/mysql_stores（bag/currency/equip/faculty/fwz/jingmai/
   mount/qinggong/shortcut 全套 Load/Save）、role_store、account_key、
   login_credentials/identity/password（MySQL 账号体系）、json_repository（JSON 回退）。
2. **物品经济**：item_catalog、bag_grant/pack/store、drop_table（roll 掉落）、
   equip_catalog/grant/store/wear、exchange_catalog（兑换）、currency_store（银两/银票/银卡/金币）、
   shop_buy（70/71 购销）。
3. **任务**：quest_system/loader/accept_gate（接取/提交/杀怪计数/对话目标/奖励发放）。
4. **装备玩法**：weapon_quench（861 淬火：淬火池/书/预览/结果帧）、
   weapon_recast（933-938 重铸：preview/lock/unlock/confirm）。
5. **风物志**：fwz_card/catalog（收集/穿戴/套装/位掩码行/外观覆盖）。
6. **武学**：skill_book（书学习）、skill_combo（连招链）、skill_cultivation（修炼消耗表）、
   skill_replace（替换规则）、skill_use_limit、skill_desc_flags/levels、skill_proppack、
   grant_all_skills、jingmai bonuses/progress/records/store、qinggong_store、qinggong_gj（御风/冲锋/跳击）。
7. **系统**：life_job（864 生活职业：学习组/经验表/工具）、mount（243/244/266/961 坐骑：
   所有权/骑乘/合并技能）、title_system（295 称号：分类/授予/穿戴）、jyz_step（770 九阴志）、
   character_system、school（997/973 门派）、freshman_guide（972 新手）、
   chat_player（1 附近聊天）、server_announcement（公告）、switch_sync（825 开关）、
   food_use（食物 buff）、faculty_level。
8. **NPC/场景**：npc_creator、npc_talk（对话目录/greeting）、npc_template_table、
   npc_transport（传送目录）、npc_spawn_frame、npc_path、map_path、viewport（视口同步）、
   entity/actor、combat_tuning、scene、server_all_dest、snapshot_608、
   official_data + official_schema_query（官方数据预取缓存）、string_names、
   batch_connection（批量帧/限流）、recording_conn（抓包）。

**本仓库独有**（fork 没有）：capture_dump、task_catalog（任务目录装载）、
resource_store（MPQ 三层仓）、ecs/world 架构、server_moving、内功特效脚本桥。

## 四、还原优先级建议

按「客户端可感知 + 依赖链短」排序：

| P | 消息/功能 | 理由 |
|---|---|---|
| P0 | 431 选目标 + 414/415 回帧 + updateSkillRangeHints | 技能选靶前置；今天抓包已钉死格式 |
| P0 | 0x18 ping → 0x38 回复 | combat-gap-audit D11：3 秒 HUD 延迟根因 |
| P1 | 15 移动上报全量（冲锋落地续招 212） | 冲锋暂存探测到位逻辑 |
| P1 | 54 格挡 / 218 招架 / 292 预警 | active_parry 三件套 |
| P1 | 30/31/34/36 背包装备四件套 | equip_wear.go 同源，一次还原 |
| P2 | 70/71 商店 + 180 风物志 + 295 称号 | 经济/收集 |
| P2 | 150/161 经脉 + 240 打坐 + 216/217 轻功激活 | 修炼闭环 |
| P3 | 861 淬火 / 933-938 重铸 / 864 生活职业 / 972-997 门派新手 | 深度玩法 |
| P3 | MySQL 持久化层 | 需求驱动再定 |
