# 九阴兼容服务端（可运行、可编译副本）

这个目录只保留服务端运行和编译所需内容，不包含 IDA 数据库、反汇编、
审计缓存、历史 EXE 或旧客户端副本。

## 运行

双击 `启动本地九阴服务.bat`。固定端口：

- 区服列表：`127.0.0.1:4000`
- 游戏服务：`127.0.0.1:19061`
- GM 页面：`http://127.0.0.1:19062/`

账号：test001
密码：test001

## 编译

双击 `编译服务端.bat`，或在本目录运行：

```powershell
go test ./...
go build -trimpath -o .\build\9yin-game-native-menu.exe ./cmd/protocol-probe
```

编译产物写入 `build`，不会自动覆盖根目录中正在测试的服务端 EXE。

源码目录：

- `cmd`：服务端和配套命令
- `internal`：协议、角色、资源及网络实现
- `migrations`：MySQL 表结构迁移
- `spec`：协议生成规格
- `testdata`：测试所需的最小协议夹具


