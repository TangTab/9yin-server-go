package fscomm

import (
	"errors"
	"sync"
)

// ErrFIFOFull 对齐官方 "(Adapter::ClientSend)put message error" /
// "(UserSock::SendMsg)send buffer overflow" 的满队列语义。
var ErrFIFOFull = errors.New("fscomm: sock fifo full")

// msgFIFO 复刻 CSockFIFO（sockfifo.cpp）：消息队列，双上限
// （条数 MaxDataNum / 字节数 MaxBufLen）封顶，Put 满即拒收。
// 官方为固定环 + 大缓冲池；Go 以切片队列传拷贝值，语义等价。
// 官方另有 ShrinkDataNum/ShrinkBufLen 收缩策略，GC 天然回收，不复刻。
type msgFIFO struct {
	mu       sync.Mutex
	q        [][]byte
	bufBytes int
	maxNum   int // 官方 MaxDataNum（≤0 = 不限条数）
	maxBytes int // 官方 MaxBufLen（≤0 = 不限字节）
}

// newMsgFIFO 建队列。
func newMsgFIFO(maxNum, maxBytes int) *msgFIFO {
	return &msgFIFO{maxNum: maxNum, maxBytes: maxBytes}
}

// Put 入队（拷贝语义，对齐官方 memcpy 入池）。满返回 ErrFIFOFull。
func (f *msgFIFO) Put(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if (f.maxNum > 0 && len(f.q) >= f.maxNum) ||
		(f.maxBytes > 0 && f.bufBytes+len(data) > f.maxBytes) {
		return ErrFIFOFull
	}
	b := make([]byte, len(data))
	copy(b, data)
	f.q = append(f.q, b)
	f.bufBytes += len(b)
	return nil
}

// Peek 取队首（不出队）；空返回 nil。
func (f *msgFIFO) Peek() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.q) == 0 {
		return nil
	}
	return f.q[0]
}

// Pop 出队队首；空返回 nil。
func (f *msgFIFO) Pop() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.q) == 0 {
		return nil
	}
	s := f.q[0]
	f.q[0] = nil
	f.q = f.q[1:]
	f.bufBytes -= len(s)
	return s
}

// Len 在队条数。
func (f *msgFIFO) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.q)
}

// Bytes 在队字节数。
func (f *msgFIFO) Bytes() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.bufBytes
}
