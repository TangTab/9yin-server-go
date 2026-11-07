package main

import (
	"net"
	"sync"
	"testing"
	"time"
)

// TestCloseLiveSessionsClosesAndClears 验证关服路径能把注册表中的连接
// 全部关闭并清空，且已退出的连接不会再被重复计数。
func TestCloseLiveSessionsClosesAndClears(t *testing.T) {
	clientA, serverA := net.Pipe()
	clientB, serverB := net.Pipe()
	defer clientA.Close()
	defer clientB.Close()
	mu := new(sync.Mutex)
	conns := map[net.Conn]struct{}{serverA: {}, serverB: {}}
	if got := closeLiveSessions(mu, conns); got != 2 {
		t.Fatalf("closeLiveSessions closed %d conns, want 2", got)
	}
	if len(conns) != 0 {
		t.Fatalf("registry not drained: %d left", len(conns))
	}
	for _, client := range []net.Conn{clientA, clientB} {
		client.SetReadDeadline(time.Now().Add(time.Second))
		buffer := make([]byte, 1)
		if _, err := client.Read(buffer); err == nil {
			t.Fatal("peer side of closed conn still readable")
		}
	}
	if got := closeLiveSessions(mu, conns); got != 0 {
		t.Fatalf("second sweep closed %d conns, want 0", got)
	}
}

// TestWaitGroupTimeoutReturnsOnTime 验证 drain 等待在 WaitGroup 清零时
// 及时返回 true。
func TestWaitGroupTimeoutReturnsOnTime(t *testing.T) {
	var group sync.WaitGroup
	group.Add(1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		group.Done()
	}()
	if !waitGroupTimeout(&group, 5*time.Second) {
		t.Fatal("waitGroupTimeout reported timeout before group settled")
	}
}

// TestWaitGroupTimeoutTimesOut 验证超时分支返回 false 而非阻塞。
func TestWaitGroupTimeoutTimesOut(t *testing.T) {
	var group sync.WaitGroup
	group.Add(1)
	defer group.Done()
	if waitGroupTimeout(&group, 20*time.Millisecond) {
		t.Fatal("waitGroupTimeout reported success for unsettled group")
	}
}
