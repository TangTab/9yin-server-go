//go:build windows

package fssololog

import (
	"syscall"
	"unsafe"
)

// acquireInstanceMutex Windows 命名互斥体（官方 CMutex::Exclusive 等价）：
// CreateMutexW(bInitialOwner=TRUE)；ERROR_ALREADY_EXISTS → 互斥失败。
// 返回释放函数（ReleaseMutex + CloseHandle）。
func acquireInstanceMutex(name string) (func(), bool) {
	pn, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, false
	}
	k32 := syscall.NewLazyDLL("kernel32.dll")
	createM := k32.NewProc("CreateMutexW")
	releaseM := k32.NewProc("ReleaseMutex")
	closeH := k32.NewProc("CloseHandle")

	h, _, e := createM.Call(0, 1, uintptr(unsafe.Pointer(pn)))
	if h == 0 {
		return nil, false
	}
	if e == syscall.ERROR_ALREADY_EXISTS { // 已有实例持有
		closeH.Call(h)
		return nil, false
	}
	return func() {
		releaseM.Call(h)
		closeH.Call(h)
	}, true
}
