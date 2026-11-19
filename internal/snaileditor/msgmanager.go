package snaileditor

import "sync"

// MsgManager 官方 MsgManager：sMsg{type,x,data} deque + CLockUtil
// （CRITICAL_SECTION）。
type MsgManager struct {
	mu   sync.Mutex
	msgs []SMsg
}

// SMsg 官方 sMsg 布局（int type; int x; char* data——此处 data 内联为串）。
type SMsg struct {
	Type int
	X    int
	Data string
}

// NewMsgManager MsgManager::ctor。
func NewMsgManager() *MsgManager { return &MsgManager{} }

// Empty 官方 Empty。
func (m *MsgManager) Empty() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs) == 0
}

// PushMsg 官方 PushMsg(type, x, data)。
func (m *MsgManager) PushMsg(typ, x int, data string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, SMsg{Type: typ, X: x, Data: data})
}

// PopMsg 官方 PopMsg：队首取出；空返回零值 + false。
func (m *MsgManager) PopMsg() (SMsg, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.msgs) == 0 {
		return SMsg{}, false
	}
	s := m.msgs[0]
	m.msgs = m.msgs[1:]
	return s, true
}

// Len 便利：队列长度。
func (m *MsgManager) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs)
}
