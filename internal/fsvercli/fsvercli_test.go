package fsvercli_test

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/fsvercli"
	"github.com/lizhanyin/9yin-server-go/internal/fsverify"
)

// 传输 XOR：dword 道 + 尾字节，Encode/Decode 互逆且与官方常量一致。
func TestTransportXor(t *testing.T) {
	cases := [][]byte{
		bytes.Repeat([]byte{0}, 64),
		bytes.Repeat([]byte{0xCC}, 17),
		{1, 2, 3, 4, 5},
	}
	for i, src := range cases {
		buf := append([]byte(nil), src...)
		want := append([]byte(nil), src...)
		fsvercli.EncodeMsgBuf(buf)
		if bytes.Equal(buf, want) && len(buf) > 0 {
			t.Fatalf("case %d: xor 无效果", i)
		}
		// 手算 dword 道
		if len(buf) >= 4 {
			v := binary.LittleEndian.Uint32(want) ^ 0xAE5C37A5
			if got := binary.LittleEndian.Uint32(buf); got != v {
				t.Fatalf("case %d: dword 道 = %08x want %08x", i, got, v)
			}
		}
		fsvercli.DecodeMsgBuf(buf)
		if !bytes.Equal(buf, want) {
			t.Fatalf("case %d: 往返失真", i)
		}
	}
}

// SetCondValue 恒等 LSB 位展开（PDB 符号位序全量核对结论）。
func TestSetCondValueIdentity(t *testing.T) {
	c := fsvercli.New()
	seed := []byte{0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80, 0xFF, 0x00, 0xAA, 0x55, 0x0F, 0xF0, 0x99, 0x66}
	c.SetCondValue(seed)
	for n := 0; n < 128; n++ {
		want := int(seed[n/8]>>(n%8)) & 1
		if got := c.CondBit(n); got != want {
			t.Fatalf("cond_%d = %d want %d", n, got, want)
		}
	}
}

// 完整握手：服务端按 fsverify 表下发 SERVER_SET_ENCODE → 客户端应答
// CLIENT_RET_ENCODE，令牌必须等于 EncodeFunc(id, [key‖f28‖echo])。
func TestHandshakeReplyToken(t *testing.T) {
	for _, id := range []uint32{0, 1, 7, 100, 2047} {
		cond, _ := fsverify.ClientCond(id)
		dynamicKey := uint32(0x11223344)
		f28 := uint32(0xDEADBEEF)

		frame := make([]byte, 0x4D)
		frame[0] = 2
		copy(frame[1:0x21], bytes.Repeat([]byte{0xA5}, 0x20)) // 回显区
		binary.LittleEndian.PutUint32(frame[0x21:], dynamicKey)
		binary.LittleEndian.PutUint32(frame[0x25:], id)
		binary.LittleEndian.PutUint32(frame[0x29:], f28)
		copy(frame[0x2D:0x3D], cond)
		copy(frame[0x3D:0x4D], bytes.Repeat([]byte{0x5C}, 16)) // code16（记录表起点 0x41 落在 code 区，游走按越界截断）

		wire := append([]byte(nil), frame...)
		fsvercli.EncodeMsgBuf(wire)

		c := fsvercli.New()
		reply, records, handled := c.RecvMessage(wire)
		if !handled {
			t.Fatalf("id %d: op02 未被消化", id)
		}
		if len(records) != 0 {
			t.Fatalf("id %d: 空 记录表应游走 0 项", id)
		}
		if reply[0] != 1 || len(reply) != 0x35 {
			t.Fatalf("id %d: 应答帧形态错", id)
		}
		if !bytes.Equal(reply[1:0x21], frame[1:0x21]) || binary.LittleEndian.Uint32(reply[0x21:]) != f28 {
			t.Fatalf("id %d: 应答回显错", id)
		}
		// 环回经 SendMessage 注令牌。官方语义：m_EncodeFunc(md5 摘要) ——
		// 即 encode_func_id(md5(preimage))，encode_func 内部还会再 md5 一次。
		out := c.SendMessage(reply)
		fsvercli.DecodeMsgBuf(out)
		pre := make([]byte, 0, 40)
		pre = binary.LittleEndian.AppendUint32(pre, dynamicKey)
		pre = binary.LittleEndian.AppendUint32(pre, f28)
		pre = append(pre, frame[1:0x21]...)
		digest := md5.Sum(pre)
		want, err := fsverify.EncodeFunc(int(id), digest[:])
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out[0x25:0x35], want[:]) {
			t.Fatalf("id %d: 令牌 %x want %x", id, out[0x25:0x35], want)
		}
	}
}

// 登录帧：版本 300 + 双槽密文 = fsverify.EncodeAccountFields（服务器
// DecodeAccount/DecodePassword 应解回原文）。
func TestLoginSendRoundtrip(t *testing.T) {
	const account = "test001"
	const pwd = "drUGtwMnNPbVE9PQ=="
	msg := fsvercli.LoginSend(account, pwd)
	if msg[0] != 2 {
		t.Fatalf("opcode = %d", msg[0])
	}
	if v := binary.LittleEndian.Uint32(msg[1:]); v != 300 {
		t.Fatalf("版本 = %d want 300", v)
	}
	accLen := binary.LittleEndian.Uint32(msg[0x15:])
	acc := msg[0x19 : 0x19+accLen]
	pwdLen := binary.LittleEndian.Uint32(msg[0x19+accLen:])
	pwdSlot := msg[0x19+accLen+4:]
	if int(pwdLen) != len(pwdSlot) {
		t.Fatalf("密码槽长 %d != %d", pwdLen, len(pwdSlot))
	}
	if got := fsverify.DecodeAccount(acc); got != account {
		t.Fatalf("账号解出 %q want %q", got, account)
	}
	if got := fsverify.DecodePassword(pwdSlot); got != pwd {
		t.Fatalf("密码解出 %q want %q", got, pwd)
	}
}

// 未握手时发送不注令牌（m_EncodeFunc == NULL 方言）。
func TestSendWithoutHandshakeSkipsToken(t *testing.T) {
	c := fsvercli.New()
	msg := make([]byte, 0x35)
	msg[0] = 1
	copy(msg[1:0x21], bytes.Repeat([]byte{0x11}, 0x20))
	orig := append([]byte(nil), msg...)
	out := c.SendMessage(msg)
	fsvercli.DecodeMsgBuf(out)
	if !bytes.Equal(out[0x25:0x35], orig[0x25:0x35]) {
		t.Fatal("未握手却改写了令牌区")
	}
}

// 记录表游走：{addr,skip,count} + 变长体，越界截断。
func TestWalkClientCodeRecords(t *testing.T) {
	c := fsvercli.New()
	frame := make([]byte, 0x60)
	frame[0] = 2
	binary.LittleEndian.PutUint32(frame[0x21:], 1)
	binary.LittleEndian.PutUint32(frame[0x25:], 3)
	// 记录 @0x41：addr=0x401000 skip=8 count=2 → 12+8+8=28 字节
	binary.LittleEndian.PutUint32(frame[0x41:], 0x401000)
	binary.LittleEndian.PutUint32(frame[0x45:], 8)
	binary.LittleEndian.PutUint32(frame[0x49:], 2)
	for i := 0; i < 16; i++ {
		frame[0x4D+i] = byte(i)
	}
	fsvercli.EncodeMsgBuf(frame)
	_, records, handled := c.RecvMessage(frame)
	if !handled {
		t.Fatal("op02 未消化")
	}
	if len(records) != 1 {
		t.Fatalf("记录数 %d want 1", len(records))
	}
	r := records[0]
	if r.Addr != 0x401000 || r.SkipLen != 8 || r.Count != 2 || !bytes.Equal(r.Body, frame[0x4D:0x55]) {
		t.Fatalf("记录解析错: %+v", r)
	}
}
