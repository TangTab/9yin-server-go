//go:build windows

package genutil

import (
	"errors"
	"syscall"
)

// Windows 错误码：共享冲突、锁冲突、文件被别的进程映射着（user-mapped
// section open）。三者都是「有人正在用这个文件」，稍等通常即释放。
const (
	errorSharingViolation = syscall.Errno(32)
	errorLockViolation    = syscall.Errno(33)
	errorUserMappedFile   = syscall.Errno(1224)
)

// isTransientWriteError 报告写文件错误是否属于短暂占用、值得重试。
func isTransientWriteError(err error) bool {
	return errors.Is(err, errorSharingViolation) ||
		errors.Is(err, errorLockViolation) ||
		errors.Is(err, errorUserMappedFile)
}
