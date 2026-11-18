# nx_base save_data 容器格式（FsSoloStore 逆向定案）

> 逆向对象：`F:\Game\Online\9yin\BGserverCJ\Debug\FsSoloStore.dll`（.map/.pdb 全带）
> 2026-11-19 定案。配套工具：`tmp_research/zip0_probe.py`、`tmp_research/disas_tool.py`、`tmp_research/disas_world.py`

## 一、结论：外层容器（已实证，反汇编级）

`nx_roles.save_data` / `nx_pubdata.save_data` / `nx_chunks.save_data` / `nx_auction.save_data`
四列同一容器格式：

```
offset  size  含义
0       4     magic = 'ZIP0'（u32 LE 0x3050495A，字节序 5A 49 50 30）
4       4     原始（未压缩）长度 u32 LE
8       ...   zlib 流（compress() 默认级别 6，标准 78 9C 头）
```

- 生产：`CAccess::CompressData` @ FsSoloStore.dll VA `0x1008d5d0`
  （写 magic → 写 origLen → `compress(dst+8, cap-8, src, srcLen)` → 返回 8+压缩长）
- 校验：`CAccess::GetUncompressBufferSize` @ `0x1008d6e0`
  （len<8→0；magic≠ZIP0→0；origLen>0x1000000(16MB)→0；否则返回 origLen）
- 解压侧 `UncompressData` @ `0x1008d770` 走 zlib `uncompress()`
- 日志指纹：`"role %s origin %d bytes compressed %d bytes"`（FsSoloStore.dll 0x10157270）

**Go 侧等价**：`zlib.NewWriterLevel(w, 6)`（对拍字节级一致需用 level 6，即 `compress/flate.DefaultCompression`）。

## 二、存取链路与 SQL（全部字符串实锚）

宿主进程：`game_store.exe`(FsSoloStore)。世界进程 `game_world.exe`(FsSoloWorld) 经
`CInnerHook::SendToStore(connId, msgId, data, len)` 发 IPC，`CAccess::ProcessMsg`
@ `0x1008d8a0` 按 msgId 跳表（表在 `0x1008e0ac`，0x00–0x42）分派：

| msgId | 处理函数 | SQL |
|-------|----------|-----|
| 0x02 | `MessageLoad` | SELECT 后解压 save_data 回传世界 |
| 0x04 | `MessageSave` | `UPDATE nx_roles SET total_sec=%d, letter_max=%d, scene_id=%d, location='%s', charge_info='%s',\tsave_para1='%s', save_time=NOW(), save_data=? WHERE name='%s'` |
| 0x0a | `MessageCreate` | `INSERT INTO nx_roles (name, uid, account, ..., save_para1, save_time, save_data) VALUES (..., '%s', NOW(), ?)` |
| 0x16 | `SavePlayer`（世界侧 0x100ce390，0x50 字节元数据包） | 同 0x04 的 UPDATE 但**不带 save_data**（周期存元数据/离线） |

- `location` = `"%.3f,%.3f,%.3f,%.3f"`（消息内 +0x80/+0x88/+0x90/+0x98 四个 double）
- 角色名在消息内 data+8 起 0x80 字节区；charge_info 在 data+0x50
- blob 在 world→store 消息内**未压缩**，压缩发生在 store 侧落库前
- 摘要表 `nx_role_dumpinfo` 由 dumpconfig/dump_role_dbinfo.xml ORM 描述（与 save_data 无关）

## 三、内层载荷编码原语（TStoreArchive / CLoadArchive，已实证）

解压后的字节流是 Fs 框架 `TStoreArchive`（写）/`CLoadArchive`（读）序列化的角色实体
（props/records）。原语编码（FsSoloWorld.dll Routine.obj 实锚）：

| 类型 | 编码 |
|------|------|
| Bool | 1 字节 |
| Byte | 1 字节 |
| Word | 2 字节 LE |
| Int | 4 字节 LE |
| Int64 | 8 字节 LE |
| Float | 4 字节 IEEE LE |
| Double | 8 字节 IEEE LE |
| String | u32 LE 长度（**含结尾 NUL**，=strlen+1）+ 字节 |
| WideStr | 宽串（UCS2，长度前缀同理） |
| Object | 8 字节 = PERSISTID{u32 ident, u32 serial} |

IVarList 通用编码（`put_vary_message`/`get_vary_message`，IPC/存档共用值编码）：
每元素 = 1 tag 字节 + 值：

```
1=bool  2=int32  3=int64  4=float32  5=float64  6=string  7=widestr  8=object(PERSISTID)
```

（`get_vary_message` @ FsSoloWorld 0x100a7a30：ReadByte tag → switch(tag-1) 八路 →
对应 Add* 虚调用；未知 tag 直接放弃。）

## 四、内层实体树布局（OBJ3 格式，2016 官方真档实证，2026-11-19 定案）

> 样本：本机 nx_base.nx_roles 2016 年官方生产档（~80 角色）。工具 `tmp_research/obj3_parse.py`
> 两档（星蓝 34168B 新角 / 詹妮弗劳伦斯 84423B 熟档）**零剩余全量解析**。
> **304 属性 + 112 记录表逐项全名单**：`docs/official/save-data-OBJ3-player-schema.txt`
> （属性含新角/熟档双列对照，≠ 标记值有差异的 88 项）。

解压后的字节流 = 一棵递归实体树：

```
entity := '3JBO'(u32 LE 0x4F424A33) u32 ver cstr 类名 cstr 实体名
          u32 propCount u32 recCount u32 childCount u32 F u32 G
          prop[propCount] record[recCount] entity[childCount]     ← 子实体紧随,无分隔符

prop   := cstr 属性名 u8 tag value          ← tag 即第三节 1..8 值编码
record := u32 nameLen name\0(NUL计入) u32 cap u32 rows u32 cols
          cols×u8 列tag (rows×cols)×value    ← 行主序,值同样按 tag 编码
cstr   := u32 字节长(含结尾NUL) + 内容       ← 窄串 GBK, 宽串 UTF-16LE(NUL=2字节)
```

- `ver`：player=2，容器/物品/buff=8，个别 box=32（值不影响布局）
- `F`：容器容量（ToolBox=150、EquipBox=32、SkillContainer=512/32、叶实体=0）
- `G`：实体序列号（新角 player=198、熟档=1367；叶内从 1 递增，有跳号）
- player 固定 **304 props + 112 records**，子容器 28~29 个：
  BufferContainer(buff) / SkillContainer×7(各技能池,子=SkillNormal[技能名]) /
  ToolBox(背包) / EquipBox / FashionBox / MountBox / EnhanceBox / InscriptionBox /
  DepotBox(仓库) / 各邮购交易盒 / UsingItemSkill…
- 物品实体：类名=物品大类（Enchase 宝石 / Material 材料 / …），实体名=模板 ID
  （`Baoshi_0001D_1`），固定 8 props：Owner(widestr)/UniqueID(string)/Amount/BindStatus/
  NotValid/LeftTime/BeginTime(int64)/ClearTime(int64)
- buff 实体：类=Buffer，名=`Buff_In1031_01_1`，31 props（LifeTime/UniqueID/LifeTimeAdd/…）
- 代表性 record（官方表 dump_role_record.xml 交叉验证）：
  capital_rec(3列 6,2,3=货币)、cooldown_rec(4列含 int64 时戳)、shortcut_rec(快捷栏)、
  equip_skill_rec_all(18行×5列)、friend_rec(19列)、clone_info_record(40行×13列)、
  ForceReputeRecord(64行×9列=门派声望)、open_prompt_rec(63行)

Go 复刻建议：`internal/persist` 落 ZIP0 读写 + OBJ3 实体树编解码（值编码即 TStoreArchive 原语），
player 侧按「固定 304 prop 槽 + 112 record 表名清单」建 schema，两档样本可作 golden。

遗留（未实证，不影响复刻）：
- `lazy_data` 列**全快照死列**：仅存在于 DDL（nx_base.sql/nx_basecj.sql 两份，nx_roles 独有），
  全部 DLL/EXE 字符串、全部 .map 符号、Res 全部 XML/INI/Lua 中 "lazy" 子串零命中；
  FsSoloStore 的 INSERT 列清单 = DDL 全列恰好减 lazy_data（21/22 一一对应）→ schema 比二进制新。
  本快照网络版 FsStore.dll 也不用它（网络版走 `%s_save_data` 通用镜像表体系，如 nx_roles_save_data）。
  语义推断：与 save_data 相对的角色冷数据第二 blob（惰性加载/延迟落库），Go 复刻可无视。

## 五、复现锚点速查

```
FsSoloStore.dll (base 0x10000000):
  0x1008d570 GetCompressBufferSize   0x1008d5d0 CompressData
  0x1008d6e0 GetUncompressBufferSize 0x1008d770 UncompressData
  0x1008d8a0 CAccess::ProcessMsg     0x1008e0ac 跳表(67 项)
  0x100a13f0 MessageLoad             0x100a24e0 MessageSave
  0x100a4240 MessageCreate           0x100ce390(世界)SavePlayer→msg 0x16
FsSoloWorld.dll (base 0x10000000):
  0x100a7580 put_vary_message        0x100a7a30 get_vary_message
  0x100a9900 WriteString(len=strlen+1 含NUL)  0x100a8b40 ReadObject(8B)
  0x100ccd30 InnerLoadPlayer         0x100b5180 CInnerHook::SendToStore
```
