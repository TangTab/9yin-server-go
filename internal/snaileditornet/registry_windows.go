//go:build windows

package snaileditornet

import (
	"strings"
	"syscall"
	"unsafe"
)

// readRegistryString HKLM 路径字符串值（官方 ReadRegValue 等价，
// advapi32 直调：RegOpenKeyExW + RegQueryValueExW，REG_SZ/REG_EXPAND_SZ）。
func readRegistryString(path, name string) (string, bool) {
	adv := syscall.NewLazyDLL("advapi32.dll")
	open := adv.NewProc("RegOpenKeyExW")
	query := adv.NewProc("RegQueryValueExW")
	closek := adv.NewProc("RegCloseKey")

	const hklm = 0x80000002 // HKEY_LOCAL_MACHINE
	const read = 0x20019    // KEY_READ
	var h uintptr
	pk, _ := syscall.UTF16PtrFromString(path)
	if r, _, _ := open.Call(hklm, uintptr(unsafe.Pointer(pk)), 0, read, uintptr(unsafe.Pointer(&h))); r != 0 {
		return "", false
	}
	defer closek.Call(h)

	pn, _ := syscall.UTF16PtrFromString(name)
	var typ, size uint32
	if r, _, _ := query.Call(h, uintptr(unsafe.Pointer(pn)), 0,
		uintptr(unsafe.Pointer(&typ)), 0, uintptr(unsafe.Pointer(&size))); r != 0 || size == 0 {
		return "", false
	}
	if typ != 1 && typ != 2 { // REG_SZ / REG_EXPAND_SZ
		return "", false
	}
	buf := make([]uint16, size/2+1)
	if r, _, _ := query.Call(h, uintptr(unsafe.Pointer(pn)), 0,
		uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size))); r != 0 {
		return "", false
	}
	s := syscall.UTF16ToString(buf)
	return strings.TrimSpace(s), true
}
