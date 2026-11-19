//go:build !windows

package fssololog

import "fmt"

// mapSharedMemory 非 Windows 平台无命名共享内存（官方 Win32 专有），
// 返回错误；宿主以 NewLogShareMemInMemory 兜底（协议对齐，仅无
// 跨进程能力——见包文档"已知取舍"）。
func mapSharedMemory(name string, size int) ([]byte, func(), error) {
	return nil, nil, fmt.Errorf("命名共享内存 %s 为 Win32 专有，本平台请用 NewLogShareMemInMemory", name)
}
