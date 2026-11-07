# 内功栏绑定持久化（NeiGongBind，2026-11-07）

> 真机反馈：「内功栏，放置武器或者内功绑定的快捷栏序号没有存储」。
> 根因：内功栏的**绑定关系**（每格绑到哪个武器快捷栏序号）走的是独立协议
> C2S 199 / 属性 NeiGongBind(201)，与快捷栏放置本身（200/201 → role_shortcuts）
> 完全不同域。Go 服与 Java 参考服此前都未实现 199（Java
> docs/lua64-protocol-dict.md 仅录得协议号），语义按客户端
> form_main_shortcut.lua / custom_sender.lua 直读钉死。

## 一、客户端语义（lua64 unpacked 直读）

- 内功栏 = `form_main_shortcut` 的 `grid_shortcut_3`（300 起，3 页各 5 格）
  + `grid_shortcut_8`（副武器栏，紧接其后 10 格），共 **25 格**
  （`G_NG_GRID_COUNT = 25`，`G_NG_BEGININDEX = 300`）。
- 放置武器/内功本体走 `set_shortcut(index, kind, id)` → **C2S 200**（item 行
  id=UniqueID、neigong 行 id=ConfigID），这部分早已闭环（role_shortcuts）。
- 放置后弹出的**绑定框**（gbox_bind）选择该格绑定的**快捷栏序号**
  （DataSource，0=清除；文本 `ui_mail_N` 即页码族文案）：
  `on_btn_bind_click` → `set_bind_by_index(slot, value)` 更新
  `form.ng_bind_str` → `update_to_srv()` 比对服务端属性后
  `custom_update_ng_bind(info)` → **C2S 199**。
- **读回**：主窗体打开时 `get_svr_ng_bind()` =
  `client_player:QueryProp("NeiGongBind")` 整串快照；拆段数 ≠ 26 就整串
  回落全零。即：服务端不回存档 → 重登绑定全丢。
- 服务器驱动同链路：custom_handler.lua `neigong_bing_weapon(bind, btn)`
  （引导/GM 脚本入口，服务端无需实现）。

## 二、协议

| 方向 | ID | 载荷 | 语义 |
|---|---|---|---|
| C2S | **199** | `[199, "v1;…;v25;"]`（tag6 窄串） | CLIENT_CUSTOMMSG_NEIGONGBIND：25 个值各对应内功栏一格（grid_3 页内序 + grid_8 序），值=绑定序号（0..99，实测 0/1/2；0=未绑定），尾分号固定（拆 26 段、末段空） |
| S2C | 0x10 属性 | NeiGongBind = 0x09 表索引 **201**，type7 | 线上串 = value + " "（u32=len+1，键位 198 同方言）；客户端 update_to_srv 靠与它比对决定是否再上传，不回推两端永不收敛 |
| S2C | 0x0D 档内嵌 | 同 201 | 引擎建对象档内无尾空格（198 同款两形态）；主窗体打开即快照，post-ready 补推可能晚于窗体打开 |

全零基线串 = `"0;" × 25`（string.rep，50 字节）。

## 三、实现（Go 服）

- `custom_config.go`：`clientCustomNeiGongBind=199`、`neiGongBindPropertyIndex=201`、
  `validNeiGongBind`（25 值 + 尾分号、各值 0..99，脏串整拒防把客户端档位
  校验打回全零）、`neiGongBindPropertyFrame`、`handleNeiGongBindCustom`
  （校验 → 回推 0x10 → 落内存 → 落库）、`pushKeybindFrames` 尾随 201 帧
  （ClientReady 与切图重推同闸，keybindPushed 复用）。
- `player_actor.go` addObject 档内嵌 201（无尾空格，无档零属性）。
- `main.go` 两处分发（0x0A 活动信封白名单 + 顶层 0x1E/0x27）各加
  `case clientCustomNeiGongBind`。
- `custom_config_db.go`：`keybindDBStore.Load` 增读 `neigong_bind`（NULL=
  从未绑定，零帧）、`SaveNeiGongBind`（UPSERT 首插补 bind_value 空串，
  role_keybinds NOT NULL 老坑）。

## 四、SQL

- `migrations/0015_add_keybind_neigong_bind_20261107.sql.gz`：
  `role_keybinds` 增列 `neigong_bind varchar(128) NULL AFTER operate_mode`
  （0008 同款 information_schema 幂等探列）。共享库三服共存，可空列对他服
  无影响。已于 2026-11-07 随 `-force` 全量重放落库并验证列存在。

## 五、测试

- `TestValidNeiGongBind`：全零/带值合法；缺尾分号、段数错、空值、越界、
  非数字、空串全拒。
- `TestHandleNeiGongBindCustomValidatesAndEchoes`：合法 → 单帧 201
  （值+尾空格）+ 内存；非法/选角前零帧。
- `TestNeiGongBindPushAndArchiveEmbed`：pushKeybindFrames 有档才发 201；
  0x0D 档内嵌无尾空格、无档不嵌。
- `TestKeybindDBLoadDistinguishesNullOperateMode` / `TestKeybindDBSaveNeiGongBindBootstrapsBindValue`：
  DB 读 NULL 语义与 UPSERT 方言。

## 六、验尸记录（为何抓包里找不到范本）

- F 盘 9yin_server1（newer 服）抓包 49840/62311：无 199 上行；0x10 无 201
  下行；0x0D 大档（count≤95）不含 201/198——会话内玩家未做过内功绑定，
  该服对 198 也走 post-ready 推送，证明档内非唯一形态。
- 官方 61114（Java §十记录）0x0D 大档 count=121 含 198——201 大概率同为
  官方档内属性，但无直接样本；实现取「档内嵌 + post-ready 单帧」双发
  （198 同款双发形态，幂等）。
