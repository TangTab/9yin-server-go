package net

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
)

// memPipe 自环内存管道（测试用）：写入的字节即随后读出的字节，
// 模拟一条自环链路；读写可跨 goroutine 并发。
type memPipe struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	cond   *sync.Cond
	closed bool
}

func newLoopPipe() *memPipe {
	p := &memPipe{}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *memPipe) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for p.buf.Len() == 0 && !p.closed {
		p.cond.Wait()
	}
	if p.buf.Len() == 0 && p.closed {
		return 0, io.EOF
	}
	return p.buf.Read(b)
}

func (p *memPipe) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, errors.New("closed")
	}
	n, err := p.buf.Write(b)
	p.cond.Broadcast()
	return n, err
}

func (p *memPipe) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	p.cond.Broadcast()
	return nil
}

func TestEscapeRoundTrip(t *testing.T) {
	p := newLoopPipe()
	f, err := NewEscapeFramer(p, 1024)
	if err != nil {
		t.Fatal(err)
	}
	payloads := [][]byte{
		{0x01, 0xEE, 0x02},       // 含 EE 字面量
		{},                        // 空帧（写侧应拒绝空帧？转义帧允许空体）
		bytes.Repeat([]byte{0xEE}, 100), // 全 EE
	}
	for _, want := range payloads {
		if err := f.WriteFrame(want); err != nil {
			t.Fatalf("写 %v: %v", want, err)
		}
		got, err := f.ReadFrame()
		if err != nil {
			t.Fatalf("读: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("往返不一致: got %x want %x", got, want)
		}
	}
}

func TestEscapeWireFormat(t *testing.T) {
	// 帧体 {0xEE, 0x01} → 线路 {0xEE,0x00, 0x01, 0xEE,0xEE}
	wire := EncodeFrameBody([]byte{0xEE, 0x01})
	if !bytes.Equal(wire, []byte{0xEE, 0x00, 0x01, 0xEE, 0xEE}) {
		t.Fatalf("线路格式错误: %x", wire)
	}
}

func TestInvalidEscape(t *testing.T) {
	p := newLoopPipe()
	f, _ := NewEscapeFramer(p, 1024)
	p.Write([]byte{0x01, 0xEE, 0x05, 0xEE, 0xEE})
	if _, err := f.ReadFrame(); !errors.Is(err, ErrInvalidEscape) {
		t.Fatalf("应报非法转义: %v", err)
	}
}

func TestXorPipelineAndRotation(t *testing.T) {
	p := newLoopPipe()
	c, err := NewConnection(p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	msg1 := []byte{0xEE, 0x00, 0xFF, 0x11}
	if err := c.WriteMsg(msg1); err != nil {
		t.Fatal(err)
	}
	got1, err := c.ReadMsg()
	if err != nil || !bytes.Equal(got1, msg1) {
		t.Fatalf("默认键往返失败: %x %v", got1, err)
	}

	// 键轮换：双侧切到新键后继续收发
	newKey := NewKey(0xDEADBEEF)
	c.SetWriteKey(newKey)
	c.SetReadKey(newKey)
	msg2 := []byte("after-rotation")
	if err := c.WriteMsg(msg2); err != nil {
		t.Fatal(err)
	}
	got2, err := c.ReadMsg()
	if err != nil || !bytes.Equal(got2, msg2) {
		t.Fatalf("轮换后往返失败: %x %v", got2, err)
	}
}

func TestIdentityPlaintextMode(t *testing.T) {
	// 明文 + EE 定界：帧体明文可在线路上直接观察到（CJ 分支对拍基线）
	p := newLoopPipe()
	c, _ := NewConnection(p, Options{
		ReadCipher:  IdentityCipher{},
		WriteCipher: IdentityCipher{},
	})
	defer c.Close()

	msg := []byte{0xAA, 0xBB}
	if err := c.WriteMsg(msg); err != nil {
		t.Fatal(err)
	}
	p.mu.Lock()
	wire := append([]byte{}, p.buf.Bytes()...)
	p.mu.Unlock()
	// 线路应为 AA BB EE EE（无转义体、终止符原样）
	if !bytes.Equal(wire, []byte{0xAA, 0xBB, 0xEE, 0xEE}) {
		t.Fatalf("明文线路错误: %x", wire)
	}
	got, err := c.ReadMsg()
	if err != nil || !bytes.Equal(got, msg) {
		t.Fatalf("明文模式失败: %x %v", got, err)
	}
}

func TestPrefixFramer(t *testing.T) {
	p := newLoopPipe()
	f, _ := NewPrefixFramer(p, 1024)

	body := []byte{0xEE, 0x00, 0x01}
	if err := f.WriteFrame(body); err != nil {
		t.Fatal(err)
	}
	// 布局校验（回读消费前抓取）：[4B 小端长度=3] + body
	p.mu.Lock()
	wire := append([]byte{}, p.buf.Bytes()...)
	p.mu.Unlock()
	if !bytes.Equal(wire, append([]byte{3, 0, 0, 0}, body...)) {
		t.Fatalf("前缀布局错误: %x", wire)
	}

	got, err := f.ReadFrame()
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("前缀帧往返失败: %x %v", got, err)
	}

	// 超限与空帧拒绝
	if err := f.WriteFrame(nil); err == nil {
		t.Fatal("空帧应拒绝")
	}
	if err := f.WriteFrame(bytes.Repeat([]byte{1}, 2048)); err == nil {
		t.Fatal("超限帧应拒绝")
	}
}

func TestPrefixPlaintextConnection(t *testing.T) {
	// proc 总线目标形态：Prefix + Identity
	p := newLoopPipe()
	c, _ := NewConnection(p, Options{
		Framer:     PrefixFramerFactory,
		ReadCipher: IdentityCipher{},
		WriteCipher: IdentityCipher{},
	})
	defer c.Close()

	for i := 0; i < 3; i++ {
		msg := []byte{byte(i), 0xEE, 0xEE}
		if err := c.WriteMsg(msg); err != nil {
			t.Fatal(err)
		}
		got, err := c.ReadMsg()
		if err != nil || !bytes.Equal(got, msg) {
			t.Fatalf("第 %d 帧失败: %x %v", i, got, err)
		}
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	p := newLoopPipe()
	c, _ := NewConnection(p, Options{})
	defer c.Close()

	const n = 200
	go func() {
		for i := 0; i < n; i++ {
			msg := bytes.Repeat([]byte{byte(i), 0xEE}, 8)
			if err := c.WriteMsg(msg); err != nil {
				t.Errorf("写 %d: %v", i, err)
				return
			}
		}
	}()
	for i := 0; i < n; i++ {
		got, err := c.ReadMsg()
		if err != nil {
			t.Fatalf("读 %d: %v", i, err)
		}
		want := bytes.Repeat([]byte{byte(i), 0xEE}, 8)
		if !bytes.Equal(got, want) {
			t.Fatalf("第 %d 帧内容错乱", i)
		}
	}
}
