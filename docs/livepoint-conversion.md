# 历练→修为转化系统（LiveGroove 全链）

官方机制（用户资料 + F 盘客户端源码双重钉死）：

- **1 点历练 = 1000 修为**（毫历练单位存储，1 毫历练 = 1 修为）
- 历练槽 5 个，每槽上限 **999 点**；总历练上限 999 点
- 转化速度（修为/100秒，槽非空才计）：槽 1..5 基础速度 **10/20/50/100/200**
- 五槽全满名俊 = **380 修为/100秒**（10+20+50+100+200，无损）
- 普通玩家效率 **70%**，损失 30% 进**潜能修为**（黄金按价赎回）

## 客户端依据（form_faculty.lua:87-111）

```lua
speed = Σ(槽i有历练 ? base_i : 0)          -- 10/20/50/100/200
speed = speed + speed * LiveSpeedPower / 100 -- 卦象等加成
if not is_vip then speed = int(speed * 0.7) end
```

tooltip（tips.idres）：`一级历练：{@0}/999<br>转换速度：{@1}/100秒`；总条 `ui_faculty_turnspeed=转化速度：{@0}/100秒`。

## 属性索引（0x09 官方属性表）

| 属性 | 索引 | 语义 | 实现 |
|---|---|---|---|
| HoldFaculty | 800 | 潜能修为存量 | `PlayerProgress.holdFaculty` |
| LiveGroove_1..5 | 804..808 | 历练槽（毫历练，UI /1000） | `PlayerProgress.liveGroove[5]` |
| LiveSpeedPower | 809 | 转化速度加成 % | `PlayerProgress.liveSpeedPower` |
| Faculty | 757 | 修为存量（修炼界面"修为：xxx"） | `PlayerProgress.faculty`（原有） |

7 个新属性随 `properties()` 进 0x10 全量帧（vitalUpdate 自动携带）。

## 服务端模型（PlayerProgress）

`livepointTick(now, vip)` 100 秒整步进（`lastLivepoint` 独立基准，余数留存，与
`advanceFaculty` 分钟节拍并行；1s ticker 同拍驱动，有变化才发帧+持久化）：

1. **累积**：总历练未达上限时 +1000 毫进第一个未满槽（在线每 100 秒 +1 点历练）
2. **转化**（先累积后转化）：每槽 `full = base_i × (100+power)/100`，
   `actual = min(槽余, full)`，槽 -= actual；
   名俊 `faculty += actual`；普通档 `gain = actual×7/10`，`faculty += gain`、
   `holdFaculty += actual - gain`

修为链闭环：历练 → Faculty(757) → `advanceFaculty` 每分钟折半消耗涨内功 CurFillValue。
修为耗尽自动停修（原有官服语义不变）。

## custom 235（CLIENT_CUSTOMMSG_LIVEPOINT_INFO）

客户端 `custom_sender.custom_hold_faculty(sub, value)`（form_holdfaculty_buy.lua）；
S2C 回包与 191 同构：`1E 03 00 | 02 <235> | 02 <sub> | 02 <arg>`。

| sub | 方向 | 语义 |
|---|---|---|
| 1 | C2S | GETINFO → 服务端回 `235(1, price)`（价格默认 500） |
| 2 | C2S | BUY count → 校验 count ≤ 黄金(CapitalType0) 且 count×price ≤ HoldFaculty；扣黄金、潜能转修为；回 [CapitalType0 帧, 0x10 vital]；越界静默防刷 |
| 3 | C2S | 修为兑换（form_faculty_exchange）——受理不实现（后续） |

## GM 动作

| 动作 | 参数 | 说明 |
|---|---|---|
| add_livepoint | points 1..999 | 发 N 点历练（×1000 毫按序填槽） |
| set_hold_faculty | value ≥0 | 潜能修为直设 |
| set_live_speed_power | value 0..1000 | 转化加成 %（卦象效果等价物） |

## 开关

| 开关 | 默认 | 说明 |
|---|---|---|
| -Dnineyin.livepoint.vip | false | true=全员名俊无损；false=普通档 70/30+潜能赎回 |
| -Dnineyin.livepoint.price | 500 | 潜能赎回价（N 黄金 → N×price 潜能修为） |
| -Dnineyin.faculty.seed | 10000 | 新角色修为种子（存量角色按快照保留；原 9,999,999 无限种子废弃） |

## 持久化

`FacultyStore`（faculty.json 文件主档 + role_faculty DB 快照）增 8 字段：
`live_groove_1..5 / hold_faculty / live_speed_power / last_livepoint_utc`。
旧档缺字段宽松解析归零，`last_livepoint_utc` 哨兵（0001-01-01）回退 now 防离线结算爆走。

## 边界（一期不做）

- 黄金会话级不持久化（vip.json 待接）；团练修为 TeamFacultyValue@777；
  修为兑换 235/3；历练药/卦象对 LiveSpeedPower 的动态来源（用 GM set_live_speed_power 代替）。
- 离线补结算：重登时对 `last_livepoint_utc` 以来的整步照结（累积+转化，与
  advanceFaculty 离线分钟结算同语义）。
