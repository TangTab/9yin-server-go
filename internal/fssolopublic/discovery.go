package fssolopublic

import (
	"strings"
	"time"
)

// 发现监督器（Go 扩展，非官方行为；fssoloworld/discovery.go 同构）：
// 以对端 TCP 连接建立（identify 随 open 即发）为"应答"，未就绪期间
// 以 200ms 快间隔重播自宣（"没收到预期结果就一直广播"），缺口每 2s
// 限频汇报；就绪打点退出，此后回落宿主 1s 心跳拍自宣（永续，晚加入
// 者仍可被发现）。public 无内建定时器（官方 CreateTimer 由宿主给），
// 监督器自驱不受影响。
const (
	discoveryFastEvery = 200 * time.Millisecond
	discoveryReportGap = 2 * time.Second
)

// startDiscoveryWatch Start 尾部挂发现监督器（广播口未启用不启动；
// 谓词经 DiscoveryPending 仍可观测）。
func (s *Server) startDiscoveryWatch() {
	if s.cfg.BroadPort == 0 {
		return
	}
	s.watchStop = make(chan struct{})
	go s.discoveryWatchLoop()
}

func (s *Server) discoveryWatchLoop() {
	start := time.Now()
	t := time.NewTicker(discoveryFastEvery)
	defer t.Stop()
	lastReport := start.Add(-discoveryReportGap) // 首拍即汇报
	for {
		pending := s.discoveryPending()
		if len(pending) == 0 {
			s.logf("(SoloPublic::DiscoveryWatch)发现就绪（耗时 %.1fs）",
				time.Since(start).Seconds())
			return
		}
		s.broadcastServerInfo() // 快播自宣：帮对端加速发现我
		if time.Since(lastReport) >= discoveryReportGap {
			s.logf("(SoloPublic::DiscoveryWatch)等待发现：%s",
				strings.Join(pending, ", "))
			lastReport = time.Now()
		}
		select {
		case <-t.C:
		case <-s.watchStop:
			return
		}
	}
}

// discoveryPending 启动期预期对端缺口（应答 = 对端连接建立）：store
// （装载链 0x2D 首闩）+ entry0。world 由 world 主动连入，不属本侧
// 预期。
func (s *Server) discoveryPending() []string {
	var pending []string
	if !s.inner.HasStore() {
		pending = append(pending, "store")
	}
	if !s.inner.Connected(CommEntryMin) {
		pending = append(pending, "entry0")
	}
	return pending
}

// DiscoveryPending 发现就绪缺口（宿主/测试观测面；空 = 全部预期
// 对端已应答连入）。
func (s *Server) DiscoveryPending() []string { return s.discoveryPending() }
