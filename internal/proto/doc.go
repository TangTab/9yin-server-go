// Package proto 定义消息号与序列化。
//
// 三层消息体系（事实源各不相同）：
//
//  1. 引擎级消息（message.go，手写）：帧首字节即消息号
//     （0x18/0x38 心跳、0x05 worldInfo、0x1C SERVER_MENU、0x1E/0x27 自定义
//     直发…）。定义在原版 C++ 中，线索源 = FsGame.pdb 符号 + 抓包对拍。
//
//  2. C2S 自定义消息（custom_gen.go，protogen 生成）：数值 ID + 格式注释，
//     事实源 = 原版 lua 目录 testdata/lua/share/client_custom_define.lua。
//
//  3. S2C 自定义消息（custom_gen.go，protogen 生成）：字符串名 + 格式注释，
//     事实源 = server_custom_define.lua。
//
// 消息体统一为 TVarList（varlist.go）——Fs 框架 IVarList 的线路形态。
//
//go:generate go run ../../tools/protogen -in ../../testdata/lua/share -out .
package proto
