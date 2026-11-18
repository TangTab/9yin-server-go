// Package proto 定义消息号与序列化。产物两部分：
// 手写部分（封包头、基础类型编解码）+ protogen 生成部分
// （消息结构体、序列化/反序列化、消息号注册表）。
//
// 事实源：tools/protogen 的输入规格（消息号 ↔ handler 映射、字段顺序），
// 规格内容来自两个线索源：
//   - 原版 FsGame.pdb 符号（OnXxxMsg 系列，枚举全部 handler）
//   - cmd/probe 对原服的抓包对拍（字段宽度/顺序/符号性验证）
//
// 纪律：抓包切片沉淀为 testdata/captures/ 的 golden 用例，
// 生成的 codec 必须通过与原始字节的逐字节比对测试。
//
//go:generate go run ../../tools/protogen -spec proto.yaml -out .
package proto
