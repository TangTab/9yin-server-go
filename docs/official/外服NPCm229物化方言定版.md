# 外服 NPC m229 物化方言定版（c130 大族）

> 对象：ready_golden.bin 帧 #2-63 的 **m229「NPC 列表推送」族**（透传清单原记
> ×63 系含 #64 误计——#64 实为 m596 [2]；m229 c130 实际 **62 条**）。
> 2026-11-22 全量逐字节解码定版，native 移植落地
> （internal/entergen/npc.go + internal/npcdata + fssoloworld 接线）。
> 相关：《透传数据清单.md》m229 行、fork-era/id-system-analysis.md §6.6
> （NPC owner 段位教训）。

## 一、帧布局（16 槽 TVarList，62/62 全量命中）

`0x1E [u16 argc=16]` + 值槽（tag：2=int32 4=f32 6=窄串 7=宽串 8=obj）：

| 槽 | tag | 实证值域 | 语义（解码定案/推测） |
|---|---|---|---|
| 0 | 2 | 229 | 消息号 |
| 1 | 2 | 130 | 子命令 c130 |
| 2 | 8 | obj(0x03002091.., 0x209e..0x24xx) | wire ident（obj 0x03 段 / own 0x2xxx 段，均唯一） |
| 3 | 6 | `npc\Guildqizhi01` 等 | 模型资源路径（客户端按路径装载） |
| 4-6 | 4 | x/y/z（y=高度 20..444） | 坐标 |
| 7 | 4 | **wrap 到 [0,2π)** | yaw float（62/62 == wrap(槽15 中位)） |
| 8 | 6 | `1.100,1.100,1.100` / 空 | Scale 缩放串（NpcConfig Scale 原样直传；空=1:1） |
| 9 | 2 | 301×31 / 302×31 | 层号——多资源**双发**（同资源 301+302 各一条；语义待真机定案） |
| 10 | 2 | 0..14（旗帜 12/特效 6） | 组号小枚举（语义待定案） |
| 11 | 2 | 0/1..8 | aux（多 0） |
| 12 | 2 | 0（3 条=5） | aux（多 0） |
| 13 | 7 | 空 / `Yaochi` | 宽串显示名（62 条中物件多空、13 条旗帜 "Yaochi" 瑶池） |
| 14 | 6 | 空 / `7.png#7.png#255,17,130,178` | 旗帜公会图标串 |
| 15 | 6 | `0.0,3.141,0.0` 等 | 三轴欧拉串（pitch 可非零：2 条 0.698） |

- **62 条构成**：26×`npc\Guildqizhi01`（公会旗帜）+ 9×`npc\itemnpc057`（复活特效）
  + 27×`npc\1701_*`（演武堂/聚义堂/木人桩/牌匾/仓库等建筑）——**全为物件类，
  无人物型 NPC、无战斗怪**。人物型属本仓方言扩展（用户拍板一步到位）。
- **朝向 wrap 双表达**：槽 15 中位是**原始域** yaw（可 >2π/负值，如 9.424/-1.571），
  槽 7 float 是 wrap 到 [0,2π) 的值。f32 wrap 丢精度**推不回原始**（9.4244 wrap 后
  +2π ≠ 9.424）——编码侧两槽独立承载（`NpcEntry.Yaw`/`OrientText`）。
- **十进制格式**（槽 15/Scale）：`%.3f` 四舍五入 → 去尾零 → 整数化保 `.0`
  （"3.141"/"1.571"/"0.0"）；Scale 串从 NpcConfig 原样直传（"1.100" 定长形）。

## 二、相位与 ident

- **相位**：ready 段最前（ClientReady 应答 #2-63 连发，先于 m596/m416/0x1F）。
  native 编排：`ReadyFrames` 开头（internal/entergen/sequence.go）。
- **ident 段位**：obj=0x03 段、own=0x2xxx 段（外服动态分配观察值）。native 公式：
  `NpcObjectID(idx)=0x03000000|idx`、`NpcOwnerID(idx)=0x2000+idx`（场景内 1 起
  确定性、跨进场稳定——点击路由前提）。避开 owner=1（外服客户端解析为
  「1 号玩家化身」→ PvP 分支坑）与玩家 0x11 段/0x7A5C0B。
- **enter 段变体帧**（enter_golden #105-107）：`m229 [229,144,0]`（argc 3）、
  `[229,145]`、`[229,154]`（argc 2）——语义未定案，**本期不发**。

## 三、native 数据源与筛选（internal/npcdata + fssoloworld/npc.go）

- 摆放：`testdata/ini/Scene/born03_yanyuzhuang/npc_creator.xml`（5401 行；静态项
  1907：commonnpc 1448/funcnpc 229/lightnpc 146/gathernpc 61/itemnpc 21/
  guildbuildingnpc 1/other 1；attacknpc 族与 *_creator 随机组不发）。
- 模板联查：`ini/Npc/NpcConfig` 树（27MB/76 档）id→Resource/Name/Scale/NpcType，
  born03 命中 1759/1907（92%；未命中≈lightnpc 火盆族 NpcConfig 无模板，跳过）。
- 筛选：**全量渲染（2026-11-22 拍板「先全部渲染，不分距离」）**——缺省半径/
  上限均不限，仅跳过模板联查未命中项（born03 全量 = 联查命中 1759 条）。
  需收窄时 `NINEYIN_NPC_RADIUS`（如 500）/ `NINEYIN_NPC_LIMIT`（如 200 ≤
  MaxVisuals 230）设回；全量对客户端视觉上限（idx1468 MaxVisuals=230）的
  实际行为待真机观察。
- 槽值策略：Layer=302（众数，不双发）、Group/Aux=0、物件名空/人物填模板 Name
  （**行为待真机验证**）、Icon 空。
- 数据根（两级语义，2026-11-22 拍板）：`NINEYIN_DATA_ROOT` > cwd 探测
  `<dir>/ini/Npc/NpcConfig`——**正式 `$root\Res`**（`Res`/`../Res`/`../../Res`，
  官方服务端 ini 布局，与 testdata/ini 同构；bin\ 或部署根起服均可命中）优先；
  **测试兜底仓内 testdata**（`testdata`/`../testdata`/`../../testdata`，Go 测试
  internal/<pkg>、build\ 直跑、源码树 cmd）。真档 27MB 不随二进制部署
  （deploy.bat 不发 ini）——正式环境在部署根放 `Res\ini`（npcgen 产出或官方
  ini），缺席时 NPC 静默不下发。
- 总开关：`NINEYIN_NPC_PUSH=off` 一键关停（方言演进期保护）。

## 四、事故记录：reactor 队满静默丢帧（2026-11-22 修复）

NPC 200 帧接入后 fsunite e2e 首次暴露**既有传输层缺陷**：

- 现象：客户端仅收到 ~23% 帧（进场段 0x11×1552 → ~286，稀疏随机丢）；
  world 侧 1999 帧全部发出、entry ClientSend 零失败、转义/写锁全查无恙。
- 根因：`reactor.Send` 的 out 队列 256 槽满即**静默拒发**（`default` 分支），
  且 world 的 `reactor.New()` 无 Logf——「Send queue full」**从不打印**。
  native 进场 1663+329 帧突发远超 256；此前 replay 模式 272+85=357 帧
  **恰好未打满**，掩盖至今。
- 修复：队满改**阻塞背压**（TCP 流控天然限速；`MaxOutBufferBytes` 4MB 字节
  硬限仍护内存）；rconn 加 `drain` 通道（writeLoop 退出即 close）解除阻塞
  防死锁。回归测试 `TestSendBurstNoDrop`（2000 帧突发慢消费不丢）。

## 五、待真机验证项

（2026-11-22 上午已验——见 §六，m229 在本客户端不渲染，以下仅存档）

1. 物件+人物 200 条混合下发：渲染效果（旗帜/火盆/建筑/村民站立）、Layer=302
   单发是否够（golden 双发 301+302 之谜）、Group/Aux=0 是否影响交互态。
2. 人物 Name 宽串（golden 无先例）：模板名 vs 空串。
3. 槽 4-6 轴序（x/y/z 直传 npc_creator 坐标，外服 y 观察值 444 与本仓 0..44
   的地图基准差异未解——渲染位置偏差时先查这里）。
4. 点击 NPC（C2S 0x07 obj@21 回传 0x03 段 ident）→ 本期未接路由，客户端
   应无响应/静默，不应崩溃。

## 六、真机判决与 0x0D 渲染通道（2026-11-22 上午）

m229 全量 1759 条真机**零渲染**。客户端 `bin64\trace.log`（排障第一站）：
进场全绿、玩家渲染正常、**m229 零报错零痕迹**——与 2433 工作树
《外服模式NPC渲染收束.md》终局结论吻合：**fork 系客户端把 m229 当
EscortRec 记录帧静默吃掉，不是物化通道**（子码=记录表 id：144/145=
job_skill_share_rec、130=EscortRec）；m229 十六值设施族是外服家族
客户端语义。trace 另见就绪段海量 `ServerViewAdd Out range`（视图行
方言问题，独立课题）。

**NPC 渲染正解 = NPC 0x0D**（2433 线真机验证过的配方，本仓表已备
2433+61 尾追=2494 头行通道）：63B 头（obj/own/x/y/z/orient）+ 头行块
（默认层 39 项值集 + 模板层同名绑定 + 摆放 PosiX/Y/Z/Orient 四行，
按 0x09 表序）+ 官方 16-prop 块（CollideRadius 寻路避障/State=stand/
**Type=4**/Resource+ConfigID+Photo 三件套/ActionSpeed=1.0/Level/
BossLevel=1/IDCard 五空槽/BornAction/WeaponMode=hand1_1/三比例=100/
宝箱 OpenRange=2.0 采集 GatherRange=3.0）。ident：obj=0x11000000|idx、
own=0x00660000|低16位（2433 真机形态）。落码 internal/entergen/npc0d.go
（默认层直抄 VisibleNPCSpawnDefaultsCompatV1；模板层脏值如
WalkSpeed="#N/A" 单字段跳过默认层兜底）。

双通道开关：0x0D 默认开（`NINEYIN_NPC_0D=off` 关）；m229 默认关
（`NINEYIN_NPC_MARK_229=1` 开——外服家族客户端接入时对照）。
编排：0x0D×N 在 PreReady 尾（官方登录序：玩家 0x0D→记录行→NPC 0x0D×N）。

**模型资源待验**：客户端 res\obj 无 npc 散目录（资源在 PackFileSys 包/
patch .rp，玩家 b_jianghu 现代名渲染成功证明 char 库吃现代命名——
worldnpc 系大概率在包内）。0x0D 下发后若 trace 出现 FxModelAdv 装载
失败即资源缺失实锤，届时按 2433 线判别实验 `NINEYIN_NPC_RESOURCE_OVERRIDE`
换名验证。
