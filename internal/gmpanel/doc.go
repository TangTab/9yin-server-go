// Package gmpanel 自建 Web 运维面板（独立进程 cmd/panel，默认
// 127.0.0.1:19064）：服务启停、日志查看、在线玩家、GM 命令入口。
//
// 形态：supervisor 托管 + 观察双模式。
//   - 托管模式：面板按 server-cmd/start-cluster.bat 同序拉起集群子进程，
//     stdout/stderr 重定向 logs/<name>-<YYYYMMDD>.log（按天滚动）——
//     Go 集群原本日志只落在各自 start 窗口，窗口关即丢，面板顺带补上落盘；
//   - 观察模式：进程由 bat/手工启动时，面板经 echo/业务口 TCP 探活识别，
//     只读展示，停止单服降级 taskkill /PID（netstat -ano 端口→PID 解析，
//     失败回落 /IM）。面板自身退出不死集群（Windows 子进程无 job object），
//     重启面板自动转观察模式。
//
// 数据/控制通道复用既有设施，不侵入 fs* 各服：
//   - 各服 echo 维护口（world 15001 status/players/plist/kick、
//     monitor 15012 state/tables、quit 优雅停族）；
//   - MySQL 直连（fssolostore.OpenMySQLStore：nx_roles/dc_accounts/
//     nx_monitor 查询，SetAccountStatus 封号）。
//
// 前端 web/ 编译期内嵌（go:embed），AXUI 组件库（web/static/ax.esm.js
// + ax.css）。ESM 构建不自动注册 Web Component，main.js 启动时遍历
// 命名空间 regComp 逐个 define（wt1653 线实证坑）。
package gmpanel
