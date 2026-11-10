package main

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// 杀怪掉落双形态（官服方言，e.pcapng 实证 + 客户端 lua 考据 2026-11-08）：
//   - 普通怪 = 掉落挂尸体（attachLootToCorpseLocked）：不生成新实体，
//     掉落运行时挂在尸体本身；客户端冒光/拾取光标由 0x10 属性
//     CanPick@76=1 + PickMember@1592=击杀者名 驱动（scene_obj_prop.lua
//     on_change_pick_state → DeadPickup 特效），点击尸体开掉落窗，
//     内容懒加载——死亡时零掉落下发；
//   - 宝箱怪（AIEditorPackID 死亡生箱事件）与 GM 世界宝箱 = 独立实体
//     （spawnChestEntityAtLocked，BoxNpc/BossBox 模板，0x0D 物化）。
//
// 掉落窗协议两形态共用（View 80 方言，考据自客户端 lua）：
//   - 点击（0x07 FuncID=0）→ 服务端建 View 80（VIEWPORT_DROP_BOX，
//     capacity 16，视图属性 DropType=0），行属性 ConfigID/Amount/
//     ColorLevel（官方索引 1440/1894/1892），随后名字寻址 custom
//     show_hide_drop_view(1) 开窗——帧序敏感：视图帧必须先于开窗帧；
//   - C2S 380=拾取槽位（int）、382=关箱；1008（拾银）/381（任务掉落）
//     本批无对应行，不处理。
const (
	viewportDropBox            uint16 = 80  // share/view_define.lua VIEWPORT_DROP_BOX
	dropBoxCapacity            uint16 = 16  // form_droppick.lua MAX_DROP_BOX_CAPACITY
	clientCustomPickupDropItem int32  = 380 // CLIENT_CUSTOMMSG_PICKUP_DROPITEM
	clientCustomCloseDropBox   int32  = 382 // CLIENT_CUSTOMMSG_CLOSE_DROPBOX
	lootBoxTemplateConfigID           = "stormbox_10"
)

// 属性索引按名解析（0x09 表方言门控换表自动跟随；fork 位见各行注记）。
var (
	propDropType   = clientdata.MustPlayerProperty("DropType").Index   // DropType Int32（视图域）（fork=2224）
	propCanPick    = clientdata.MustSceneProperty("CanPick").Index     // CanPick Byte（场景对象域，尸体冒光开关）（fork=76）
	propPickMember = clientdata.MustPlayerProperty("PickMember").Index // PickMember WideString（拾取归属名单，逗号分隔玩家名）（fork=1592）
)

// lootBoxRuntime 是一份在场掉落的共享状态（挂 sceneRun，多连接共见；
// 拾取互斥经 run.mu 下的 taken 标记结算）。官服双形态：
//   - standalone=false：普通怪掉落挂尸体（键=尸体 id，点击尸体开窗，
//     CanPick/PickMember 属性驱动客户端冒光+拾取光标，懒加载方言）；
//   - standalone=true：宝箱怪（AIEditorPackID 生箱事件）或 GM 世界宝箱
//     的独立实体（stormbox_10/BoxNpc 模板，0x0D 物化，拾空/超时 0x0E）。
type lootBoxRuntime struct {
	id         uint32
	ownerID    uint32
	contents   []grantItem
	taken      []bool
	spawnedAt  time.Time // 过期按生成时刻+当前 lootBoxTTL() 推算（改值即刻生效）
	standalone bool      // true=独立宝箱实体；false=掉落挂尸体本体
	picker     string    // 挂尸形态的拾取归属（击杀者名，死亡中继时补写）
}

func (b *lootBoxRuntime) remainingLocked() int {
	count := 0
	for _, taken := range b.taken {
		if !taken {
			count++
		}
	}
	return count
}

// openDropBoxState 是一条连接正开着的掉落箱窗口：slots 把 View 80
// 槽位映射回宝箱内容下标（槽位=原始序号+1，拾取后留空档不复用）。
type openDropBoxState struct {
	boxID uint32
	box   *lootBoxRuntime
	slots map[uint16]int
}

// ---- 宝箱模板 ----

var (
	lootBoxTemplateOnce sync.Once
	lootBoxTemplateBase *npcSpawn
)

// lootBoxTemplate 缓存解析 stormbox_10（worldnpc/monsternpc_commonnpc.txt）。
// 资源缺失/解析失败时回落手工最小模板（测试环境无 npcconfig 时仍可生成）。
func lootBoxTemplate() npcSpawn {
	lootBoxTemplateOnce.Do(func() {
		path := filepath.Join(defaultModernShareRoot, "npc", "npcconfig", "worldnpc", "monsternpc_commonnpc.txt")
		file, err := openResourceReader(path)
		if err != nil {
			log.Printf("loot box template %s unavailable (%v); using fallback", lootBoxTemplateConfigID, err)
			return
		}
		table, loadErr := clientdata.LoadNPCTemplateTable(file, path)
		_ = file.Close()
		if loadErr != nil {
			log.Printf("loot box template table %s: %v; using fallback", path, loadErr)
			return
		}
		resolved, resolveErr := clientdata.ResolveNPC(table, clientdata.NPCCreatorInstance{TemplateID: lootBoxTemplateConfigID}, clientdata.ResolveOptions{
			Defaults: clientdata.VisibleNPCSpawnDefaultsCompatV1(),
		})
		if resolveErr != nil {
			log.Printf("loot box template %s resolve: %v; using fallback", lootBoxTemplateConfigID, resolveErr)
			return
		}
		lootBoxTemplateBase = &npcSpawn{resolved: resolved}
	})
	if lootBoxTemplateBase != nil {
		return *lootBoxTemplateBase
	}
	return fallbackLootBoxSpawn()
}

// fallbackLootBoxSpawn 手工最小模板（与 testNPCSpawn 同属性面，
// npcAddObject 可编码）。
func fallbackLootBoxSpawn() npcSpawn {
	properties := map[string]clientdata.PropertyValue{}
	for name, value := range map[string]clientdata.Value{
		"Type":       clientdata.ByteValue(4),
		"ConfigID":   clientdata.StringValue(lootBoxTemplateConfigID),
		"Resource":   clientdata.StringValue(`npc\itemnpc004`),
		"PosiX":      clientdata.Float32Value(0),
		"NpcType":    clientdata.Int32Value(229),
		"Name":       clientdata.WideStringValue("特殊财宝箱"),
		"WeaponMode": clientdata.StringValue("hand10_1"),
	} {
		properties[name] = clientdata.PropertyValue{Value: value}
	}
	return npcSpawn{resolved: clientdata.ResolvedNPC{
		ConfigID:    lootBoxTemplateConfigID,
		ScriptClass: "CommonNpc",
		Properties:  properties,
	}}
}

// ---- 生成与移除 ----

// rollLootForNPC 按怪物模板 DropID（attacknpc 掉落列，字符串形如
// NormalMon00101，落在 Extensions["template.DropID"]）掉一个权威
// 掉落包；缺包/缺列走 drops_mvp 回落阶梯（与 msg31 开箱同口径）。
func rollLootForNPC(npc npcSpawn) []grantItem {
	dropID := ""
	if npc.resolved.Extensions != nil {
		dropID = npc.resolved.Extensions["template.DropID"]
	}
	return currentDropPacks().roll(dropID, 1, itemUseRandom())
}

// attachLootToCorpseLocked 普通怪掉落挂尸体（官服懒加载方言）：掉落
// 运行时挂在尸体实体本身（键=尸体 id，不生成新实体），玩家点击尸体
// 开 View 80，客户端冒光/拾取光标由 CanPick/PickMember 属性驱动（死亡
// 中继时补发）。roll 为空=无掉落，尸体走普通 corpse TTL。调用方必须
// 已持有 r.mu。
func (r *sceneRun) attachLootToCorpseLocked(deadID, ownerID uint32, now time.Time) {
	index := forkNPCIndex(deadID) - 2
	if index < 0 || index >= len(r.catalog) {
		return // 目录外实体（测试摆放）不掉落
	}
	contents := rollLootForNPC(r.catalog[index])
	if len(contents) == 0 {
		return
	}
	meta, ok := r.npcMeta(worldcore.EntityID(deadID))
	if !ok {
		return
	}
	box := &lootBoxRuntime{
		id: deadID, ownerID: ownerID,
		contents:   contents,
		taken:      make([]bool, len(contents)),
		spawnedAt:  now,
		standalone: false,
	}
	meta.lootBox = box
	r.writeNpcMeta(worldcore.EntityID(deadID), meta)
	r.lootBoxes[deadID] = box
	log.Printf("scene %s/%s: loot attached to corpse %d items=%d ttl=%s",
		r.key.config, r.key.resource, deadID, len(contents), lootBoxTTL())
}

// spawnBoxNPCEntityLocked 宝箱怪路径：击杀点尸体位置生成独立宝箱实体
// （AIEditorPackID 死亡生箱事件指定的 BoxNpc/BossBox 模板），掉落内容
// 仍按怪自己的 DropID roll。调用方必须已持有 r.mu。
func (r *sceneRun) spawnBoxNPCEntityLocked(deadID uint32, boxConfigID string, now time.Time) {
	index := forkNPCIndex(deadID) - 2
	if index < 0 || index >= len(r.catalog) {
		return
	}
	contents := rollLootForNPC(r.catalog[index])
	if len(contents) == 0 {
		return
	}
	transform, ok := r.npcTransform(worldcore.EntityID(deadID))
	if !ok {
		return
	}
	r.spawnChestEntityAtLocked(transform, contents, boxConfigID, deadID, now)
}

// spawnChestEntityAtLocked 生成一只独立宝箱实体（宝箱怪/GM 世界宝箱，
// id 取 nextWireID 段永不复用），返回实体 id（0=失败）。visibleWith 是
// 物化帧的可见性锚对象（宝箱怪=尸体 id；0=对全部在场成员直发）。
// 调用方必须已持有 r.mu；物化帧排队 pendingObjectFrames，由
// flushPendingObjectFrames 锁外扇出。
func (r *sceneRun) spawnChestEntityAtLocked(transform worldcore.Transform, contents []grantItem, boxConfigID string, visibleWith uint32, now time.Time) uint32 {
	spawn, _ := chestEntityTemplate(boxConfigID)
	spawn.x, spawn.y, spawn.z, spawn.orient = transform.X, transform.Y, transform.Z, transform.Orient
	spawn.route, spawn.schedule = nil, nil // 宝箱不巡逻
	id := forkNPCID(r.nextWireID)
	if id == forkNPCID(0) {
		id = forkNPCID(2)
	}
	r.nextWireID++
	if err := r.registerNPCLocked(id, spawn); err != nil {
		log.Printf("scene %s/%s: register chest entity %s: %v", r.key.config, r.key.resource, boxConfigID, err)
		return 0
	}
	meta, ok := r.npcMeta(worldcore.EntityID(id))
	if !ok {
		return 0
	}
	box := &lootBoxRuntime{
		id: id, ownerID: meta.ownerID,
		contents:   contents,
		taken:      make([]bool, len(contents)),
		spawnedAt:  now,
		standalone: true,
	}
	meta.lootBox = box
	r.writeNpcMeta(worldcore.EntityID(id), meta)
	r.lootBoxes[id] = box
	r.pendingObjectFrames = append(r.pendingObjectFrames, pendingObjectFrame{
		id: id, ownerID: meta.ownerID, visibleWith: visibleWith,
		payload: append([]byte(nil), meta.payload...), transform: transform,
	})
	log.Printf("scene %s/%s: chest entity %d (%s) spawned items=%d ttl=%s",
		r.key.config, r.key.resource, id, boxConfigID, len(contents), lootBoxTTL())
	return id
}

// chestEntityTables 是 BoxNpc/BossBox 模板的两张权威表
// （clonenpc/funnpc/boxnpc.txt 的 BossBox* 与 worldnpc/funnpc/boxnpc.txt
// 的野外 BoxNpc*），Once 装载缓存。
var (
	chestEntityTablesOnce sync.Once
	chestEntityTables     []*clientdata.NPCTemplateTable
)

// chestEntityTemplate 按ConfigID 解析宝箱实体模板（BoxNpc/BossBox）；
// 缺模板回落 stormbox_10（含手工 fallback 模板，测试环境无 npcconfig
// 也能生成）。
func chestEntityTemplate(boxConfigID string) (npcSpawn, bool) {
	if boxConfigID != "" && boxConfigID != lootBoxTemplateConfigID {
		chestEntityTablesOnce.Do(func() {
			for _, rel := range []string{
				filepath.Join("npc", "npcconfig", "clonenpc", "funnpc", "boxnpc.txt"),
				filepath.Join("npc", "npcconfig", "worldnpc", "funnpc", "boxnpc.txt"),
			} {
				path := filepath.Join(defaultModernShareRoot, rel)
				file, err := openResourceReader(path)
				if err != nil {
					log.Printf("chest entity table %s unavailable (%v)", path, err)
					continue
				}
				table, loadErr := clientdata.LoadNPCTemplateTable(file, path)
				_ = file.Close()
				if loadErr != nil {
					log.Printf("chest entity table %s: %v", path, loadErr)
					continue
				}
				chestEntityTables = append(chestEntityTables, table)
			}
		})
		for _, table := range chestEntityTables {
			if len(table.LookupAll(boxConfigID)) == 0 {
				continue
			}
			resolved, err := clientdata.ResolveNPC(table, clientdata.NPCCreatorInstance{TemplateID: boxConfigID}, clientdata.ResolveOptions{
				Defaults: clientdata.VisibleNPCSpawnDefaultsCompatV1(),
			})
			if err == nil {
				return npcSpawn{resolved: resolved}, true
			}
			log.Printf("chest entity template %s resolve: %v", boxConfigID, err)
		}
		log.Printf("chest entity template %s unresolved; falling back to %s", boxConfigID, lootBoxTemplateConfigID)
	}
	return lootBoxTemplate(), false
}

// removeLootBoxLocked 注销一份掉落并登记 0x0E（out 为 nil 时不登记，
// 由 despawnLootBox 直发路径自行扇出）。挂尸形态注销的就是尸体本体：
// 必须同步 deadNPCs[id].removed=true，否则批量扫描会对已注销实体走
// reviveNPCLocked 缺 Vitals 兜底 delete(deadNPCs)，把重生吞掉。调用方
// 必须已持有 r.mu。
func (r *sceneRun) removeLootBoxLocked(out *refreshTickOutcome, id, ownerID uint32) {
	if err := r.unregisterNPC(worldcore.EntityID(id)); err != nil {
		log.Printf("scene %s/%s: remove loot box %d: %v", r.key.config, r.key.resource, id, err)
		return
	}
	delete(r.lootBoxes, id)
	if rec, ok := r.deadNPCs[id]; ok && !rec.removed {
		rec.removed = true
		r.deadNPCs[id] = rec
	}
	if out != nil {
		out.despawns = append(out.despawns, npcDespawnUpdate{id: id, ownerID: ownerID})
	}
}

// relayCorpsePickFrames 挂尸掉落的冒光属性下发（官服方言）：死亡时
// 补发尸体 CanPick@76=1 + PickMember@1592=击杀者名（宽串）。客户端
// scene_obj_prop.lua on_change_pick_state 据此挂 DeadPickup 特效并
// 切拾取光标。击杀者首发（致死 0x10 的 link.WriteFrame 不含这两帧）、
// 视口内其他成员走中继。实体宝箱（standalone）不发。
func (s *sceneLifecycle) relayCorpsePickFrames(id uint32, killerName string) {
	s.run.mu.Lock()
	if s.run.closed {
		s.run.mu.Unlock()
		return
	}
	box, has := s.run.lootBoxes[id]
	if !has || box.standalone {
		s.run.mu.Unlock()
		return
	}
	if killerName != "" {
		box.picker = killerName // 迟到者补帧时读回
	}
	s.run.mu.Unlock()
	canPick := sceneObjectByteProperty(id, box.ownerID, propCanPick, 1)
	pickMember := sceneObjectWideStringProperty(id, box.ownerID, propPickMember, box.picker)
	for _, frame := range [][]byte{canPick, pickMember} {
		if err := s.conn.WriteFrame(frame); err != nil {
			log.Printf("%s: send corpse pick property %d: %v", s.remote, id, err)
		}
		s.run.relayNPCPropertyFrame(id, frame, s)
	}
}

// despawnLootBox 拾空后的直发移除：锁内注销（挂尸形态=清尸体本体，
// removeLootBoxLocked 内部同步 deadNPCs removed 放行重生）、锁外把
// 0x0E 扇出给可见该对象的成员。
func (r *sceneRun) despawnLootBox(id, ownerID uint32) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	if _, exists := r.lootBoxes[id]; !exists {
		r.mu.Unlock()
		return
	}
	r.removeLootBoxLocked(nil, id, ownerID)
	members := r.sortedMembersLocked()
	r.mu.Unlock()
	frame := sceneRemoveObject(id, ownerID)
	for _, member := range members {
		if member.view == nil || !member.view.npcVisible(id) {
			continue
		}
		if err := member.view.conn.WriteFrame(frame); err != nil {
			log.Printf("%s: despawn loot box %d: %v", member.view.remote, id, err)
		}
	}
}

// flushPendingObjectFrames 把 apply* 锁内排队的宝箱物化帧（0x0D+0x1F）
// 在锁外扇出给看得见锚对象的成员，并把宝箱号注入其可见集（站着不动
// 的成员也能收到后续的过期/拾空 0x0E）。锚=尸体（宝箱怪路径）；
// visibleWith==0（GM 世界宝箱）对全部在场成员直发。击杀中继后必调；
// refreshTick 兜底再排空。
func (r *sceneRun) flushPendingObjectFrames() {
	r.mu.Lock()
	if r.closed || len(r.pendingObjectFrames) == 0 {
		r.mu.Unlock()
		return
	}
	pending := r.pendingObjectFrames
	r.pendingObjectFrames = nil
	members := r.sortedMembersLocked()
	r.mu.Unlock()
	for _, object := range pending {
		location := serverLocation(object.id, object.ownerID, object.transform)
		for _, member := range members {
			if member.view == nil {
				continue
			}
			if object.visibleWith != 0 && !member.view.npcVisible(object.visibleWith) {
				continue
			}
			if err := member.view.conn.WriteFrame(object.payload); err != nil {
				log.Printf("%s: materialize loot box %d: %v", member.view.remote, object.id, err)
				continue
			}
			if err := member.view.conn.WriteFrame(location); err != nil {
				log.Printf("%s: locate loot box %d: %v", member.view.remote, object.id, err)
				continue
			}
			member.view.mu.Lock()
			member.view.visibleNPCs[object.id] = struct{}{}
			member.view.mu.Unlock()
		}
	}
}

// ---- 开箱 / 拾取 / 关箱（View 80） ----

// openDropBoxLocked 点击宝箱 → 掉落箱窗口。帧序敏感：先 0x15 建视图、
// 0x18 逐行，再 custom show_hide_drop_view(1) 开窗。调用方必须已持有
// run.mu（objectRequest 路径）。
func (s *sceneLifecycle) openDropBoxLocked(entity sceneEntity) error {
	box := entity.lootBox
	if box == nil {
		return nil
	}
	create, err := serverCreateViewWithProperties(serverViewSpec{ID: viewportDropBox, Capacity: dropBoxCapacity},
		[]serverViewProperty{viewInt(propDropType, 0)})
	if err != nil {
		return fmt.Errorf("encode drop box view: %w", err)
	}
	if err := s.conn.WriteFrame(create); err != nil {
		return fmt.Errorf("create drop box view: %w", err)
	}
	slots := make(map[uint16]int, len(box.contents))
	for index, item := range box.contents {
		if box.taken[index] {
			continue
		}
		slot := uint16(index + 1)
		row, err := serverViewAdd(viewportDropBox, slot, []serverViewProperty{
			viewString(propItemConfigID, item.configID),
			viewInt(propAmount, item.amount),
			viewInt(propColorLevel, 1),
		})
		if err != nil {
			return fmt.Errorf("encode drop row %d: %w", slot, err)
		}
		if err := s.conn.WriteFrame(row); err != nil {
			return fmt.Errorf("send drop row %d: %w", slot, err)
		}
		slots[slot] = index
	}
	open, err := serverCustomStringMessage("show_hide_drop_view", customInt(1))
	if err != nil {
		return fmt.Errorf("encode drop view open: %w", err)
	}
	if err := s.conn.WriteFrame(open); err != nil {
		return fmt.Errorf("open drop view: %w", err)
	}
	s.openDropBox = &openDropBoxState{boxID: entity.id, box: box, slots: slots}
	log.Printf("%s: drop box %d opened slots=%d", s.remote, entity.id, len(slots))
	return nil
}

// closeDropBoxLocked 关闭掉落箱窗口（382 或拾空收尾）。宝箱留场至
// TTL/拾空，可再次打开，已拾行不重现。返回是否需要移除宝箱本体。
func (s *sceneLifecycle) closeDropBoxLocked(removeBox bool) {
	if s.openDropBox == nil {
		return
	}
	state := s.openDropBox
	s.openDropBox = nil
	if err := s.conn.WriteFrame(serverDeleteView(viewportDropBox)); err != nil {
		log.Printf("%s: delete drop box view: %v", s.remote, err)
	}
	if close, err := serverCustomStringMessage("show_hide_drop_view", customInt(0)); err == nil {
		if err := s.conn.WriteFrame(close); err != nil {
			log.Printf("%s: close drop view: %v", s.remote, err)
		}
	}
	if removeBox {
		s.run.despawnLootBox(state.boxID, state.box.ownerID)
	}
}

// pickupDropItemSlot 处理 C2S 380：槽位入包 → 0x19 删行；全部拾完
// 关窗并移除宝箱本体（0x0E 广播）。
func (s *sceneLifecycle) pickupDropItemSlot(session *itemSession, slot int32) {
	state := s.openDropBox
	if state == nil {
		return
	}
	index, ok := state.slots[uint16(slot)]
	if !ok {
		log.Printf("%s: pickup slot %d not in open drop box", s.remote, slot)
		return
	}
	item := state.box.contents[index]
	entry, resolved := session.store.lookupGrantable(session.ctx, item.configID)
	if !resolved {
		log.Printf("%s: pickup %s unknown config", s.remote, item.configID)
		return
	}
	if session.grantBagItem(entry, item.amount) == 0 {
		log.Printf("%s: pickup %s failed (bag full)", s.remote, item.configID)
		return
	}
	s.run.mu.Lock()
	taken := false
	if index < len(state.box.taken) && !state.box.taken[index] {
		state.box.taken[index] = true
		taken = true
	}
	remaining := state.box.remainingLocked()
	s.run.mu.Unlock()
	if !taken {
		// 并发拾取已被他人结算：物品已入包属于重复发放，这里保守不再
		// 回滚（单机规模可接受），只跳过删行避免视图错位。
		log.Printf("%s: pickup slot %d raced on box %d", s.remote, slot, state.boxID)
		return
	}
	delete(state.slots, uint16(slot))
	if err := s.conn.WriteFrame(serverViewRemove(viewportDropBox, uint16(slot))); err != nil {
		log.Printf("%s: remove drop row %d: %v", s.remote, slot, err)
	}
	log.Printf("%s: picked %s x%d from box %d (remaining=%d)", s.remote, item.configID, item.amount, state.boxID, remaining)
	if remaining == 0 {
		s.closeDropBoxLocked(true)
	}
}

// ---- C2S 自定义消息入口（main.go 两处 switch 调用） ----

// handlePickupDropItemCustom 处理 msg380 [380, slot(int)]。
func handlePickupDropItemCustom(world *sceneLifecycle, session *itemSession, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) < 2 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomPickupDropItem {
		return false, nil
	}
	slot, ok := custom.Values[1].exactInt32()
	if !ok {
		return true, nil
	}
	if world == nil {
		return true, nil
	}
	world.pickupDropItemSlot(session, slot)
	return true, nil
}

// handleCloseDropBoxCustom 处理 msg382 [382]（关箱；宝箱留场）。
func handleCloseDropBoxCustom(world *sceneLifecycle, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomCloseDropBox {
		return false, nil
	}
	if world == nil {
		return true, nil
	}
	world.closeDropBoxLocked(false)
	return true, nil
}
