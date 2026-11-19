package snailsharemem

import (
	"encoding/binary"
	"errors"
	"sync"
)

// 官方常量（SharingMemory.dll 定案）。
const (
	// BufferMaxSize 单帧载荷缓冲 0x800（GetBufferMaxSize）。
	BufferMaxSize = 0x800
	// MaxCountReader 读者位图宽 0x20（GetMaxCountReader）。
	MaxCountReader = 0x20
	// MapSize 映射总大小 0x814 = 0x14 头尾字段 + 0x800 缓冲。
	MapSize = 0x800 + 0x14

	offData     = 0x004 // data[0x800]
	offDataLen  = 0x804 // uint32 当前帧长
	offValid    = 0x808 // byte   有新帧
	offReadable = 0x80C // uint32 可读位图
	offRegisted = 0x810 // uint32 注册位图

	// MapNamePrefix 官方映射名前缀（L"SNAILEDITOR_%s"）。
	MapNamePrefix = "SNAILEDITOR_"
)

// ErrNoReaderSlot 读者位满（CreateReaderID 置 0xFF 返回假）。
var ErrNoReaderSlot = errors.New("snailsharemem: reader slots exhausted (32)")

// ErrIllegalUserID CheckIllegalUserID：id < 0 || id >= 0x20。
type errIllegalUserID struct{ id byte }

func (e errIllegalUserID) Error() string { return "snailsharemem: illegal reader id" }

// Mailbox 官方 CSharingData 的布局兼容实现。
// 零值不可用；用 NewMailbox（进程内）或映射层（跨进程）构造。
// data 指向 0x814 字节区域（共享内存或堆），mu 仅进程内多 goroutine 防竞
// （跨进程语义仍靠 valid/位图字段，与官方一致——官方无锁）。
type Mailbox struct {
	data []byte
	mu   sync.Mutex
}

// NewMailbox 进程内信箱：按官方 ctor 语义零初始化（data 清零、
// dataLen=0、valid=0、readable=registered=0）。
func NewMailbox() *Mailbox {
	m := &Mailbox{data: make([]byte, MapSize)}
	m.initHeader()
	return m
}

// initHeader 复刻 CSharingData ctor（memset(this+4,0,0x800) + 尾字段清零）。
func (m *Mailbox) initHeader() {
	for i := offData; i < offData+BufferMaxSize; i++ {
		m.data[i] = 0
	}
	m.setValid(0)
	binary.LittleEndian.PutUint32(m.data[offDataLen:], 0)
	binary.LittleEndian.PutUint32(m.data[offReadable:], 0)
	binary.LittleEndian.PutUint32(m.data[offRegisted:], 0)
}

// attach 挂接已存在的 0x814 区域（映射层用）。不重置内容。
func attach(b []byte) (*Mailbox, error) {
	if len(b) < MapSize {
		return nil, errors.New("snailsharemem: backing region < 0x814")
	}
	return &Mailbox{data: b[:MapSize]}, nil
}

func (m *Mailbox) valid() byte     { return m.data[offValid] }
func (m *Mailbox) setValid(v byte) { m.data[offValid] = v }

func (m *Mailbox) readableBitmap() uint32 { return binary.LittleEndian.Uint32(m.data[offReadable:]) }
func (m *Mailbox) registedBitmap() uint32 { return binary.LittleEndian.Uint32(m.data[offRegisted:]) }

func checkBit(bm uint32, id byte) bool  { return id < 32 && bm&(1<<id) != 0 }
func setBit(bm uint32, id byte) uint32  { return bm | (1 << id) }
func zeroBit(bm uint32, id byte) uint32 { return bm &^ (1 << id) }

// GetBufferMaxSize 官方 GetBufferMaxSize()=0x800。
func (m *Mailbox) GetBufferMaxSize() int { return BufferMaxSize }

// GetBufferSize 官方 GetBufferSize()=当前帧长。
func (m *Mailbox) GetBufferSize() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int(binary.LittleEndian.Uint32(m.data[offDataLen:]))
}

// GetMaxCountReader 官方 GetMaxCountReader()=0x20。
func (m *Mailbox) GetMaxCountReader() int { return MaxCountReader }

// GetWritable 官方 GetWritable() == (valid==0)。
func (m *Mailbox) GetWritable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.valid() == 0
}

// GetReadable 官方 GetReadable() == (valid!=0)。
func (m *Mailbox) GetReadable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.valid() != 0
}

// CreateReaderID 官方语义：扫 0..0x1F 找注册位图首个空位，置位返回；
// 满则返回 ErrNoReaderSlot（官方 *id=0xFF / 返回假）。
func (m *Mailbox) CreateReaderID() (byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id := 0; id < MaxCountReader; id++ {
		if !checkBit(m.registedBitmap(), byte(id)) {
			binary.LittleEndian.PutUint32(m.data[offRegisted:], setBit(m.registedBitmap(), byte(id)))
			return byte(id), nil
		}
	}
	return 0xFF, ErrNoReaderSlot
}

// FreeReaderID 官方 FreeReaderID：清注册位。
func (m *Mailbox) FreeReaderID(id byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	binary.LittleEndian.PutUint32(m.data[offRegisted:], zeroBit(m.registedBitmap(), id))
}

// Write 官方 CSharingData::Write(buf,len)：
// ASSERT(有读者注册)；len 截 0x800；写 dataLen+data；
// readableBitmap=registeredBitmap；valid=1；返回实际帧长。
// 无读者注册时按官方断言语义返回错误（release 版仅弹 _CrtDbgReport 仍写）。
func (m *Mailbox) Write(buf []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := len(buf)
	if n > BufferMaxSize {
		n = BufferMaxSize
	}
	binary.LittleEndian.PutUint32(m.data[offDataLen:], uint32(n))
	copy(m.data[offData:offData+n], buf[:n])
	// SetAllUserReadable: readable = registered; valid = 1
	binary.LittleEndian.PutUint32(m.data[offReadable:], m.registedBitmap())
	m.setValid(1)
	return n, nil
}

// Read 官方 CSharingData::Read(id)：前置 valid/注册位/可读位三查；
// 通过则拷出当前帧（官方返回 data+4 指针，此处拷贝以避免暴露内部切片）。
// 未就绪返回 nil。
func (m *Mailbox) Read(id byte) []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.valid() == 0 {
		return nil
	}
	if !checkBit(m.registedBitmap(), id) || !checkBit(m.readableBitmap(), id) {
		return nil
	}
	n := int(binary.LittleEndian.Uint32(m.data[offDataLen:]))
	if n > BufferMaxSize {
		n = BufferMaxSize
	}
	out := make([]byte, n)
	copy(out, m.data[offData:offData+n])
	return out
}

// ReadEnd 官方 CSharingData::ReadEnd(id)：同 Read 前置；
// 清自身可读位；若 readableBitmap==0（GetAllUserReaded）则 valid=0。
func (m *Mailbox) ReadEnd(id byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.valid() == 0 {
		return
	}
	if !checkBit(m.registedBitmap(), id) || !checkBit(m.readableBitmap(), id) {
		return
	}
	binary.LittleEndian.PutUint32(m.data[offReadable:], zeroBit(m.readableBitmap(), id))
	if m.readableBitmap() == 0 {
		m.setValid(0)
	}
}

// GetAllUserReaded 官方语义：readableBitmap == 0。
func (m *Mailbox) GetAllUserReaded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readableBitmap() == 0
}

// RegisteredCount 便利方法：注册读者数（位计数）。
func (m *Mailbox) RegisteredCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return popcount(m.registedBitmap())
}

func popcount(v uint32) int {
	n := 0
	for ; v != 0; v &= v - 1 {
		n++
	}
	return n
}
