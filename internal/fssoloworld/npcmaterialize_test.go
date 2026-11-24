package fssoloworld

import (
	"testing"
	"time"
)

func TestMaterializeOnActivity(t *testing.T) {
	s := newNpcTestServer(t)
	pl := s.players.AddPlayer(9, 1, "tester")
	s.players.SetPlayerState(pl, PlOnline)
	pl.haveRoleState = true
	pl.roleState.Resource = "born03"
	s.npcMaterializeArm(pl)
	s.npcMaterializeOnActivity(pl) // 首触
	npcPendingMu.Lock()
	_, pending := npcPendingSet[pl.Conn1]
	npcPendingMu.Unlock()
	if pending {
		t.Fatalf("挂账未清: conn %d 仍在待物化集", pl.Conn1)
	}
	// 二触幂等（不应重发——以 trace 观察为准，此处仅验证不炸）。
	s.npcMaterializeOnActivity(pl)
	time.Sleep(100 * time.Millisecond)
}
