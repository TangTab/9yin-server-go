//go:build windows

package genutil

import "syscall"

// transientLockError 一个「文件被别的进程映射着」的样本错误。
func transientLockError() error { return syscall.Errno(1224) }
