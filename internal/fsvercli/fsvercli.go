// Package fsvercli 是官方 FsVerCli.dll（BGserverCJ 2014 快照，带 PDB 全符号）
// 的 Go 复刻。原 DLL 是九阴客户端侧的消息校验钩子模块（模块名 "FxClient"，
// 与服务端 FsVerify.dll 及生成工具 FsVerEnc.dll 同族），导出四个入口：
//
//	FxModule_GetType   → "FxClient"
//	FxClient_SetApi(core, send)  客户端注入 VAPI 核心与原始发送函数
//	FxClient_GetSendApi → SendMessageFunc（替换客户端发送路径）
//	FxClient_GetRecvApi → RecvMessageFunc（替换客户端接收路径）
//
// 协议语义（全部反汇编实证）：
//
//   - 传输层：EncodeMsgBuf/DecodeMsgBuf = 32 位静态 XOR（g_encode_key ==
//     g_decode_key == 0xAE5C37A5，.data 固定常量，无运行期写入者）。
//   - 发送 SendMessageFunc(sock, buf, len)：按 buf[0] 分发（1/2/4/7/10 注
//     令牌，其余直通），令牌 = m_EncodeFunc(md5(m_nDynamicKey ‖ 选定字段))，
//     写回帧内固定偏移，再整体 XOR 后交原始发送。
//   - 接收 RecvMessageFunc(sock, buf, len)：先 XOR 解包；buf[0]==2
//     （SERVER_SET_ENCODE）且 len≥0x3E 时消化帧：动态键=msg[0x21]、编码
//     id=msg[0x25]（2048 项跳表选 m_EncodeFunc）、cond16=msg[0x2d..0x3d)
//     （set_cond_value 恒等位展开）、msg[0x41..) 记录表游走（{addr,字节数,
//     dword数} 三元组 + 变长体，官方仅解析不落地），并回 0x35 字节
//     CLIENT_RET_ENCODE 帧。
//   - 应答环回：官方 RecvMessageFunc 直接调 g_send_msg_func，但客户端把
//     原始发送入口做成先经 SendMessageFunc 钩子（官方 golden 线路帧
//     login-connect-001.bin 的 op01 令牌可被 FsSoloEntry VerifyMessage 验证
//     通过即为此环回），故应答帧同样获得令牌与 XOR。
//   - login_send(pdata, len)：op02 ClientLogin 帧构建——[1:5)=版本 300、
//     [0x15]=账号槽 {u32 长度 ‖ DES 阶梯密文}、随后密码槽；键/掩码与
//     internal/fsverify 的 field1/field2 逐字节同源。
//   - encode_func_N 家族（bool 返回 + cond 分支 + 反调试诱饵）在服务端下发
//     的 cond16[id] 种子下与 FsVerify.EncodeFunc(id, in) 完全等价——
//     unicorn oracle 全量对拍 2048×3 输入零失配（tools/fsvercli_oracle.py）。
//     本包直接复用 internal/fsverify 的程序表。
package fsvercli

import (
	"crypto/md5"
	"encoding/binary"
	"strconv"

	"github.com/lizhanyin/9yin-server-go/internal/fsverify"
)

// TransportKey 对应 g_encode_key/g_decode_key 的 .data 固定初值。
const TransportKey uint32 = 0xAE5C37A5

// ClientVersion 对应 login_send 的 pmsg->nVersion = 0x12C（=FsVerify GetVersion）。
const ClientVersion uint32 = 300

// MinSetEncodeLen 对应 RecvMessageFunc 的 len≥0x3E 门。
const MinSetEncodeLen = 0x3E

// Client 模拟 FsVerCli 的全局状态机（m_nDynamicKey/m_EncodeFunc/g_msg_serial/
// 128 个 g_cond_N/g_chaos_N）。
type Client struct {
	dynamicKey  uint32
	encodeID    uint32
	encodeReady bool
	serial      uint32
	condSeed    [16]byte
	// retToken 为 op02 帧的 msg[0x29]（f28），供构建应答回显。
	retToken uint32
}

// New 建立未初始化握手的客户端（m_EncodeFunc == NULL 方言）。
func New() *Client { return &Client{} }

// ---------------------------------------------------------------------------
// 传输 XOR（EncodeMsgBuf/DecodeMsgBuf @0x10259320/0x10259410）
// ---------------------------------------------------------------------------

// EncodeMsgBuf 原地按 32 位道 XOR（dword 道对齐 + 尾字节按道内下标）。
func EncodeMsgBuf(buf []byte) { xorTransport(buf) }

// DecodeMsgBuf 与 EncodeMsgBuf 同构（官方两键同值）。
func DecodeMsgBuf(buf []byte) { xorTransport(buf) }

func xorTransport(buf []byte) {
	k := TransportKey
	key := [4]byte{byte(k), byte(k >> 8), byte(k >> 16), byte(k >> 24)}
	dwords := len(buf) / 4
	for i := 0; i < dwords; i++ {
		v := binary.LittleEndian.Uint32(buf[i*4:]) ^ TransportKey
		binary.LittleEndian.PutUint32(buf[i*4:], v)
	}
	for i := dwords * 4; i < len(buf); i++ {
		buf[i] ^= key[i-dwords*4]
	}
}

// ---------------------------------------------------------------------------
// cond 展开（set_cond_value @0x100F7280）
// ---------------------------------------------------------------------------

// SetCondValue 对应 set_cond_value(p)：16 字节种子按恒等映射展开到 128 位——
// g_cond_N = (p[N/8] >> (N%8)) & 1（LSB 先，PDB 符号位序全量核对）。
func (c *Client) SetCondValue(seed []byte) {
	copy(c.condSeed[:], seed[:16])
}

// CondBit 返回 g_cond_N（0/1）。
func (c *Client) CondBit(n int) int {
	if n < 0 || n >= 128 {
		return 0
	}
	return int(c.condSeed[n>>3]>>(n&7)) & 1
}

// ---------------------------------------------------------------------------
// 令牌（encode_func_id）
// ---------------------------------------------------------------------------

// token 对应 m_EncodeFunc(md5 摘要)：等价 fsverify.EncodeFunc(encodeID, digest)。
// m_EncodeFunc == NULL（尚未握手）时跳过注入。
func (c *Client) token(digest16 []byte) ([]byte, bool) {
	if !c.encodeReady {
		return nil, false
	}
	out, err := fsverify.EncodeFunc(int(c.encodeID), digest16)
	if err != nil {
		return nil, false
	}
	return out[:], true
}

func (c *Client) md5Preimage(preimage []byte) []byte {
	sum := md5.Sum(preimage)
	if tok, ok := c.token(sum[:]); ok {
		return tok
	}
	return nil
}

// ---------------------------------------------------------------------------
// 发送路径（SendMessageFunc @0x10259680，跳表 @0x10259C44）
// ---------------------------------------------------------------------------

// SendMessage 模拟 SendMessageFunc：按 opcode 注令牌 → EncodeMsgBuf。
// 返回线路帧（msg 原地修改）；无令牌路径仅 XOR。
func (c *Client) SendMessage(msg []byte) []byte {
	if len(msg) == 0 {
		return msg
	}
	switch msg[0] {
	case 1: // CLIENT_RET_ENCODE：preimage=[key ‖ msg[0x21] ‖ msg[1..0x21)]
		if len(msg) >= 0x35 {
			pre := make([]byte, 0, 4+4+0x20)
			pre = binary.LittleEndian.AppendUint32(pre, c.dynamicKey)
			pre = binary.LittleEndian.AppendUint32(pre, binary.LittleEndian.Uint32(msg[0x21:]))
			pre = append(pre, msg[1:0x21]...)
			if tok := c.md5Preimage(pre); tok != nil {
				copy(msg[0x25:0x35], tok)
			}
		}
	case 2: // ClientLogin：preimage=[key ‖ msg[1..0x43)]，令牌 @0x43，帧拓至 0x53
		if len(msg) >= 0x43 {
			pre := make([]byte, 0, 4+0x42)
			pre = binary.LittleEndian.AppendUint32(pre, c.dynamicKey)
			pre = append(pre, msg[1:0x43]...)
			if tok := c.md5Preimage(pre); tok != nil {
				if len(msg) < 0x53 {
					msg = append(msg, make([]byte, 0x53-len(msg))...)
				}
				copy(msg[0x43:0x53], tok)
			}
		}
	case 4: // choose_role：preimage=[key ‖ serial ‖ msg[0x1d] ‖ msg[0x15] ‖ msg[0x19)]，令牌 @1
		if len(msg) >= 0x21 {
			binary.LittleEndian.PutUint32(msg[0x11:], c.serial)
			c.serial++
			pre := make([]byte, 0, 20)
			pre = binary.LittleEndian.AppendUint32(pre, c.dynamicKey)
			pre = binary.LittleEndian.AppendUint32(pre, binary.LittleEndian.Uint32(msg[0x11:]))
			pre = binary.LittleEndian.AppendUint32(pre, binary.LittleEndian.Uint32(msg[0x1d:]))
			pre = binary.LittleEndian.AppendUint32(pre, binary.LittleEndian.Uint32(msg[0x15:]))
			pre = binary.LittleEndian.AppendUint32(pre, binary.LittleEndian.Uint32(msg[0x19:]))
			if tok := c.md5Preimage(pre); tok != nil {
				copy(msg[1:0x11], tok)
			}
		}
	case 7, 10: // select / 重置类：preimage=[key ‖ serial ‖ word@0x15 ‖ 载荷(≤8 全拷，否则首4+尾4)]
		if len(msg) >= 0x17 {
			binary.LittleEndian.PutUint32(msg[0x11:], c.serial)
			c.serial++
			pre := make([]byte, 0, 4+4+4+8)
			pre = binary.LittleEndian.AppendUint32(pre, c.dynamicKey)
			pre = binary.LittleEndian.AppendUint32(pre, binary.LittleEndian.Uint32(msg[0x11:]))
			pre = binary.LittleEndian.AppendUint32(pre, uint32(binary.LittleEndian.Uint16(msg[0x15:])))
			payloadLen := len(msg) - 0x17
			if payloadLen <= 8 {
				pre = append(pre, msg[0x17:]...)
			} else {
				pre = append(pre, msg[0x17:0x17+4]...)
				pre = append(pre, msg[len(msg)-4:]...)
			}
			if tok := c.md5Preimage(pre); tok != nil {
				copy(msg[1:0x11], tok)
			}
		}
	}
	EncodeMsgBuf(msg)
	return msg
}

// ---------------------------------------------------------------------------
// 接收路径（RecvMessageFunc @0x10259DF0）
// ---------------------------------------------------------------------------

// ClientCodeRecord 对应 op02 帧 [0x41..) 的记录表项（官方仅游走解析）：
// {目标 RVA, 跳过字节数, 尾随 dword 数} + 变长体。
type ClientCodeRecord struct {
	Addr    uint32
	SkipLen uint32
	Count   uint32
	Body    []byte
}

// RecvMessage 模拟 RecvMessageFunc：XOR 解包；op02 时更新握手状态并返回
// 应答帧（需再经 SendMessage 上线路，官方经由客户端发送钩子环回）。
// 非 op02 返回 handled=false（官方恒 return true，语义上由客户端续处理）。
func (c *Client) RecvMessage(wire []byte) (reply []byte, records []ClientCodeRecord, handled bool) {
	DecodeMsgBuf(wire)
	if len(wire) == 0 || wire[0] != 2 {
		return nil, nil, false
	}
	if len(wire) < MinSetEncodeLen {
		return nil, nil, true
	}
	// m_EncodeFunc = NULL; m_nDynamicKey = msg[0x21]
	c.encodeReady = false
	c.dynamicKey = binary.LittleEndian.Uint32(wire[0x21:])
	id := binary.LittleEndian.Uint32(wire[0x25:])
	c.retToken = binary.LittleEndian.Uint32(wire[0x29:])
	if id < fsverify.EncodeNum {
		c.encodeID = id
		c.encodeReady = true
	}
	c.SetCondValue(wire[0x2D : 0x2D+16])
	records = walkClientCode(wire[0x41:])
	// 应答 CLIENT_RET_ENCODE：[0]=1 ‖ [1..0x21)=回显 ‖ [0x21]=msg[0x29]，共 0x35
	reply = make([]byte, 0x35)
	reply[0] = 1
	copy(reply[1:0x21], wire[1:0x21])
	binary.LittleEndian.PutUint32(reply[0x21:], c.retToken)
	return reply, records, true
}

// walkClientCode 游走官方记录表：{u32 目标, u32 跳过字节数, u32 dword 数,
// skipLen 字节, count 个 dword}，直到越过帧尾（记录不得越界，截断即止）。
func walkClientCode(tail []byte) []ClientCodeRecord {
	var out []ClientCodeRecord
	p := tail
	for len(p) >= 12 {
		addr := binary.LittleEndian.Uint32(p)
		skip := binary.LittleEndian.Uint32(p[4:])
		count := binary.LittleEndian.Uint32(p[8:])
		p = p[12:]
		if uint64(skip) > uint64(len(p)) {
			break
		}
		body := p[:skip]
		p = p[skip:]
		if uint64(count)*4 > uint64(len(p)) {
			break
		}
		p = p[count*4:]
		out = append(out, ClientCodeRecord{Addr: addr, SkipLen: skip, Count: count, Body: body})
	}
	return out
}

// ---------------------------------------------------------------------------
// 登录帧（login_send @0x100F89C0）
// ---------------------------------------------------------------------------

// LoginSend 构建 op02 ClientLogin 明文帧（未注令牌、未 XOR）：
//
//	[0]=2 [1:5)=版本300 [5:0x15)零 [0x15]=u32 账号密文长 ‖ 账号槽
//	随后 u32 密码密文长 ‖ 密码槽
//
// 双槽加密即 fsverify.EncodeAccountFields（field1/field2 DES 阶梯 + 奇偶
// 掩码，客户端方向）。
func LoginSend(account, derivedPassword string) []byte {
	acc, pwd := fsverify.EncodeAccountFields(account, derivedPassword)
	msg := make([]byte, 0x15)
	msg[0] = 2
	binary.LittleEndian.PutUint32(msg[1:], ClientVersion)
	msg = binary.LittleEndian.AppendUint32(msg, uint32(len(acc)))
	msg = append(msg, acc...)
	msg = binary.LittleEndian.AppendUint32(msg, uint32(len(pwd)))
	msg = append(msg, pwd...)
	return msg
}

// String 便于调试。
func (c *Client) String() string {
	return "fsvercli.Client{key:" + strconv.FormatUint(uint64(c.dynamicKey), 16) +
		" id:" + strconv.FormatUint(uint64(c.encodeID), 10) +
		" ready:" + strconv.FormatBool(c.encodeReady) + "}"
}
