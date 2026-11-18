// Package persist 实现持久化，对应原版 FsSql：MySQL 存取队列、
// 角色数据序列化（nx_chunks 分块 blob）。
//
// 已知原版线索：
//   - 完整 DDL 见原版 Res/mysql/*.sql（nx_base/nx_district/nx_log），
//     本仓库 Go 侧建库脚本以此为输入
//   - nx_chunks 表 + save_data longblob → 对象存档走分块 blob 序列化，
//     具体二进制格式是唯一需逆向的格式黑盒（FsSql.pdb 在手，符号齐全）
//   - 原版有异步存档线程（logger.UseThread 同款思路）——本实现用
//     后台 goroutine + 写队列，禁止在 tick 内直接落库
package persist
