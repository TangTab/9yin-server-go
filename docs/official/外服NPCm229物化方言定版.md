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
- 筛选：距出生点平面半径（缺省 500，`NINEYIN_NPC_RADIUS`）+ 总量上限（缺省 200
  ≤ MaxVisuals 230，`NINEYIN_NPC_LIMIT`），距离升序截断。
- 槽值策略：Layer=302（众数，不双发）、Group/Aux=0、物件名空/人物填模板 Name
  （**行为待真机验证**）、Icon 空。
- 数据根：`NINEYIN_DATA_ROOT` > cwd 探测（`.`/`..`/`../..`/`testdata`/
  `../testdata`/`../../testdata` 下的 ini/Npc/NpcConfig）。真档 27MB 不随二进制
  部署（deploy.bat 不发 ini）——**真机验证需在目标根放 testdata\ini 或设 env**。
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

1. 物件+人物 200 条混合下发：渲染效果（旗帜/火盆/建筑/村民站立）、Layer=302
   单发是否够（golden 双发 301+302 之谜）、Group/Aux=0 是否影响交互态。
2. 人物 Name 宽串（golden 无先例）：模板名 vs 空串。
3. 槽 4-6 轴序（x/y/z 直传 npc_creator 坐标，外服 y 观察值 444 与本仓 0..44
   的地图基准差异未解——渲染位置偏差时先查这里）。
4. 点击 NPC（C2S 0x07 obj@21 回传 0x03 段 ident）→ 本期未接路由，客户端
   应无响应/静默，不应崩溃。
