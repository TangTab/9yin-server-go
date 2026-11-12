package main

// 武器重铸（C2S 933-938 / S2C 467/468；form_recast_attribute_weapon.lua
// + custom_sender.lua:4343-4420 + custom_handler.lua:4742-4770 客户端方言，
// menu2 weapon_recast.go 函数族反汇编对齐）：
//
//	C2S 933 EQUIPRANDPROP_LOCK(view,pos,sel_op)   武器入台（穿戴视图 1/
//	    背包装备页 121；sel_op=3 武器随机/4 武器定向/5 技能随机/9 技能定向）
//	C2S 934 UNLOCK(view,pos,sel_op)               取出（会话清）
//	C2S 935 YURANDPROP(view,pos,sel_op,cost_type) 随机预览 → S2C 467
//	    [op_type, newpackprop]（newpackprop="0,词条:0,词条:…"，op5 技能
//	    类客户端逆序渲染）
//	C2S 936 YUDESIGNATEPROP                       定向预览（dire 表按属性名
//	    路由；一期 form 主走 935，日志静默）
//	C2S 937 RESUMEEQUIPPROP                       恢复原词条（一期静默）
//	C2S 938 SUREEQUIPPROP(view,pos,sel_op)        确认 → 词条落档 +
//	    穿戴帧重发 → S2C 468 [op_type]
//
// 词条通道 = 穿戴行 affixes JSON（镇派兵器同款 [{"pack_rec":0,
// "prop_built":词条,"kind":1|2}]）+ PropBuilt@1954；词条池 =
// share/item/senior_weapon_rand.ini（随机）/senior_weapon_dire.ini
// （按属性名分组）；词条→属性+数值 = modifypack/proppack.ini（13040
// 族）/skillpack.ini（13234 族）。第一期只支持穿戴中武器（VIEWPORT_EQUIP
// 槽 22）；背包武器（121 页无 affixes 列）拒绝提示先穿戴。

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// C2S/S2C 消息号（client_custom_define.lua:632-637；S2C 467/468 由
// custom_handler.lua custom_recastattribute_randprop_succeed/_sure 路由）。
const (
	clientCustomEquipRandPropLock   int32 = 933 // CLIENT_CUSTOMMSG_EQUIPRANDPROP_LOCK
	clientCustomEquipRandPropUnlock int32 = 934 // CLIENT_CUSTOMMSG_EQUIPRANDPROP_UNLOCK
	clientCustomYuRandProp          int32 = 935 // CLIENT_CUSTOMMSG_YURANDPROP
	clientCustomYuDesignateProp     int32 = 936 // CLIENT_CUSTOMMSG_YUDESIGNATEPROP
	clientCustomResumeEquipProp     int32 = 937 // CLIENT_CUSTOMMSG_RESUMEEQUIPPROP
	clientCustomSureEquipProp       int32 = 938 // CLIENT_CUSTOMMSG_SUREEQUIPPROP
	serverCustomRecastPreview       int32 = 467 // 预览回执 [op_type, newpackprop]
	serverCustomRecastConfirm       int32 = 468 // 确认回执 [op_type]
)

// RECAST_TYPE（form_recast_attribute_weapon.lua:7-10 本地表）。
const (
	recastTypeWeaponRand  int32 = 3
	recastTypeWeaponDire  int32 = 4
	recastTypeSkillRand   int32 = 5
	recastTypeSkillDirect int32 = 9
)

// 重铸寻址视图（view_define.lua：VIEWPORT_EQUIP=1 / VIEWPORT_EQUIP_TOOL=121）。
const (
	recastViewEquipWorn int32 = 1
	recastViewEquipBag  int32 = 121
)

// weaponRecastSlotWeapon 穿戴武器槽（equipbox.ini weapon=22）。
const weaponEquipSlot = int32(22)

// ---- 目录 ----

// weaponRecastCatalog 重铸词条池（两表 + 词条归属）。
type weaponRecastCatalog struct {
	randPools  map[string][]int            // 武器 config → 随机池（保持表序）
	direRoutes map[string]map[string][]int // 武器 config → 属性名 → 定向池
	propIDs    map[int]bool                // proppack 节号集（词条 kind=1）
	skillIDs   map[int]bool                // skillpack 节号集（词条 kind=2）
}

var (
	weaponRecastOnce sync.Once
	weaponRecast     *weaponRecastCatalog
)

// currentWeaponRecastCatalog 懒加载全局目录（randomEquipCatalogOf 同模式）。
func currentWeaponRecastCatalog() *weaponRecastCatalog {
	weaponRecastOnce.Do(func() {
		if weaponRecast != nil {
			return // 测试注入（setWeaponRecastCatalogForTest）优先
		}
		weaponRecast = loadWeaponRecastCatalog(
			filepath.Join(defaultModernShareRoot, "item", "senior_weapon_rand.ini"),
			filepath.Join(defaultModernShareRoot, "item", "senior_weapon_dire.ini"),
			filepath.Join(defaultModernShareRoot, "modifypack", "proppack.ini"),
			filepath.Join(defaultModernShareRoot, "modifypack", "skillpack.ini"))
	})
	return weaponRecast
}

// setWeaponRecastCatalogForTest 单测注入合成目录（生产代码不得调用；
// applyItemTestOverrides 同约定）。
func setWeaponRecastCatalogForTest(catalog *weaponRecastCatalog) {
	weaponRecastOnce.Do(func() {})
	weaponRecast = catalog
}

// loadWeaponRecastCatalog 装载两表 + 词条归属（prop/skillpack 节号集）。
func loadWeaponRecastCatalog(randPath, direPath, propPackPath, skillPackPath string) *weaponRecastCatalog {
	catalog := &weaponRecastCatalog{
		randPools:  make(map[string][]int),
		direRoutes: make(map[string]map[string][]int),
		propIDs:    make(map[int]bool),
		skillIDs:   make(map[int]bool),
	}
	catalog.propIDs = loadINICollection(propPackPath)
	catalog.skillIDs = loadINICollection(skillPackPath)

	// 随机池：[weapon] r=词条,词条,…（一节可能多行 r，全部收）。
	if file, err := openResourceReader(randPath); err == nil {
		parseINISections(file, func(section, key, value string) {
			if _, ok := parseUintSection(section); ok {
				return // 节名须是武器 config 串（非数字）
			}
			if !strings.EqualFold(key, "r") {
				return
			}
			for _, id := range splitIntList(value) {
				if id != 0 {
					catalog.randPools[section] = append(catalog.randPools[section], id)
				}
			}
		})
		file.Close()
	} else {
		log.Printf("weapon recast catalog: senior_weapon_rand unavailable: %v", err)
	}
	// 定向池：[weapon] r=属性名,词条,词条,…（多行=多属性路由）。
	if file, err := openResourceReader(direPath); err == nil {
		parseINISections(file, func(section, key, value string) {
			if _, ok := parseUintSection(section); ok {
				return
			}
			if !strings.EqualFold(key, "r") {
				return
			}
			parts := strings.Split(value, ",")
			if len(parts) < 2 {
				return
			}
			propName := strings.TrimSpace(parts[0])
			if propName == "" {
				return
			}
			route := catalog.direRoutes[section]
			if route == nil {
				route = make(map[string][]int)
				catalog.direRoutes[section] = route
			}
			for _, id := range splitIntList(strings.Join(parts[1:], ",")) {
				if id != 0 {
					route[propName] = append(route[propName], id)
				}
			}
		})
		file.Close()
	} else {
		log.Printf("weapon recast catalog: senior_weapon_dire unavailable: %v", err)
	}
	return catalog
}

// loadINICollection 收集 INI 数字节号集合（proppack/skillpack 词条归属判定）。
func loadINICollection(path string) map[int]bool {
	ids := make(map[int]bool)
	file, err := openResourceReader(path)
	if err != nil {
		return ids
	}
	defer file.Close()
	parseINISections(file, func(section, key, value string) {
		if id, ok := parseUintSection(section); ok && id != 0 {
			ids[int(id)] = true
		}
	})
	return ids
}

// splitIntList 逗号列表 → int 列表（非数字/空项丢弃）。
func splitIntList(raw string) []int {
	parts := strings.Split(raw, ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil && parsed != 0 {
			values = append(values, parsed)
		}
	}
	return values
}

// packKind 词条归属（1=proppack / 2=skillpack；未知 0；nil 目录 0）。
func (catalog *weaponRecastCatalog) packKind(id int) int {
	if catalog == nil {
		return 0
	}
	switch {
	case catalog.propIDs[id]:
		return 1
	case catalog.skillIDs[id]:
		return 2
	default:
		return 0
	}
}

// hasPools 武器是否可重铸（任一表在册）。
func (catalog *weaponRecastCatalog) hasPools(configID string) bool {
	if catalog == nil {
		return false
	}
	_, inRand := catalog.randPools[configID]
	_, inDire := catalog.direRoutes[configID]
	return inRand || inDire
}

// rollPreview 随机预览：池均匀抽 count 条（prop 段优先保留序，与客户端
// 渲染无强约束——全量交给 tips 文案）。
func (catalog *weaponRecastCatalog) rollPreview(configID string, opType int32, count int, rnd *rand.Rand) []int {
	pool := catalog.randPools[configID]
	if opType == recastTypeSkillRand {
		pool = catalog.skillOnly(pool)
	} else if opType == recastTypeWeaponRand {
		pool = catalog.propOnly(pool)
	}
	if len(pool) == 0 || count <= 0 {
		return nil
	}
	picked := make([]int, 0, count)
	used := make(map[int]bool, count)
	for len(picked) < count && len(used) < len(pool) {
		id := pool[rnd.Intn(len(pool))]
		if used[id] {
			continue
		}
		used[id] = true
		picked = append(picked, id)
	}
	return picked
}

// propOnly / skillOnly 按词条归属过滤池。
func (catalog *weaponRecastCatalog) propOnly(pool []int) []int {
	return catalog.filterPool(pool, 1)
}

func (catalog *weaponRecastCatalog) skillOnly(pool []int) []int {
	return catalog.filterPool(pool, 2)
}

func (catalog *weaponRecastCatalog) filterPool(pool []int, kind int) []int {
	filtered := make([]int, 0, len(pool))
	for _, id := range pool {
		if catalog.packKind(id) == kind {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

// ---- 会话 ----

// weaponRecastSession 重铸台会话（933 锁定 → 935 预览 → 938 确认）。
type weaponRecastSession struct {
	active   bool
	viewID   int32
	slot     int32
	opType   int32
	configID string
	preview  []int // 935 预览词条（938 应用；op5 技能逆序渲染由客户端处理）
}

// recastRandomSource 重铸随机源（进程级，itemUseRandom 同口径）。
var recastRandomSource = rand.New(rand.NewSource(time.Now().UnixNano()))

// ---- 处理器 ----

// handleRecastCustom 933-938 入口（0x1E/0x27 直达与活动信封双路汇入）。
func handleRecastCustom(h *itemSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 {
		return false, nil
	}
	switch custom.Values[0].Int32 {
	case clientCustomEquipRandPropLock,
		clientCustomEquipRandPropUnlock,
		clientCustomYuRandProp,
		clientCustomYuDesignateProp,
		clientCustomResumeEquipProp,
		clientCustomSureEquipProp:
	default:
		return false, nil
	}
	args := customIntArgs(custom)
	if len(args) < 3 {
		log.Printf("%s: recast malformed args=%d (need msg,view,pos,op)", h.remote, len(args))
		return true, nil
	}
	view, slot, opType := args[0], args[1], args[2]
	switch custom.Values[0].Int32 {
	case clientCustomEquipRandPropLock:
		recastLock(h, view, slot, opType)
	case clientCustomEquipRandPropUnlock:
		recastUnlock(h, view, slot, opType)
	case clientCustomYuRandProp:
		recastPreviewRand(h, view, slot, opType)
	case clientCustomYuDesignateProp, clientCustomResumeEquipProp:
		log.Printf("%s: recast msg=%d view=%d pos=%d op=%d not implemented (phase-2)",
			h.remote, custom.Values[0].Int32, view, slot, opType)
	case clientCustomSureEquipProp:
		recastConfirm(h, view, slot, opType)
	}
	return true, nil
}

// recastLock 933：校验会话视图 + 武器目录命中 → 绑定重铸台。
func recastLock(h *itemSession, view, slot, opType int32) {
	if !recastOpKnown(opType) {
		log.Printf("%s: recast lock unknown op=%d", h.remote, opType)
		return
	}
	// 背包装备行定位（00020 affixes 列就位，洗炼二期扩展 121 页）；
	// 其余视图拒绝。
	configID := ""
	switch view {
	case recastViewEquipWorn:
		worn, ok := h.player.wornItemAt(weaponEquipSlot)
		if !ok {
			log.Printf("%s: recast lock rejected: no weapon worn at slot %d", h.remote, weaponEquipSlot)
			return
		}
		configID = worn.configID
	case recastViewEquipBag:
		item, _, ok := bagItemAtForView(h.player, uint16(view), slot)
		if !ok {
			log.Printf("%s: recast lock rejected: empty bag cell view=%d pos=%d", h.remote, view, slot)
			return
		}
		configID = item.configID
	default:
		log.Printf("%s: recast lock rejected: view=%d not supported", h.remote, view)
		return
	}
	if !currentWeaponRecastCatalog().hasPools(configID) {
		log.Printf("%s: recast lock rejected: %s not in senior weapon tables", h.remote, configID)
		return
	}
	h.player.mu.Lock()
	h.player.recast = weaponRecastSession{
		active: true, viewID: view, slot: slot, opType: opType, configID: configID,
	}
	h.player.mu.Unlock()
	log.Printf("%s: recast locked config=%s view=%d pos=%d op=%d", h.remote, configID, view, slot, opType)
}

// recastUnlock 934：清台。
func recastUnlock(h *itemSession, view, slot, opType int32) {
	h.player.mu.Lock()
	session := h.player.recast
	h.player.recast = weaponRecastSession{}
	h.player.mu.Unlock()
	if session.active {
		log.Printf("%s: recast unlocked config=%s op=%d", h.remote, session.configID, opType)
	}
}

// recastPreviewRand 935：roll 预览词条 → S2C 467 [op, newpackprop]。
// 词条数 = 现有 affixes 条数（无词条按 3 拍板——真机校准点：官方重铸
// 条数口径待抓包）。
func recastPreviewRand(h *itemSession, view, slot, opType int32) {
	h.player.mu.Lock()
	session := h.player.recast
	h.player.mu.Unlock()
	if !recastSessionMatches(session, view, slot, opType) {
		log.Printf("%s: recast preview rejected: session mismatch (view=%d pos=%d op=%d)", h.remote, view, slot, opType)
		return
	}
	count := recastAffixCount(h, session)
	catalog := currentWeaponRecastCatalog()
	preview := catalog.rollPreview(session.configID, opType, count, recastRandomSource)
	if len(preview) == 0 {
		log.Printf("%s: recast preview empty pool config=%s op=%d", h.remote, session.configID, opType)
		return
	}
	h.player.mu.Lock()
	h.player.recast.preview = preview
	h.player.mu.Unlock()
	frame, err := serverCustomIntMessageWithOpcode(0x1E, serverCustomRecastPreview,
		customInt(opType), customString(recastPackList(preview)))
	if err != nil {
		log.Printf("%s: encode recast preview: %v", h.remote, err)
		return
	}
	if err := h.link.WriteFrame(frame); err != nil {
		log.Printf("%s: write recast preview: %v", h.remote, err)
	}
	log.Printf("%s: recast preview config=%s op=%d packs=%v", h.remote, session.configID, opType, preview)
}

// recastConfirm 938：预览词条落档（affixes JSON + UPDATE + 穿戴帧重发）
// → S2C 468 [op]。
func recastConfirm(h *itemSession, view, slot, opType int32) {
	h.player.mu.Lock()
	session := h.player.recast
	h.player.mu.Unlock()
	if !recastSessionMatches(session, view, slot, opType) {
		log.Printf("%s: recast confirm rejected: session mismatch (view=%d pos=%d op=%d)", h.remote, view, slot, opType)
		return
	}
	if len(session.preview) == 0 {
		log.Printf("%s: recast confirm without preview", h.remote)
		return
	}
	affixes := recastAffixesJSON(currentWeaponRecastCatalog(), session.preview)
	built := propBuiltOfAffixes(affixes)
	switch session.viewID {
	case recastViewEquipBag:
		// 背包装备行（00020 通道）：落档 + 内存行 + 背包帧重发。
		if err := h.store.updateBagAffixes(h.ctx, h.roleID, session.slot, affixes); err != nil {
			log.Printf("%s: recast confirm bag persist: %v", h.remote, err)
			return
		}
		h.player.updateBagRowPropBuilt(uint16(session.viewID), session.slot, built)
		if row, _, ok := bagItemAtForView(h.player, uint16(session.viewID), session.slot); ok {
			if err := pushBagItemFrames(h.link, h.roleID, row); err != nil {
				log.Printf("%s: recast confirm bag frames: %v", h.remote, err)
			}
			// 同 qianghua：清格+重发后按原槽位补推属性 record。
			h.pushRandomPropsForBag(row.configID, row.uid, row.viewID, row.slot)
		}
	default:
		if err := h.store.updateEquipAffixes(h.ctx, h.roleID, weaponEquipSlot, affixes); err != nil {
			log.Printf("%s: recast confirm persist: %v", h.remote, err)
			return
		}
		// 内存行 propBuilt 刷新 + 穿戴帧重发（PropBuilt@1954 tooltip 通道）。
		if row, ok := h.player.wornItemAt(weaponEquipSlot); ok {
			row.propBuilt = built
			h.player.replaceEquipSlot(row)
			if err := pushWearFrames(h.link, h.roleID, h.player.visualSexOf(),
				effectiveShowEquipType(h.player), row, h.player.sessionEquips()); err != nil {
				log.Printf("%s: recast confirm wear frames: %v", h.remote, err)
			}
		}
	}
	h.player.mu.Lock()
	h.player.recast = weaponRecastSession{}
	h.player.mu.Unlock()
	frame, err := serverCustomIntMessageWithOpcode(0x1E, serverCustomRecastConfirm, customInt(opType))
	if err != nil {
		log.Printf("%s: encode recast confirm: %v", h.remote, err)
		return
	}
	if err := h.link.WriteFrame(frame); err != nil {
		log.Printf("%s: write recast confirm: %v", h.remote, err)
	}
	log.Printf("%s: recast confirmed config=%s op=%d packs=%v", h.remote, session.configID, opType, session.preview)
}

// ---- 纯函数 ----

// recastOpKnown op 白名单（form 本地表 RECAST_TYPE_*）。
func recastOpKnown(op int32) bool {
	switch op {
	case recastTypeWeaponRand, recastTypeWeaponDire, recastTypeSkillRand, recastTypeSkillDirect:
		return true
	}
	return false
}

// recastSessionMatches 会话归属校验（view+slot+op 三元一致）。
func recastSessionMatches(session weaponRecastSession, view, slot, opType int32) bool {
	return session.active && session.viewID == view && session.slot == slot && session.opType == opType
}

// recastAffixCount 现有词条条数（穿戴/背包按会话视图取行；无 affixes
// 按 3 拍板——真机校准点）。
func recastAffixCount(h *itemSession, session weaponRecastSession) int {
	propBuilt := ""
	switch session.viewID {
	case recastViewEquipBag:
		if row, _, ok := bagItemAtForView(h.player, uint16(session.viewID), session.slot); ok {
			propBuilt = row.propBuilt
		}
	default:
		if row, ok := h.player.wornItemAt(weaponEquipSlot); ok {
			propBuilt = row.propBuilt
		}
	}
	if propBuilt != "" {
		if count := len(strings.Split(propBuilt, ";")); count > 0 {
			return count
		}
	}
	return 3
}

// recastPackList newpackprop 串（"0,词条:0,词条"——col1=pack_rec 行号 0
// 起，col2=词条 id；客户端渲染取 col2 拼 tips 文案）。
func recastPackList(packs []int) string {
	segments := make([]string, 0, len(packs))
	for _, pack := range packs {
		segments = append(segments, fmt.Sprintf("0,%d", pack))
	}
	return strings.Join(segments, ":")
}

// recastAffixesJSON 词条列表 → affixes JSON（镇派兵器同款形态）。
func recastAffixesJSON(catalog *weaponRecastCatalog, packs []int) string {
	type affixEntry struct {
		PackRec   int `json:"pack_rec"`
		PropBuilt int `json:"prop_built"`
		Kind      int `json:"kind"`
	}
	entries := make([]affixEntry, 0, len(packs))
	for _, pack := range packs {
		kind := catalog.packKind(pack)
		if kind == 0 {
			kind = 1 // 未知归属按属性包（渲染兜底）
		}
		entries = append(entries, affixEntry{PackRec: 0, PropBuilt: pack, Kind: kind})
	}
	encoded, err := json.Marshal(entries)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

// propBuiltOfAffixes affixes JSON → PropBuilt 串（inventory.propBuiltOf
// 同语义，输入已是 string）。
func propBuiltOfAffixes(affixes string) string {
	if affixes == "" {
		return ""
	}
	var entries []struct {
		PackRec   int64 `json:"pack_rec"`
		PropBuilt int64 `json:"prop_built"`
		Kind      int64 `json:"kind"`
	}
	if err := json.Unmarshal([]byte(affixes), &entries); err != nil {
		return ""
	}
	segments := make([]string, 0, len(entries))
	for _, entry := range entries {
		kind := entry.Kind
		if kind < 1 || kind > 5 {
			kind = 1
		}
		segments = append(segments, fmt.Sprintf("%d,%d", kind, entry.PropBuilt))
	}
	return strings.Join(segments, ";")
}

// wornItemAt 穿戴行查询（p.mu 外调用——内部自锁）。
func (p *playerActor) wornItemAt(slot int32) (equipRow, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, row := range p.equip {
		if row.slot == slot {
			return row, true
		}
	}
	return equipRow{}, false
}

// updateEquipAffixes 重铸词条落档（UPDATE affixes；缺列/无行仅告警——
// 内存与帧仍以调用方为准）。
func (s *inventoryStore) updateEquipAffixes(ctx context.Context, roleID role.RoleID, slot int32, affixes string) error {
	if s == nil || s.db == nil {
		return nil // 内存降级：无库不阻断
	}
	if roleID == 0 {
		return fmt.Errorf("recast store: zero role id")
	}
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := s.db.ExecContext(queryCtx,
		"UPDATE role_equip_items SET affixes = ? WHERE role_id = ? AND slot = ?",
		affixes, roleID, slot)
	if err != nil {
		if unknownColumn(err, "affixes") {
			return nil // 旧库无列：降级
		}
		return fmt.Errorf("update affixes role=%d slot=%d: %w", roleID, slot, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return fmt.Errorf("recast store: no worn row role=%d slot=%d", roleID, slot)
	}
	return nil
}

// updateBagAffixes 背包装备行词条落档（00020 列；缺列/无行降级——
// 内存与帧仍权威，重铸/强化调用方记日志）。
func (s *inventoryStore) updateBagAffixes(ctx context.Context, roleID role.RoleID, slot int32, affixes string) error {
	if s == nil || s.db == nil {
		return nil // 内存降级：无库不阻断
	}
	if roleID == 0 {
		return fmt.Errorf("recast store: zero role id")
	}
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := s.db.ExecContext(queryCtx,
		"UPDATE role_bag_items SET affixes = ? WHERE role_id = ? AND slot = ?",
		affixes, roleID, slot)
	if err != nil {
		if unknownColumn(err, "affixes") {
			return nil // 0020 前旧库：降级（内存/帧仍生效）
		}
		return fmt.Errorf("update bag affixes role=%d slot=%d: %w", roleID, slot, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return fmt.Errorf("recast store: no bag row role=%d slot=%d", roleID, slot)
	}
	return nil
}


// bagItemAtForView 客户端视图寻址背包行（bagViewOf(row.viewID)==view，
// tersure bagItemAtLocked 同口径；p.mu 外调用——内部自锁）。
func bagItemAtForView(p *playerActor, view uint16, slot int32) (bagRow, int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for index := range p.bag {
		if p.bag[index].slot == slot && bagViewOf(p.bag[index].viewID) == view {
			return p.bag[index], index, true
		}
	}
	return bagRow{}, -1, false
}

// updateBagRowPropBuilt 背包行词条内存镜像更新（重铸/强化落档后随帧
// 重发对齐；p.mu 保护）。
func (p *playerActor) updateBagRowPropBuilt(view uint16, slot int32, propBuilt string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for index := range p.bag {
		if p.bag[index].slot == slot && bagViewOf(p.bag[index].viewID) == view {
			p.bag[index].propBuilt = propBuilt
			return true
		}
	}
	return false
}
