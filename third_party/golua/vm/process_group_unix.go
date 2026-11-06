//go:build !windows

package vm

import (
	"os"
	"os/exec"
	"syscall"
)

// applyProcessGroup 把子进程放进独立进程组，使取消上下文时能
// 击杀整棵进程树（如 sh -c "sleep 60" 会同时杀掉 sh 和 sleep）。
func applyProcessGroup(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup 击杀整个进程组（负 PID）。
func killProcessGroup(p *os.Process) error {
	return syscall.Kill(-p.Pid, syscall.SIGKILL)
}
