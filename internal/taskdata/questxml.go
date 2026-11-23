package taskdata

// questxml.go 服务端任务域 XML 装载（<root>/ini/Quest，tools/npcgen 由
// 客户端 share/task 任务域树生成——任务 ID 与客户端文案键/官服 0x29 记录
// 帧同一体系）：
//
//	Task/Task.xml            任务主表（15 域）：Title 为 npcgen 按 TitleId
//	                         解析出的客户端显示文本（无文案留空）；
//	                         AcceptNpc/SubmitNpc + 两侧 SceneID 给
//	                         NPC↔任务绑定
//	SubFunction/SubFunction.xml   接线表：ID=任务 ID、Type=步骤域、
//	                              TypeID=步骤行 ID（story 域 → StoryTaskList
//	                              组 ID）
//	StoryTaskList/StoryTaskList.xml  剧情段行：同 ID 多行=同剧情多段
//	                              （行序即段序），NpcID/contextId/titleId/
//	                              LeaveTextID/SceneID 与 txt 链
//	                              storytasklist 同词汇
//
// 与快照链（share/task txt + idres 文本派生 tasks.json）互为数据源：
// 目录装载优先走本链（XML 为正典），快照作旧部署回退。无 Title 的主表行
// 不计入目录——客户端没有该任务的文案，任务行发了也无处渲染——与快照链
// 「绑定且已加载文本的任务」同语义。

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// LoadQuestXMLDir 从服务端任务域 XML 目录（<root>/ini/Quest）构建目录。
// Task.xml 缺席或无有效任务行报错（调用方视为候选源失败）；剧情链
// （SubFunction × StoryTaskList）缺档降级为无剧情绑定——剧情行是增强
// 下发，缺数据不阻塞任务行。
func LoadQuestXMLDir(dir string) (*Catalog, error) {
	rows, err := loadQuestPropRows(filepath.Join(dir, "Task", "Task.xml"))
	if err != nil {
		return nil, err
	}
	tasks := make(map[int]Task)
	var bindings []Binding
	// 链式接取目标集：被某任务 AutoAcceptID 指向的任务只随前序任务完成
	// 接取（如 27509 随 1648），不算链根。
	chainDriven := make(map[int]bool)
	for _, row := range rows {
		if next := iniInt(row["AutoAcceptID"]); next > 0 {
			chainDriven[next] = true
		}
	}
	for _, row := range rows {
		id, err := strconv.Atoi(strings.TrimSpace(row["ID"]))
		if err != nil || id <= 0 {
			continue
		}
		title := strings.TrimSpace(row["Title"])
		if title == "" {
			continue
		}
		tasks[id] = Task{
			ID: id, Title: title,
			AcceptMenuKey: strings.TrimSpace(row["CanAcceptMenu"]),
			ChainRoot:     strings.TrimSpace(row["PreTaskLimit"]) == "" && !chainDriven[id],
			AutoAcceptID:  iniInt(row["AutoAcceptID"]),
			AutoSubmit:    strings.TrimSpace(row["AutoSubmit"]) == "1",
		}
		for _, side := range []struct{ npc, scene, source string }{
			{row["AcceptNpc"], row["AcceptSceneID"], "quest-accept"},
			{row["SubmitNpc"], row["SubmitSceneID"], "quest-submit"},
		} {
			if npc := strings.TrimSpace(side.npc); npc != "" {
				bindings = append(bindings, Binding{
					TaskID:      id,
					NPCConfigID: npc,
					SceneID:     strings.TrimSpace(side.scene),
					Source:      side.source,
				})
			}
		}
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("taskdata: %s 无带文案的任务主表行（Task/Task.xml）", dir)
	}
	if story, err := loadQuestStoryBindings(dir); err == nil {
		bindings = append(bindings, story...)
	}
	return NewCatalog(tasks, bindings, nil), nil
}

// loadQuestStoryBindings SubFunction（Type=story）× StoryTaskList 合成剧情
// 段绑定：同组多行逐行成段（StoryID=组 ID，ContextID/TitleID/LeaveTextID
// 同 txt 链词汇），行序即段序。
//
// 源表只在一组剧情的首段行写 NpcID，续段行留空——沿用组内上一行的 NPC；
// 没有任何 NPC 的行（空行/残行）不成段。完全重复的行（跨档复制）由
// Catalog 去重。
func loadQuestStoryBindings(dir string) ([]Binding, error) {
	storyRows, err := loadQuestPropRows(filepath.Join(dir, "StoryTaskList", "StoryTaskList.xml"))
	if err != nil {
		return nil, err
	}
	storyByID := make(map[string][]map[string]string)
	for _, row := range storyRows {
		if id := strings.TrimSpace(row["ID"]); id != "" {
			storyByID[id] = append(storyByID[id], row)
		}
	}
	subRows, err := loadQuestPropRows(filepath.Join(dir, "SubFunction", "SubFunction.xml"))
	if err != nil {
		return nil, err
	}
	var bindings []Binding
	for _, sf := range subRows {
		if strings.TrimSpace(sf["Type"]) != "story" {
			continue
		}
		taskID, err := strconv.Atoi(strings.TrimSpace(sf["ID"]))
		if err != nil || taskID <= 0 {
			continue
		}
		npc := "" // 组内上一个带 NPC 的行
		for _, row := range storyByID[strings.TrimSpace(sf["TypeID"])] {
			if v := strings.TrimSpace(row["NpcID"]); v != "" {
				npc = v
			}
			if npc == "" || strings.TrimSpace(row["contextId"]) == "" {
				continue
			}
			bindings = append(bindings, Binding{
				TaskID:      taskID,
				NPCConfigID: npc,
				SceneID:     strings.TrimSpace(row["SceneID"]),
				StoryID:     strings.TrimSpace(row["ID"]),
				ContextID:   strings.TrimSpace(row["contextId"]),
				TitleID:     strings.TrimSpace(row["titleId"]),
				LeaveTextID: strings.TrimSpace(row["LeaveTextID"]),
				Source:      "storytasklist",
			})
		}
	}
	return bindings, nil
}

// loadQuestPropRows 读 <Object><Property k="v"/></Object> 属性表文件。
func loadQuestPropRows(path string) ([]map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rows, err := parseQuestPropRows(data)
	if err != nil {
		return nil, fmt.Errorf("taskdata: %s: %w", path, err)
	}
	return rows, nil
}

// parseQuestPropRows 解析官方 CXmlFile Property 属性表方言（与
// fsgame.ParsePropTable 同实现——本包保持叶包不引 fsgame）。gb2312/gbk
// 声明按 GB18030 超集解码。
func parseQuestPropRows(data []byte) ([]map[string]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = gbCharsetReader
	var rows []map[string]string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("prop table: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Property" {
			continue
		}
		row := make(map[string]string, len(se.Attr))
		for _, a := range se.Attr {
			row[a.Name.Local] = a.Value
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// gbCharsetReader 官方 gb2312/gbk 声明统一按 GB18030 超集解码（与
// internal/fsgame 同决策）。
func gbCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(charset) {
	case "gb2312", "gbk", "gb18030":
		return simplifiedchinese.GB18030.NewDecoder().Reader(input), nil
	}
	return nil, fmt.Errorf("taskdata: 不支持的字符集 %q", charset)
}
