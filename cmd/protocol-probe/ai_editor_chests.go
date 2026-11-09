package main

import (
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync"
)

// 宝箱怪判定（官服 AI 编辑器方言，F 盘资源考据 2026-11-08）：
//
//	attacknpc 第 82 列 / bossnpc 第 73 列 AIEditorPackID →
//	share/rule/ai_editor_package.ini [包id] EventGroup=逗号分隔的事件节 id →
//	share/rule/ai_editor_event.ini [事件id] EventType=3（死亡）且
//	OpEvent=25（创建 NPC）时 Para1=死后生成的 BoxNpc/BossBox 宝箱模板
//	ConfigID（全表 1738 处 OpEvent=25、其中 1605 处挂在死亡事件）。
//
// 普通怪没有生箱事件：掉落按 DropID 挂尸体（CanPick/PickMember 属性驱动
// 客户端冒光+拾取光标，scene_obj_prop.lua on_change_pick_state）。
// 资源缺失时判定表为空（全部走挂尸），测试环境不依赖资源树。
var (
	aiEditorChestsOnce sync.Once
	aiEditorChests     map[string][]string
)

// chestSpawnConfigForNPC 返回怪物 AI 编辑包登记的死后宝箱模板列表
// （Extensions["template.AIEditorPackID"]，npc_resolver 对表列自动透传）。
func chestSpawnConfigForNPC(npc npcSpawn) []string {
	if npc.resolved.Extensions == nil {
		return nil
	}
	packID := npc.resolved.Extensions["template.AIEditorPackID"]
	if packID == "" {
		return nil
	}
	return aiEditorChestTemplates(packID)
}

// aiEditorChestTemplates 惰性装载双 ini 后按包 id 查死后宝箱模板。
func aiEditorChestTemplates(packID string) []string {
	aiEditorChestsOnce.Do(func() {
		packs, packErr := openResourceReader(filepath.Join(defaultModernShareRoot, "rule", "ai_editor_package.ini"))
		if packErr != nil {
			log.Printf("ai editor chest package table unavailable (%v); chest-monster detection disabled", packErr)
			aiEditorChests = map[string][]string{}
			return
		}
		events, eventErr := openResourceReader(filepath.Join(defaultModernShareRoot, "rule", "ai_editor_event.ini"))
		if eventErr != nil {
			_ = packs.Close()
			log.Printf("ai editor chest event table unavailable (%v); chest-monster detection disabled", eventErr)
			aiEditorChests = map[string][]string{}
			return
		}
		aiEditorChests = loadAIEditorChests(packs, events)
		_ = packs.Close()
		_ = events.Close()
	})
	return aiEditorChests[packID]
}

// loadAIEditorChests 解析 package/event 双 ini：包 id → 死亡事件(3)且
// 创建 NPC(25) 的 Para1 宝箱模板列表。readers 由调用方负责关闭。
func loadAIEditorChests(packageReader, eventReader io.Reader) map[string][]string {
	// 事件节：EventType=3 + OpEvent=25 → Para1 宝箱模板。
	chestEvents := map[string]string{}
	for section, fields := range parseAIEditorINI(eventReader) {
		if fields["EventType"] == "3" && fields["OpEvent"] == "25" && fields["Para1"] != "" {
			chestEvents[section] = fields["Para1"]
		}
	}
	// 包节：EventGroup=逗号分隔事件 id → 命中的宝箱模板列表。
	chests := map[string][]string{}
	for section, fields := range parseAIEditorINI(packageReader) {
		var templates []string
		for _, eventID := range strings.Split(fields["EventGroup"], ",") {
			eventID = strings.TrimSpace(eventID)
			if configID, ok := chestEvents[eventID]; ok {
				templates = append(templates, configID)
			}
		}
		if len(templates) > 0 {
			chests[section] = templates
		}
	}
	return chests
}

// parseAIEditorINI 极简 ini 解析（[节] key=value，# 或 ; 注释行）。
func parseAIEditorINI(reader io.Reader) map[string]map[string]string {
	sections := map[string]map[string]string{}
	current := ""
	var builder strings.Builder
	_, _ = io.Copy(&builder, reader)
	for _, line := range strings.Split(builder.String(), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(line[1 : len(line)-1])
			if _, ok := sections[current]; !ok {
				sections[current] = map[string]string{}
			}
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || current == "" {
			continue
		}
		sections[current][strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return sections
}
