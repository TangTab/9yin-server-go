//go:build windows

package fssololog

import (
	"encoding/binary"
	"testing"
)

// TestOpenLogShareMemWindows 真命名映射回归（Win11 24H2+ 上
// FILE_MAP_ALL_ACCESS 掩码坑的哨兵）：同名双视图互通 + 全尺寸可写。
func TestOpenLogShareMemWindows(t *testing.T) {
	m1, err := OpenLogShareMem("memory_fs_solo_log_probe_test_0")
	if err != nil {
		t.Fatal(err)
	}
	defer m1.Close()
	b := m1.data
	t.Logf("data len=%d cap=%d", len(b), cap(b))
	binary.LittleEndian.PutUint32(b[slotWriteCur:], 7)
	if got := binary.LittleEndian.Uint32(b[slotWriteCur:]); got != 7 {
		t.Fatalf("写读不符：%d", got)
	}
	// 第二视图（同名，模拟生产进程）。
	m2, err := OpenLogShareMem("memory_fs_solo_log_probe_test_0")
	if err != nil {
		t.Fatal(err)
	}
	defer m2.Close()
	if got := binary.LittleEndian.Uint32(m2.data[slotWriteCur:]); got != 7 {
		t.Fatalf("跨视图读取不符：%d", got)
	}
	// 尾槽写入（越界即崩 → 证明视图全尺寸）。
	binary.LittleEndian.PutUint32(b[(LogSlotCount-1)*LogSlotSize+slotWriteCur:], 9)
	if got := binary.LittleEndian.Uint32(m2.data[(LogSlotCount-1)*LogSlotSize+slotWriteCur:]); got != 9 {
		t.Fatalf("尾槽不符：%d", got)
	}
}
