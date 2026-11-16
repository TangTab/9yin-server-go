# fxgamelogic 格挡门逆向：CustomSend 双路架构与 m54 发送链定位

> 源：`F:\Game\Online\9yin\九阴资源\tmp_carve\fxgamelogic_rebuilt.dll`（48MB 内存抠取件，零空洞）。
> IDA ImageBase=0x3AF0000，**文件偏移==RVA，EA=RVA+0x3AF0000**（换算勿心算，本轮曾错 0x50000 反出无关函数）。
> 逆向窗口：2026-11-17 两个 session；配套工具 `tmp/ida_xref.py`（改 TARGETS 复用）、产物 `tmp/*_disasm.txt`、
> lua API 全量名册 `tmp/gv_ent_names.txt`。
> 姊妹篇：[fxnet2-op-dispatch-table](fxnet2-op-dispatch-table.md)（op 分发器）、
> [fxgamelogic-s2c-custommsg-table](fxgamelogic-s2c-custommsg-table.txt)（S2C 722 槽 handler 表）、
> [lua64-protocol-dict](../client/lua64-protocol-dict.md)（716 custom ID 语义字典，其中 184 个引擎直发 ID 含 218 格挡）。

## 一句话结论

**CustomSend 有两条路：lua 路被 1024 项 per-id 白名单闸死（54 不在名单 → lua 永远发不出格挡），
原生路（`game_visual->vt+120`）无闸直达 Sender；而 m54 的发送器与格挡本地进入链在所有模块的静态码里
均不存在 → 位于 .vmp0/.vmp1 虚拟化区。** 恒 1 每帧上报证明发送器活着，卡死的是 VMP 里的"本地进入格挡"判定。

## §一 CustomSend 双路架构（本轮核心发现）

```
lua 路（脚本可发）:
  game_visual:CustomSend(id, ...)            [lua]
    → GameConfigEnt "CustomSend" 注册项      （注册器 sub_4666BC0）
    → sub_3B19B63 → sub_4666BA0 → sub_3B4785B → sub_465E440
        id = VarList->GetInt(0)              （首值 = custom id）
        if (id >= 1024) return;
        if ( *(this + id + 0x5149DC) == 0 ) return;   ★ 1024 项 per-id 使能表
        → 找 "Sender" 对象（root sub_3B4F970() + 名字表）→ Sender->vt+192(varlist, out)

原生路（引擎直发，无闸）:
  任意 C++ 调 game_visual->vt+120(varlist)
    → vtable @0x54B2CF8 的 +120 槽 = thunk 0x3B54FE2 → sub_466B880
        → 同样找 "Sender" → Sender->vt+192(varlist, out)   ← 无白名单检查
```

- `Sender->vt+192` 是唯一的真发送汇点；两条路只差中间那道闸。
- **白名单只约束 lua**：引擎原生发送完全不受限。这与 [lua64-protocol-dict] 的"184 个 ID 引擎直发"结论互证
  ——这批 ID 的 lua 侧不可达并非"没写 lua"，而是被白名单结构性排除。

## §二 1024 白名单：构造器与 54 的缺席

- **构造器 = sub_466ECA0**（类 ctor，0x119A 字节）：先 `memset(this+0x5149DC, 0, 1024)`，
  再对数百个 id 逐个 `*(this+0x5149DC+N) = 1` 点亮。
- 对照验证（搜各表项 disp32 即点亮指令）：

| 表项 | disp32 | 点亮指令 | 结论 |
|---|---|---|---|
| table[218]（ACTIVE_PARRY/SetParryType） | 0x514A86 | @0x466fa42 | ✓ 点亮（它能工作，自证） |
| table[292]（CHANGEWARNING/Z拔刀） | 0x514AD0 | @0x466eedb | ✓ 点亮（Z 已修通，自证） |
| **table[54]（BLOCK_STATE/m54）** | 0x514A0A | **全模块零命中** | ✗ **不在白名单** |

- 全表区（0x5149xx/0x514Axx）静态引用扫描：**白名单构造器之外无任何写者**。
  0x514994~0x514998 的命中是同对象的其他字段（计数器族），非表项；0x540f548 是指令流字节巧合。
- **推论：lua 层永远发不出 54**。`CLIENT_CUSTOMMSG_*` 定义表（`share__client_custom_define.lua`）
  中 50/52/53 连号存在而 **54 空缺**，与白名单互证 —— 54 是纯引擎通道。
- game_visual 的 lua API 全量 = **6990 个注册函数**（全局 ent 链头 off_5923ED0 的 dref 枚举，
  6460 具名，落盘 `tmp/gv_ent_names.txt`）：**零 parry/block 项、无 EnableCustomMsg 类开闸 API** →
  白名单无运行时使能路径。

## §三 m292（Z 拔刀）原生发送器 = sub_51964F0

与 lua `custom_change_warning`（QueryRoleStateIndex 闸）双轨并存：

```c
SendChangeWarning():
  v16 = game_visual->vt+96()            // GetPlayer
  v1  = scene->FindObj(v16); body = v1[7]
  if ( body+1472 == 1 || body+1472 == 2 )      // ★ 原生门 = RoleStateIndex ∈ {静态, 移动}
      if ( sub_3B38167()->vt+64(v16) ) return 0 // 前置重复检查
      VarList{292} → game_visual->vt+120       // 发送（无闸原生路）
```

- **body+1472 就是 RoleStateIndex**（lua `QueryRoleStateIndex` 的数据源）。
  该字段在 InParry 处理器（§六）里同一语义出现，双点互证。
- 原生 custom id 全量清单 = **99 个**（全模块 sub_3B43963 调用点 edx-imm 扫描）：
  218=sub_5194CB0、292=sub_51964F0 在列，**48-62 整段空缺** → fxgamelogic 内没有 m54 的静态构造点。

## §四 SetParryType（custom 218）全链

```
设置界面（form_system_interface_fightfreedom.lua）:
  parry_type: 0=手动(rbtn_manualparry, 默认) / 2=自动(rbtn_autoparry)
  fight:SetParryType(nx_int(parry_type)) → save_config_to_server
    ↓
lua 桥 glue 0x503A090（vt+0x28 判型 / vt+0x98 取 int）
    ↓
sub_5039910: VarList{2, type} → sub_3AF6992(thunk)
    ↓
sub_5194CB0: VarList{218, {2, type}} → game_visual->vt+120   ← 即 C2S custom 218
```

- `CLIENT_CUSTOMMSG_ACTIVE_PARRY = 218`（`share__client_custom_define.lua:137` 官方自证）。
- Go 侧 `handleActiveParryCustom`（protocol-probe）已受理该消息——链路对齐 ✓。
- 0x503A120 是 218 的静态注册壳；"sub_1549910 引擎真身" 旧记法有误（该地址是 VarList 打包壳，
  见 §九 勘误）。

## §五 fightparry GameConfigEnt 与 body 字段地图

fight 对象的格挡模式是双向绑定的配置实体：

| 角色 | 函数 | 行为 |
|---|---|---|
| 本地写（fightparry_A） | sub_3EADAA0 | `body+1460 = type` + body+0x38 map 挂 "fightparry" 标志 |
| S2C 推写（fightparry_B） | sub_3EC4630 | 属性变体 → body+1460（**服务端个性配置回推通道**） |
| 上报导出 | sub_3E9E2B0 | body+1460 → 属性对象（vt+144） |
| 注册器（fightparry_C） | sub_3EC4720 | GameConfigEnt {name:"fightparry", 存, 取} |

fight 对象 glue 表 @0x68f0e90（12 字节行 {funcA_rva, funcB_rva, desc_rva}，**DWORD RVA 存储**，
+0x3AF0000 才是 EA）；方法名簇 RVA 0x1a30f78..0x1a30fe8：
`SelectCancel / GetNormalAnqiAttackSkillID / CheckExtraSkillBeginWhack / CheckExtraSkillEndWhack / SetParryType`。

**body（场景可视体）字段地图**（0x1440~0x147C 每 4 字段一个 getter 阶梯 @0x3E9C7xx）：

| 偏移 | 语义 |
|---|---|
| +0x1460 | parryType（fightparry 配置值） |
| +0x1468 | NpcType |
| +0x1472 | RoleStateIndex（1/2=静态/移动；亦作移动态判据） |
| +0x1480 | InParry 运行时值 |
| +0x1184 | 当前动作名（char*） |
| +0x1508 | WeatherPag |
| +0x2124 | RideLinkState |

## §六 InParry 生态（属性三件套 + 全部消费者）

"InParry" 字符串（VA 0x5448EC8）全模块仅 3 个消费者：

1. **建体初始化**：sub_46E3C90 = `SceneObj::ShowSceneObjStep2`（错误串自证）。
   两处调用（0x46e40d5 Type==4 人形武器体 / 0x46e462c Type==2）同走
   thunk sub_3B11909 → **sub_41B9420 = InParry 属性处理器**：
   - 按名取属性 → 写 body+0x1480；
   - `InParry<=0` 且 body+0x1472==2 → 清 `move_angle`/`face_angle`；
   - 移动态分发动作：ground / `buf_qg_fast` 且速度=(body+0x1624^body+0x1628)/100 **>6.8** →
     `qg_fastruning`+`fight\other\fight_qg_sx_g|b` 特效 / jumping / swim；
   - 退出时动作名含 "parry" → 对场景 `ClearBlendAction`。
2. **HUD 血条**：sub_3DF7580（ParryRatio/lbl_parry 控件显隐+赋值；Type==4 门）。
3. **虚招（Whack）闸门**：sub_503B2C0 —— 技能 StaticData `NoLimit/CanAirUse/IsAirSkill>0`
   或身体空中态时，按目标 `InParry>0` 返回 1/4 分流（虚招克格挡的判定点）。

属性三件套（均挂 0x67exxxx/0x67fxxx 属性注册表，12 字节行 {funcA,funcB,desc} RVA 三元组）：

| 方向 | 函数 |
|---|---|
| S2C 推写（属性→body） | sub_3EC54B0（RVA 0x3d54b0，行 @0x67f117c） |
| 上报导出（body→属性） | sub_3E9E430（行 @0x67ed6fc） |
| 本地写 | sub_3EAE260（写 body+1480；尾部为 std::string 构造**非**事件派发——旧"事件18"记法勘误） |

## §七 ClaspParryLockTime 家族（擒抱克格挡）

- **sub_5092BC0 = "该技能能否锁格挡"**：当前动作名剥 `_parry` 后缀得基础技能名 →
  查其 StaticData `ClaspParryLockTime > 0` 才返 1。
- **sub_5093090 = 擒抱克格挡（ClaspRush）**：目标 InParry>0 且技能有锁时 → 双发 `clasp_state=2`
  + clasp_rush_h/v_distance 断言（失败路径带 `__debugbreak`，发布版即静默）。
- **sub_50963A0 = play_hitout**：擒抱被打脱出机（hitclasp_vv/va + hit_clasp_dropdown/uploop/up 动作族）。
- LastParryTime → sub_3E85770 = `CS_wd_tjq02`（太极剑舞二式）专用 5-tick 窗口检查，非通用门。
- 格挡动作名族（VA 0x54ac2f0 起）：`parrying` + `parrying_up/f/r/l/b` + `phyparry_s` + jump/swim 变体。

## §八 m54 发送链排除法定位：VMP 区

| 模块 | imm 54 命中（edx/ecx/r8d/eax 全变体） | 结论 |
|---|---|---|
| fxgamelogic | 12 处 | 全是 UI 数值巧合（Height/Width=54、ride_stay=54 等） |
| fxprotect | 30 处 | 全是日志调用（`mov e?x,54; lea r8,[rip+str]; call` 行号形态） |
| fxnet2（rebuilt+原版） | 0 | 无 |
| fxcore / fxworld | 1 / 1 | 无关 |

叠加证据：99 个原生 custom id 无 48-62 段、lua 定义表 54 空缺、白名单无 54、
game_visual lua API 无格挡项 —— **m54 的 VarList 构造与格挡本地进入判定都在 .vmp0/.vmp1 虚拟化区**
（.vmp0 @0x5AAA000 / .vmp1 @0x5CFF000，共约 17MB，静态不可读；与
[rush-autowalk 结论](../client/) "FxGameLogic .text 虚拟化静态死路" 一致）。

**判读**：官服族客户端 m54=每帧状态上报（基调=未格挡），"恒 1"说明上报循环活着；
真正不发生的是**本地进入格挡态**（进入后上报值才会翻转）——该判定被 VMP 保护（输入层反外挂）。

## §九 工具坑与方法论（本轮新增）

1. **hex 心算地雷**：RVA+base 一律用工具算。本轮 0x1549910+0x3AF0000 心算成 0x4FE9910
   （正解 0x5039910），反出一个无关的 JSON 析构函数，浪费一跳。
2. **mcp find_bytes ≫ search_text**：后者逐指令渲染全图，47MB 镜像 30s 超时；
   字节级 pattern（含 `??` 通配）秒回。
3. **表值 DWORD RVA 存储**：rebuilt 件里的 jump/注册表大量以 4 字节 RVA 存指针（+0x3AF0000 才是 EA），
   按 8 字节 qword 读会把两行并成一行。
4. **xref 死路形态**：纯虚调用（vtable 槽引用）与 ILT thunk 链会让 XrefsTo 断链；
   换 dref 链（get_first_dref_to/get_next_dref_to）或 MCP xref_query 可破（本轮 13979 个引用
   python XrefsTo 返回 0，dref 链全量拿到）。
5. **py_eval 作用域**：生成器/推导式里的变量在 exec(globals, locals) 分离下不可见
   （NameError: chunk）——展开成普通循环；控制台 GBK，输出非 ASCII 须 encode('ascii','replace')。
6. 属性注册表（0x6818640 / 0x67exxx / 0x68f0e90 家族）第三列指向**二进制描述符**
   （序列化 TVarList 参数流，形如 `01 06 02 00 06 32 02 30`），不是名字串。

## §十 下一步三选

- **A**：IDA 载 `fxprotect_rebuilt.dll` 挖右键→格挡进入链（输入层有大量原生码，且 motion_func 白名单
  已有先例可循）。
- **B**：运行时断点 sub_466B880（vt+120 无闸发送入口）抓 m54 caller 栈——每帧必触发，一试便知；
  风险=SDProtect 反调试（历史经验：live RPM/MiniDump 均被剥权，任务管理器手工 dump 是唯一取证）。
- **C**：黑盒回归——对照 server1（可格挡的参照服）给客户端发的使能类 S2C：它能格挡
  = 它点亮了客户端某个本地态；沿 `格挡.pcapng`/server1 抓包差集找这帧。

## 附录：关键地址速查（EA=RVA+0x3AF0000，均记 RVA）

| 对象 | RVA | 说明 |
|---|---|---|
| lua CustomSend 处理器（带闸） | 0xb6e440 → 0x465E440 | sub_465E440 |
| 白名单表 | this+0x5149DC | 1024 字节 |
| 白名单构造器（类 ctor） | 0xb7eca0 → 0x466ECA0 | memset+点亮数百 id |
| 类 vtable | 0x1952CF8 → 0x54B2CF8 | 对象 5,328,304B，分配器 sub_0xC32800 |
| vt+120 无闸发送 | 0x64FE2 → 0x3B54FE2 → sub 0xB6B880 | thunk 链 |
| 对象分配点 | 0xb82800/0xb82890 | 5328304 字节经 qword_5923EC8->vt+16 |
| m292 原生发送器 | 0x14a4f0 → 0x51964F0 | 门=body+1472∈{1,2} |
| m218 发送器 | 0x14acb0 → 0x5194CB0 | VarList{218,args} |
| SetParryType glue | 0x154a090（壳 0x154a120） | → 218 |
| InParry 属性处理器 | 0xcc420 → 0x41B9420 | 建体走 0x46e40d5/0x46e462c |
| InParry S2C 写/导出/本地写 | 0x3d54b0 / 0x2fe430 / 0x3fe260 | sub_3EC54B0/3E9E430/3EAE260 |
| fightparry A/B/C | 0x3bdaa0 / 0x3d4630 / 0x3d4720 | 本地写/推写/注册 |
| 虚招闸门 | 0x154b2c0 | sub_503B2C0 |
| ClaspParryLock 检查 | 0x143bc0 | sub_5092BC0 |
| 擒抱 engage / hitout | 0x143090 / 0x1463a0 | sub_5093090 / sub_50963A0 |
| body getter 阶梯 | 0x3ec700..0x3ec7d0+ | 0x1440..0x147C 每 4B |
| game_visual 名字串 | 0x19593a8 → 0x54293A8 | 查找=sub_3B2B728→sub_5211E00 |
| game_visual lua API 全量 | off_5923ED0 链 | 6990 项，名册 tmp/gv_ent_names.txt |
