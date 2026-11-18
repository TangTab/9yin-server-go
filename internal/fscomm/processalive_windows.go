//go:build windows

package fscomm

import "syscall"

// processAlive Windows 实现：OpenProcess + STILL_ACTIVE。
// 与官方 named-mutex 存在性判定等价——进程一死两者同灭。
const processQueryLimitedInfo = 0x1000 // PROCESS_QUERY_LIMITED_INFORMATION

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := syscall.OpenProcess(processQueryLimitedInfo, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}
