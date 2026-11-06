# lua64 客户端脚本资源分析（对服务端的价值）

> 源：`F:\Game\Online\9yin\九阴资源\lua64_unpacked\lua64`（2296 个 lua，28MB，64 位客户端完整脚本）。
> 配套生成物：[lua64-protocol-dict.md](lua64-protocol-dict.md)（710 个 custom ID 全字典 + 665 个下行 handler 名册），
> 提取脚本 `tools/lua64_extract.py`（可重复执行：`python -I tools/lua64_extract.py <lua64根> docs/lua64-protocol-dict.md`）。

## 一句话结论

这是迄今拿到过的**最完整的协议语义权威源**：716 个 custom 消息号、812 个客户端发送调用的精确参数类型、
665 个服务端下行命令的载荷布局，全部带函数名和触发 UI 场景。已实现的功能（心跳/格挡/背包/内功/战斗一期）
全部能在其中找到对应定义，且能直接指导下一批功能（交易/摆摊/商店回购/掉落箱/邮寄/交易行）。

## 架构级发现（影响所有功能实现）

1. **下行是"命令名寻址"，不是 ID 寻址**。服务端→客户端 custom 消息携带命令名字符串，
   引擎按名调用 `custom_handler.lua` 同名函数，签名为 `func(chander, arg_num, msg_type, ...)`，
   `msg_type` 之后的形参即载荷布局（例：`custom_update_player_pos(player,x,y,z,o)`）。
   Go/Java 侧需要一个"命令名→编码器"表，而不是数字分发表。
2. **上行是 ID 寻址 + 显式类型**：`game_visual:CustomSend(nx_int(ID), 参数...)`，
   参数类型由 `nx_int/nx_string/nx_widestr/nx_int64/nx_float/nx_object` 定型。
   习惯约定：shopid/configid/serialno/uniqueid/strpos 一律 string；金额一律 int64（单位=文）；槽位从 1 起。
3. **184 个 ID 客户端 Lua 不发送**（引擎 C++ 直发，见字典第二节），其中包括战斗主链：
   211 出招 / 212 恢复 / 215 内功 / 216 轻功 / 218 格挡 / 914 招式落点 / 924 心跳 / 260-261 阵法。
   这些的收包格式只能靠抓包（现有 F 盘抓包链路）+ exe 逆向，lua64 提供**语义**而非字节格式。
4. **物品移动主通道不在 custom 里**：拖拽/穿脱装备/拆堆/右键使用全走引擎 GameHand→GoodsGrid 的
   view-move 协议（`form_rp_arm.lua:1170` 无任何 CustomSend）。custom 只承载有业务语义的操作。
5. **服务端→客户端四类通道**：(a) View/ViewObj 增删改（additem/delitem/updateitemprop/updateview，
   回调 optype 全枚举见 `form_bag.lua:437-494`）；(b) 角色属性（CapitalType0-4、LogicState、HP 等）；
   (c) 玩家记录表（RecvLetterRec、taolu_rec、cooldown_rec、buffer_rec 等）；(d) custom 命令名消息。

## 对已实现功能的印证与修正

| 服务端现状（记忆） | lua64 证据 | 结论 |
|---|---|---|
| 0x38 心跳 seq 回显 | 924=MSG_DELAY 全 Lua 零引用，引擎 MessageDelay 模块专用；客户端不主动断线，等服务器断 | 现有实现正确；踢人 error_code 用 20116/20122/51004 可阻止自动重连，否则客户端自动重连 3 次（25s+10~15s 随机） |
| 294 时间广播 | 喂 `MessageDelay:GetServerNowTime()`，20+ UI 依赖；另有 custom `custom_asyn_time(server_time)` | 保持广播；进图时可补发 custom_asyn_time |
| 格挡 218 / 招式 914 | 均引擎直发；`SYNC_STATE_ZHAOSHI_END_STOP` 佐证 914=招式结束位置校正上报 | 语义确认；字节格式继续以抓包为准 |
| CD 双行 + GCD 50000 | `gui.lua:81-91`：cool id 段 0(100-2000)/1(2000-19999)/2(24000-29999)；**50000=GCD 基值，50001-51000=按 CoolDownTeam 分组公共 CD，其中 50001-50100 无特效**；每技能双键=CoolDownCategory+CoolDownTeam（skill_static 列 3/5） | 与已实现一致，且补全了分段细节 |
| 背包 1-18+扩展 19-90 锚点 | `form_bag.lua:740-778`：主包视口属性 **BaseCap**；扩展视口内小包条目（ItemType=31）带 **BeginPos/EndPos** 声明锚点区间，客户端拼成连续网格；小包有耐久（msg 38 修理） | 结构确认；新增 38/53/862/626/627 消息签名见字典 |
| msg 819 | 客户端定义 `ASK_OPEN_DAILY`（离线收益询问），非协议同意 | 819 兼做同意协议是参考服行为，注意区分 |
| 内功 215/1006 | 215 引擎直发；View 43=NEIGONG、46=QINGGONG | 一致 |

## 新解锁的高价值功能链（按实现成本排序）

1. **玩家交易（exchange）**：发起走 `custom_request(REQUESTTYPE_EXCHANGE=2, 目标名)`→对方 `custom_be_request`
   弹窗→`101 REQUEST_ANSWER`；双方同意后服务端建 View 30/31。锁定仲裁全在服务端：
   ExchangeIsLock/ExchangeIsOk/ExchangeMoney/ExchangeTarget 视口属性同步双方状态；322 取消**带 1 个 int 参数**；
   324 放物品**不带数量**（整叠）。交割=原子交换 View 30/31 内容+货币后销毁视口。
2. **掉落箱（View 80）**：`show_hide_drop_view(1/0)` 开关；条目属性 ConfigID/Amount/EquipType/ColorLevel；
   View 属性 DropType、TaskDropItem(`"configid,playername;..."` 只显示给自己的任务掉落)；容量 16（江湖场景 100）；
   收 380(槽位)/1008(拾银)/382(关箱)/381(任务掉落 drop_id 字符串)。
3. **商店回购（View 64）**：SELL 时服务端记录；`73 custom_buy_back(index)` 单参数即可，成本低。
4. **货币四件套收尾**：金额 int64 文为单位；银子三段显示 1铮=1000000文；银行存取 205/206(int type,int value)；
   仓库余额=View 4 属性 DepotSilver/DepotSilverCard。
5. **摆摊/离线摊**：650-672 全参数已提取（652 加货 5 参含 i64、655 开摊 widestr 摊名+时长+离线标志、658 买他人摊）。
6. **邮寄**：数据面=玩家记录 RecvLetterRec 15 列（非 View）；附件走 View 35/36；serialno 全部 string；
   附件描述 XML 格式 `<Object Config=..><Prop Amount=../></Object>`。
7. **NPC 预创建**：scene 对象记录 `prep_show_npc_rec` 8 列（configid,x,y,z,o,sx,sy,sz）进图预建模型，
   大流量场景防卡顿可直接用。

## 战斗侧关键补充（衔接战斗二期）

- **伤害数字/闪避/招架字样不在 custom 通道**——665 个 handler 无任何 damage/hit 命名函数，
  表现全在引擎 fight 协议。短期替代：属性同步 HP（血条）+ `custom_trigger_skill_effect(attacker,skillID,target...)`
  （群体特效）+ `custom_specific_sysinfo(tips_type,string_id,参数...)`（飘字）。
- **打断下行**：`stop_use_skill`（无参）→ 客户端 `fight:StopUseSkill()+StopCheckCurSkill()`。
- **buff 下行**：`on_buffer_change(type,buffer_id,owner,sender,owner_name,sender_name,end_time,category)`
  + 引擎 buffer 数据（BufferListStr 属性/buffer_rec 记录）驱动图标层数倒计时；1021=玩家主动驱散（buffer_id string）。
- **公式权威在 ini.package 不在 lua64**：`share\Skill\formula.ini + damage_calculate.ini + skill_*_varprop.ini`；
  客户端 tooltip 有完整求值器（`tips_func_skill.lua:309-487`）：varprop 的 DamageCalculate 索引→
  damage_calculate.ini r 行→formula.ini 的 Src*/Prop*/表达式（`loadstring("return "..expr)` 求值，
  Prop 从玩家属性取值，argv 支持标量与 `a;b` 二元组）。**下一步建议：解包 ini.package 移植这套求值器。**
  （父目录已有 `jiuyin-unpack.exe` + `jiuyin-keys.txt`，工具链现成。）
- **LogicState 属性是服务端权威战斗状态**：LS_NORMAL=0/LS_FIGHTING=1/LS_DIED=120 等
  （`share\logicstate_define.lua`），客户端大量逻辑以它为开关，服务端必须维护并随 0x0D/0x10 下发。
- **受击状态机**：LOGIC_ATTACK=3/HITBREAK=4/HITBACK=5/HITOUT=6/HITDOWN=7/GETUP=8/DIE=9/RELIVE=10
  （`player_state\logic_const.lua`），由服务端 fight 包驱动切换。

## 网络层（game_sock.lua）

- 分包/加密在 `FxCli.dll`（`sock:LoadEncodeDll("FxCli.dll")`），Lua 不可见——现状（按抓包复刻）不变。
- 登录版本串：`"627a&&po@&a$#%+b*^@."`（client.lua:1024）。
- 引擎级命令全集（Lua 可见）：Login/LoginByShield/LoginByMobileShield/LoginByString/LoginByPhone/LoginByQR/
  LoginReconnect(account,ReconKey)/ChooseRole/ChooseRoleAndScene/CreateRole/DeleteRole/ClientReady/ReviveRole/GetWorldInfo。
- 重连续传：服务端先发 `custorm_player_oncontinue(scene)`（拼写如此），客户端进图后回 **979** 请求继续推场景状态。
- 跨服切图：`custom_reconnect_server(ip,port,server_id,scene_id,cross_type,cross_server_type)`→客户端回 16(1,...) 重连。

## 后续动作建议

1. [ ] 解包 `ini.package`（工具就在同目录），移植 `tips_func_skill.lua:309-487` 的公式求值器 → 战斗二期伤害权威。
2. [ ] 把字典第一节的 string/int/i64 类型约定落进 Go/Java 的 custom 消息编码层（现有 211/54/819 已对齐可验证）。
3. [ ] 交易（View 30/31）+ 掉落箱（View 80）+ 回购（View 64）为下一批功能，参数全在字典里。
4. [ ] 命令名寻址的下行通道建 `名字→编码器` 注册表，逐功能补充（先补 `custom_asyn_time`/`stop_use_skill`/`on_buffer_change`）。
