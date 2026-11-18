// Package net 实现通讯层，对应原版 FsComm：TCP 监听/连接、封包帧、
// 编码/校验/加密。
//
// 已知原版线索（待对拍确认细节）：
//   - game_unite.ini 的 MessageEncodeKey（通讯编码加密键，须与客户端一致）
//   - MessageVerifyDll=fsverify.dll（校验模块挂载点）
//   - 内网/外网/广播三组地址配置段
//
// 职责边界：本包只管字节流 ↔ 消息帧，不认识消息号语义（那是 proto 的事），
// 也不管理连接会话状态（那是 engine/session 的事）。
package net
