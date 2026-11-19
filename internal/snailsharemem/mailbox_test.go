package snailsharemem

import (
	"bytes"
	"math/rand"
	"sync"
	"testing"
)

// 协议单测：与官方 SharingMemory.dll 语义逐条对齐。

func TestMailboxLayoutConstants(t *testing.T) {
	if BufferMaxSize != 0x800 || MaxCountReader != 0x20 || MapSize != 0x814 {
		t.Fatalf("官方常量漂移: %x %x %x", BufferMaxSize, MaxCountReader, MapSize)
	}
	if MapNamePrefix != "SNAILEDITOR_" {
		t.Fatalf("映射名前缀不符: %q", MapNamePrefix)
	}
}

func TestMailboxSingleFrameHandshake(t *testing.T) {
	m := NewMailbox()
	// 初始：不可读、可写、无读者
	if m.GetReadable() || !m.GetWritable() || m.GetBufferSize() != 0 {
		t.Fatal("初始态不符")
	}
	id, err := m.CreateReaderID()
	if err != nil || id != 0 {
		t.Fatalf("首个读者应为 0: %d %v", id, err)
	}
	// 写帧（官方 ASSERT 有读者——此处有 1 个）
	payload := []byte("hello editor")
	n, err := m.Write(payload)
	if err != nil || n != len(payload) {
		t.Fatalf("Write: %d %v", n, err)
	}
	if m.GetWritable() || !m.GetReadable() {
		t.Fatal("写后 valid=1 应不可写可读")
	}
	if got := m.Read(id); !bytes.Equal(got, payload) {
		t.Fatalf("Read 不符: %q", got)
	}
	if m.GetWritable() {
		t.Fatal("ReadEnd 前不可写")
	}
	m.ReadEnd(id)
	if m.GetReadable() || !m.GetWritable() {
		t.Fatal("唯一读者 ReadEnd 后应回到可写")
	}
	if !m.GetAllUserReaded() {
		t.Fatal("GetAllUserReaded 应为真")
	}
}

func TestMailboxMultiReaderIsolation(t *testing.T) {
	m := NewMailbox()
	var ids []byte
	for i := 0; i < 5; i++ {
		id, err := m.CreateReaderID()
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if m.RegisteredCount() != 5 {
		t.Fatalf("注册数 %d", m.RegisteredCount())
	}
	if _, err := m.Write([]byte("frame")); err != nil {
		t.Fatal(err)
	}
	// 未注册读者不可读
	if m.Read(6) != nil {
		t.Fatal("未注册读者不应读到")
	}
	// 逐个读完：前 4 个读完仍不可写
	for i, id := range ids[:4] {
		if m.Read(id) == nil {
			t.Fatal("注册读者应可读")
		}
		m.ReadEnd(id)
		if m.GetWritable() {
			t.Fatalf("第 %d 个读完不应可写", i)
		}
	}
	m.ReadEnd(ids[4])
	if !m.GetWritable() {
		t.Fatal("全部读完应可写")
	}
	// FreeReaderID 后新帧不含该读者
	m.FreeReaderID(ids[4])
	m.Write([]byte("second"))
	if m.Read(ids[4]) != nil {
		t.Fatal("已释放读者不应可读")
	}
	m.ReadEnd(ids[4]) // 无效 ReadEnd（未注册/未可读）应安全
	m.ReadEnd(ids[0])
	m.ReadEnd(ids[1])
	m.ReadEnd(ids[2])
	m.ReadEnd(ids[3])
	if !m.GetWritable() {
		t.Fatal("清算失败")
	}
}

func TestMailboxReaderSlotExhaustion(t *testing.T) {
	m := NewMailbox()
	for i := 0; i < 32; i++ {
		if _, err := m.CreateReaderID(); err != nil {
			t.Fatalf("第 %d 个注册失败", i)
		}
	}
	if _, err := m.CreateReaderID(); err != ErrNoReaderSlot {
		t.Fatalf("应报满: %v", err)
	}
}

func TestMailboxOversizeTruncation(t *testing.T) {
	m := NewMailbox()
	id, _ := m.CreateReaderID()
	big := make([]byte, BufferMaxSize+100)
	rand.Read(big)
	n, _ := m.Write(big)
	if n != BufferMaxSize {
		t.Fatalf("应截断 0x800: %x", n)
	}
	if got := m.Read(id); len(got) != BufferMaxSize || !bytes.Equal(got, big[:BufferMaxSize]) {
		t.Fatal("截断内容不符")
	}
}

func TestMailboxSequentialFrames(t *testing.T) {
	m := NewMailbox()
	id, _ := m.CreateReaderID()
	for round := 0; round < 100; round++ {
		frame := []byte{byte(round), byte(round >> 8)}
		if _, err := m.Write(frame); err != nil {
			t.Fatal(err)
		}
		got := m.Read(id)
		if got[0] != byte(round) {
			t.Fatalf("round %d 帧错位", round)
		}
		m.ReadEnd(id)
	}
}

func TestMailboxConcurrentReaders(t *testing.T) {
	m := NewMailbox()
	const R = 8
	ids := make([]byte, R)
	for i := range ids {
		id, err := m.CreateReaderID()
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = id
	}
	var wg sync.WaitGroup
	for i := 0; i < R; i++ {
		wg.Add(1)
		go func(id byte) {
			defer wg.Done()
			for round := 0; round < 50; round++ {
				for m.Read(id) == nil {
				}
				m.ReadEnd(id)
			}
		}(ids[i])
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for round := 0; round < 50; round++ {
			for !m.GetWritable() {
			}
			m.Write([]byte{byte(round)})
		}
	}()
	wg.Wait()
}
