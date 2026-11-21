// Package net 实现线路层：成帧（Framer）与编码（Cipher）两个正交维度
// 的自由组合，对应 Fs 框架通讯层（FsComm/FsVerify）的机制形态。
//
// # 五项框架能力
//
//  1. 键注入机制：XOR 键不编译进代码，由配置/握手注入（对应原版 ini 的
//     MessageEncodeKey）；Connection.SetReadKey/SetWriteKey 支持登录协商
//     后的运行中键轮换，读写方向独立。
//
//  2. 可插拔编码层：Cipher 接口（对应原版 MessageVerifyDll 可插拔语义）。
//     内置 XorCipher（九阴现代客户端线路）与 IdentityCipher（明文，对应
//     黑金 CJ 分支 MessageEncodeKey 注释掉的默认运行形态）。
//
//  3. 前缀式内部协议：PrefixFramer（[4B 小端长度][帧体]），对应黑金
//     FsComm 反汇编确认的内部前缀帧形态（UserSock::SendMsg 的前缀缓冲），
//     用于 proc 总线等可信链路；客户端线路用 EscapeFramer（EE 定界）。
//
//  4. 代码生成器：编解码结构由 tools/protogen 生成（规格→代码），
//     对应原版 FsVerEnc 的 gen_encode_cpp/h/imp 生成器机制；本包只提供
//     稳定的 Framer/Cipher 接缝，生成物在其上组装。
//
//  5. 线路抓取：NINEYIN_CAPTURE_DIR 指定目录时，NewConnection 自动把
//     客户端线路两方向原始字节（EE+XOR 后的 wire 流）按到达顺序追加
//     落盘为 {ip点换杠}-{端口}_c2s/_s2c.bin——与外服对拍基准
//     （captures/local）逐字节同构，可直接进解帧/对拍工具链。
//     详见 capture.go。
//
// # 帧格式速查
//
//	EscapeFramer：帧体 EE 00=字面 EE、EE EE=帧终止，无长度前缀
//	PrefixFramer：[4B LE len][body]，无转义
//
// 本包不包含任何登录或游戏消息语义（那属于 proto/login）。
package net
