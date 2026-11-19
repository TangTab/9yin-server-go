// Package fssolostore 复刻官方 FsSoloStore.dll（fssolostore 模块，
// game_store.exe 宿主，MainEntity=SoloStore）——solo 单机集群的存储/DB
// 枢纽：集群内一切持久化（角色存档/场景块/信件/公共数据/管理配置/
// SNS/交易/拍卖/运营日志）经 inner 网汇入本进程落库。
//
// 权威逆向文档：docs/official/FsSoloStore全解与Go复刻.md。
//
// 复刻范围（框架/协议/SQL 语义层；DB 以 Backend 接口注入）：
//   - reactor 装配：inner listener + UDP 广播自宣（kind2 0x28B，
//     +0x24=StoreId 原值，member 端 CheckStoreConnect 内 +4 得 comm）
//   - echo 维护口（15100+StoreID）+ 每秒 TimerCheck；
//   - 命令路由：ProcessInnerMsg 四分支 + SendAccessRequest 67 型
//     （0x00..0x42）通道分片 + serial 编解码（高 16=源 comm，低 16=
//     请求序号；0/1/2=world/public/auction 直连）；
//   - 回执路由：CheckResponse 等价——Backend 异步执行，回执按 serial
//     还源（SendToWorld/Public/Auction/Member）；
//   - 自定义日志表 29 张（main.lua init_custom_log 全量 schema）+
//     分周分表语义（SHOW TABLES LIKE 缺则建 + suffix 广播）；
//   - AppRecord 状态行（每 5s <info state database db_err tick/>）。
//
// 不复刻（与官方差异）：TStoreArchive/CLoadArchive 对象内核（payload
// 透传 Backend）、CTaskQueue/CThread 具体实现（goroutine+chan 等价）、
// SQLOLEDB 方言（Backend 出内存/MySQL 两实现）、双进程形态 UseThread。
package fssolostore
