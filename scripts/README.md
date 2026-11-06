# 服务端 Lua 脚本

本目录下的 `*.lua` 会在服务端启动时按文件名序加载（`-scripts` 指定目录，
默认即本目录；传空串禁用）。GM 面板的"重载脚本目录"按钮可热重载。

引擎是内嵌的 Lua 5.5 运行时（`third_party/golua`，纯 Go）。脚本在沙盒内
执行：**没有 io / os / exec / debug / chan / time 模块**，每次调用有
500ms 超时与 5,000,000 条指令预算；运行期出错只会让当次交互失败，随后
VM 连同脚本目录整体重建，不会影响服务端进程。

## 事件注册

```lua
game.on("事件名", function(ctx) ... end)
```

同一事件多次注册以最后一次为准。处理器返回 `false` 表示交回默认路径
（例如维持"服务未实现"）；无返回值或真值视为已处理。

| 事件 | 触发点 |
| --- | --- |
| `npc_service` | 玩家在 NPC 菜单选中一项原生未实现的服务（任务/传送/职业/宝石/帮派/宅邸等 mark） |

## 调试输出

```lua
game.log("格式 %s=%d", "键", 42)   -- 进服务端日志，带 lua: 前缀
```

## 调用上下文 ctx（仅 npc_service）

- `ctx:player()` → `{ id, name, faction, silver, sp, scene, x, y, z }`
- `ctx:npc()` → `{ id, config_id, name }`
- `ctx:service()` → `{ mark, label, value, source }`
- `ctx:notify(brief, chapter_name?)` → 给玩家发剧情提示卡
  （chapter_name 缺省 `ui_shop`，须是客户端已装的文本 ID；brief 可为任意文本）
- `ctx:bubble(text_id)` → 让该 NPC 头顶冒五秒说话气泡（text_id 须已装）

## GM 面板

"Lua 调试"卡片可在沙盒内执行任意代码并回显返回值（`return 1+2` → `3`），
输入上限 16KB。注意：卡片里没有 ctx，玩家相关副作用请写在事件处理器里。

## Lua 5.5 注意事项

chunk 内一旦使用 `global x = 1` 声明，该 chunk 就进入严格模式——
所有全局引用（包括 `game`）都必须先声明。除非需要严格检查，
沿用传统的隐式全局即可（本目录示例即如此）。
