package fssoloworld

import (
	"fmt"
	"strings"
	"time"
)

// 发现监督器（Go 扩展，非官方行为）：官方发现模型是每 1s 无条件
// 自宣 + 收方 TCP 外连 identify，但没有任何一方判定"预期对端是否都
// 已应答"——UDP 偶丢或对端晚起时只能干等下一拍（world 卡 LoadScenes
// 即此隐式阻塞）。此处以 identify 入登记为"应答"，未就绪期间以
// 200ms 快间隔重播自宣（"没收到预期结果就一直广播"），缺口每 2s
// 限频汇报；就绪打点退出，此后回落官方 1s 心跳自宣（永续，晚加入
// 者仍可被发现，onBroadcast 不受就绪状态影响）。
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
			s.trace("(SoloWorld::DiscoveryWatch)发现就绪（耗时 %.1fs）",
				time.Since(start).Seconds())
			return
		}
		s.broadcastWorldInfo() // 快播自宣：帮对端加速发现我
		if time.Since(lastReport) >= discoveryReportGap {
			s.trace("(SoloWorld::DiscoveryWatch)等待发现：%s",
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
// 入登记）：≥1 store（装载链首闩 SendToAllStore）+ scenes 映射的全部
// member（LoadScenes 门）+ entry0 + public。auction 为可选对端不计
// （缺席不阻开服，连上仅 trace）。
func (s *Server) discoveryPending() []string {
	var pending []string
	if !s.anyStoreConnected() {
		pending = append(pending, "store")
	}
	for _, m := range s.scenes.Members() {
		if !s.inner.GetConnected(commMemberMin + uint32(m)) {
			pending = append(pending, fmt.Sprintf("member%d", m))
		}
	}
	if !s.inner.GetConnected(commEntryMin) {
		pending = append(pending, "entry0")
	}
	if !s.inner.GetConnected(commPublic) {
		pending = append(pending, "public")
	}
	return pending
}

func (s *Server) anyStoreConnected() bool {
	for i := 0; i < storeNumMax; i++ {
		if s.inner.GetConnected(commStoreMin + uint32(i)) {
			return true
		}
	}
	return false
}

// DiscoveryPending 发现就绪缺口（宿主/测试观测面；空 = 全部预期
// 对端已应答连入）。
func (s *Server) DiscoveryPending() []string { return s.discoveryPending() }
