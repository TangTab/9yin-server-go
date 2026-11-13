package main

// 副本波次引擎 + Lua 玩法事件接线。
//
// 数据驱动默认引擎：creator.NpcProgress 是场景内波次刻度（clone003
// 实测值域 {2,3,4,5,7}，有间隔——推进一律跳到目录中下一个存在的
// 值）。populate 只注册首波，波次全灭由 waveTick 结算：默认推进下一
// 波（registerNPCLocked 按原目录下标注册，ID 布局不变），末波全灭
// 后延时收尾（finishHook → dungeonManager.markFinished，由属主会话
// ticker 执行回城+销毁）。
//
// Lua 接管：dungeon_wave_cleared 处理器返回真值即接管推进节奏，
// ctx:spawn_wave(n) 立即物化指定波、ctx:after(sec, event) 声明延时
// 回调（定时器由 sceneRun.timers 持有，epoch 防世代穿越）。事件在
// 持 run.mu 的路径上发起；宿主 mu 是叶子锁，回调（写帧/注册）在
// 调用方锁纪律下执行——与 npc_service 同口径。

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/local/9yin-go-server/internal/luascript"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

const (
	luaEventDungeonEnter        = "dungeon_enter"
	luaEventDungeonNPCKilled    = "dungeon_npc_killed"
	luaEventDungeonWaveCleared  = "dungeon_wave_cleared"
	luaEventDungeonDestroy      = "dungeon_destroy"
	luaEventDungeonTimerDefault = "dungeon_timer"

	// dungeonFinishDelay 末波全灭到收尾回城的默认缓冲（捡拾掉落）。
	dungeonFinishDelay = 10 * time.Second
	// dungeonAfterMaxSeconds ctx:after 的单次延时上限。
	dungeonAfterMaxSeconds = 600
)

// npcWaveValue 读取出生点的波次刻度（creator.NpcProgress 透传到
// Extensions；非数字或缺席返回 false）。
func npcWaveValue(npc npcSpawn) (int, bool) {
	raw, exists := npc.resolved.Extensions["creator.NpcProgress"]
	if !exists {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

// initWaveGroupsLocked 从目录归纳波次组（仅合格出生点）。无任何波次
// 数据的实例保持全量注册口径（waveMode=false）。调用方持有 r.mu。
func (r *sceneRun) initWaveGroupsLocked(catalog []npcSpawn) {
	groups := map[int][]int{}
	for index, npc := range catalog {
		if !npcSpawnEligible(npc) {
			continue
		}
		if wave, ok := npcWaveValue(npc); ok {
			groups[wave] = append(groups[wave], index)
		}
	}
	if len(groups) == 0 {
		return
	}
	values := make([]int, 0, len(groups))
	for wave := range groups {
		values = append(values, wave)
	}
	sort.Ints(values)
	r.waveMode = true
	r.waveGroups = groups
	r.currentWave = values[0]
	r.pendingWaves = values[1:]
	r.waveAlive = make(map[uint32]struct{})
	r.waveClearedFired = -1
	log.Printf("dungeon %s instance=%s wave engine armed waves=%v initial=%d spawns=%d",
		r.key.config, r.key.instance, values, r.currentWave, len(groups[r.currentWave]))
}

// trackWaveNPCLocked 把已注册的波次 NPC 计入在场集合。调用方持有 r.mu。
func (r *sceneRun) trackWaveNPCLocked(id uint32) {
	if r.waveAlive == nil {
		r.waveAlive = make(map[uint32]struct{})
	}
	r.waveAlive[id] = struct{}{}
}

// onDungeonNPCKilledLocked 击杀结算：出波次在场集合并递增计数，再发
// npc_killed 事件。调用方持有 r.mu。
func (r *sceneRun) onDungeonNPCKilledLocked(id uint32, configID string) {
	delete(r.waveAlive, id)
	r.dungeonKills++
	r.callDungeonScriptLocked(luaEventDungeonNPCKilled, func(ec *luascript.EventContext) {
		ec.NPC = luascript.NPCSnapshot{ID: id, ConfigID: configID}
	})
}

// spawnWaveLocked 物化一个波次组（幂等：在场/已注册的跳过），并标记
// viewsDirty 等锁外 reconcile。调用方持有 r.mu。
func (r *sceneRun) spawnWaveLocked(wave int) {
	indexes, ok := r.waveGroups[wave]
	if !ok {
		return
	}
	for i, pending := range r.pendingWaves {
		if pending == wave {
			r.pendingWaves = append(r.pendingWaves[:i], r.pendingWaves[i+1:]...)
			break
		}
	}
	r.currentWave = wave
	r.waveClearedFired = -1
	spawned := 0
	for _, index := range indexes {
		id := (uint32(index) + 2) | 0x11000000
		if _, alive := r.waveAlive[id]; alive {
			continue
		}
		if _, exists := r.npcMeta(worldcore.EntityID(id)); exists {
			continue
		}
		if err := r.registerNPCLocked(id, r.catalog[index]); err != nil {
			log.Printf("dungeon %s instance=%s: spawn wave %d index %d: %v", r.key.config, r.key.instance, wave, index, err)
			continue
		}
		r.trackWaveNPCLocked(id)
		spawned++
	}
	if spawned > 0 {
		r.viewsDirty = true
		log.Printf("dungeon %s instance=%s wave %d spawned=%d alive=%d pending=%v",
			r.key.config, r.key.instance, wave, spawned, len(r.waveAlive), r.pendingWaves)
	}
}

// waveTick 是 tickOnce 的波次段：当前波全灭时结算。默认推进下一波或
// 排程收尾；dungeon_wave_cleared 被脚本接管（返回真值）时按脚本节奏
//（ctx:spawn_wave / ctx:after）走。调用方不得持有 r.mu。
func (r *sceneRun) waveTick(now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.waveMode || r.finished || len(r.members) == 0 || len(r.waveAlive) > 0 {
		return
	}
	if r.waveClearedFired == r.currentWave {
		// 本波结算事件已发过（脚本接管节奏中），等待脚本动作。
		return
	}
	r.waveClearedFired = r.currentWave
	if handled := r.callDungeonScriptLocked(luaEventDungeonWaveCleared, nil); handled {
		return
	}
	if len(r.pendingWaves) > 0 {
		r.spawnWaveLocked(r.pendingWaves[0])
		return
	}
	r.scheduleFinishLocked(dungeonFinishDelay)
}

// scheduleFinishLocked 排程末波收尾：finishHook 在锁外、epoch 防护后
// 执行（dungeonManager.markFinished，只动 manager 锁）。调用方持有 r.mu。
func (r *sceneRun) scheduleFinishLocked(delay time.Duration) {
	if r.finished {
		return
	}
	r.finished = true
	epoch := r.epoch
	timer := time.AfterFunc(delay, func() {
		r.mu.Lock()
		if r.closed || r.epoch != epoch {
			r.mu.Unlock()
			return
		}
		hook := r.finishHook
		r.mu.Unlock()
		if hook != nil {
			hook()
		}
	})
	r.timers = append(r.timers, timer)
	log.Printf("dungeon %s instance=%s all waves cleared; auto-finish in %s", r.key.config, r.key.instance, delay)
}

// scheduleDungeonTimerLocked ctx:after 的引擎侧：延时后在本锁内重发
// 一个脚本事件（携带全套副本回调）。调用方持有 r.mu。
func (r *sceneRun) scheduleDungeonTimerLocked(seconds float64, event string) error {
	if seconds <= 0 || seconds > dungeonAfterMaxSeconds {
		return fmt.Errorf("ctx:after seconds must be in (0, %d]", dungeonAfterMaxSeconds)
	}
	if event == "" {
		return fmt.Errorf("ctx:after requires an event name")
	}
	epoch := r.epoch
	timer := time.AfterFunc(time.Duration(seconds*float64(time.Second)), func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.closed || r.epoch != epoch {
			return
		}
		r.callDungeonScriptLocked(event, nil)
	})
	r.timers = append(r.timers, timer)
	return nil
}

// callDungeonScript 在未持锁路径（GM 进本/销毁命令）发一个副本事件。
func (r *sceneRun) callDungeonScript(event string, extra func(*luascript.EventContext)) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.callDungeonScriptLocked(event, extra)
}

// callDungeonScriptLocked 发一个副本事件：快照 + 广播回调 + 波次/延时
// 能力回调。返回脚本是否接手（无处理器/显式 false = 默认引擎路径）。
// 调用方持有 r.mu；宿主 mu 是叶子锁，本调用安全。
func (r *sceneRun) callDungeonScriptLocked(event string, extra func(*luascript.EventContext)) bool {
	if r.scripts == nil || !r.scripts.HasHandler(event) {
		return false
	}
	ec := &luascript.EventContext{Dungeon: r.dungeonSnapshotLocked()}
	ec.Notify = func(brief, chapterName string) error {
		frame, err := encodeDramaPrompt(dramaPrompt{Chapter: 1, ChapterName: chapterName, Explore: 1, Challenge: 1, Brief: brief})
		if err != nil {
			return err
		}
		return r.broadcastFrameLocked(frame)
	}
	ec.SpawnWave = func(wave int) error {
		if !r.waveMode {
			return fmt.Errorf("dungeon has no wave data")
		}
		r.spawnWaveLocked(wave)
		return nil
	}
	ec.After = func(seconds float64, name string) error {
		return r.scheduleDungeonTimerLocked(seconds, name)
	}
	if extra != nil {
		extra(ec)
	}
	handled, err := r.scripts.Call(event, ec)
	if err != nil {
		log.Printf("dungeon %s instance=%s: lua %s failed: %v", r.key.config, r.key.instance, event, err)
		return true
	}
	return handled
}

// broadcastFrameLocked 向全部在场成员直发一帧（锁内写帧与
// broadcastNPCLocked 同纪律）。调用方持有 r.mu。
func (r *sceneRun) broadcastFrameLocked(frame []byte) error {
	var firstErr error
	for _, member := range r.members {
		if member.view == nil {
			continue
		}
		if err := member.view.conn.WriteFrame(frame); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// dungeonSnapshotLocked 构造脚本可见的副本快照。调用方持有 r.mu。
func (r *sceneRun) dungeonSnapshotLocked() luascript.DungeonSnapshot {
	next := 0
	if len(r.pendingWaves) > 0 {
		next = r.pendingWaves[0]
	}
	return luascript.DungeonSnapshot{
		Config:     r.key.config,
		Resource:   r.key.resource,
		Instance:   r.key.instance,
		Difficulty: r.dungeonDifficulty,
		Wave:       r.currentWave,
		NextWave:   next,
		Killed:     r.dungeonKills,
	}
}

// reconcileMemberViews 把新物化的波次 NPC 对账到全体视口（锁外：
// reconcileViewport 自取 run.mu）。viewsDirty 由 spawnWaveLocked 置位。
func (r *sceneRun) reconcileMemberViews() {
	r.mu.Lock()
	if r.closed || len(r.members) == 0 {
		r.viewsDirty = false
		r.mu.Unlock()
		return
	}
	views := make([]*sceneLifecycle, 0, len(r.members))
	for _, member := range r.members {
		if member.view != nil {
			views = append(views, member.view)
		}
	}
	r.viewsDirty = false
	r.mu.Unlock()
	for _, view := range views {
		if err := view.reconcileViewport(); err != nil {
			log.Printf("dungeon %s instance=%s: wave reconcile: %v", r.key.config, r.key.instance, err)
		}
	}
}
