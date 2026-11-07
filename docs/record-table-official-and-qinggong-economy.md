# 记录表官方化 + 轻功经济对齐（2026-11-07）

> 真机反馈：① 轻功无限跳；② 快捷栏下发数据有问题（参考 Java）。
> 权威参考：Java 参考服 `RecordSchemas/RecordMessages/QingGongHandler/Actor`
> 与 `docs/customizing-shortcut-persist.md`（G:\pojie\ai\9yin-server）。

## 一、记录表官方化（快捷栏不上显的根因）

### 根因

本服此前在 0x0A 协商自建 11 张记录表（shortcut_rec 落在自定下标 4）。真机
客户端的 C++ 原生记录消费者（CoolLogicManager、快捷栏等）按**官方表下标**
绑定——行帧字节级正确却错绑记录，表象即快捷栏数据异常。Java 参考服已于
2026-11-02 切换为整表透传官方方言（`official_record_table.bin`，447 schema，
源自 52795 抓包逐字节提取）。

### 修复

- `internal/clientdata/record_table.go`：内嵌 `official_record_table.bin`
  （与 Java 同源），`OfficialRecordTableFrame()` 逐字节透传 0x0A 帧；
  按名解析官方下标（首现语义）。
- `cmd/protocol-probe` 全部记录下标常量改官方值，init 自检防漂移：

| 记录 | 旧自协商 | 官方 | 列型变化 |
|---|---|---|---|
| Task_Accepted | 2（7 列） | **29** | 官方 28 列；前缀 7 列 `[int,int,string,int,string,string,int]`（顺序已按官方类型修正），尾列按线型补零 |
| Task_Completed | 3 | **31** | [3] 不变 |
| shortcut_rec | 4 | **93** | [3,7,7] 不变 |
| title_rec | 10（[3,3,6,1,6] 5 列） | **98** | 官方 **[3,1,4]** 3 列（u32 id / u8 佩戴 / i64 到期） |
| cooldown_rec | 5（[3,4,4,3]） | **109** | 末列 team **u16**（0xFFFF 个人行口径，低 16 位截断） |
| QingGongRec | 0 | **116** | [7] 不变 |
| ActiveQGSkillRec | 1（[7]） | **117** | 官方 **[7,1]**：行必须满宽，尾列 byte 补零（短行触发 Out range） |
| active_jingmai_rec | 9 | **162** | [7] 不变 |
| FashionSuitRec | 6 | **315** | [7,3,3,4,3,3] 不变 |
| CardRec | 7 | **322** | [3] 不变 |
| UseCardRec | 8 | **324** | [2,2] 不变 |

- `serverRecordAddCells` 改按官方 schema 编码：允许**前缀行**（缺失尾列按
  线型补零/空串，Java recordAddCells 同语义）；超宽报错；type 8 宽串/9
  object 暂无消费者，命中即显式报错。

## 二、轻功经济对齐 Java（无限跳根因）

### 根因

`internal/sim/qinggong.go` 的 `Use` 带「余量 <50 自动回满」本地测试规则
（150 条打个七折就回满）+ 被动 10 点/秒回复——条永不见底，真机表象即
无限跳。Java 已于 2026-11-03 删除同款规则（官方回复是分档慢回复）。

### 修复（Java Actor/QingGongHandler/PlayerActor 同口径）

- `Use`：纯扣费不回满；`Drain`：租约秒耗、付不起清零 false；
  `RecoverTiered`：官方分档回复一步——101..上限 +10 / 51..100 +5 /
  0..50 +3，封顶 max+add。
- `playerActor`：持续轻功租约（UseType=1）三态 + 3s 回复累积器 +
  即时型 500ms 同 id 重报闸；复活时租约/累积器一并清。
- `handleQingGongCustom`（216）：
  - flag=0 end：同 id 租约摘除，不再扣费（双扣=一次按放双倍耗）；
  - flag=1 begin：①持续型同 id 续约抑制（按住冲刺引擎逐帧重发 begin，
    逐条扣费会秒清轻功条）；②即时型 500ms 重报闸；
  - begin 扣 qgdefine Consume 后，UseType=1 挂秒耗租约（异 id 直接顶替，
    踏波行→上天梯不叠加双份秒耗）。
- 秒耗表 `qingGongDrainPerSecond`（= Java buff 表 |CCQGP| 实证值内联）：
  buf_qg_light（qinggong_9 缓降）10/s、buf_qg_drift（踏波行）7/s、
  buf_qg_climb（壁虎游墙/上天梯）7/s、其余 0（租约仍挂，保续约/暂停语义）。
- 1s ticker（facultyTicker 同拍）：租约期每秒扣点且回复暂停；点变化推
  `QingGongPoint@692` 单属性 0x10 帧；耗尽强摘租约。

### 已知未做（战斗一期 backlog，docs/combat-traffic-compare.md 同源）

- qinggong_9 缓降的 msg169 timedBuff 图标链（add/remove + 槽帧）未移植：
  经济口径已对齐（10 开局 + 10/s），仅客户端缓降图标不显示。

## 三、测试

- `internal/sim/sim_test.go`：删除回满用例，新增不回满/分档三档/租约耗尽。
- `cmd/protocol-probe/qinggong_economy_test.go`（新增 6 例）：即时 500ms 闸
  只扣一次、持续租约 begin/秒耗/续约/end、耗尽强摘+回复恢复、点数不足拒、
  单属性帧形、秒耗表。
- `qinggong_records_test.go`：官方 447 表透传 + 关键下标/列型断言。
- `title_system_test.go`/`skill_combat_test.go`/`jingmai_progress_test.go`：
  官方下标与新列型断言。
