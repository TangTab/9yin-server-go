package main

// GM 武学三动作（Java LoginFlow.applyLearnAllTaolu/applyForgetAllTaolu/
// applyLearnAllNeigong 移植，面板「门派/武学套路」区块按钮的服务端落点）：
//
//   - learn_all_taolu   已学全部武学升 varprop 满级，档值落 data/武学等级.json
//     重登保持，逐行 0x10 刷 Level/MaxLevel/CurFill/TotalFill/VarPropNo；
//   - forget_all_taolu  回登录基线集（Go 模型=starter 并集基线），清档、删
//     role_skills 武学行、View40 整视图重开、摘除指向已遗忘技能的快捷栏；
//   - learn_all_neigong neigong.ini 全目录补书（level 1/曲线首级 total/power）
//     + View43 整视图重建 + role_faculty 快照落库。

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

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
