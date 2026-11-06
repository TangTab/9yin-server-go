//go:build windows

package vm

import (
	"os"
	"os/exec"
	"syscall"
)

// applyProcessGroup 在 Windows 上创建新进程组；此语义弱于 Unix 的
// 进程组击杀，但足以让取消路径表现为进程终止。
func applyProcessGroup(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// killProcessGroup Windows 没有向负 PID 发信号的等价物，退化为
// 击杀单个进程。沙盒宿主默认不注册 exec 能力，此路径仅为编译兼容。
func killProcessGroup(p *os.Process) error {
	return p.Kill()
}
