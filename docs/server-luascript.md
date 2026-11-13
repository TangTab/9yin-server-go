# 服务端内嵌 Lua 脚本宿主（internal/luascript）

2026-11-06 落地。Go 服务端（`cmd/protocol-probe`）内嵌 **Lua 5.5** 运行时，
玩法逻辑（当前是原生未实现的 NPC 服务）可用仓库自有 `.lua` 脚本编写。

与另两条 Lua 线索的关系，注意不要混淆：

| 线索 | 方言 | 位置 | 说明 |
|---|---|---|---|
| **本文**：服务端脚本宿主 | Lua 5.5 | `internal/luascript` + 本仓 `scripts/` | 我们自己写的服务端玩法脚本 |
| 客户端字节码加解密 | Lua 5.1 | `internal/luacodec` + `cmd/lua-batch` | FxCore XOR 格式逆向，见 [lua64-analysis.md](lua64-analysis.md) |
| Java 参考服技能脚本框架 | 客户端 5.1 方言 | java 侧 `scripts/` 契约 | 跑游戏自带技能脚本，见 [skill-script-framework.md](skill-script-framework.md) |

## 引擎与内嵌方式

引擎选 [iceisfun/golua](https://github.com/iceisfun/golua) v2（纯 Go 的
Lua 5.5 实现，MIT，sandbox-first）。备选过 gopher-lua（纯 Go 但只有 5.1）、
Shopify/go-lua（能载 luac 字节码但停止维护）、cgo LuaJIT（破坏纯 Go 构建），
均不如它满足"5.5 + 纯 Go"。

- 以源码形式内嵌在 `third_party/golua/`（仿 `internal/ecs` 惯例），
  `go.mod` 经 `replace` 指向本地目录，上游 commit `36de3be`（2026-09-01）。
- 上游在 Windows 编译不过（exec 模块无条件用 `Setpgid`/`syscall.Kill`），
  本地补丁把这 3 处拆成按构建标签区分的 `vm/process_group_{unix,windows}.go`；
  补丁只服务 exec 模块，宿主沙盒不注册该模块，运行期不触发。
  差异与同步方法见 [../third_party/golua/FORK.md](../third_party/golua/FORK.md)。
- 裁掉了 docs/tests/examples/bench/wontfix/cmd/check 目录（约 14MB）。

## 架构与锁纪律

```
handle() ──newSharedSceneLifecycle──> sceneLifecycle.scripts（*luascript.Host）
selectNPCMenu / selectCinematicTalkLocked（持 run.mu）
  └─ openSelectedServiceLocked 原生实现缺席的 default 分支
       └─ runNpcServiceScriptLocked ──> Host.Call("npc_service", ctx)
             └─ 单 VM + Host.mu（叶子锁）；副作用经 EventContext 回调写帧
```

- 单 VM 由 `Host.mu` 串行化；**锁序：调用方已持 run.mu → Host.mu → player.mu**，
  宿主函数绝不反向获取场景锁，副作用（发帧）一律经回调由调用方执行。
- 玩家/NPC/服务快照在调用前装配进 `EventContext`，脚本只读数据 + 调回调。

## 沙盒与防护

| 防线 | 值 |
|---|---|
| 模块面 | `stdlib.Open` 不注册任何 provider → **io/os/exec/debug/chan/time 不存在** |
| wall-clock | 每次调用 500ms context 超时（死循环 → 错误返回） |
| 指令预算 | 5,000,000 条（与超时互为冗余） |
| panic | 引擎自身 panic 被 recover 成错误，不会带崩服务进程 |
| 脏状态 | 运行期出错（含超时）后 VM 连同脚本目录整体重建 |
| GM eval | 输入 ≤16KB，回显 ≤4KB；空输入 HTTP 侧直接 400 |

Lua 5.5 语义注意：chunk 内一旦出现 `global x = 1` 声明即进入严格模式，
该 chunk 所有全局引用（含 `game`）都必须先声明；沿用隐式全局则与 5.1 无异。

## 事件与 API

事件当前两组：

- `npc_service`（原生未实现的 NPC 菜单服务，markTask 0x1008 等）。处理器
  `return false` 显式交回默认"未实现"路径。
- `dungeon_*` 副本玩法组（2026-11-13，见 [dungeon-clone-system.md](dungeon-clone-system.md)）：
  `dungeon_enter` / `dungeon_npc_killed` / `dungeon_wave_cleared` /
  `dungeon_destroy` + `ctx:dungeon()` 快照与 `ctx:spawn_wave(n)` /
  `ctx:after(sec, event)` 能力回调（`dungeon_wave_cleared` 返回真值即接管
  波次推进节奏）。

```lua
game.on("npc_service", function(ctx)
    if ctx:service().mark ~= 0x1008 then return false end
    ctx:notify("任务暂未开放", "ui_shop")   -- 剧情提示卡（custom 383）
    -- ctx:bubble("ui_talk")               -- NPC 气泡（custom 43）
    return true
end)
game.log("格式 %s=%d", "键", 42)            -- 服务端日志，lua: 前缀
```

ctx 快照字段与脚本侧用法详见 [../scripts/README.md](../scripts/README.md)。

## GM 面板

"Lua 调试"卡片（19062）：`lua_eval` 沙盒执行任意代码并回显返回值
（`return 1+2` → `3`），`lua_reload` 热重载脚本目录。命令仍走
`hub.enqueue → 会话循环 processGMCommand` 的既有链路，需有在线角色。

## 验证记录（2026-11-06, windows/amd64 + Go 1.27）

- 引擎 spike 15/15：基本链路、5.5 语法、沙盒、闭包回调、错误/超时恢复、
  panic 隔离、`string.format`。
- 引擎自带套件：vm/compiler/parser/lexer/ast 全绿；stdlib 与 vm 中少量
  io/符号链接测试因 Windows 普通权限限制失败（TempDir 清理与 symlink
  特权），均在与本宿主无关的 io provider 区域。
- 集成测试：`cmd/protocol-probe` 证实有脚本时 markTask 写出剧情卡帧
  （0x1E），无脚本维持 "not yet implemented"；单测覆盖
  `internal/luascript/host_test.go`。
- 实机冒烟：启动日志 `Lua scripts ready count=1`；GM API 校验/路由正确。

## 后续方向

- 事件面扩展：战斗命中钩子、物品使用（msg31 链，见
  [item-use-msg31.md](item-use-msg31.md)）都可以走同一 `game.on` 机制
  （tick 定时器已由副本层的 `ctx:after` 落地一种形态）。
- 若需要按 mark/config 细分事件或一次性多处理器（多副本脚本必需，
  `game.on` 单 handler 后者覆盖前者），在 `Host.Call` 之上加路由即可，
  不必动引擎绑定层。
- 引擎升级：重放 FORK.md 的两个补丁文件 + 重跑其测试套件即可。
