// Package sync 实现可见性与脏同步系统：把组件/属性变更按
// public/private/realtime 元数据转成协议推送，并处理断线重连的增量同步。
//
// 这是全项目风险最高的模块（对拍老风险项）：
//   - realtime 属性的推送时机（帧末聚合？立即？）
//   - 进出视野时的属性全量集与顺序
//   - 断线重连的增量集与顺序
//   - Record 行操作的推送粒度
//
// 以上语义原版二进制里只能看到"是哪个函数"，时机要靠 cmd/probe 对原服
// 抓包验证（golden 用例沉淀在 testdata/captures/）。
// FsGame 的 PDB 符号（OnStore/OnRecover/OnPlayerReady 等）是函数级线索源。
package sync
