//go:build !windows

package fscomm

import "syscall"

// processAlive POSIX 实现：kill(pid, 0) 探进程存在。
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
