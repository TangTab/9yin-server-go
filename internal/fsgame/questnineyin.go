// questnineyin.go 九阴任务域配置装载（npcgen 由客户端 share/task 生成的
// testdata/ini/Quest/**.xml——任务 ID 与客户端文案键同一体系）→ 黑金复刻
// 引擎词汇（Quests/Criteria 两表）。
//
// 数据模型对齐（与黑金三表 Quest/QuestCriteria/Award 的差异）：
//
//	Task.xml                任务主表（Title 已按 TitleId 解析为客户端显示
//	                       文本）；九阴无独立子模板表——
//	SubFunction             接线表：ID=任务ID、Type=步骤域、TypeID=步骤行
//	                       ID、Order=步骤序，一条任务多行
//	HuntList/CollectList/   步骤域行（ID 域内唯一、跨域撞号，经 SubFunction
//	UseItemList/            TypeID 关联）
//	InteractiveList/Question
//
// 主表列映射（九阴 → 引擎词汇，映射不到的九阴列原样随行保留）：
//
//	Name←Title              AcceptNpc/SubmitNpc 直传
//	preQuestList←PreTaskLimit    postQuestList←NextTaskLimit
//	GiveUp←CanGiveup 反转（九阴 1=不能放弃；引擎 1=可放弃）
//	Share←ShareType≠0       TimeType←TimeLimitLength≠0（时限长由引擎面定）
//	CriteriaList            由 SubFunction 链合成（引擎侧无此九阴列）
//
// 子模板合成：引擎 QuestCriteriaRecord.qcID 是 word，九阴步骤行 ID 跨域
// 撞号（HuntList/CollectList 都有 4001），不能直用——按任务号升序 + 行序
// 确定性重编号（1..N），重启后同一任务同一编号，玩家在办进度不漂移。
//
// 步骤域 → 引擎 CriteriaType：
//
//	hunter   → 1 杀怪（InterNpc=NpcId，MaxCount=Count）
//	collect  → 5 收集（ItemID=ItemId，MaxCount=Count）
//	useitem  → 7 用物（ItemID=ItemID，MaxCount=ProCount）
//	interact → 4 对话（InterNpc=NpcId）
//	question → 4 对话（InterNpc=QuestionNpc）
//	story    → 4 对话（每行一段剧情：InterNpc=StoryTaskList 该组行的 NPC
//	           （源表只在首段行写 NpcID，续段行沿用组内上一行）；同 TypeID
//	           多行=多段对话，各成一条子模板，按 Order 顺序推进——
//	           advanceByType 的 talk 面对同任务剧情段门控，一次对话只推
//	           最前未完成段）
//	convoy   → 占位子模板（CriteriaType 留空：不被事件面误推，完成走
//	           cmd 934 通用推进面/未来护送运行时——物化 ConvoyNpc 沿
//	           PathID 走到终点后推进；NpcId/PathID/SceneID 溯源随行，
//	           护送 NPC 在 NpcConfig/NpcTask_ConvoyNpc 的命中校验入统计）
//	special/choose → 不合成（引擎侧无对应子模板类型，仅统计；#REF! 坏值同）。
package fsgame

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/sharedata"
)

// stepDomain SubFunction.Type → 步骤域文件（相对 ini/Quest）。
var stepDomain = map[string]string{
	"hunter":     "HuntList/HuntList.xml",
	"collect":    "CollectList/CollectList.xml",
	"useitem":    "UseItemList/UseItemList.xml",
	"interact":   "InteractiveList/InteractiveList.xml",
	"question":   "Question/Question.xml",
}

// nineyinQuestStats 装载统计（Tracef/排障面）。
type nineyinQuestStats struct {
	Tasks        int // 主表任务数
	SubFunctions int // SubFunction 总行
	WithSteps    int // 有可推进步骤（CriteriaList 非空）的任务
	Synthesized  int // 合成子模板数
	SkippedType  int // 未映射类型行（special/#REF!）
	MissingStep  int // TypeID 在步骤域查不到的行
	AwardsBuilt  int // 合成奖励行（TaskItem IsGive=0 给予物）
	ConvoyLinked  int // 护送 NPC 在 NpcConfig 命中的子模板
	ConvoyOrphan  int // 护送 NPC 未命中（NpcConfig 缺档）
}

// nineyinQuestData 九阴任务域装载产物（引擎词汇 + NPC 面）。
type nineyinQuestData struct {
	Quests, Criteria, Awards map[string]map[string]string
	// NpcAccept/NpcSubmit NPC → 任务号索引（任务侧 AcceptNpc/SubmitNpc
	// 反查；与 NpcConfig 的 TaskCanAccept/TaskCanSubmit 侧互为校验）。
	NpcAccept, NpcSubmit map[string][]string
	Stats                nineyinQuestStats
}

// loadNineyinQuests 装载九阴任务域并合成引擎词汇两表。
func loadNineyinQuests(resourcePath string) (*nineyinQuestData, error) {
	out := &nineyinQuestData{
		Quests:    make(map[string]map[string]string),
		Criteria:  make(map[string]map[string]string),
		Awards:    make(map[string]map[string]string),
		NpcAccept: make(map[string][]string),
		NpcSubmit: make(map[string][]string),
	}
	quests, criteria, stats := out.Quests, out.Criteria, &out.Stats

	// 1. 主表。
	taskRows, err := loadPropRows(resourcePath, "ini/Quest/Task/Task.xml")
	if err != nil {
		return nil, err
	}
	for _, row := range taskRows {
		id := row["ID"]
		if id == "" {
			continue
		}
		quests[id] = nineyinTaskRow(row)
		// NPC → 任务号索引（任务侧声明）。
		if npc := row["AcceptNpc"]; npc != "" {
			out.NpcAccept[npc] = append(out.NpcAccept[npc], id)
		}
		if npc := row["SubmitNpc"]; npc != "" {
			out.NpcSubmit[npc] = append(out.NpcSubmit[npc], id)
		}
	}
	stats.Tasks = len(quests)

	// 2. SubFunction 接线（文件序保留步骤顺序）。
	subRows, err := loadPropRows(resourcePath, "ini/Quest/SubFunction/SubFunction.xml")
	if err != nil {
		return nil, err
	}
	subfuncs := map[string][]map[string]string{}
	for _, row := range subRows {
		if row["ID"] == "" {
			continue
		}
		subfuncs[row["ID"]] = append(subfuncs[row["ID"]], row)
		stats.SubFunctions++
	}

	// 3. 步骤域装载（域内 ID 索引；StoryTaskList 同 ID 多行保序）。
	steps := map[string]map[string]map[string]string{}
	for _, domain := range stepDomain {
		rows, err := loadPropRows(resourcePath, "ini/Quest/"+domain)
		if err != nil {
			return nil, err
		}
		index := make(map[string]map[string]string, len(rows))
		for _, row := range rows {
			if row["ID"] != "" {
				index[row["ID"]] = row
			}
		}
		steps[domain] = index
	}
	storyRows, err := loadPropRows(resourcePath, "ini/Quest/StoryTaskList/StoryTaskList.xml")
	if err != nil {
		return nil, err
	}
	storyByID := map[string][]map[string]string{}
	for _, row := range storyRows {
		id := row["ID"]
		if id == "" {
			continue
		}
		// 源表只在首段行写 NpcID：续段行沿用组内上一行的 NPC。空行（无
		// contextId）不成段；与组内已有行完全相同的行是跨档复制重复，不
		// 重复成段（否则同一段要对两次话）。
		if row["NpcID"] == "" {
			if prev := storyByID[id]; len(prev) > 0 {
				row["NpcID"] = prev[len(prev)-1]["NpcID"]
			}
		}
		if row["contextId"] == "" {
			continue
		}
		repeated := false
		for _, prev := range storyByID[id] {
			if prev["NpcID"] == row["NpcID"] && prev["contextId"] == row["contextId"] {
				repeated = true
				break
			}
		}
		if repeated {
			continue
		}
		storyByID[id] = append(storyByID[id], row)
	}
	// 护送域与护送 NPC 配置命中集（NpcConfig 侧 best-effort：缺档只影响
	// 统计，不阻断任务装载）。
	convoyRows, err := loadPropRows(resourcePath, "ini/Quest/Convoy/Convoy.xml")
	if err != nil {
		return nil, err
	}
	convByID := map[string]map[string]string{}
	for _, row := range convoyRows {
		if row["ID"] != "" {
			convByID[row["ID"]] = row
		}
	}
	convoyNpcIDs := map[string]struct{}{}
	if rows, err := loadPropRows(resourcePath, "ini/Npc/NpcConfig/WorldNpc/NpcTask_ConvoyNpc.xml"); err == nil {
		for _, row := range rows {
			if row["ID"] != "" {
				convoyNpcIDs[row["ID"]] = struct{}{}
			}
		}
	}

	// 4. 按任务号升序合成 CriteriaList（确定性编号）。
	ids := make([]string, 0, len(quests))
	for id := range quests {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, aok := strconv.Atoi(ids[i])
		b, bok := strconv.Atoi(ids[j])
		if aok == nil && bok == nil {
			return a < b
		}
		return ids[i] < ids[j]
	})
	next := int32(0)
	var list []string
	emit := func(crit map[string]string) {
		next++
		cid := fmt.Sprint(next)
		crit["ID"] = cid
		criteria[cid] = crit
		list = append(list, cid)
	}
	storySeq := map[string]int32{} // 任务 → 剧情段播放序（合成序即播放序）
	for _, id := range ids {
		list = nil
		for _, sf := range subfuncs[id] {
			if sf["Type"] == "story" {
				// 剧情：同 TypeID 多行=多段对话，逐行成对话类子模板，
				// 播放序取任务内递增序（SubFunction 的 Order 对整段
				// story 是同值，不能作段间序）。
				rows := storyByID[sf["TypeID"]]
				if len(rows) == 0 {
					stats.MissingStep++
					continue
				}
				for _, row := range rows {
					storySeq[id]++
					emit(map[string]string{
						"TaskID": id, "StepType": "story", "StepID": sf["TypeID"],
						"Order": fmt.Sprint(storySeq[id]),
						"CriteriaType": CriteriaTalkNpc, "InterNpc": row["NpcID"],
						"ContextId": row["contextId"], "SceneID": row["SceneID"],
					})
				}
				continue
			}
			if sf["Type"] == "convoy" {
				// 护送：占位子模板（CriteriaType 留空不被事件面推进），
				// NpcId/PathID/SceneID 溯源 + NpcConfig 命中校验。
				row := convByID[sf["TypeID"]]
				if row == nil {
					stats.MissingStep++
					continue
				}
				crit := map[string]string{
					"TaskID": id, "StepType": "convoy", "StepID": sf["TypeID"],
					"Order": sf["Order"],
					"NpcId": row["NpcId"], "PathID": row["PathID"], "SceneID": row["SceneID"],
				}
				if _, ok := convoyNpcIDs[row["NpcId"]]; ok {
					stats.ConvoyLinked++
				} else {
					stats.ConvoyOrphan++
				}
				emit(crit)
				continue
			}
			domainPath, mappable := stepDomain[sf["Type"]]
			if !mappable {
				stats.SkippedType++
				continue
			}
			row := steps[domainPath][sf["TypeID"]]
			if row == nil {
				stats.MissingStep++
				continue
			}
			crit := map[string]string{"TaskID": id, "StepType": sf["Type"], "StepID": sf["TypeID"], "Order": sf["Order"]}
			switch sf["Type"] {
			case "hunter":
				crit["CriteriaType"], crit["InterNpc"], crit["MaxCount"] = CriteriaKillNpc, row["NpcId"], row["Count"]
			case "collect":
				crit["CriteriaType"], crit["ItemID"], crit["MaxCount"] = CriteriaCollect, row["ItemId"], row["Count"]
			case "useitem":
				crit["CriteriaType"], crit["ItemID"], crit["MaxCount"] = CriteriaUseItem, row["ItemID"], row["ProCount"]
			case "interact":
				crit["CriteriaType"], crit["InterNpc"] = CriteriaTalkNpc, row["NpcId"]
			case "question":
				crit["CriteriaType"], crit["InterNpc"] = CriteriaTalkNpc, row["QuestionNpc"]
			}
			emit(crit)
		}
		if len(list) > 0 {
			quests[id]["CriteriaList"] = strings.Join(list, ",")
			stats.WithSteps++
			stats.Synthesized += len(list)
		}
	}

	// 5. 奖励合成：Task.TaskItemList → TaskItem 行（独立 ID 空间）。
	// 经验/货币/物品奖励在 Prize 域（Prize.xml 已随 npcgen 出档，接线
	// 待做），此处 Exp/Money 置 0；给予物取 IsGive=0（无需上交的奖励行，
	// IsGive=1 是接取时给、交时按 back 回收的任务道具——由接取面发放，
	// 不进交任务奖励）。
	taskItems, err := loadPropRows(resourcePath, "ini/Quest/TaskItem/TaskItem.xml")
	if err != nil {
		return nil, err
	}
	itemByID := make(map[string]map[string]string, len(taskItems))
	for _, row := range taskItems {
		if row["ID"] != "" {
			if _, exists := itemByID[row["ID"]]; !exists {
				itemByID[row["ID"]] = row
			}
		}
	}
	for _, id := range ids {
		raw := quests[id]["TaskItemList"]
		if strings.TrimSpace(raw) == "" {
			continue
		}
		var items []string
		for _, ref := range strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == ',' }) {
			ref = strings.TrimSpace(ref)
			if row := itemByID[ref]; row != nil && row["IsGive"] == "0" && row["ItemId"] != "" {
				count := row["Count"]
				if count == "" {
					count = "1"
				}
				items = append(items, row["ItemId"]+","+count)
			}
		}
		if len(items) > 0 {
			out.Awards[id] = map[string]string{
				"ID": id, "Exp": "0", "Money": "0",
				"DefaultItemList": strings.Join(items, ";"),
			}
			// SubmitQuest 按 cfg["AwardID"] 查奖励——九阴无独立奖励号，
			// 以任务号自指。
			quests[id]["AwardID"] = id
			stats.AwardsBuilt++
		}
	}
	return out, nil
}

// nineyinTaskRow 九阴任务主表行 → 引擎词汇（九阴原列随行保留）。
func nineyinTaskRow(row map[string]string) map[string]string {
	cfg := make(map[string]string, len(row)+8)
	for k, v := range row {
		cfg[k] = v
	}
	cfg["Name"] = row["Title"]
	cfg["preQuestList"] = row["PreTaskLimit"]
	cfg["postQuestList"] = row["NextTaskLimit"]
	// GiveUp 语义反转：九阴 CanGiveup 1=不能放弃；引擎 GiveUp 1=可放弃。
	cfg["GiveUp"] = "1"
	if atoi32(row["CanGiveup"]) == 1 {
		cfg["GiveUp"] = "0"
	}
	cfg["Share"] = "0"
	if t := row["ShareType"]; t != "" && t != "0" {
		cfg["Share"] = "1"
	}
	cfg["TimeType"] = "0"
	if t := row["TimeLimitLength"]; t != "" && t != "0" {
		cfg["TimeType"] = "1"
	}
	return cfg
}

// loadPropRows 读 Object/Property XML 属性行（资源根 + 相对路径）。
//
// 数据根走 share 源时改道直读（2026-11-25，见 internal/npcfeed/source.go）：
// 任务域、任务文案、护送 NPC 模板表这几类产物路径由
// sharedata.BuiltRows 直接读客户端表给出同样的行（share/task/task 合并、
// text/chineses 文案档）；不归直读管的路径照旧读 XML 产物。
func loadPropRows(resourcePath, rel string) ([]map[string]string, error) {
	if rows, ok, err := sharedata.BuiltRows(resourcePath, rel); ok {
		if err != nil {
			return nil, fmt.Errorf("fsgame: 九阴任务域 %s（直读）: %w", rel, err)
		}
		return rows, nil
	}
	data, err := readResource(resourcePath, rel)
	if err != nil {
		return nil, fmt.Errorf("fsgame: 九阴任务域 %s: %w", rel, err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return nil, fmt.Errorf("fsgame: 九阴任务域 %s: %w", rel, err)
	}
	return rows, nil
}
