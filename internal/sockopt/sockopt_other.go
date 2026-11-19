//go:build !windows

// sockopt 是 UDP 广播口共绑助手（非 Windows 形态）：Linux 需额外
// SO_REUSEPORT 才允许多套接字双绑同口（best-effort，旧内核忽略）。
package sockopt

import (
	"fmt"
	"net"
	"syscall"
)

// ListenUDPReusable 建 UDP 口：bind 前置 SO_REUSEADDR + SO_BROADCAST
// （+ Linux SO_REUSEPORT）。
func ListenUDPReusable(network, addr string) (*net.UDPConn, error) {
	lc := net.ListenConfig{Control: func(_, _ string, c syscall.RawConn) error {
		var serr error
		if err := c.Control(func(fd uintptr) {
			fdi := int(fd)
			if serr = syscall.SetsockoptInt(fdi, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); serr != nil {
				return
			}
			if serr = syscall.SetsockoptInt(fdi, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1); serr != nil {
				return
			}
			_ = syscall.SetsockoptInt(fdi, syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
		}); err != nil {
			return fmt.Errorf("sockopt: control: %w", err)
		}
		return serr
	}}
	pc, err := lc.ListenPacket(nil, network, addr)
	if err != nil {
		return nil, err
	}
	return pc.(*net.UDPConn), nil
}
