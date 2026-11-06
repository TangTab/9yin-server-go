# 本目录的来源与本地修改

本目录是 [github.com/iceisfun/golua](https://github.com/iceisfun/golua)
v2 分支的**内嵌副本**（纯 Go 实现的 Lua 5.5 运行时，MIT 许可，见
LICENSE）。做法与本仓库内嵌 `internal/ecs`（unitoftime/ecs）一致：
以源码形式随仓库分发，构建不依赖任何外部托管。

## 上游版本

- commit `36de3be5500e`（pseudo-version `v2.0.6-0.20260901081125-36de3be5500e`，
  2026-09-01 master）
- 上游在 v2.0.5 tag 之后、该 commit 之前无 Windows 适配，本副本在其
  基础上打了下述补丁。

## 本地补丁（相对上游的全部差异）

上游 `vm/process_provider.go` 无条件使用仅 Unix 存在的
`syscall.SysProcAttr.Setpgid` 与 `syscall.Kill`，导致 Windows 编译失败。
本副本将这三处进程组操作拆到按构建标签区分的两个新文件，行为不变：

- `vm/process_group_unix.go`（`//go:build !windows`）：原样使用
  `Setpgid` + `syscall.Kill(-pid, SIGKILL)`；
- `vm/process_group_windows.go`（`//go:build windows`）：
  `CREATE_NEW_PROCESS_GROUP` + `Process.Kill()` 退化实现。

该代码只服务于 exec 模块（`DefaultProcessProvider`）；本仓库的脚本宿主
（`internal/luascript`）**不注册任何 io/os/exec provider**，补丁路径在
服务端运行时不会执行，仅为 Windows 编译兼容。

## 裁剪

相对上游删除了与服务端嵌入无关的目录：`docs/`、`tests/`、`examples/`、
`bench/`、`wontfix/`、`cmd/`、`check/`，以及 CONTRIBUTING.md、
SKILL.md、TESTING.md、PERF.md。全部 Go 包（含其测试）保持上游原样。

## 验证与同步

- 在本目录运行 `go build ./...` 与 `go test ./vm/... ./compiler/... ./parser/... ./lexer/... ./ast/...`
  可验证（2026-11 在 windows/amd64 + Go 1.27 验证通过；`stdlib` 与
  `vm` 中少量 io/符号链接相关测试在 Windows 普通权限下因系统限制失败，
  与本仓库的使用方式无关——沙盒不注册 io）。
- 重新同步上游：拷入新版本源码后，重放"本地补丁"两个文件并重跑上述
  验证。根 `go.mod` 的 `replace` 指向本目录，升级只需改这里。
