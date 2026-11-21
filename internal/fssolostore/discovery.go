package fssolostore

import (
	"strings"
	"time"
)

// 发现监督器（Go 扩展，非官方行为；fssoloworld/discovery.go 同构）。
// store 纯被连（只自宣不外连）：以"任一对端连接 + identify 入登记"
// 为应答，未就绪期间以 200ms 快间隔重播自宣（"没收到预期结果就一直
// 广播"），缺口每 2s 限频汇报；就绪打点退出，此后回落官方 1s
// TimerCheck 自宣（永续，晚加入者仍可被发现）。退出判定沿用
// tickLoop 的 stateClosed 约定。
const (
	discoveryFastEvery = 200 * time.Millisecond
	discoveryReportGap = 2 * time.Second
)

// startDiscoveryWatch Start 尾部挂发现监督器（广播口未启用或
// ManualTick 测试形态不启动；谓词经 DiscoveryPending 仍可观测）。
func (s *Server) startDiscoveryWatch() {
	if s.cfg.BroadPort == 0 || s.opt.ManualTick {
		return
	}
	go s.discoveryWatchLoop()
}

func (s *Server) discoveryWatchLoop() {
	start := time.Now()
	t := time.NewTicker(discoveryFastEvery)
	defer t.Stop()
	lastReport := start.Add(-discoveryReportGap) // 首拍即汇报
	for {
		s.mu.Lock()
		closed := s.state == stateClosed
		s.mu.Unlock()
		if closed {
			return
		}
		pending := s.discoveryPending()
		if len(pending) == 0 {
			s.logf("(SoloStore::DiscoveryWatch)发现就绪（耗时 %.1fs）",
				time.Since(start).Seconds())
			return
		}
		s.BroadcastStoreInfo() // 快播自宣：帮对端加速发现我
		if time.Since(lastReport) >= discoveryReportGap {
			s.logf("(SoloStore::DiscoveryWatch)等待发现：%s",
				strings.Join(pending, ", "))
			lastReport = time.Now()
		}
		<-t.C
	}
}

// discoveryPending 启动期预期对端缺口（应答 = 任一对端 identify 入
// 登记；world/member/public/auction 任一即证广播链通）。
func (s *Server) discoveryPending() []string {
	if s.inner.AnyConnected() {
		return nil
	}
	return []string{"any-peer"}
}

// DiscoveryPending 发现就绪缺口（宿主/测试观测面；空 = 已有对端
// 应答连入）。
func (s *Server) DiscoveryPending() []string { return s.discoveryPending() }
