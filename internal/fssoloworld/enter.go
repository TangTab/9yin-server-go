package fssoloworld

// 进场链双模式（NINEYIN_ENTER_MODE=native|replay，默认 native）：
//
//	native（2433 方言原生生成，2026-11-21 决策）：0x02 回执 ok 后下发
//	  0x09/0x0A 官方表（official2433 方言）→ entergen.PreReadyFrames
//	  （0x0B/记录对/0x29/0x28 化身/物理流/基础属性批，数据源=store 回执
//	  RST1 尾段的 role_state_json——建号 rolestate.BuildRoleState 同源）；
//	  ClientReady 后 entergen.ReadyFrames（视图簇/登录信息帧/位置）。
//	replay（回退）：真机镜像 golden bins 整段透传（初始账号教程态快照，
//	  见 docs/official/透传数据清单.md 处置台账）。RST1 缺席（官方
//	  store/旧档）自动回落 replay。
//
// 表前 0x32 推送（inner_push_golden ×2）两模式共用（会话票据透传保留）。

import (
	_ "embed"
	"encoding/binary"
	"os"
	"sync/atomic"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcdata"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

//go:embed enter_golden.bin
var enterGoldenFrames []byte

//go:embed inner_push_golden.bin
var innerPushGoldenFrames []byte // 表前 0x32 LZ 域配置块×2（INNER 推送）

//go:embed ready_golden.bin
var readyGoldenFrames []byte

//go:embed login_reply_golden.bin
var loginReplyGolden []byte // 外服 op02 登录应答镜像（369B：会话锚 0xB8B8 + 反外挂校验码段——客户端场景对象表/校验初始化源；2026-11-21 ter0 报错定位的缺失帧）

// enterSeq 0x0B 进场流水号（官方每次进场自增——0x1521/0x2F05 实证
// 非常量；从 1 起）。
var enterSeq atomic.Uint32

// nativeEnterEnabled 进场模式开关（运行时读 env——测试可 t.Setenv 切换）。
func nativeEnterEnabled() bool {
	return os.Getenv("NINEYIN_ENTER_MODE") != "replay"
}

// propertyTableFrame 0x09 属性表：默认 official2433 方言（2433+61 尾追
// = 2494 项，fork/7074 线客户端，b-line 真机跑通）；NINEYIN_TABLE_DIALECT
// =golden 时逐字节透传真机镜像帧（2086 项，回退开关）。
func propertyTableFrame() []byte {
	if clientdata.GoldenTableActive() {
		return clientdata.GoldenPropertyTableFrame()
	}
	return clientdata.OfficialPropertyTableFrame()
}

// homeRecordsFrame 0x0A 记录表：默认官方 446 schema 镜像帧（外服登录链
// 实证形态，帧头 16B 兼作 C2S 0x0A 信封头回显；编码侧 447 schema 的
// recId≤446 与其完全兼容）；golden 方言透传 344 帧。
func homeRecordsFrame() []byte {
	if clientdata.GoldenTableActive() {
		return clientdata.GoldenRecordTableFrame()
	}
	return clientdata.HomeRecordsMirrorFrame()
}

// sendLoginReplyGolden op02 登录应答镜像回放（golden 方言模式；
// accountLogonReply 过门后紧跟，外服时序=登录应答位）。
func (s *Server) sendLoginReplyGolden(pl *Player) {
	if !clientdata.GoldenTableActive() {
		return // 2433/native 模式不需要（fork 0x04 方言已完整）
	}
	s.sendToClient(pl, loginReplyGolden)
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

// enterContext 组装原生生成上下文（身份/会话/角色状态/场景 NPC 目录）。
// NPC 物化不再走 PreReady（2026-11-22 晚定案：客户端 stage_main lua
// scene_obj:RegisterCallBack 注册前收到的对象只建视觉层（game_visual）、
// 不进逻辑层（game_client）——form_talk.open_npc_talk 的 GetSceneObj 与
// 点击判定全在逻辑层，PreReady 物化=看得见点不中）；改由 clientReady
// 后延迟下发（npcEnterDefer，见 clientReady）。
func (s *Server) enterContext(pl *Player, materializeNPC bool) entergen.Context {
	_ = materializeNPC // 兼容参数：物化时机已挪 clientReady 后
	return entergen.Context{
		Identity: entergen.Identity{
			ObjectID: 1, // fork 客户端恒 1-1（stage_main 开图/记录绑定锚）
			OwnerID:  1,
			Sequence: enterSeq.Add(1),
		},
		Session: entergen.Session{
			ServerID:   int32(s.cfg.ServerID),
			ServerName: s.cfg.ServerName,
			Account:    pl.Account,
		},
		RoleName:   pl.Name,
		State:      pl.roleState,
		SceneIndex: s.sceneGlobalIndexOf(pl.roleState.Resource), // 开图位图/迷雾行（-1=scenes.ini 缺席）
	}
}

// sceneGlobalIndexOf 场景 resource → 客户端 scenes.ini 全局段号（npcdata
// 装载缓存）。缺席 -1：开图降级只发基线位图，不阻塞进场。
func (s *Server) sceneGlobalIndexOf(resource string) int {
	if resource == "" {
		return -1
	}
	root := npcfeed.ResolveRoot()
	if root == "" {
		return -1
	}
	if index, ok := npcdata.SceneGlobalIndex(root, resource); ok {
		return int(index)
	}
	return -1
}

// storeLoadEnter 0x02 Load 回执 {client,serial,u32 ok,arch-wstr para
// [,RST1 扩展尾段]}：ok → 0x32×2 → 0x09/0x0A 表 → 进场段（native 或
// golden 回放）→ PlOnline（等 op09 ClientReady）。
func (s *Server) storeLoadEnter(id1 uint32, p []byte) bool {
	if len(p) < 12 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	serial := binary.LittleEndian.Uint32(p[4:])
	ok := binary.LittleEndian.Uint32(p[8:])
	pl := s.players.FindByConnect(clientID, serial)
	if pl == nil {
		// GM 离线邮件合成 clientID 段回执（name-only 查档 → 补信 → 0x04）。
		if clientID >= offlineSynthClientBase && len(p) > 12 {
			s.offlineMailReply(clientID, p[12:])
		}
		return true
	}
	if ok == 0 {
		s.trace("(SoloWorld::StoreMsgLoad)load denied client=%d", clientID)
		return true
	}
	// RST1 尾段（Go store 扩展）：role_state_json → 原生生成数据源。
	if rest := parseRST1(p[12:]); rest != nil {
		if state, err := rolestate.UnmarshalRoleState(*rest); err != nil {
			s.trace("role state unmarshal failed client=%d: %v", clientID, err)
		} else {
			pl.roleState, pl.haveRoleState = state, true
		}
	}
	if nativeEnterEnabled() && pl.haveRoleState {
		// 重登序对齐 2433 fork 官服（2026-11-22 晚五轮终案，capture 对拍
		// 61036）：`09→0a→0b→0d→0x11 海`——**不发 0x0C/0x0E 拆除批**
		//（重登后客户端场景本空，2491 个无效 0x0E 反成进场风暴，
		// all_msg[276] 积压之源）、**不发 0x32×2/0x14×48**（aow 遗物，
		// fork 客户端错误消费致 Out range 越界族）。fork 客户端重登自清，
		// 无需服务端清场。
		s.sendToClient(pl, propertyTableFrame())
		s.sendToClient(pl, homeRecordsFrame())
		ctx := s.enterContext(pl, true)
		frames, err := entergen.PreReadyFrames(ctx, time.Now())
		if err != nil {
			s.trace("native enter generate failed client=%d: %v（回退 golden）", clientID, err)
			s.npcEnterAborted(pl) // 物化未落地：撤销视口簿记（golden 回放无 NPC，防幽灵 0x21）
			n := s.replayGoldenFrames(pl, enterGoldenFrames)
			s.players.SetPlayerState(pl, PlOnline)
			s.trace("enter golden replay %d frames client=%d name=%q", n, clientID, pl.Name)
			return true
		}
		for _, frame := range frames {
			s.sendToClient(pl, frame)
		}
		// 开图位图回写（玩家级累积）：生效位（存档/基线 + 当前场景位）落
		// roleState.OpenMaps，随位置节流/下线存档持久化——重登/切图后
		// 走过的场景保持点亮（2433 线无此持久化，每次重派生基线）。
		bits := entergen.EffectiveOpenMapBits(pl.roleState, ctx.SceneIndex)
		pl.roleState.OpenMaps = bits[:]
		s.players.SetPlayerState(pl, PlOnline)
		s.trace("enter native %d frames client=%d name=%q scene=%s", len(frames), clientID, pl.Name, pl.roleState.Scene)
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

// clientReady op09 ClientReady → 就绪段（native：视图簇+登录信息帧；
// 回放：ready_golden）。切图重入（sceneSwitch 在途）走 reentryReady 分支。
func (s *Server) clientReady(pl *Player) {
	if pending := s.pendingSceneSwitch(pl); pending != nil {
		s.reentryReady(pl, pending)
		return
	}
	if nativeEnterEnabled() && pl.haveRoleState {
		frames, err := entergen.ReadyFrames(s.enterContext(pl, false), time.Now())
		if err != nil {
			s.trace("native ready generate failed client=%d: %v（回退 golden）", pl.Conn1, err)
			n := s.replayGoldenFrames(pl, readyGoldenFrames)
			s.trace("ready golden replay %d frames client=%d", n, pl.Conn1)
			return
		}
		for _, frame := range frames {
			s.sendToClient(pl, frame)
		}
		// 邮箱整表重放（2433 线 ClientReady 后、View43/40 前窗口；空表零帧）。
		letters := s.pushLetterRows(pl)
		s.trace("ready native %d frames client=%d letters=%d", len(frames), pl.Conn1, letters)
		// NPC 物化挂账：等客户端首条上行活动消息（stage_main lua
		// scene_obj:RegisterCallBack 注册完成的就绪屏障——逻辑层
		// game_client 才收对象）触发；30s 兜底强制。
		s.npcMaterializeArm(pl)
		return
	}
	n := s.replayGoldenFrames(pl, readyGoldenFrames)
	s.trace("ready golden replay %d frames client=%d", n, pl.Conn1)
}

// parseRST1 从 0x02 回执 para 之后的尾段解出 role_state_json。
// 布局：para 宽串（u32 码元数+NUL）后 [magic 'RST1' u32][len u32][json]。
func parseRST1(rest []byte) *string {
	for off := 0; off+8 <= len(rest); off++ {
		if binary.LittleEndian.Uint32(rest[off:]) != 0x31545352 { // 'RST1'
			continue
		}
		ln := int(binary.LittleEndian.Uint32(rest[off+4:]))
		if ln <= 0 || off+8+ln > len(rest) {
			return nil
		}
		v := string(rest[off+8 : off+8+ln])
		return &v
	}
	return nil
}
