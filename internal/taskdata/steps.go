package taskdata

// steps.go 任务步骤行（Task_Record 目标行的来源）与奖励串的合成。
//
// SubFunction 接线表（ID=任务号、Type=步骤域、TypeID=步骤行 ID、Order=步骤
// 序）× 各步骤域行 → 每任务一组 Step；Prize 域同号行 → 奖励串。列映射按
// 官服引导抓包的 Task_Record 行逐列定版（杀怪/交互/剧情/特殊四类有样本）：
//
//	hunter   子类型 1：Key=NpcId，MaxNum=Count，TrackInfo=TrackInfo，无超链接
//	interact 子类型 3：Key=NpcId，Scene=DialogID，TrackInfo=TrackInfo
//	story    子类型 6：Index=剧情组 ID，Key=组内首个 NpcID，TrackInfo=TrackInfoID
//	special  子类型 8：Index=SpecialType，Key=TypeID，TrackInfo=TrackInfoID
//
// 采集/用物/问答/护送没有样本，按同构类推（Key=步骤域的对象列、MaxNum=次数
// 列）。目标行里 scene 与超链接列的空值形态见 buildStep。

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// stepTables 各步骤域的行，键为 SubFunction.Type（story 域用剧情段表）。
type stepTables map[string][]map[string]string

// buildSteps SubFunction 接线行 × 步骤域行 → 任务号 → 步骤行（步骤序升序，
// 同序保持接线行序；步骤序重编为 1..n 连续值——目标行下标按它推算）。
// 接线表缺席或步骤行在域里查不到的步骤不出行。
func buildSteps(subRows []map[string]string, tables stepTables) map[int][]Step {
	if len(subRows) == 0 {
		return nil
	}
	byID := make(map[string]map[string]map[string]string, len(tables))
	for typ, rows := range tables {
		if typ == "story" {
			continue
		}
		index := make(map[string]map[string]string, len(rows))
		for _, row := range rows {
			id := strings.TrimSpace(row["ID"])
			if id != "" {
				if _, exists := index[id]; !exists {
					index[id] = row
				}
			}
		}
		byID[typ] = index
	}
	// 剧情组：组 ID → 首个带 NpcID 的行（续段行 NpcID 留空）。
	story := make(map[string]map[string]string)
	for _, row := range tables["story"] {
		id := strings.TrimSpace(row["ID"])
		if id == "" {
			continue
		}
		if story[id] == nil {
			story[id] = row
		} else if strings.TrimSpace(story[id]["NpcID"]) == "" && strings.TrimSpace(row["NpcID"]) != "" {
			story[id] = row
		}
	}

	steps := make(map[int][]Step)
	seen := make(map[string]bool, len(subRows))
	for _, sf := range subRows {
		taskID, err := strconv.Atoi(strings.TrimSpace(sf["ID"]))
		if err != nil || taskID <= 0 {
			continue
		}
		typ, typeID := strings.TrimSpace(sf["Type"]), strings.TrimSpace(sf["TypeID"])
		order := strings.TrimSpace(sf["Order"])
		// 接线表里有整行重复（1597 的 interact 496 两行同序）：官服只发一条
		// 目标行——同任务、同域、同步骤行、同序的只算一步。
		wire := strings.Join([]string{strconv.Itoa(taskID), typ, typeID, order}, "\x00")
		if seen[wire] {
			continue
		}
		seen[wire] = true
		step, ok := buildStep(typ, typeID, byID[typ][typeID], story[typeID])
		if !ok {
			continue
		}
		step.Order = iniInt(sf["Order"])
		steps[taskID] = append(steps[taskID], step)
	}
	for taskID, list := range steps {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Order < list[j].Order })
		for i := range list {
			list[i].Order = i + 1
		}
		steps[taskID] = list
	}
	return steps
}

// buildStep 一个接线行的步骤；步骤行在域里查不到返回 false。
func buildStep(typ, typeID string, row, story map[string]string) (Step, bool) {
	if typeID == "" {
		return Step{}, false
	}
	cell := func(key string) string {
		if row == nil {
			return ""
		}
		return strings.TrimSpace(row[key])
	}
	positive := func(value string) int {
		if n := iniInt(value); n > 0 {
			return n
		}
		return 1
	}
	// 目标行编码口径（官服四种样本）：交互 scene=DialogID；剧情/特殊
	// scene 是空串（长度 1）、带超链接；杀怪 scene 是 0 长度串、无超链接。
	// 采集/用物/护送/问答无样本，计数类目标（采集/用物）从杀怪、NPC 目标
	// 类（问答/护送）从交互。
	npc := Step{Index: iniInt(typeID), MaxNum: 1, HasLink: true}
	counted := Step{Index: iniInt(typeID), MaxNum: 1, NullScene: true}
	empty := Step{Index: iniInt(typeID), MaxNum: 1, HasLink: true}
	switch typ {
	case "hunter":
		if row == nil {
			return Step{}, false
		}
		counted.Subtype, counted.Key, counted.MaxNum, counted.TrackInfo = StepHunter, cell("NpcId"), positive(cell("Count")), cell("TrackInfo")
		return counted, true
	case "collect":
		if row == nil {
			return Step{}, false
		}
		counted.Subtype, counted.Key, counted.MaxNum, counted.TrackInfo = StepCollect, cell("ItemId"), positive(cell("Count")), cell("TrackInfo")
		return counted, true
	case "useitem":
		if row == nil {
			return Step{}, false
		}
		counted.Subtype, counted.Key, counted.MaxNum, counted.TrackInfo = StepUseItem, cell("ItemID"), positive(cell("ProCount")), cell("TrackInfoID")
		return counted, true
	case "interact":
		if row == nil {
			return Step{}, false
		}
		npc.Subtype, npc.Key, npc.Scene, npc.TrackInfo = StepInteract, cell("NpcId"), cell("DialogID"), cell("TrackInfo")
		npc.OkMenu = cell("OkMenu")
		if npc.OkMenu == "" {
			npc.OkMenu = "okmenu_" + typeID
		}
		return npc, true
	case "question":
		if row == nil {
			return Step{}, false
		}
		npc.Subtype, npc.Key, npc.TrackInfo = StepQuestion, cell("QuestionNpc"), cell("TrackInfoID")
		return npc, true
	case "convoy":
		if row == nil {
			return Step{}, false
		}
		npc.Subtype, npc.Key, npc.TrackInfo = StepConvoy, cell("NpcId"), cell("TrackInfo")
		return npc, true
	case "story":
		if story == nil {
			return Step{}, false
		}
		empty.Subtype, empty.Key, empty.TrackInfo = StepStory, strings.TrimSpace(story["NpcID"]), strings.TrimSpace(story["TrackInfoID"])
		return empty, true
	case "special":
		if row == nil {
			return Step{}, false
		}
		empty.Subtype, empty.Index, empty.Key, empty.TrackInfo = StepSpecial, iniInt(cell("SpecialType")), typeID, cell("TrackInfoID")
		return empty, true
	default:
		return Step{}, false
	}
}

// prizeStrings Prize 域行 → 任务号 → 奖励串（accept_rec_prize_base）。
func prizeStrings(rows []map[string]string) map[int]string {
	out := make(map[int]string, len(rows))
	for _, row := range rows {
		id, err := strconv.Atoi(strings.TrimSpace(row["ID"]))
		if err != nil || id <= 0 {
			continue
		}
		out[id] = prizeBase(row)
	}
	return out
}

// prizeBase Prize 行 → 23 字段奖励串（官服已接行逐字段定版）：
//
//	[0]CapitalType1 [1]CapitalType2 [2]ReputeName1 [3]ReputeCount1
//	[4..9] 0  [10..17] Item1,Num1 … Item4,Num4  [18] 0
//	[19]LiveGroove [20]LiveGrooveCount [21..22] 0
//
// 空列写 0（官服无声望的行第 2、3 位即 "0,0"）。
func prizeBase(row map[string]string) string {
	fields := make([]string, 23)
	for i := range fields {
		fields[i] = "0"
	}
	set := func(index int, value string) {
		if value = strings.TrimSpace(value); value != "" {
			fields[index] = value
		}
	}
	set(0, row["CapitalType1"])
	set(1, row["CapitalType2"])
	set(2, row["ReputeName1"])
	set(3, row["ReputeCount1"])
	for k := 0; k < 4; k++ {
		set(10+2*k, row[fmt.Sprintf("Item%d", k+1)])
		set(11+2*k, row[fmt.Sprintf("Num%d", k+1)])
	}
	set(19, row["LiveGroove"])
	set(20, row["LiveGrooveCount"])
	return strings.Join(fields, ",")
}
