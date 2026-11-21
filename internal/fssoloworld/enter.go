package fssoloworld

// 进场链（真机镜像回放方言）：0x02 存档载入回执后下发 0x09/0x0A 镜像
// 表 + 外服 golden 进场段（enter_golden.bin，#5..#284：0x0B 五属性 →
// 0x14/0x11 记录对 → 0x29/0x28 LZ 档案 → 0x18 视图行…），ClientReady 后
// 回放就绪段（ready_golden.bin）。帧序/字节与用户外服抓包逐帧一致
// （心跳 seq7/#284 为就绪边界锚）；fork 方言自构帧（0x0D/0x1F/物理流/
// View/m366 族）已全部退役——外服官方方言不发这些，自构形态在官方
// 客户端上解引用崩溃（2026-11-21 capture 对拍定案）。

import (
	_ "embed"
	"encoding/binary"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

//go:embed enter_golden.bin
var enterGoldenFrames []byte

//go:embed inner_push_golden.bin
var innerPushGoldenFrames []byte // 外服表前 0x32 LZ 域配置块×2（INNER 推送）

//go:embed ready_golden.bin
var readyGoldenFrames []byte

// propertyTableFrame 0x09 属性表（真机镜像逐字节透传，2086 项）。
func propertyTableFrame() []byte {
	return clientdata.GoldenPropertyTableFrame()
}

// homeRecordsFrame 0x0A 记录表（真机镜像透传，344 schema）。
func homeRecordsFrame() []byte {
	return clientdata.GoldenRecordTableFrame()
}

// replayGoldenFrames 长度前缀包（[u32 len][plain]×N）逐帧下行。
func (s *Server) replayGoldenFrames(pl *Player, blob []byte) int {
	sent := 0
	for off := 0; off+4 <= len(blob); {
		n := int(binary.LittleEndian.Uint32(blob[off:]))
		off += 4
		if n <= 0 || off+n > len(blob) {
			break
		}
		s.sendToClient(pl, blob[off:off+n])
		off += n
		sent++
	}
	return sent
}
// storeLoadEnter 0x02 Load 回执 {client,serial,u32 ok,arch-wstr para}：
// ok → 0x09/0x0A 镜像表 → 外服进场段整段回放（enter_golden）→ PlOnline
// （等 op09 ClientReady 回放就绪段）。
func (s *Server) storeLoadEnter(id1 uint32, p []byte) bool {
	if len(p) < 12 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	ok := binary.LittleEndian.Uint32(p[8:])
	pl := s.players.FindByConnect(clientID, serial)
	if pl == nil {
		return true
	}
	if ok == 0 {
		s.trace("(SoloWorld::StoreMsgLoad)load denied client=%d", clientID)
		return true
	}
	s.replayGoldenFrames(pl, innerPushGoldenFrames) // 0x32 域配置块（外服序位于表前）
	s.sendToClient(pl, propertyTableFrame())
	s.sendToClient(pl, homeRecordsFrame())
	n := s.replayGoldenFrames(pl, enterGoldenFrames)
	s.players.SetPlayerState(pl, PlOnline)
	s.trace("enter golden replay %d frames client=%d name=%q", n, clientID, pl.Name)
	return true
}

// clientReady op09 ClientReady → 外服就绪段回放（ready_golden）。
func (s *Server) clientReady(pl *Player) {
	n := s.replayGoldenFrames(pl, readyGoldenFrames)
	s.trace("ready golden replay %d frames client=%d", n, pl.Conn1)
}
