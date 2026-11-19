//go:build windows

// sockopt 是 UDP 广播口共绑助手：官方 CCore CreateBroadcast/
// BroadSock 均以 setsockopt(SO_REUSEADDR + SO_BROADCAST) 建口，
// 集群内任意数量实体（多进程或合一进程内多实体）共绑同一广播口，
// 广播包全量投递到每个套接字（单播只投其一——官方同此语义）。
package sockopt

import (
	"fmt"
	"net"
	"syscall"
)

// ListenUDPReusable 建 UDP 口：bind 前置 SO_REUSEADDR + SO_BROADCAST。
func ListenUDPReusable(network, addr string) (*net.UDPConn, error) {
	lc := net.ListenConfig{Control: func(_, _ string, c syscall.RawConn) error {
		var serr error
		if err := c.Control(func(fd uintptr) {
			h := syscall.Handle(fd)
			if serr = syscall.SetsockoptInt(h, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); serr != nil {
				return
			}
			serr = syscall.SetsockoptInt(h, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
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
