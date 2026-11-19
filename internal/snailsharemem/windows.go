package snailsharemem

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// Windows 映射层：复刻官方 CreateFileMappingW/OpenFileMappingW 路径。
// 官方细节：大小 0x814、掩码 0xF001F（FILE_MAP_ALL_ACCESS/SECTION_ALL_ACCESS）、
// 名字 "SNAILEDITOR_"+name（UTF-16）、读端创建且首建者零初始化头部、
// 写端仅打开（不存在即失败）。

const (
	fileMapAllAccess = 0xF001F
	// fileMapViewAccess 官方 MapViewOfFile 传 0xF001F；但 Win11 24H2+
	// （build 26300 实测，同 FsSoloLog 复刻的 LogShareMem 坑）该掩码视图
	// 写入即 AV，回落 FILE_MAP_READ|FILE_MAP_WRITE。
	fileMapViewAccess = 0x02 | 0x04
	pageReadWrite     = 0x04
	errAlreadyExists  = 183 // ERROR_ALREADY_EXISTS
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procCreateFileMappingW         = kernel32.NewProc("CreateFileMappingW")
	procOpenFileMappingW           = kernel32.NewProc("OpenFileMappingW")
	procMapViewOfFile              = kernel32.NewProc("MapViewOfFile")
	procUnmapViewOfFile            = kernel32.NewProc("UnmapViewOfFile")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procGetLastError               = kernel32.NewProc("GetLastError")
	invalidHandlePtr       uintptr = ^uintptr(0) // INVALID_HANDLE_VALUE
)

// Mapping 一段命名共享内存（对应官方 CSharingMemory<CSharingData> 模板对象）。
type Mapping struct {
	name   string
	hMap   uintptr
	view   []byte
	mbox   *Mailbox
	closed bool
}

func mapName(name string) (string, []uint16) {
	full := MapNamePrefix + name
	return full, syscall.StringToUTF16(full)
}

func lastErr() uintptr { ret, _, _ := procGetLastError.Call(); return ret }

// OpenRead 官方读端：CreateFileMappingW(0x814) + MapViewOfFile；
// 首建者（GetLastError != ERROR_ALREADY_EXISTS）按官方 ctor 零初始化头部。
func OpenRead(name string) (*Mapping, error) {
	_, wname := mapName(name)
	h, _, _ := procCreateFileMappingW.Call(
		invalidHandlePtr, 0, pageReadWrite, 0, MapSize,
		uintptr(unsafe.Pointer(&wname[0])),
	)
	if h == 0 || h == invalidHandlePtr {
		return nil, fmt.Errorf("snailsharemem: CreateFileMappingW(%s) failed: %d", name, lastErr())
	}
	first := lastErr() != errAlreadyExists
	return finishOpen(name, h, first)
}

// OpenWrite 官方写端：OpenFileMappingW + MapViewOfFile；不存在即失败。
func OpenWrite(name string) (*Mapping, error) {
	_, wname := mapName(name)
	h, _, _ := procOpenFileMappingW.Call(
		fileMapAllAccess, 0,
		uintptr(unsafe.Pointer(&wname[0])),
	)
	if h == 0 {
		return nil, fmt.Errorf("snailsharemem: OpenFileMappingW(%s) failed: %d", name, lastErr())
	}
	return finishOpen(name, h, false)
}

func finishOpen(name string, h uintptr, init bool) (*Mapping, error) {
	p, _, _ := procMapViewOfFile.Call(h, fileMapViewAccess, 0, 0, 0)
	if p == 0 {
		procCloseHandle.Call(h)
		return nil, fmt.Errorf("snailsharemem: MapViewOfFile(%s) failed: %d", name, lastErr())
	}
	view := unsafe.Slice((*byte)(unsafe.Pointer(p)), MapSize)
	mbox, err := attach(view)
	if err != nil {
		procUnmapViewOfFile.Call(p)
		procCloseHandle.Call(h)
		return nil, err
	}
	if init {
		mbox.mu.Lock()
		mbox.initHeader()
		mbox.mu.Unlock()
	}
	return &Mapping{name: name, hMap: h, view: view, mbox: mbox}, nil
}

// Box 返回布局兼容信箱（读写均经此）。
func (mp *Mapping) Box() *Mailbox { return mp.mbox }

// Close UnmapViewOfFile + CloseHandle（幂等）。
func (mp *Mapping) Close() error {
	if mp == nil || mp.closed {
		return nil
	}
	mp.closed = true
	procUnmapViewOfFile.Call(uintptr(unsafe.Pointer(&mp.view[0])))
	procCloseHandle.Call(mp.hMap)
	return nil
}

// ErrNotWindows 非 Windows 平台的映射层错误。
var ErrNotWindows = errors.New("snailsharemem: named mapping requires windows")
