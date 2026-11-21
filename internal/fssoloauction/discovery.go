package fssoloauction

import (
	"time"
)

// 发现监督器（Go 扩展，非官方行为；fssoloworld/discovery.go 同构）：
// 以对端 TCP 连接建立（identify 随 open 即发）为"应答"，未就绪期间
// 以 200ms 快间隔重播自宣（"没收到预期结果就一直广播"），缺口每 2s
// 限频汇报；就绪打点退出，此后回落官方 1s TimerCheck 自宣（永续，
// 晚加入者仍可被发现）。停口复用 stopCh（Stop 已关）。
const (
	discoveryFastEvery = 200 * time.Millisecond
	discoveryReportGap = 2 * time.Second
)

// startDiscoveryWatch Start 尾部挂发现监督器（广播口未启用或
// ManualTick 测试形态不启动；谓词经 DiscoveryPending 仍可观测）。
func (s *Server) startDiscoveryWatch() {
	if s.opt.BroadPort == 0 || s.opt.ManualTick {
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
		if len(s.discoveryPending()) == 0 {
			s.logf("(SoloAuction::DiscoveryWatch)发现就绪（耗时 %.1fs）",
				time.Since(start).Seconds())
			return
		}
		s.TimerCheck(0) // 快拍驱动自宣（含状态机 tick，幂等）
		if time.Since(lastReport) >= discoveryReportGap {
			s.logf("(SoloAuction::DiscoveryWatch)等待发现：store0")
			lastReport = time.Now()
		}
		select {
		case <-t.C:
		case <-s.stopCh:
			return
		}
	}
}

// discoveryPending 启动期预期对端缺口（应答 = 对端连接建立）：store0
// （注册/装载链之源）。LocalStore 垫片在场即视作已连（Start 内
// evStoreConnected 同语义）；world 由对方发起（实际拨向 entry 内网
// 口，官方 announceWorld 语义如此），不属本侧可判定预期。
func (s *Server) discoveryPending() []string {
	if s.opt.LocalStore != nil || s.inner.Connected(commStoreMin) {
		return nil
	}
	return []string{"store0"}
}

// DiscoveryPending 发现就绪缺口（宿主/测试观测面；空 = 全部预期
// 对端已应答连入）。
func (s *Server) DiscoveryPending() []string { return s.discoveryPending() }
