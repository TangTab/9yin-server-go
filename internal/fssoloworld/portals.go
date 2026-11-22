package fssoloworld

// portals.go 传送门走近触发（2433 portalAt 同构：X/Z 圆判定 + 2s 场景级
// 冷却）。数据=internal/doordata（客户端 door_data.ini 提取，gen_door_routes
// 生成）；半径 0 = 点击专用路线（无走近语义，跳过）。执行器与 gm scene
// 共用 transferPlayer：同场景目标仅 0x1F，跨场景走完整重入序列。
//
// 门点击路径（op0x07 门对象直接传送）待 NPC 交互域并行工作落地后接线
// （数据/执行器/圆判定本次全部就位）。

import (
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/doordata"
)

// portalCooldown 传送门触发冷却（2433 portalCooldown=2s：触发圆内连续位移
// 包不重复传送；切图重入后玩家已远离触发点亦不误触）。
const portalCooldown = 2 * time.Second

// checkPortal 位移收编尾钩：当前场景路线表内首个命中触发圆的路线执行
// 传送。切换在途由 absorbPlayerPosition 前置守卫拦截（不会重入）。
func (s *Server) checkPortal(pl *Player, pos [4]float64) {
	if !pl.haveRoleState {
		return
	}
	routes := doordata.RoutesForScene(pl.roleState.Resource)
	if len(routes) == 0 {
		return
	}
	now := time.Now()
	if now.Before(pl.portalCoolAt) {
		return
	}
	for i := range routes {
		route := routes[i]
		if route.Radius <= 0 { // 点击专用路线（无触发点）
			continue
		}
		dx, dz := pos[0]-route.TriggerX, pos[2]-route.TriggerZ
		if dx*dx+dz*dz > route.Radius*route.Radius {
			continue
		}
		pl.portalCoolAt = now.Add(portalCooldown)
		dest := sceneDestination{
			Config:   route.TargetConfig,
			Resource: doordata.ResourceOfConfig(route.TargetConfig),
			Position: [4]float64{route.TargetX, route.TargetY, route.TargetZ, route.TargetOrient},
		}
		if err := s.transferPlayer(pl, dest, "portal:"+route.ID); err != nil {
			s.trace("portal %s transfer failed client=%d: %v", route.ID, pl.Conn1, err)
			return
		}
		s.trace("portal %s client=%d %s → %s", route.ID, pl.Conn1, pl.roleState.Resource, dest.Resource)
		return
	}
}
