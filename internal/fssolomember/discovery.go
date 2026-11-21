package fssolomember

import (
	"strings"
	"time"
)

// 发现监督器（Go 扩展，非官方行为；fssoloworld/discovery.go 同构）：
// 以对端连接 + identify 入登记为"应答"，未就绪期间以 200ms 快间隔
// 重播自宣（"没收到预期结果就一直广播"），缺口每 2s 限频汇报；就绪
// 打点退出，此后回落官方 1s 心跳自宣（永续，晚加入者仍可被发现）。
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
			s.opt.Logf("(SoloMember::DiscoveryWatch)发现就绪（耗时 %.1fs）",
				time.Since(start).Seconds())
			return
		}
		s.broadcastMemberInfo() // 快播自宣：帮对端加速发现我
		if time.Since(lastReport) >= discoveryReportGap {
			s.opt.Logf("(SoloMember::DiscoveryWatch)等待发现：%s",
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

// discoveryPending 启动期预期对端缺口（应答 = 对端连接 + identify 已
// 入登记）：store0（stateUnknown 首闩 SendToStore(0x29)）+ public
// （注册帧/三闩之源）+ entry0（SendToClient 下行通道）。auction 为
// 可选对端不计；world 由 world 主动连入，不属本侧预期。
func (s *Server) discoveryPending() []string {
	var pending []string
	if !s.inner.GetStoreConnected(0) {
		pending = append(pending, "store0")
	}
	if !s.inner.GetPublicConnected() {
		pending = append(pending, "public")
	}
	if !s.inner.GetEntryConnected(0) {
		pending = append(pending, "entry0")
	}
	return pending
}

// DiscoveryPending 发现就绪缺口（宿主/测试观测面；空 = 全部预期
// 对端已应答连入）。
func (s *Server) DiscoveryPending() []string { return s.discoveryPending() }
