// housekeeper.go 后台巡检：CheckServers（keep 5 分钟超时断连，官方
// 0.00347 天阈值）与 CheckUsers（AwakeTime 秒，对计费类型 1/3 用户
// 主动推空回显 renew 帧）；AutoDumpMin 导出仅保留钩子（官方写
// charge\*.log 文本，Go 侧以日志呈现）。
package charge

import "time"

// housekeeper 每秒巡检（官方 DoWait 心跳 1s + 每 60 圈 CheckUsers）。
func (s *Server) housekeeper(stop <-chan struct{}) {
	awakeEvery := time.Duration(s.Config.AwakeSec) * time.Second
	if awakeEvery <= 0 {
		awakeEvery = 60 * time.Second
	}
	var lastAwake time.Time
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			now := s.now()
			s.checkServers(now)
			if now.Sub(lastAwake) >= awakeEvery {
				lastAwake = now
				s.checkUsers(now)
			}
		}
	}
}

// checkServers CheckServers@0x6fe40：keep 超时先断底层连接（由读循环
// defer 收尸 RemoveServer）。
func (s *Server) checkServers(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sv := range s.servers {
		if sv == nil {
			continue
		}
		if now.Sub(sv.lastKeep) > keepTimeout {
			s.logf("Server ID: %d keep timeout, closing", sv.serverID)
			if sess := s.sessions[sv.connid]; sess != nil {
				sess.conn.Close() //nolint:errcheck — 触发 OnClose 清理
			}
		}
	}
}

// checkUsers CheckUsers@0x766b0：对计费类型 1/3 且距上次提醒超过
// AwakeTime 的用户推 ["","renew",accID,type,unitType,deadline]。
func (s *Server) checkUsers(now time.Time) {
	nowDays := daysSinceEpoch(now)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u == nil {
			continue
		}
		if u.chargeTyp != 1 && u.chargeTyp != 3 {
			continue
		}
		if u.kicked {
			continue
		}
		if nowDays-u.lastAwake <= float64(s.Config.AwakeSec)/86400 {
			continue
		}
		sess := s.sessions[u.connid]
		if sess == nil {
			continue
		}
		u.lastAwake = nowDays
		s.logf("Account: %s ID: %d awake renew push", u.account, u.accID)
		s.renewUser(sess, "", u)
	}
}
