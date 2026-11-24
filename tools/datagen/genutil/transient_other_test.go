//go:build !windows

package genutil

import "syscall"

// transientLockError 一个「文件正被占用」的样本错误。
func transientLockError() error { return syscall.ETXTBSY }
