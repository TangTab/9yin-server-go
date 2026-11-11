package main

// GM 武学三动作（Java LoginFlow.applyLearnAllTaolu/applyForgetAllTaolu/
// applyLearnAllNeigong 移植，面板「门派/武学套路」区块按钮的服务端落点）：
//
//   - learn_all_taolu   已学全部武学升 varprop 满级，档值落 data/武学等级.json
//     重登保持，逐行 0x10 刷 Level/MaxLevel/CurFill/TotalFill/VarPropNo；
//   - forget_all_taolu  回登录基线集（Go 模型=starter 并集基线），清档、删
//     role_skills 武学行、View40 整视图重开、摘除指向已遗忘技能的快捷栏；
//   - learn_all_neigong neigong.ini 全目录补书（level 1/曲线首级 total/power）
//     + View43 整视图重建 + role_faculty 快照落库；
//   - learn_taolu      整套学某套路（school_taolu.ini 精选集）直达指定层，
//     逐技钳 varprop 满级，秘籍链同款落库/重建，档值并入武学等级.json；
//   - learn_neigong    学单本内功直达指定层（钳 varprop 满级，只升不降），
//     曲线逐层推进 total/power，View43 重建，role_faculty 由调用方落库；
//   - forget_taolu     移除整套套路（learn_taolu 逆操作）：非基线招删
//     DB 行+等级档+快捷栏，基线招重置回 1 层，View40 重开+CanUse 批；
//   - forget_neigong   移除单本内功（learn_neigong 逆操作）：删书+装备
//     书/修炼目标重接（215 换书同款簇），基线书重登由种子复活回初始档。

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// skillLevelFileName 武学等级主档（Java SkillLevelStore 同款口径：
// 无 DB 快照——role_skills 无 level 列且由外部 dump 维护，不加列）。
const skillLevelFileName = "武学等级.json"

// skillLevelFileVersion 与 Java STORE_VERSION 对齐；不兼容版本拒载不覆写。
const skillLevelFileVersion = 1

// skillLevelFileStore 是 data/武学等级.json 主档：{version, roles:{<roleId>:
// {<skillId>: level}}}，仅存 >1 的等级（learn_all_taolu 满级档）。写入走
// 临时文件原子替换（Java Files.move ATOMIC_MOVE 同款），读档失败时拒写
// 防止把好档覆盖成空档（jyz gate 同方言）。
type skillLevelFileStore struct {
	path string
	mu   sync.Mutex
	// loadFailed 置位后 apply/clear 只改内存不落盘。
	loadFailed bool
	roles      map[role.RoleID]map[string]int32
}

var (
	skillLevelStoreOnce sync.Once
	skillLevelStoreData *skillLevelFileStore
)

// skillLevelStoreOf 惰性装载主档（首用时 NINEYIN_SERVER_ROOT 已定型；
// modernNeiGongCatalog 同款 sync.Once 模式）。文件缺失=空档正常态。
func skillLevelStoreOf() *skillLevelFileStore {
	skillLevelStoreOnce.Do(func() {
		skillLevelStoreData = openSkillLevelStore(filepath.Join(runtimeProjectPath("data"), skillLevelFileName))
	})
	return skillLevelStoreData
}

type skillLevelFileDoc struct {
	Version int32                       `json:"version"`
	Roles   map[string]map[string]int32 `json:"roles,omitempty"`
}

func openSkillLevelStore(path string) *skillLevelFileStore {
	store := &skillLevelFileStore{
		path:  path,
		roles: make(map[role.RoleID]map[string]int32),
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			store.loadFailed = true
			log.Printf("wuxue level store: read %s failed: %v (levels degrade to 1, refuse future writes)", path, err)
		}
		return store
	}
	var doc skillLevelFileDoc
	if err := json.Unmarshal(stripJSONBOM(raw), &doc); err != nil {
		store.loadFailed = true
		log.Printf("wuxue level store: decode %s failed: %v (levels degrade to 1, refuse future writes)", path, err)
		return store
	}
	if doc.Version != skillLevelFileVersion {
		store.loadFailed = true
		log.Printf("wuxue level store: unsupported version %d in %s (refuse future writes)", doc.Version, path)
		return store
	}
	for roleKey, levels := range doc.Roles {
		roleID, parseErr := strconv.ParseInt(roleKey, 10, 64)
		if parseErr != nil || roleID == 0 || len(levels) == 0 {
			continue
		}
		effective := make(map[string]int32, len(levels))
		for skillID, level := range levels {
			if level > 1 {
				effective[skillID] = level
			}
		}
		if len(effective) > 0 {
			store.roles[role.RoleID(roleID)] = effective
		}
	}
	return store
}

// levelOf 单技能档值；无档/无该技能 → 0（调用方默认 1）。
func (s *skillLevelFileStore) levelOf(roleID role.RoleID, skillID string) int32 {
	if s == nil || roleID == 0 || skillID == "" {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if levels, exists := s.roles[roleID]; exists {
		return levels[skillID]
	}
	return 0
}

// apply 整角色覆写（learn_all_taolu；空表等价 clear）。
func (s *skillLevelFileStore) apply(roleID role.RoleID, levels map[string]int32) error {
	if roleID == 0 {
		return fmt.Errorf("wuxue level store: zero role id")
	}
	effective := make(map[string]int32, len(levels))
	for skillID, level := range levels {
		if level > 1 {
			effective[skillID] = level
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(effective) == 0 {
		delete(s.roles, roleID)
	} else {
		s.roles[roleID] = effective
	}
	return s.persistLocked()
}

// clear 清角色档（forget_all_taolu；无档幂等）。
func (s *skillLevelFileStore) clear(roleID role.RoleID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.roles[roleID]; !exists {
		return nil
	}
	delete(s.roles, roleID)
	return s.persistLocked()
}

// persistLocked MarshalIndent 输出 + 临时文件原子替换（roleId → skillId → level
// 均字典序稳定，Java GoJson 同构）。调用方持有 s.mu。
func (s *skillLevelFileStore) persistLocked() error {
	if s.loadFailed {
		return fmt.Errorf("wuxue level store: load previously failed, refuse to overwrite %s", s.path)
	}
	doc := skillLevelFileDoc{Version: skillLevelFileVersion}
	if len(s.roles) > 0 {
		doc.Roles = make(map[string]map[string]int32, len(s.roles))
		for roleID, levels := range s.roles {
			if len(levels) == 0 {
				continue
			}
			doc.Roles[fmt.Sprintf("%d", int64(roleID))] = levels
		}
	}
	encoded, err := json.MarshalIndent(doc, "", "\t")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".wuxue-levels-*.tmp")
	if err != nil {
		return err
	}
	tempName := temporary.Name()
	if _, err := temporary.Write(append(encoded, '\n')); err != nil {
		temporary.Close()
		os.Remove(tempName)
		return err
	}
	if err := temporary.Close(); err != nil {
		os.Remove(tempName)
		return err
	}
	if err := os.Rename(tempName, s.path); err != nil {
		os.Remove(tempName)
		return err
	}
	return nil
}

// totalFillDisplay 修炼总量显示值（View40 行 TotalFillValue）。Java
// SkillInitTable 口径在现代数据下的等价实现：skill_new 无 total 键、
// skill_init 八键样板 total 又被 skill_new 同名节覆写为 0 → 实际全部
// 回落 starter 内联 total（156550），目录外技能 0。
func totalFillDisplay(configID string) int32 {
	for _, skill := range starterSkillViews {
		if skill.configID == configID {
			return skill.total
		}
	}
	return 0
}

// skillRowLevelUpFrame 是 View40 技能行等级帧（Java FightMessages.
// skillRowLevelUp 同构）：0x10 | isView=1 | u32 40 | u32 行号 | u16 1 |
// Level(Byte@1465)/MaxLevel(2083)/CurFillValue(764)/TotalFillValue(765)
// 同帧更新；varPropNo 非空时随帧更新（按级 varprop 行号），nil=保持现值。
func skillRowLevelUpFrame(row uint16, level, maxLevel, curFill, totalFill int32, varPropNo *int32) ([]byte, error) {
	if level < 1 {
		return nil, fmt.Errorf("skill row level-up: level %d < 1", level)
	}
	properties := []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("Level").Index, Name: "Level", Value: clientdata.ByteValue(byte(level))},
		{Index: propMaxLevel, Name: "MaxLevel", Value: clientdata.Int32Value(maxLevel)},
		{Index: propCurFillValue, Name: "CurFillValue", Value: clientdata.Int32Value(curFill)},
		{Index: propTotalFillValue, Name: "TotalFillValue", Value: clientdata.Int32Value(totalFill)},
	}
	if varPropNo != nil {
		properties = append(properties, clientdata.IndexedProperty{
			Index: propVarPropNo, Name: "VarPropNo", Value: clientdata.Int32Value(*varPropNo),
		})
	}
	return sceneObjectProperties(uint32(viewportSkill), uint32(row), 1, properties)
}

// applyStoredSkillLevels 登录档值合并（Java learnedSkillLevels + 带等级
// grantStarterProgress 合体）：View40 行 Level/MaxLevel 与释放闸
// learnedSkills 出自同一档值，修为显示 CurFill=TotalFill=totalFillDisplay
// （仅 >1 级行）。幂等可重放（0x0C 重入每代都过一遍）。返回副本——
// 入参可能是包级 starterSkillViews 本体（mergeLearnedSkills 空集直返），
// 原地改写会污染全局基线（跨会话/跨角色泄漏等级）。
func applyStoredSkillLevels(player *playerActor, roleID role.RoleID, store *skillLevelFileStore, rows []learnedSkillView) []learnedSkillView {
	if player == nil || store == nil || roleID == 0 {
		return rows
	}
	levels := make(map[string]int32)
	player.mu.Lock()
	for id := range player.learnedSkills {
		if stored := store.levelOf(roleID, id); stored > 1 {
			levels[id] = stored
		}
	}
	for id, level := range levels {
		player.learnedSkills[id] = level
	}
	player.mu.Unlock()
	if len(levels) == 0 {
		return rows
	}
	merged := append([]learnedSkillView(nil), rows...)
	for index := range merged {
		level, stored := levels[merged[index].configID]
		if !stored || level <= merged[index].level {
			continue
		}
		merged[index].level = level
		if merged[index].maxLevel < level {
			merged[index].maxLevel = level
		}
		merged[index].fill = totalFillDisplay(merged[index].configID)
		merged[index].total = merged[index].fill
	}
	return merged
}

// applyGMLearnAllTaolu 一键学满：已学全部技能升 varprop 满级
// （catalog.maxLevelOf），档值落 data/武学等级.json，逐行等级帧刷新。
// 出招链 definition(id, level) 即时按高级编译——伤害/消耗/CD 全生效。
func applyGMLearnAllTaolu(link sceneMessageConnection, player *playerActor, roleID role.RoleID, store *skillLevelFileStore, remote string) (string, error) {
	current := player.learnedSkillsSnapshot()
	if len(current) == 0 {
		return "", fmt.Errorf("尚未学会任何武学")
	}
	levels := make(map[string]int32, len(current))
	raised := 0
	for id, existing := range current {
		max := installedCombatSkillCatalog.maxLevelOf(id)
		if max < 1 {
			max = 1
		}
		levels[id] = max
		if max > existing {
			raised++
		}
	}
	player.replaceLearnedSkills(levels)
	if err := store.apply(roleID, levels); err != nil {
		log.Printf("%s: persist wuxue levels: %v", remote, err)
	}
	order := player.skillViewRowOrder()
	for _, id := range order {
		if id == "" {
			continue
		}
		level := levels[id]
		if level <= 1 {
			continue
		}
		var varPropNo *int32
		if supplement := installedCombatSkillCatalog.skillRowData(id, level); supplement != nil {
			value := supplement.varPropNo
			varPropNo = &value
		}
		totalFill := totalFillDisplay(id)
		frame, frameErr := skillRowLevelUpFrame(player.skillViewRow(id), level, level, totalFill, totalFill, varPropNo)
		if frameErr != nil {
			return "", frameErr
		}
		if err := link.WriteFrame(frame); err != nil {
			return "", err
		}
	}
	message := fmt.Sprintf("已把 %d 门武学升至满级（其中 %d 门升档）", len(levels), raised)
	log.Printf("%s: GM learn all taolu role=%d skills=%d raised=%d", remote, roleID, len(levels), raised)
	return message, nil
}

// applyGMForgetAllTaolu 删除全部武学：回 Go 登录基线集（starter 并集——
// Go 的 View40 恒为 starter∪DB 模型，删净 role_skills 武学/暗器行后重登
// 即 starter 集）。View40 整视图重开（0x16→0x15→行帧→CanUse 批），
// 等级档清空，指向已遗忘技能的快捷栏槽同步摘除。
func applyGMForgetAllTaolu(link sceneMessageConnection, player *playerActor, roleID role.RoleID, store *skillLevelFileStore, inventory *inventoryStore, shortcuts *shortcutDBStore, ctx context.Context, remote string) (message string, err error) {
	// ① DB 武学/暗器行整删（best-effort：缺库/失败只记日志，内存照常重置）。
	if inventory != nil {
		if deleteErr := inventory.deleteSkillsByTypes(ctx, roleID, []string{"wuxue", "anqi"}); deleteErr != nil {
			log.Printf("%s: delete role_skills wuxue rows: %v", remote, deleteErr)
		}
	}
	// ② 等级档清空。
	if clearErr := store.clear(roleID); clearErr != nil {
		log.Printf("%s: clear wuxue level store: %v", remote, clearErr)
	}
	// ③ 内存基线重置（starter 集，全 1 级）。
	baseline := make([]learnedSkillView, len(starterSkillViews))
	learned := make(map[string]int32, len(starterSkillViews))
	for index, skill := range starterSkillViews {
		baseline[index] = skill
		learned[skill.configID] = 1
	}
	player.replaceLearnedSkills(learned)
	// ④ View40 整视图重开 + 行号映射刷新。
	if deleteErr := link.WriteFrame(serverDeleteView(viewportSkill)); deleteErr != nil {
		return "", deleteErr
	}
	player.noteSkillViewRows(baseline)
	frames, framesErr := learnedSkillViewFrames(baseline)
	if framesErr != nil {
		return "", framesErr
	}
	for _, frame := range frames {
		if writeErr := link.WriteFrame(frame); writeErr != nil {
			return "", writeErr
		}
	}
	for row := uint16(1); row <= uint16(len(baseline)); row++ {
		frame, canUseErr := skillRowCanUseFrame(row, true)
		if canUseErr != nil {
			return "", canUseErr
		}
		if writeErr := link.WriteFrame(frame); writeErr != nil {
			return "", writeErr
		}
	}
	// ⑤ 摘除指向已遗忘技能的快捷栏槽（msg201 同款：内存删 + 整表重推 + DB 删行）。
	removed := 0
	snapshot := player.shortcutSnapshot()
	for _, shortcut := range snapshot {
		if shortcut.kind != "skill" {
			continue
		}
		if _, learnedStill := player.learnedSkillLevel(shortcut.id); learnedStill {
			continue
		}
		if _, _, exists := player.removeShortcutIndex(shortcut.index); exists {
			removed++
			if shortcuts != nil {
				if deleteErr := shortcuts.DeleteIndex(roleID, shortcut.index); deleteErr != nil {
					log.Printf("%s: delete forgotten shortcut %d: %v", remote, shortcut.index, deleteErr)
				}
			}
		}
	}
	if removed > 0 {
		if rebuildErr := rebuildShortcutRecord(link, player, len(snapshot)); rebuildErr != nil {
			log.Printf("%s: rebuild shortcut record after forget: %v", remote, rebuildErr)
		}
	}
	message = fmt.Sprintf("已删除全部武学，重置为初始套路集（剩余 %d 项）", len(baseline))
	log.Printf("%s: GM forget all taolu role=%d baseline=%d shortcuts_removed=%d", remote, roleID, len(baseline), removed)
	return message, nil
}

// applyGMLearnAllNeiGong 一键收录全部内功：neigong.ini 全目录补书
// （level 1 / 曲线首级 total/power，varprop 效果 best-effort 回填）→
// View43 整视图重建 + vital。已有书等级/修为不动；重登按 role_faculty
// 快照恢复。
func applyGMLearnAllNeiGong(link sceneMessageConnection, player *playerActor, roleID role.RoleID, remote string) (string, error) {
	catalog, catalogErr := loadModernNeiGongCatalog()
	if catalogErr != nil {
		return "", fmt.Errorf("内功目录为空（share 资源不可用）")
	}
	ids := make([]string, 0, len(catalog.books))
	for id := range catalog.books {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	player.mu.Lock()
	added := 0
	for _, id := range ids {
		def := catalog.books[id]
		if player.progress.book(def.configID) != nil {
			continue
		}
		capLevel := catalog.maxLevelOf(def.staticData)
		if capLevel < 1 {
			capLevel = 1
		}
		player.progress.learnOrRaiseBook(def, capLevel, 0, 0)
		added++
	}
	player.mu.Unlock()
	if added == 0 {
		message := fmt.Sprintf("全部内功已收录（共 %d 本，无新增）", player.progressBookCount())
		log.Printf("%s: GM learn all neigong role=%d added=0", remote, roleID)
		return message, nil
	}
	session := &itemSession{link: link, player: player}
	if rebuildErr := rebuildNeiGongView(session); rebuildErr != nil {
		return "", rebuildErr
	}
	message := fmt.Sprintf("已收录全部内功：新增 %d 本（共 %d 本）", added, player.progressBookCount())
	log.Printf("%s: GM learn all neigong role=%d added=%d", remote, roleID, added)
	return message, nil
}

// applyGMLearnTaolu 整套学某套路：school_taolu.ini 精选集全招式入已学表
// 并直达目标层（wantLevel≤0=逐技各自 varprop 满级；超过该技上限按上限
// 给，learn_all_taolu 同款夹制）。新学行走秘籍链同款流程：starter 基线
// 物化 → role_skills 落库 → View40 整视图重开；等级档（武学等级.json）
// 并入整角色覆写，重登保持。
func applyGMLearnTaolu(link sceneMessageConnection, player *playerActor, roleID role.RoleID, store *skillLevelFileStore, inventory *inventoryStore, ctx context.Context, taoluKey string, wantLevel int32, remote string) (string, error) {
	taolu, found := gmTaolus.resolve(taoluKey)
	if !found {
		return "", fmt.Errorf("未知套路：%s（目录共 %d 套）", taoluKey, len(gmTaolus.rows))
	}
	skillRows, err := inventory.gameSkillRows(ctx, taolu.Skills)
	if err != nil {
		log.Printf("%s: GM learn taolu %s: load skill rows: %v", remote, taolu.Key, err)
	}
	type grantRow struct {
		id        string
		level     int32
		skillType string
	}
	grants := make([]grantRow, 0, len(taolu.Skills))
	skipped := make([]string, 0, 4)
	clamped := 0
	for _, id := range taolu.Skills {
		row, exists := skillRows[id]
		if !exists || (row.skillType != "wuxue" && row.skillType != "anqi") {
			skillType := row.skillType
			if skillType == "" {
				skillType = "目录缺席"
			}
			skipped = append(skipped, fmt.Sprintf("%s(%s)", id, skillType))
			continue
		}
		level := wantLevel
		if level <= 0 {
			level = installedCombatSkillCatalog.maxLevelOf(id)
		} else if max := installedCombatSkillCatalog.maxLevelOf(id); max > 0 && level > max {
			level = max
			clamped++
		}
		if level < 1 {
			level = 1
		}
		current, learned := player.learnedSkillLevel(id)
		if learned && current >= level {
			continue
		}
		grants = append(grants, grantRow{id: id, level: level, skillType: row.skillType})
	}
	if len(grants) == 0 {
		var message string
		if len(skipped) >= len(taolu.Skills) && len(skipped) > 0 {
			// 占位/目录缺席套路（真实目录装载时已过滤，这里兜底防御）。
			message = fmt.Sprintf("《%s》无可学招式：%s", taolu.Label, strings.Join(skipped, "、"))
		} else {
			message = fmt.Sprintf("《%s》%d 招已全部学至目标层", taolu.Label, len(taolu.Skills))
			if len(skipped) > 0 {
				message += fmt.Sprintf("（跳过 %d 招：%s）", len(skipped), strings.Join(skipped, "、"))
			}
		}
		log.Printf("%s: GM learn taolu %s role=%d no-op skipped=%v", remote, taolu.Key, roleID, skipped)
		return message, nil
	}
	session := &itemSession{link: link, player: player, store: inventory, ctx: ctx, roleID: roleID, remote: remote}
	// starter 回退会话首学（DB 无 wuxue 行）：先物化基线，防重登切 DB 模式丢 starter。
	if err := materializeStarterSkills(session); err != nil {
		log.Printf("%s: materialize starter skill baseline: %v", remote, err)
	}
	player.mu.Lock()
	for _, grant := range grants {
		player.learnedSkills[grant.id] = grant.level
	}
	player.mu.Unlock()
	for _, grant := range grants {
		if err := inventory.upsertSkill(ctx, roleID, grant.id, grant.skillType); err != nil {
			log.Printf("%s: upsert skill %s: %v", remote, grant.id, err)
		}
	}
	if err := rebuildWuxueSkillView(session, skillRows); err != nil {
		return "", err
	}
	// 等级档整角色覆写（与 learn_all_taolu 同档）：>1 层落 data/武学等级.json。
	levels := player.learnedSkillsSnapshot()
	if err := store.apply(roleID, levels); err != nil {
		log.Printf("%s: persist wuxue levels: %v", remote, err)
	}
	message := fmt.Sprintf("已学《%s》：%d 招直达目标层", taolu.Label, len(grants))
	if clamped > 0 {
		message += fmt.Sprintf("（%d 招超过上限按上限给）", clamped)
	}
	if wantLevel > 0 {
		highest := int32(0)
		for _, grant := range grants {
			if grant.level > highest {
				highest = grant.level
			}
		}
		message += fmt.Sprintf("，本次最高 %d 层", highest)
	}
	if len(skipped) > 0 {
		message += fmt.Sprintf("（跳过 %d 招：%s）", len(skipped), strings.Join(skipped, "、"))
	}
	log.Printf("%s: GM learn taolu %s role=%d granted=%d clamped=%d skipped=%v",
		remote, taolu.Key, roleID, len(grants), clamped, skipped)
	return message, nil
}

// applyGMLearnNeiGong 学单本内功并直达目标层：wantLevel≤0=varprop 满级，
// 超上限按上限给。新书建书即达层（total/power 按曲线逐层推进，与传承
// 直升同款循环）；已有书只升不降（层与上限均单调，秘籍链同口径）。
// 修炼中/装备中的书变档后重算资源上限，View43 整视图重建；重登按
// role_faculty 快照恢复（调用方 persistFaculty）。
func applyGMLearnNeiGong(link sceneMessageConnection, player *playerActor, roleID role.RoleID, neigongID string, wantLevel int32, remote string) (string, error) {
	catalog, catalogErr := loadModernNeiGongCatalog()
	if catalogErr != nil {
		return "", fmt.Errorf("内功目录为空（share 资源不可用）")
	}
	def, hasBook := catalog.book(strings.TrimSpace(neigongID))
	if !hasBook {
		return "", fmt.Errorf("未知内功：%s（目录共 %d 本）", neigongID, len(catalog.books))
	}
	capLevel := catalog.maxLevelOf(def.staticData)
	if capLevel < 1 {
		capLevel = 1
	}
	target := wantLevel
	if target <= 0 {
		target = capLevel
	}
	if target > capLevel {
		target = capLevel
	}
	player.mu.Lock()
	book := player.progress.book(def.configID)
	created := false
	if book == nil {
		player.progress.learnOrRaiseBook(def, target, 0, 0)
		book = player.progress.book(def.configID)
		created = book != nil
	}
	changed := created
	if book != nil {
		if target > book.maxLevel {
			book.maxLevel = target
			changed = true
		}
		for book.level < target {
			book.level++
			levelUpNeiGongTotal(book)
			book.maxPower += 20
			book.power = book.maxPower
			changed = true
		}
		if changed {
			if err := book.applyModernEffect(); err != nil {
				log.Printf("%s: apply neigong effect %s: %v", remote, def.configID, err)
			}
		}
	}
	if changed {
		// 目标可能是当前装备/修炼中的书：varprop 加成变化须重算推导上限。
		player.syncEquippedNeiGongResourcesLocked()
	}
	player.mu.Unlock()
	if !changed {
		message := fmt.Sprintf("《%s》已至 %d 层（满 %d 层），无需变更", gmNeiGongName(def.configID), target, capLevel)
		log.Printf("%s: GM learn neigong %s role=%d no-op", remote, def.configID, roleID)
		return message, nil
	}
	session := &itemSession{link: link, player: player}
	if rebuildErr := rebuildNeiGongView(session); rebuildErr != nil {
		return "", rebuildErr
	}
	action := "升至"
	if created {
		action = "收录并直达"
	}
	message := fmt.Sprintf("已%s《%s》%d 层（满 %d 层）", action, gmNeiGongName(def.configID), target, capLevel)
	if wantLevel > capLevel {
		message += fmt.Sprintf("，目标 %d 层超上限已按上限给", wantLevel)
	}
	log.Printf("%s: GM learn neigong %s role=%d level=%d cap=%d created=%t",
		remote, def.configID, roleID, target, capLevel, created)
	return message, nil
}

// gmNeiGongName 内功书显示名（stringname.idres；缺行回退 configId）。
func gmNeiGongName(configID string) string {
	if name := sceneNameTablesOf().nameByKey[strings.ToLower(configID)]; name != "" {
		return name
	}
	return configID
}

// starterNeiGongBookIDs 登录基线内功书集（newPlayerProgress 种子六本；
// GM forget_neigong 删除后重登由种子复活回初始档，消息按此注明）。
var starterNeiGongBookIDs = func() map[string]bool {
	ids := make(map[string]bool)
	for _, book := range newPlayerProgress().books {
		ids[book.configID] = true
	}
	return ids
}()

// starterSkillIDsLower 登录基线招式集（小写键；starterSkillRowIDs 是
// 混合大小写、供行帧方言判定用，forget_taolu 的分类按小写比对更稳——
// school_taolu.ini 与 starter 目录的大小写偶有出入）。
var starterSkillIDsLower = func() map[string]bool {
	ids := make(map[string]bool, len(starterSkillViews))
	for _, skill := range starterSkillViews {
		ids[strings.ToLower(skill.configID)] = true
	}
	return ids
}()

// applyGMForgetTaolu 移除整套套路（learn_taolu 的逆操作）：
//   - 非 starter 招式：删已学表 + role_skills 行 + 等级档条目，View40
//     整视图重开后行消失；指向它们的快捷栏槽同步摘除；
//   - starter 基线招式（花神/心佛/太极古谱等）：登录基线不可删，重置
//     回 1 层并清档值（重登口径=mergeLearnedSkills 的 starter 集）。
//
// 返回完全删除的 ConfigID 列表（调用方过滤会话级 dbLearnedSkills，
// 防 0x0C 重入复活已删行）。
func applyGMForgetTaolu(link sceneMessageConnection, player *playerActor, roleID role.RoleID, store *skillLevelFileStore, inventory *inventoryStore, shortcuts *shortcutDBStore, ctx context.Context, taoluKey string, remote string) ([]string, string, error) {
	taolu, found := gmTaolus.resolve(taoluKey)
	if !found {
		return nil, "", fmt.Errorf("未知套路：%s（目录共 %d 套）", taoluKey, len(gmTaolus.rows))
	}
	skillRows, err := inventory.gameSkillRows(ctx, taolu.Skills)
	if err != nil {
		log.Printf("%s: GM forget taolu %s: load skill rows: %v", remote, taolu.Key, err)
	}
	removed := make([]string, 0, len(taolu.Skills))
	reset := 0
	player.mu.Lock()
	for _, id := range taolu.Skills {
		if starterSkillIDsLower[strings.ToLower(id)] {
			// 登录基线招：只重置等级（>1 才计档）。
			if current, learned := player.learnedSkills[id]; learned && current > 1 {
				player.learnedSkills[id] = 1
				reset++
			}
			continue
		}
		if _, learned := player.learnedSkills[id]; learned {
			removed = append(removed, id)
		}
		delete(player.learnedSkills, id)
	}
	player.mu.Unlock()
	// DB 行删除（含未学但残留的脏行；starter 行不删——重登 starter 集
	// 并集语义下删了也会由基线补齐，保留行等价）。
	for _, id := range taolu.Skills {
		if starterSkillIDsLower[strings.ToLower(id)] {
			continue
		}
		if deleteErr := inventory.deleteSkill(ctx, roleID, id); deleteErr != nil {
			log.Printf("%s: delete role_skills %s: %v", remote, id, deleteErr)
		}
	}
	// 等级档整角色覆写：被移除/重置的条目随之出档。
	levels := player.learnedSkillsSnapshot()
	if applyErr := store.apply(roleID, levels); applyErr != nil {
		log.Printf("%s: persist wuxue levels: %v", remote, applyErr)
	}
	session := &itemSession{link: link, player: player, store: inventory, ctx: ctx, roleID: roleID, remote: remote}
	if rebuildErr := rebuildWuxueSkillView(session, skillRows); rebuildErr != nil {
		return nil, "", rebuildErr
	}
	// 可用性心跳整批（forget_all_taolu 同款：整视图重开后逐行补发）。
	if rowOrder := player.skillViewRowOrder(); len(rowOrder) > 0 {
		for row := uint16(1); row <= uint16(len(rowOrder)); row++ {
			frame, canUseErr := skillRowCanUseFrame(row, true)
			if canUseErr != nil {
				return nil, "", canUseErr
			}
			if writeErr := link.WriteFrame(frame); writeErr != nil {
				return nil, "", writeErr
			}
		}
	}
	// 摘除指向已删招式的快捷栏槽（内存删 + 整表重推 + DB 删行）。
	shortcutsRemoved := 0
	if len(removed) > 0 {
		unlearned := make(map[string]bool, len(removed))
		for _, id := range removed {
			unlearned[id] = true
		}
		snapshot := player.shortcutSnapshot()
		for _, shortcut := range snapshot {
			if shortcut.kind != "skill" || !unlearned[shortcut.id] {
				continue
			}
			if _, _, exists := player.removeShortcutIndex(shortcut.index); exists {
				shortcutsRemoved++
				if shortcuts != nil {
					if deleteErr := shortcuts.DeleteIndex(roleID, shortcut.index); deleteErr != nil {
						log.Printf("%s: delete forgotten shortcut %d: %v", remote, shortcut.index, deleteErr)
					}
				}
			}
		}
		if shortcutsRemoved > 0 {
			if rebuildErr := rebuildShortcutRecord(link, player, len(snapshot)); rebuildErr != nil {
				log.Printf("%s: rebuild shortcut record after forget: %v", remote, rebuildErr)
			}
		}
	}
	message := fmt.Sprintf("已移除《%s》：删除 %d 招", taolu.Label, len(removed))
	if reset > 0 {
		message += fmt.Sprintf("、%d 招属登录基线重置回 1 层（重登保持基线）", reset)
	}
	if shortcutsRemoved > 0 {
		message += fmt.Sprintf("、摘除 %d 个失效快捷栏", shortcutsRemoved)
	}
	if len(removed) == 0 && reset == 0 {
		message = fmt.Sprintf("《%s》无可移除招式（未学习或均为基线初始态）", taolu.Label)
	}
	log.Printf("%s: GM forget taolu %s role=%d removed=%d reset=%d shortcuts=%d",
		remote, taolu.Key, roleID, len(removed), reset, shortcutsRemoved)
	return removed, message, nil
}

// applyGMForgetNeiGong 移除单本内功（learn_neigong 的逆操作）：从书架
// 删除该书；若为当前装备书则自动换装余下首本（215 换书同款状态清理 +
// 怒气上限分母帧），若为修炼目标则回退到装备书并退出修炼态。View43
// 整视图重建 + role_faculty 快照落库（调用方 persistFaculty）。登录
// 基线书（starterNeiGongBookIDs）重登由种子复活回初始档，消息注明。
func applyGMForgetNeiGong(link sceneMessageConnection, player *playerActor, roleID role.RoleID, neigongID string, remote string) (string, error) {
	catalog, catalogErr := loadModernNeiGongCatalog()
	if catalogErr != nil {
		return "", fmt.Errorf("内功目录为空（share 资源不可用）")
	}
	id := strings.TrimSpace(neigongID)
	if _, ok := catalog.book(id); !ok {
		return "", fmt.Errorf("未知内功：%s（目录共 %d 本）", neigongID, len(catalog.books))
	}
	// 战斗面板快照须在持锁外取（combatPanelProperties 内部自取 p.mu，
	// 215 换书链同序：先快照后变更）。
	combatBefore := combatPanelSnapshot(player)
	player.mu.Lock()
	if player.progress.book(id) == nil {
		count := len(player.progress.books)
		player.mu.Unlock()
		message := fmt.Sprintf("《%s》未收录（书架共 %d 本），无需移除", gmNeiGongName(id), count)
		log.Printf("%s: GM forget neigong %s role=%d no-op", remote, id, roleID)
		return message, nil
	}
	kept := make([]learnedNeiGong, 0, len(player.progress.books))
	for _, book := range player.progress.books {
		if book.configID != id {
			kept = append(kept, book)
		}
	}
	player.progress.books = kept
	reequipped := ""
	if player.progress.curNeiGong == id && len(player.progress.books) > 0 {
		// 换装余下首本（equipNeiGong 同款状态清理，避免旧书脚本窗跨书存活）。
		reequipped = player.progress.books[0].configID
		player.progress.curNeiGong = reequipped
		player.progress.attributes.sp = player.progress.clampSP(player.progress.attributes.sp)
		player.neigongState = nil
		player.neigongShields = nil
		player.neigongDodge = neigongDodgeWindow{}
		delete(player.buffs, neigongProcBuffSlot)
		player.neigongPassive = neigongPassiveCache{}
	}
	if player.progress.facultyName == id {
		player.progress.facultyExit()
		player.progress.facultyName = player.progress.curNeiGong
	}
	player.syncEquippedNeiGongBuffLocked(time.Now())
	player.syncEquippedNeiGongResourcesLocked()
	remaining := len(player.progress.books)
	player.mu.Unlock()
	// 换装先刷怒气上限分母（215 同序：capFrame → vital 大批）。
	if reequipped != "" {
		capFrame, capErr := player.rageCapUpdate()
		if capErr != nil {
			return "", capErr
		}
		if writeErr := link.WriteFrame(capFrame); writeErr != nil {
			return "", writeErr
		}
	}
	session := &itemSession{link: link, player: player}
	if rebuildErr := rebuildNeiGongView(session); rebuildErr != nil {
		return "", rebuildErr
	}
	// 换书战斗面板增量（CurNeiGongWuXing/五维 Add 变化项）。
	pushCombatAttributeDelta(link, player, combatBefore, remote)
	message := fmt.Sprintf("已移除《%s》（书架剩 %d 本）", gmNeiGongName(id), remaining)
	if reequipped != "" {
		message += fmt.Sprintf("，装备书已换为《%s》", gmNeiGongName(reequipped))
	}
	if starterNeiGongBookIDs[id] {
		message += "；此书属登录基线，重登将回到初始档"
	}
	log.Printf("%s: GM forget neigong %s role=%d remaining=%d reequipped=%s starter=%t",
		remote, id, roleID, remaining, reequipped, starterNeiGongBookIDs[id])
	return message, nil
}
