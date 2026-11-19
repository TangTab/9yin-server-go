package fssololog

import (
	"encoding/binary"
	"fmt"
)

// stdPrintf 官方 printf 面缺省：本进程控制台。
func stdPrintf(format string, args ...any) { fmt.Printf(format, args...) }

// 官方 CLogShareMem 槽协议定案（FsSoloLog全解 §4）：
// Create("memory_fs_solo_log_%d", 0x202300)——块 id 恒 0 → 名字固定
// memory_fs_solo_log_0；0x808C × 64 槽整除无头部。
const (
	// LogShareSize 共享内存总大小（官方 0x202300）。
	LogShareSize = 0x202300
	// LogSlotCount appid 槽数（64）。
	LogSlotCount = 64
	// LogSlotSize 每槽字节数（0x808C）。
	LogSlotSize = 0x808C
	// LogPrefixSize 槽首进程名上限（0x80；[0]!=0 ⟺ 槽在用）。
	LogPrefixSize = 0x80
	// LogLineCount 环形行仓行数（128）。
	LogLineCount = 128
	// LogLineSize 每行字节数（0x100，含 NUL）。
	LogLineSize = 0x100
	// LogLineMask 游标掩码（行号 = 游标 & 0x7F）。
	LogLineMask = LogLineCount - 1

	// 槽内偏移：
	slotPrefix   = 0    // char  prefix[0x80]
	slotCreateID = 0x80 // u32   create_new（生产者请求换档）
	slotWriteCur = 0x84 // u32   write_cursor（生产者游标，0..127 环）
	slotReadCur  = 0x88 // u32   read_cursor（消费者游标，0..127 环）
	slotLines    = 0x8C // char  lines[128][0x100]

	// DefaultLogShareName 官方块名（块 id 恒 0）。
	DefaultLogShareName = "memory_fs_solo_log_0"
)

// LogShareMem 复刻 CLogShareMem：生产、消费两侧共用本类（官方同码；
// 生产侧见 FsSoloGlobal 等各 DLL 的同款代码）。零值不可用，经
// NewLogShareMemInMemory / OpenLogShareMem / CloneView 构造。
type LogShareMem struct {
	name   string
	data   []byte
	appID  int
	close  func() // Windows 映射清理（in-memory 为 nil）
	printf func(format string, args ...any)
}

// NewLogShareMemInMemory 建进程内共享缓冲（非 Windows 回退形态；
// 布局/游标协议与官方逐字对齐，仅无跨进程能力）。
func NewLogShareMemInMemory() *LogShareMem {
	return &LogShareMem{
		name:   DefaultLogShareName,
		data:   make([]byte, LogShareSize),
		appID:  -1,
		printf: stdPrintf,
	}
}

// OpenLogShareMem 打开/创建官方同名 Windows 命名共享内存
// （CreateFileMappingW，PAGE_READWRITE，0x202300）。非 Windows 平台
// 返回错误（用 NewLogShareMemInMemory 兜底）。可与官方 game_*.exe
// 生产进程互操作（混合部署对拍）。
func OpenLogShareMem(name string) (*LogShareMem, error) {
	if name == "" {
		name = DefaultLogShareName
	}
	data, closeF, err := mapSharedMemory(name, LogShareSize)
	if err != nil {
		return nil, err
	}
	return &LogShareMem{name: name, data: data, appID: -1, close: closeF, printf: stdPrintf}, nil
}

// CloneView 以同一底层缓冲再开一个视图（模拟另一进程的 Attach；
// 测试与进程内生产/消费分离用）。
func (m *LogShareMem) CloneView() *LogShareMem {
	return &LogShareMem{name: m.name, data: m.data, appID: -1, printf: stdPrintf}
}

// Close 释放映射（in-memory 形态 no-op）。官方 Stop+delete。
func (m *LogShareMem) Close() {
	if m != nil && m.close != nil {
		m.close()
		m.close = nil
	}
}

// SetPrintf 注入 printf 面（官方 TraceLog 的本地控制台回显；
// 测试改向捕获用）。nil 恢复 fmt.Printf。
func (m *LogShareMem) SetPrintf(f func(format string, args ...any)) {
	if f == nil {
		f = stdPrintf
	}
	m.printf = f
}

// —— 生产者接口（业务进程侧，官方 §4.3）——

// SetAppId 选择本进程写入槽（官方断言 0 <= v < 0x40）。
func (m *LogShareMem) SetAppId(v int) {
	if m == nil || m.data == nil || v < 0 || v >= LogSlotCount {
		return
	}
	m.appID = v
}

// SetLogPrefix 写槽首进程名（≤0x7F+NUL）——即"注册本槽"。
func (m *LogShareMem) SetLogPrefix(s string) {
	if m == nil || m.data == nil || m.appID < 0 {
		return
	}
	copyStr(m.slot(m.appID)[slotPrefix:slotPrefix+LogPrefixSize], s)
}

// SetCreateNew 请求消费侧换新文件（写槽 +0x80）。
func (m *LogShareMem) SetCreateNew(v uint32) {
	if m == nil || m.data == nil || m.appID < 0 {
		return
	}
	binary.LittleEndian.PutUint32(m.slot(m.appID)[slotCreateID:], v)
}

// ClearLogUsed 清零槽首（"注销本槽"）。
func (m *LogShareMem) ClearLogUsed() {
	if m == nil || m.data == nil || m.appID < 0 {
		return
	}
	m.slot(m.appID)[slotPrefix] = 0
}

// GetLogUsed 报告槽 idx 是否在用（prefix[0] != 0）。
func (m *LogShareMem) GetLogUsed(idx int) bool {
	if m == nil || m.data == nil || idx < 0 || idx >= LogSlotCount {
		return false
	}
	return m.slot(idx)[slotPrefix] != 0
}

// TraceLog 生产一行日志（官方语义 @TraceLog）：
//
//	next = (write+1)&0x7F；next == read → 环满丢行返回 false；
//	否则拷入行仓 lines[write]（≤0xFF+NUL）、write = next、
//	printf 本地控制台回显、返回 true。
func (m *LogShareMem) TraceLog(info string) bool {
	if m == nil || m.data == nil || m.appID < 0 {
		return false
	}
	slot := m.slot(m.appID)
	w := binary.LittleEndian.Uint32(slot[slotWriteCur:])
	r := binary.LittleEndian.Uint32(slot[slotReadCur:])
	next := (w + 1) & LogLineMask
	if next == r {
		return false // 环满丢行
	}
	copyStr(slot[slotLines+int(w&LogLineMask)*LogLineSize:][:LogLineSize], info)
	binary.LittleEndian.PutUint32(slot[slotWriteCur:], next)
	if m.printf != nil {
		m.printf("%s\n", info)
	}
	return true
}

// TraceFunc 静态钩子适配（官方 TraceFunc(ctx, msg) = TraceLog(msg)，
// 挂全局日志钩子后业务进程所有 CORE_TRACE 自动入槽）。
func (m *LogShareMem) TraceFunc(msg string) { m.TraceLog(msg) }

// —— 消费者接口（SoloLog::CheckLog 侧）——

// SlotUsed 同 GetLogUsed（消费侧视角命名）。
func (m *LogShareMem) SlotUsed(idx int) bool { return m.GetLogUsed(idx) }

// SlotPrefix 槽 idx 的进程名（NUL 截断）。
func (m *LogShareMem) SlotPrefix(idx int) string {
	if m == nil || m.data == nil || idx < 0 || idx >= LogSlotCount {
		return ""
	}
	return readStr(m.slot(idx)[slotPrefix : slotPrefix+LogPrefixSize])
}

// SlotCreateNew 读槽 idx 的换档请求位。
func (m *LogShareMem) SlotCreateNew(idx int) uint32 {
	if m == nil || m.data == nil || idx < 0 || idx >= LogSlotCount {
		return 0
	}
	return binary.LittleEndian.Uint32(m.slot(idx)[slotCreateID:])
}

// ClearSlotCreateNew 消费侧清换档请求位（§5.1 create_new 分支）。
func (m *LogShareMem) ClearSlotCreateNew(idx int) {
	if m == nil || m.data == nil || idx < 0 || idx >= LogSlotCount {
		return
	}
	binary.LittleEndian.PutUint32(m.slot(idx)[slotCreateID:], 0)
}

// WriteCursor / ReadCursor 槽游标（0..127 环）。
func (m *LogShareMem) WriteCursor(idx int) uint32 {
	if m == nil || m.data == nil || idx < 0 || idx >= LogSlotCount {
		return 0
	}
	return binary.LittleEndian.Uint32(m.slot(idx)[slotWriteCur:])
}

func (m *LogShareMem) ReadCursor(idx int) uint32 {
	if m == nil || m.data == nil || idx < 0 || idx >= LogSlotCount {
		return 0
	}
	return binary.LittleEndian.Uint32(m.slot(idx)[slotReadCur:])
}

// SetReadCursor 消费侧推进读游标（&0x7F 环回）。
func (m *LogShareMem) SetReadCursor(idx int, v uint32) {
	if m == nil || m.data == nil || idx < 0 || idx >= LogSlotCount {
		return
	}
	binary.LittleEndian.PutUint32(m.slot(idx)[slotReadCur:], v&LogLineMask)
}

// Line 取槽 idx 行仓中第 line 行（0..127；NUL 截断，空串=空行）。
func (m *LogShareMem) Line(idx int, line uint32) string {
	if m == nil || m.data == nil || idx < 0 || idx >= LogSlotCount {
		return ""
	}
	off := slotLines + int(line&LogLineMask)*LogLineSize
	return readStr(m.slot(idx)[off : off+LogLineSize])
}

// —— 布局辅助 ——

// slot 返回槽 idx 的完整窗口（0x808C 字节）。
func (m *LogShareMem) slot(idx int) []byte {
	return m.data[idx*LogSlotSize : (idx+1)*LogSlotSize]
}

// copyStr 拷贝 s 到 dst（清零后写入，NUL 终结，len(dst)-1 内截断）。
// 官方 CopyString：超限截断且保证 NUL。
func copyStr(dst []byte, s string) {
	b := []byte(s)
	if len(b) > len(dst)-1 {
		b = b[:len(dst)-1]
	}
	for i := range dst {
		dst[i] = 0
	}
	copy(dst, b)
}

// readStr 读 NUL 截断串（窗口即上限）。
func readStr(win []byte) string {
	end := 0
	for end < len(win) && win[end] != 0 {
		end++
	}
	return string(win[:end])
}
