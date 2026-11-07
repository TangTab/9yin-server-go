package main

// 秘籍使用链（Java LoginFlow.useSkillBook :4895 + SkillGateTable）：
// tool_item.ini LearnSkillPack → item_learn_skill_static.ini TragetID（官方拼写）
// 目标 1..N 个，按 game_skills.skill_type 分发——wuxue/anqi 走 View40 整视图
// 重开 + role_skills 落库；neigong 走 playerProgress 抬修炼上限 + View43 重建；
// qinggong 走 View46 追加行 + role_skills 落库。门槛（UseCondition →
// rule/condition.ini Type3 修炼上限区间 / Type1 门派 / Type19 时代放行）未满足
// 或全部目标已学 → 不消耗防白吃；任一目标生效即消耗 1 本。

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/local/9yin-go-server/internal/role"
)

// gameSkillRow 是 game_skills 目录行（skill_type 分发 + View40 行 static 集）。
type gameSkillRow struct {
	skillID    string
	skillType  string
	staticData int32
	itemType   int32
}

// gameSkillRows 批查 game_skills（Java gameSkillRows 同构）。
func (s *inventoryStore) gameSkillRows(ctx context.Context, skillIDs []string) (map[string]gameSkillRow, error) {
	result := make(map[string]gameSkillRow, len(skillIDs))
	if s == nil || s.db == nil || len(skillIDs) == 0 {
		return result, nil
	}
	for start := 0; start < len(skillIDs); start += 200 {
		end := start + 200
		if end > len(skillIDs) {
			end = len(skillIDs)
		}
		chunk := skillIDs[start:end]
		placeholders := strings.TrimRight(strings.Repeat("?,", len(chunk)), ",")
		args := make([]any, 0, len(chunk))
		for _, id := range chunk {
			args = append(args, id)
		}
		rows, err := s.db.QueryContext(ctx,
			"SELECT skill_id, skill_type, static_data, item_type FROM game_skills WHERE skill_id IN ("+placeholders+")", args...)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			var row gameSkillRow
			if err := rows.Scan(&row.skillID, &row.skillType, &row.staticData, &row.itemType); err != nil {
				rows.Close()
				return result, err
			}
			result[row.skillID] = row
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return result, err
		}
	}
	return result, nil
}

// dbSkillsExist 报告 role_skills 是否已有该角色的行。
func (s *inventoryStore) dbSkillsExist(ctx context.Context, roleID role.RoleID) (bool, error) {
	if s == nil || s.db == nil {
		return false, nil
	}
	var count int32
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM role_skills WHERE role_id = ?", roleID).Scan(&count)
	return count > 0, err
}

// useSkillBook 秘籍使用主链（handleBodyItemUse 在 useItemEffect 之前拦截）。
func useSkillBook(h *itemSession, item bagRow, script string) {
	entry, ok := currentSkillBookCatalog().resolve(item.configID)
	if !ok || len(entry.targets) == 0 {
		log.Printf("%s: use skill book %s: no LearnSkillPack target (pack table gap)", h.remote, item.configID)
		return
	}
	if deny := checkSkillBookGate(h, entry); deny != "" {
		log.Printf("%s: use skill book %s: 门槛未满足：%s", h.remote, item.configID, deny)
		return
	}

	skillRows, err := h.store.gameSkillRows(h.ctx, entry.targets)
	if err != nil {
		log.Printf("%s: use skill book %s: load skill rows: %v", h.remote, item.configID, err)
		return
	}
	neigongCatalog, catalogErr := loadModernNeiGongCatalog()

	smax, addMax, addLevel := entry.skillMaxLevel, entry.addMaxLevel, entry.addLevel
	neigongCap := func(staticData int32) int32 {
		capLevel := smax
		if capLevel <= 0 {
			capLevel = addMax
		}
		if catalogMax := neigongCatalog.maxLevelOf(staticData); catalogMax > 0 && capLevel > catalogMax {
			capLevel = catalogMax
		}
		return capLevel
	}
	type wuxueLearn struct {
		id    string
		level int32
	}
	var newWuxue []wuxueLearn
	var newNeigong []modernNeiGongBook
	var newQingGong []string
	skipped := make([]string, 0, 4)
	for _, target := range entry.targets {
		row, exists := skillRows[target]
		if !exists {
			skipped = append(skipped, target+"(目录缺席)")
			continue
		}
		switch row.skillType {
		case "wuxue", "anqi":
			// 无修炼链：书直达 SkillMaxLevel 层（夹到 varprop 满级；无字段=学会 1 级）。
			wanted := smax
			if wanted <= 0 {
				wanted = 1
			}
			if maxLevel := installedCombatSkillCatalog.maxLevelOf(target); maxLevel > 0 && wanted > maxLevel {
				wanted = maxLevel
			}
			current, learned := h.player.learnedSkillLevel(target)
			if learned && wanted <= current {
				skipped = append(skipped, fmt.Sprintf("%s(已学至 %d 层)", target, current))
				continue
			}
			newWuxue = append(newWuxue, wuxueLearn{id: target, level: wanted})
		case "neigong":
			if catalogErr != nil {
				skipped = append(skipped, target+"(内功目录不可用)")
				continue
			}
			def, hasBook := neigongCatalog.book(target)
			if !hasBook {
				skipped = append(skipped, target+"(内功无varprop区间)")
				continue
			}
			h.player.mu.Lock()
			existing := h.player.progress.book(target)
			h.player.mu.Unlock()
			capLevel := neigongCap(def.staticData)
			setLevel := int32(0)
			if addMax > 0 && existing != nil {
				// 传承心法：直升 AddLevel 层（夹到新上限）。
				setLevel = existing.level + addLevel
				if setLevel < 0 {
					setLevel = 0
				}
				if setLevel > capLevel {
					setLevel = capLevel
				}
			}
			if existing == nil || capLevel > existing.maxLevel || setLevel > existing.level {
				newNeigong = append(newNeigong, def)
			} else {
				skipped = append(skipped, fmt.Sprintf("%s(已至 %d 层上限)", target, existing.maxLevel))
			}
		case "qinggong":
			h.player.mu.Lock()
			_, already := h.player.qingGongExtra[target]
			h.player.mu.Unlock()
			if already || inStarterQingGong(target) {
				skipped = append(skipped, target+"(已学)")
				continue
			}
			newQingGong = append(newQingGong, target)
		default:
			skipped = append(skipped, target+"("+row.skillType+" 暂未接)")
		}
	}
	if len(newWuxue) == 0 && len(newNeigong) == 0 && len(newQingGong) == 0 {
		log.Printf("%s: use skill book %s: 全部目标已学或不可解析 skipped=%v", h.remote, item.configID, skipped)
		return
	}
	h.player.mu.Lock()
	h.player.lastItemUseAt = h.now()
	h.player.mu.Unlock()

	// ① 武学/暗器手法：role_skills 落库 + 已学表追加 + View40 整视图重开。
	if len(newWuxue) > 0 {
		// starter 回退会话首学（DB 无 wuxue 行）：先物化基线，防重登切 DB 模式丢 starter。
		if err := materializeStarterSkills(h); err != nil {
			log.Printf("%s: materialize starter skill baseline: %v", h.remote, err)
		}
		h.player.mu.Lock()
		for _, learn := range newWuxue {
			h.player.learnedSkills[learn.id] = learn.level
		}
		h.player.mu.Unlock()
		for _, learn := range newWuxue {
			skillType := skillRows[learn.id].skillType
			if skillType == "" {
				skillType = "wuxue"
			}
			if err := h.store.upsertSkill(h.ctx, h.roleID, learn.id, skillType); err != nil {
				log.Printf("%s: upsert skill %s: %v", h.remote, learn.id, err)
			}
		}
		if err := rebuildWuxueSkillView(h, skillRows); err != nil {
			log.Printf("%s: rebuild wuxue view: %v", h.remote, err)
		}
	}
	// ② 内功书：建书/抬上限（传承直升）+ View43 整视图重建。
	if len(newNeigong) > 0 {
		h.player.mu.Lock()
		for _, def := range newNeigong {
			h.player.progress.learnOrRaiseBook(def, neigongCap(def.staticData), addMax, addLevel)
		}
		h.player.mu.Unlock()
		if err := rebuildNeiGongView(h); err != nil {
			log.Printf("%s: rebuild neigong view: %v", h.remote, err)
		}
	}
	// ③ 轻功：会话追加集 + role_skills 落库 + View46 追加行。
	if len(newQingGong) > 0 {
		h.player.mu.Lock()
		for _, id := range newQingGong {
			h.player.qingGongExtra[id] = struct{}{}
		}
		h.player.mu.Unlock()
		for _, id := range newQingGong {
			if err := h.store.upsertSkill(h.ctx, h.roleID, id, "qinggong"); err != nil {
				log.Printf("%s: upsert qinggong %s: %v", h.remote, id, err)
			}
		}
		if err := appendQingGongViewRows(h, newQingGong); err != nil {
			log.Printf("%s: append qinggong rows: %v", h.remote, err)
		}
	}
	// ④ 消耗 1 本 + 背包帧对。
	if !h.consumeBagItemCount(item, 1) {
		log.Printf("use skill book %s: consume failed (row missing/changed)", item.configID)
	}
	log.Printf("%s: use skill book %s: wuxue=%v neigong=%v qinggong=%v skipped=%v",
		h.remote, item.configID, newWuxue, newNeigong, newQingGong, skipped)
}

// learnOrRaiseBook 内功建书/抬上限（Java PlayerProgress.learnOrRaiseBook 同构；
// 新书 total/power 走 Go starter 新书基线 700/120，传承直升层按 total+200/
// maxPower+20 自然曲线）。调用方持有 p.mu。
func (p *playerProgress) learnOrRaiseBook(def modernNeiGongBook, capLevel, addMax, addLevel int32) {
	target := p.book(def.configID)
	if target == nil {
		initialCap := capLevel
		if initialCap <= 0 {
			initialCap = 1
		}
		initialLevel := int32(1)
		if addMax > 0 && addLevel > 0 && addLevel < initialLevel {
			initialLevel = addLevel
		}
		if initialLevel > initialCap {
			initialLevel = initialCap
		}
		itemType := def.itemType
		if itemType <= 0 {
			itemType = 1002
		}
		book := learnedNeiGong{
			configID: def.configID, staticData: def.staticData, itemType: itemType,
			level: initialLevel, maxLevel: initialCap,
			total: 700 + (initialLevel-1)*200, power: 120 + (initialLevel-1)*20,
			maxPower: 120 + (initialLevel-1)*20,
		}
		if err := book.applyModernEffect(); err != nil {
			log.Printf("learn neigong %s effect: %v", def.configID, err)
		}
		p.books = append(p.books, book)
		return
	}
	changed := false
	if capLevel > target.maxLevel {
		target.maxLevel = capLevel
		changed = true
	}
	want := addLevel
	if want > target.maxLevel {
		want = target.maxLevel
	}
	if addMax > 0 {
		for target.level < want {
			target.level++
			target.total += 200
			target.maxPower += 20
			target.power = target.maxPower
			changed = true
		}
	}
	if changed {
		if err := target.applyModernEffect(); err != nil {
			log.Printf("raise neigong %s effect: %v", def.configID, err)
		}
	}
}

// checkSkillBookGate 秘籍官方门槛：返回空串=放行，否则拒绝原因（不消耗）。
// Type19 Func/Type4 九阴志时代等未接条件放行（单机服宁松勿锁）。
func checkSkillBookGate(h *itemSession, entry skillBookEntry) string {
	if entry.useCondition <= 0 {
		return ""
	}
	gate, exists := currentSkillGateTable().gate(entry.useCondition)
	if !exists {
		return "" // 条件表缺席（1,222 个引用）：放行
	}
	if gate.gateType == "3" && strings.EqualFold(strings.TrimSpace(gate.para2), "MaxLevel") {
		capLevel := trainableCapOf(h, strings.TrimSpace(gate.para1))
		if !gate.containerCapInRange(capLevel) {
			return fmt.Sprintf("%s 修炼上限需 %s~%s，当前 %d（需先用前序篇章解锁）",
				gate.para1, gate.min, gate.max, capLevel)
		}
		return ""
	}
	if gate.gateType == "1" {
		expect := strings.TrimSpace(gate.min)
		h.player.mu.Lock()
		actual := h.player.faction
		h.player.mu.Unlock()
		if expect != "" && actual != "" && expect != actual {
			return fmt.Sprintf("门派不符（需 %s，当前 %s）", expect, actual)
		}
		return ""
	}
	return ""
}

// trainableCapOf 目标当前修炼上限：wuxue/anqi=已学等级（无修炼链，层即上限）；
// neigong=书 maxLevel；未学 0。
func trainableCapOf(h *itemSession, skillID string) int32 {
	if level, learned := h.player.learnedSkillLevel(skillID); learned {
		return level
	}
	h.player.mu.Lock()
	defer h.player.mu.Unlock()
	if book := h.player.progress.book(skillID); book != nil {
		return book.maxLevel
	}
	return 0
}

// materializeStarterSkills 会话首学（DB 无 wuxue 行）时把 starter 基线物化
// 落库，防重登切 DB 模式只剩新学行（Java 同款决策）。
func materializeStarterSkills(h *itemSession) error {
	existing, err := h.store.dbSkillsExist(h.ctx, h.roleID)
	if err != nil || existing {
		return err
	}
	h.player.mu.Lock()
	learned := make([]string, 0, len(h.player.learnedSkills))
	for id := range h.player.learnedSkills {
		learned = append(learned, id)
	}
	h.player.mu.Unlock()
	skillRows, err := h.store.gameSkillRows(h.ctx, learned)
	if err != nil {
		return err
	}
	for _, id := range learned {
		skillType := skillRows[id].skillType
		if skillType == "" {
			skillType = "wuxue"
		}
		if err := h.store.upsertSkill(h.ctx, h.roleID, id, skillType); err != nil {
			return err
		}
	}
	log.Printf("%s: materialized %d starter skills into role_skills (first book learn)", h.remote, len(learned))
	return nil
}

// rebuildWuxueSkillView View40 整视图重开（0x16 删 → 0x15 容器 → 行帧；
// 行序=starter 顺序 + 已学集按 configID 排序追加，与登录 merge 口径一致）。
func rebuildWuxueSkillView(h *itemSession, skillRows map[string]gameSkillRow) error {
	h.player.mu.Lock()
	levels := make(map[string]int32, len(h.player.learnedSkills))
	for id, level := range h.player.learnedSkills {
		levels[id] = level
	}
	h.player.mu.Unlock()
	views := append([]learnedSkillView(nil), starterSkillViews...)
	inStarter := make(map[string]bool, len(views))
	for index := range views {
		inStarter[views[index].configID] = true
		if level, learned := levels[views[index].configID]; learned && level > 1 {
			views[index].level = level
			if views[index].maxLevel < level {
				views[index].maxLevel = installedCombatSkillCatalog.maxLevelOf(views[index].configID)
			}
		}
	}
	extras := make([]string, 0, len(levels))
	for id := range levels {
		if !inStarter[id] {
			extras = append(extras, id)
		}
	}
	sort.Strings(extras)
	for _, id := range extras {
		row := skillRows[id]
		view := learnedSkillView{configID: id, level: levels[id], maxLevel: installedCombatSkillCatalog.maxLevelOf(id)}
		if row.skillID != "" {
			view.staticData, view.itemType = row.staticData, row.itemType
		} else if definition, ok := installedCombatSkillCatalog.definition(id, 1); ok {
			view.staticData, view.itemType = definition.staticData, 1000
		}
		if view.maxLevel < view.level {
			view.maxLevel = view.level
		}
		views = append(views, view)
	}
	if err := h.link.WriteFrame(serverDeleteView(viewportSkill)); err != nil {
		return err
	}
	frames, err := learnedSkillViewFrames(views)
	if err != nil {
		return err
	}
	for _, frame := range frames {
		if err := h.link.WriteFrame(frame); err != nil {
			return err
		}
	}
	return nil
}

// rebuildNeiGongView View43 整视图重建（0x16 删 → viewFrames 全推）+ vital。
func rebuildNeiGongView(h *itemSession) error {
	if err := h.link.WriteFrame(serverDeleteView(viewportNeiGong)); err != nil {
		return err
	}
	h.player.mu.Lock()
	frames, err := h.player.progress.viewFrames()
	h.player.mu.Unlock()
	if err != nil {
		return err
	}
	for _, frame := range frames {
		if err := h.link.WriteFrame(frame); err != nil {
			return err
		}
	}
	vital, err := h.player.vitalUpdate()
	if err != nil {
		return err
	}
	return h.link.WriteFrame(vital)
}

// appendQingGongViewRows View46 追加行（行号接 starter 集 + 已追加集后）+ QingGongRec。
func appendQingGongViewRows(h *itemSession, ids []string) error {
	h.player.mu.Lock()
	base := len(starterQingGongIDs) + len(starterQingGongSkillIDs) + len(h.player.qingGongExtra) - len(ids)
	h.player.mu.Unlock()
	for offset, id := range ids {
		if err := h.link.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, recordQingGong, id)); err != nil {
			return fmt.Errorf("add QingGongRec %s: %w", id, err)
		}
		frame, err := serverViewAdd(viewportQingGong, uint16(base+offset+1), []serverViewProperty{
			viewString(qingGongConfigProperty, id),
		})
		if err != nil {
			return fmt.Errorf("encode QingGongContainer %s: %w", id, err)
		}
		if err := h.link.WriteFrame(frame); err != nil {
			return fmt.Errorf("add QingGongContainer %s: %w", id, err)
		}
	}
	return nil
}

// inStarterQingGong 目标是否已在 starter 轻功集。
func inStarterQingGong(id string) bool {
	for _, starter := range starterQingGongIDs {
		if starter == id {
			return true
		}
	}
	for _, starter := range starterQingGongSkillIDs {
		if starter == id {
			return true
		}
	}
	return false
}
