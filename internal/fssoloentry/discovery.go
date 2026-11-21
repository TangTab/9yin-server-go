package fssoloentry

import (
	"strings"
	"time"
)

// 发现监督器（Go 扩展，非官方行为；fssoloworld/discovery.go 同构）。
// entry 纯被连（只自宣不外连）：以"任一对端 inner 注册"为应答，未
// 就绪期间以 200ms 快间隔经 BroadcastSink 重播自宣（"没收到预期结果
// 就一直广播"），缺口每 2s 限频汇报；就绪打点退出，此后回落官方 1s
// TimerCheck 自宣（永续，晚加入者仍可被发现）。未授权形态不广播
// （官方 not licensed 语义），监督器随之不启动。
const (
	discoveryFastEvery = 200 * time.Millisecond
	discoveryReportGap = 2 * time.Second
)

// startDiscoveryWatch Start 尾部挂发现监督器（ManualTick 测试形态、
// 未授权或无自宣出口不启动；谓词经 DiscoveryPending 仍可观测）。
func (s *Server) startDiscoveryWatch() {
	if s.opt.ManualTick || s.opt.BroadcastSink == nil || s.inner == nil {
		return
	}
	s.mu.Lock()
	licensed := s.fields[0] == OfficialMagic1
	s.mu.Unlock()
	if !licensed {
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
		pending := s.discoveryPending()
		if len(pending) == 0 {
			s.logf("(SoloEntry::DiscoveryWatch)发现就绪（耗时 %.1fs）",
				time.Since(start).Seconds())
			return
		}
		// 快播自宣（TimerCheck 同路径：license 魔数对已过门）
		s.opt.BroadcastSink(s.BroadcastInfo())
		if time.Since(lastReport) >= discoveryReportGap {
			s.logf("(SoloEntry::DiscoveryWatch)等待发现：%s",
				strings.Join(pending, ", "))
			lastReport = time.Now()
		}
		select {
		case <-t.C:
		case <-s.stop:
			return
		}
	}
}

// discoveryPending 启动期预期对端缺口（应答 = 任一对端 inner 注册；
// world/member/public/store 任一即证广播链通）。
func (s *Server) discoveryPending() []string {
	if s.inner != nil && s.inner.Registered() > 0 {
		return nil
	}
	return []string{"any-peer"}
}

// DiscoveryPending 发现就绪缺口（宿主/测试观测面；空 = 已有对端
// 应答注册）。
func (s *Server) DiscoveryPending() []string { return s.discoveryPending() }
