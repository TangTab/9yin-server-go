package fssoloworld

// neigonguse.go C2S 215 运行内功切换（2433 线 player_progress.go case 215
// 定版）：校验已学 → 换 st.CurNeiGong → 帧序 MaxSP 专帧（先刷分母）→
// 资源批（CurNeiGong/五行/FacultyName/BufferInfo4 动态——旧书 buff 图标
// 随批刷新消失）→ 战斗面板单属性逐帧（CombatAttributeValues 同一取值
// 面）→ m169 两值帧挂新书可见 buff（buff 栏图标只由 m169 驱动）→
// sendRoleSave。取数面 internal/neigongdata（客户端设计态 share 树直读）。

import (
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/neigongdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

var (
	neiGongCatalogMu     sync.Mutex
	neiGongCatalogTried  string
	neiGongCatalogCached *neigongdata.Catalog
)

// neiGongCatalogOf 内功目录（进程级缓存，skillCatalog 同款数据根锚定；
// 全缺返回 nil——调用方按缺目录降级：五行 0、BufferInfo4 空）。
func (s *Server) neiGongCatalogOf() *neigongdata.Catalog {
	roots := skillDataRoots()
	key := strings.Join(roots, "\x00")
	neiGongCatalogMu.Lock()
	defer neiGongCatalogMu.Unlock()
	if key == neiGongCatalogTried {
		return neiGongCatalogCached
	}
	neiGongCatalogTried, neiGongCatalogCached = key, nil
	for _, root := range roots {
		catalog, err := neigongdata.Load(root)
		if err == nil {
			neiGongCatalogCached = catalog
			s.trace("neigong catalog loaded from %s（%d 行五行表）", root, catalog.Count())
			break
		}
	}
	if neiGongCatalogCached == nil {
		s.trace("neigong catalog unavailable（候选根 %v）：五行/BufferInfo4 降级", roots)
	}
	return neiGongCatalogCached
}

// resetNeiGongCatalog 清目录缓存（测试缝）。
func resetNeiGongCatalog() {
	neiGongCatalogMu.Lock()
	neiGongCatalogTried, neiGongCatalogCached = "", nil
	neiGongCatalogMu.Unlock()
}

// equipBookOf 已学书中按 ConfigID 取书（nil=未学）。
func equipBookOf(st *rolestate.RoleState, configID string) *rolestate.NeiGongBook {
	for i := range st.NeiGong {
		if st.NeiGong[i].ConfigID == configID {
			return &st.NeiGong[i]
		}
	}
	return nil
}

// handleUseNeiGong C2S 215 运行内功切换（文件头帧序）。
func (s *Server) handleUseNeiGong(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomUseNeiGong) {
		return
	}
	if len(message.Values) < 2 || message.Values[1].Type != 6 || strings.TrimSpace(message.Values[1].Text) == "" {
		s.trace("neigong use: 215 需要 ConfigID string client=%d", pl.Conn1)
		return
	}
	configID := strings.TrimSpace(message.Values[1].Text)
	st := &pl.roleState
	book := equipBookOf(st, configID)
	if book == nil {
		s.trace("neigong use: 未学内功 %s 拒收 client=%d", configID, pl.Conn1)
		return
	}
	catalog := s.neiGongCatalogOf()
	previous := st.CurNeiGong
	st.CurNeiGong = configID
	id := pl.entergenIdentity()

	// 1. MaxSP 专帧：换书改怒气有效上限，先刷分母再发资源批。
	if frame, err := playerPropertyFrame(pl,
		entergen.NamedProperty("MaxSP", entergen.Int32Value(entergen.RageCapOf(configID)))); err == nil {
		s.sendToClient(pl, frame)
	} else {
		s.trace("neigong use: MaxSP 专帧编码失败: %v", err)
	}

	// 2. 资源批：CurNeiGong/五行/FacultyName + BufferInfo4 动态合成
	//（旧书 buff 图标随批刷新消失，新书静态位就位）。
	props := []entergen.IndexedProperty{
		entergen.NamedProperty("CurNeiGong", entergen.StringValue(configID)),
		entergen.NamedProperty("FacultyName", entergen.StringValue(configID)),
		entergen.NamedProperty("CurNeiGongWuXing", entergen.Int32Value(catalog.WuXingOf(configID))),
	}
	buffInfo4 := equipBuffInfo4(catalog, id, book)
	props = append(props, entergen.NamedProperty("BufferInfo4", entergen.StringValue(buffInfo4)))
	if frame, err := playerPropertyFrame(pl, props...); err == nil {
		s.sendToClient(pl, frame)
	} else {
		s.trace("neigong use: 资源批编码失败: %v", err)
	}

	// 3. 战斗面板单属性逐帧（五维/资源 Add 随新书面板值）。
	for _, spec := range entergen.CombatAttributeValues(*st) {
		if frame, err := playerPropertyFrame(pl,
			entergen.NamedProperty(spec.Name, entergen.Int32Value(spec.Value))); err == nil {
			s.sendToClient(pl, frame)
		}
	}

	// 4. m169 两值帧：登记新书可见 buff（图标只由此帧驱动）。
	if book.BuffID != "" {
		if frame, err := entergen.EquipBuffFrame(id, book.BuffID); err == nil {
			s.sendToClient(pl, frame)
		} else {
			s.trace("neigong use: m169 buff 帧编码失败 %s: %v", book.BuffID, err)
		}
	}
	s.sendRoleSave(pl)
	s.trace("neigong use: 运行内功 %s → %s（五行 %d）client=%d",
		previous, configID, catalog.WuXingOf(configID), pl.Conn1)
}

// equipBuffInfo4 装备书的 BufferInfo4 槽串（无 buff/缺 StaticData 空串）。
func equipBuffInfo4(catalog *neigongdata.Catalog, id entergen.Identity, book *rolestate.NeiGongBook) string {
	if book == nil || book.BuffID == "" || catalog == nil {
		return ""
	}
	static, ok := catalog.BuffStaticData(book.BuffID)
	if !ok {
		return ""
	}
	return entergen.BufferInfo4CSV(id, static, int32(book.BuffLevel))
}
