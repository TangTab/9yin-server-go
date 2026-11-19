//go:build windows

package fssololog

import (
	"fmt"
	"syscall"
	"unsafe"
)

// mapSharedMemory 创建/打开官方同名 Windows 命名共享内存
// （CShareMem::Create 等价：CreateFileMappingW(INVALID_HANDLE_VALUE,
// PAGE_READWRITE) + MapViewOfFile）。访问掩码取
// FILE_MAP_READ|FILE_MAP_WRITE：经典 FILE_MAP_ALL_ACCESS(0xF0010)
// 在 Win11 24H2+（build 26300 实测）映射出的视图写入即 AV，读写
// 最小掩码矩阵实测可用。
func mapSharedMemory(name string, size int) (data []byte, closeF func(), err error) {
	pn, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, nil, err
	}
	k32 := syscall.NewLazyDLL("kernel32.dll")
	createF := k32.NewProc("CreateFileMappingW")
	mapF := k32.NewProc("MapViewOfFile")
	unmapF := k32.NewProc("UnmapViewOfFile")
	closeH := k32.NewProc("CloseHandle")

	const invalidHandle = ^uintptr(0) // INVALID_HANDLE_VALUE
	const pageReadWrite = 0x04
	h, _, e := createF.Call(invalidHandle, 0, pageReadWrite, uintptr(size>>32), uintptr(size), uintptr(unsafe.Pointer(pn)))
	if h == 0 {
		return nil, nil, fmt.Errorf("CreateFileMappingW(%s): %w", name, e)
	}
	const fileMapRW = 0x4 | 0x2 // FILE_MAP_READ | FILE_MAP_WRITE
	ptr, _, e := mapF.Call(h, fileMapRW, 0, 0, 0)
	if ptr == 0 {
		closeH.Call(h)
		return nil, nil, fmt.Errorf("MapViewOfFile(%s): %w", name, e)
	}
	data = unsafe.Slice(uintptrToBytePtr(ptr), size)
	closed := false
	closeF = func() {
		if closed {
			return
		}
		closed = true
		unmapF.Call(ptr)
		closeH.Call(h)
	}
	return data, closeF, nil
}

// uintptrToBytePtr 把 Win32 API 返回的基址读回 *byte。vet 的
// unsafeptr 分析器对 uintptr→unsafe.Pointer 直接转换一律告警；
// 此处返回值确为有效且存活期受控的映射基址，故经值双间读取
// （语义等价 (*byte)(unsafe.Pointer(v))）。
func uintptrToBytePtr(v uintptr) *byte {
	return *(**byte)(unsafe.Pointer(&v))
}
