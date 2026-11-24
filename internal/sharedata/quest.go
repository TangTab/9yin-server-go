package sharedata

// quest.go 任务域（share/task/task/<域>/*.txt）与任务文案（text/chineses）
// 的直读，口径与构建器 tools/datagen/npc/{tasks.go,text.go} 逐条对齐——
// 产物是 ini/Quest/<域>/<域>.xml 与 ini/Text/TaskText.xml，对拍见
// internal/sharedata 的 parity 测试与 internal/taskdata 的测试。
//
// 任务域表：15 个域目录（task / storytasklist / subfunction / huntlist…），
// 每域几十到近三百档，2 行头（类型 + 列键）的 TAB 表，GBK / UTF-8 自适应。
// 同域各档列头不统一（主表 83 种列头变体）——按列键并集合并、行按列键
// 对齐；列头须齐含域的列键签名才归属本域（挡错置档：storytasklist/ 里的
// 主表档、interactivelist/ 里的奖励档）。目录内的 task_index_file.ini 清单
// 已过期，不作文件集依据。
//
// 行方言（与产物一致）：ID 置首；空 cell 省略；同名列取首见；无名 / 无 ID
// 行不入。主表 Title 列是设计期备注（与客户端文案七成不一致），改存
// TitleId 在文案档里解析出的显示文本，解析不到留空——任务目录据此把
// 客户端没有文案的任务挡在外。
//
// 文案不在 share 下：text/chineses/stringtask_normal + stringtask_special
// 的 .idres（key=value，UTF-8 BOM），首见胜出。

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// QuestDomain 一个任务域。
type QuestDomain struct {
	Client string   // share/task/task 下的客户端目录名
	Server string   // 域名（产物目录与文件名 Quest/<Server>/<Server>.xml）
	Keys   []string // 列键签名：列头齐含才归属本域
}

// QuestDomains 任务域清单（产物序）。Keys 取该域全部列头变体的公共键。
var QuestDomains = []QuestDomain{
	{Client: "task", Server: "Task", Keys: []string{"TitleId", "ContextId", "AcceptNpc", "SubmitNpc"}},
	{Client: "storytasklist", Server: "StoryTaskList", Keys: []string{"NpcID", "contextId", "titleId", "TaskID"}},
	{Client: "subfunction", Server: "SubFunction", Keys: []string{"Type", "TypeID"}},
	{Client: "huntlist", Server: "HuntList", Keys: []string{"NpcId", "Count", "TaskID"}},
	{Client: "collectlist", Server: "CollectList", Keys: []string{"ItemId", "NpcId"}},
	{Client: "useitemlist", Server: "UseItemList", Keys: []string{"ItemID", "ProCount"}},
	{Client: "interactivelist", Server: "InteractiveList", Keys: []string{"NpcId", "DialogID"}},
	{Client: "question", Server: "Question", Keys: []string{"QuestionNpc", "QuestionID"}},
	{Client: "refreshmonster", Server: "RefreshMonster", Keys: []string{"MonsterID"}},
	{Client: "convoy", Server: "Convoy", Keys: []string{"NpcId", "PathID"}},
	{Client: "convoypoint", Server: "Convoypoint", Keys: []string{"Xpoint", "Zpoint"}},
	{Client: "special", Server: "Special", Keys: []string{"SpecialType"}},
	{Client: "choose", Server: "Choose", Keys: []string{"ChooseNpc"}},
	{Client: "taskitem", Server: "TaskItem", Keys: []string{"ItemId", "IsGive"}},
	{Client: "prize", Server: "Prize", Keys: []string{"Faculty", "CapitalType1"}},
}

// QuestRootTables 任务域根级散表（share/task 下）→ 产物文件名
// （ini/Quest/<Output>）。
var QuestRootTables = []struct{ Source, Output string }{
	{"task_config.txt", "task_config.xml"}, // 字段字典
	{"chareffect.txt", "CharEffect.xml"},   // 人物特效表
	{"task/xxx_type_id.txt", "xxx_type_ID.xml"},
}

// QuestAudit 一个任务域的装载审计（对齐构建器报告；trace / 测试面）。
type QuestAudit struct {
	Files      int      // 入表档数
	RepeatIDs  int      // 同 ID 多行数（剧情步骤 / 分档变体，全部保留不去重）
	TitleText  int      // 主表 Title 解析到文案的行数（仅 task 域）
	Skipped    []string // 未入表档与原因
	Truncated  int      // 超列宽截断行（多余字段丢弃）
	Nameless   int      // 无名列（值丢弃）
	Duplicates int      // 重复列（取首见）
}

// QuestDomainOf 按域名（大小写不敏感）取任务域；没有返回 false。
func QuestDomainOf(server string) (QuestDomain, bool) {
	for _, domain := range QuestDomains {
		if strings.EqualFold(domain.Server, server) {
			return domain, true
		}
	}
	return QuestDomain{}, false
}

// ReadQuestDomain 读一个任务域的全部行。texts 是任务文案键值表（仅主表
// task 域用来解析 Title；其它域不使用）。域目录缺席报 fs.ErrNotExist。
func ReadQuestDomain(root string, domain QuestDomain, texts map[string]string) ([]map[string]string, QuestAudit, error) {
	audit := QuestAudit{}
	taskDir := filepath.Join(root, filepath.FromSlash(QuestDomainDir))
	entries, err := res.ReadDir(taskDir)
	if err != nil {
		return nil, audit, err
	}
	var files []string
	found := false
	for _, e := range entries {
		if !e.IsDir() || !strings.EqualFold(e.Name(), domain.Client) {
			continue
		}
		found = true
		list, err := res.ReadDir(filepath.Join(taskDir, e.Name()))
		if err != nil {
			return nil, audit, err
		}
		for _, f := range list {
			if !f.IsDir() && strings.EqualFold(filepath.Ext(f.Name()), ".txt") {
				files = append(files, filepath.Join(taskDir, e.Name(), f.Name()))
			}
		}
	}
	if !found {
		return nil, audit, &fs.PathError{Op: "open", Path: filepath.ToSlash(QuestDomainDir) + "/" + domain.Client, Err: fs.ErrNotExist}
	}
	sort.Strings(files)
	merged, audit := mergeQuestDomain(domain, files)
	if merged == nil {
		return []map[string]string{}, audit, nil
	}
	if domain.Client == "task" {
		audit.TitleText = resolveTaskTitle(merged, texts)
	}
	return passThroughRows(merged), audit, nil
}

// ReadQuestRootTable 任务域根级散表。
func ReadQuestRootTable(root, source string) ([]map[string]string, error) {
	path := filepath.Join(root, filepath.FromSlash("share/task"), filepath.FromSlash(source))
	table, err := ReadDomainTable(path, 2)
	if err != nil {
		return nil, err
	}
	return passThroughRows(table), nil
}

// mergeQuestDomain 同域各档合并为一张表：列取各档列键并集（首见序），
// 行按列键对齐（缺列留空）。同 ID 多行全部保留不去重——是合法形态
// （StoryTaskList 一条任务多段剧情、SubFunction 多段子功能）。域内无有效
// 表时表为 nil。
func mergeQuestDomain(domain QuestDomain, paths []string) (*Table, QuestAudit) {
	audit := QuestAudit{}
	var sources []*Table
	for _, p := range paths {
		if info, err := res.Stat(p); err == nil && info.Size() == 0 {
			audit.Skipped = append(audit.Skipped, filepath.Base(p)+"（空档）")
			continue
		}
		t, err := ReadDomainTable(p, 2)
		if err != nil {
			audit.Skipped = append(audit.Skipped, fmt.Sprintf("%s（%v）", filepath.Base(p), err))
			continue
		}
		first := t.First()
		if _, hasID := first["ID"]; !hasID || !hasColumnKeys(first, domain.Keys) {
			audit.Skipped = append(audit.Skipped, fmt.Sprintf("%s（错置档：列头无 ID/%s 域签名）", filepath.Base(p), domain.Client))
			continue
		}
		sources = append(sources, t)
		audit.Files++
		audit.Nameless += countBlanks(t.Columns)
		audit.Duplicates += len(t.Columns) - len(first)
		audit.Truncated += t.Truncated
	}
	if len(sources) == 0 {
		return nil, audit
	}

	merged := &Table{}
	index := make(map[string]int, 256)
	for _, t := range sources {
		for i, name := range t.Columns {
			if name == "" {
				continue
			}
			if _, seen := index[name]; seen {
				continue
			}
			index[name] = len(merged.Columns)
			merged.Columns = append(merged.Columns, name)
			declared := ""
			if i < len(t.Types) {
				declared = t.Types[i]
			}
			merged.Types = append(merged.Types, declared)
		}
	}
	// 行按列键对齐；同名列取首见（与输出取舍一致）。
	idIdx := index["ID"]
	seen := map[string]int{}
	for _, t := range sources {
		first := t.First()
		for _, row := range t.Rows {
			cells := make([]string, len(merged.Columns))
			for i, name := range t.Columns {
				if name == "" || first[name] != i || i >= len(row) {
					continue
				}
				cells[index[name]] = row[i]
			}
			merged.Rows = append(merged.Rows, cells)
			if id := cells[idIdx]; id != "" {
				seen[id]++
				if seen[id] > 1 {
					audit.RepeatIDs++
				}
			}
		}
	}
	return merged, audit
}

func hasColumnKeys(first map[string]int, keys []string) bool {
	for _, key := range keys {
		if _, ok := first[key]; !ok {
			return false
		}
	}
	return true
}

// countBlanks 列键中段的无名列数（尾部空列键读取时已裁剪）。
func countBlanks(columns []string) int {
	n := 0
	for _, name := range columns {
		if name == "" {
			n++
		}
	}
	return n
}

// resolveTaskTitle 主表 Title 改存显示文本：源表 Title 列是设计期备注，
// 显示名以 TitleId 键在文案表的解析为准（TitleId 留空回落 title_<ID>）；
// 解析不到留空。返回解析到文案的行数。
func resolveTaskTitle(t *Table, texts map[string]string) int {
	first := t.First()
	titleIdx, hasTitle := first["Title"]
	if !hasTitle {
		return 0
	}
	titleIDIdx, hasTitleID := first["TitleId"]
	resolved := 0
	for _, row := range t.Rows {
		key := ""
		if hasTitleID && titleIDIdx < len(row) {
			key = strings.TrimSpace(row[titleIDIdx])
		}
		if key == "" {
			key = "title_" + strings.TrimSpace(row[first["ID"]])
		}
		text := ""
		if texts != nil {
			text = strings.TrimSpace(texts[key])
		}
		if text != "" {
			resolved++
		}
		row[titleIdx] = text
	}
	return resolved
}

// passThroughRows 透传表 → 属性行：ID 置首，空 cell 省略，同名列取首见，
// 无名 / 无 ID 行不入。
func passThroughRows(t *Table) []map[string]string {
	first := t.First()
	idIdx, hasID := first["ID"]
	rows := make([]map[string]string, 0, len(t.Rows))
	for _, cells := range t.Rows {
		if !hasID || idIdx >= len(cells) || cells[idIdx] == "" {
			continue
		}
		row := make(map[string]string, len(t.Columns))
		row["ID"] = cells[idIdx]
		for i, name := range t.Columns {
			if first[name] != i || name == "ID" || name == "" || i >= len(cells) {
				continue
			}
			if cells[i] != "" {
				row[name] = cells[i]
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// taskTextEnv 文案目录覆盖：须含 stringtask_normal / stringtask_special。
const taskTextEnv = "NINEYIN_TASK_TEXT_DIR"

// TaskTextDirOf 任务文案目录：env NINEYIN_TASK_TEXT_DIR（数据根带了
// share/ 而文案另放时显式指定）> <数据根>/text/chineses。目录不在返回空
// ——任务主表的 Title 解析不了，任务域即回落 XML 产物。
func TaskTextDirOf(root string) string {
	if v := strings.TrimSpace(os.Getenv(taskTextEnv)); v != "" {
		if st, err := res.Stat(v); err == nil && st.IsDir() {
			return v
		}
		return ""
	}
	dir := filepath.Join(root, filepath.FromSlash(TaskTextDir))
	if st, err := res.Stat(dir); err == nil && st.IsDir() {
		return dir
	}
	return ""
}

// ReadTaskText 任务文案键值表：文案目录下的 stringtask_normal 与
// stringtask_special 两个串目录并表（先 normal 后 special，各档按文件名
// 排序，首见胜出）。返回键值表、读入档数、重复键数。缺一个串目录只是
// 少一部分文案（构建器同样只告警）；文案目录缺席、或两个串目录都不在，
// 报 fs.ErrNotExist。
func ReadTaskText(root string) (map[string]string, int, int, error) {
	textRoot := TaskTextDirOf(root)
	if textRoot == "" {
		return nil, 0, 0, &fs.PathError{Op: "open", Path: TaskTextDir, Err: fs.ErrNotExist}
	}
	entries := map[string]string{}
	files, conflicts := 0, 0
	for _, dir := range []string{"stringtask_normal", "stringtask_special"} {
		part, n, dup, err := ReadIDresDir(filepath.Join(textRoot, dir))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, files, conflicts, err
		}
		files += n
		conflicts += dup
		for key, value := range part {
			if _, exists := entries[key]; exists {
				conflicts++
				continue
			}
			entries[key] = value
		}
	}
	if files == 0 {
		return nil, 0, 0, &fs.PathError{Op: "open", Path: TaskTextDir, Err: fs.ErrNotExist}
	}
	return entries, files, conflicts, nil
}

// TextRows 键值表 → 文案属性行（ID=串键，Text=文本；键字典序，与产物
// 同序）。空文本不写 Text（产物同样省略空值——有 3 千多个串键的文本是
// 空的）。
func TextRows(entries map[string]string) []map[string]string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		row := map[string]string{"ID": key}
		if text := entries[key]; text != "" {
			row["Text"] = text
		}
		rows = append(rows, row)
	}
	return rows
}

// QuestSource 任务域的来源：主表 Title 要靠文案档解析，share/task/task 与
// 文案目录（TaskTextDirOf）两者都在才算 share 源齐备。
func QuestSource(root string) Source {
	if EffectiveSource(root, QuestDomainDir) != SourceShare {
		return SourceXML
	}
	if TaskTextDirOf(root) == "" {
		return SourceXML
	}
	return SourceShare
}

// BuiltRows 把「构建产物 XML 的路径」改道到直读：调用方照旧拿着产物路径
// 来要行，数据根走 share 源时这里直接读客户端表给出同样的行。返回
// ok=false 表示这个路径不归直读管（不是产物路径，或该域当前来源是
// xml），调用方照旧读 XML。
//
//	ini/Quest/<域>/<域>.xml、ini/Quest/<根级表>.xml   任务域（QuestSource）
//	ini/Text/TaskText.xml                              任务文案（text/chineses 在即直读）
//	ini/Npc/NpcConfig/WorldNpc/NpcTask_ConvoyNpc.xml   护送 NPC 模板表（只给 ID 行）
func BuiltRows(root, rel string) (rows []map[string]string, ok bool, err error) {
	name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(rel), "\\", "/"))
	switch {
	case strings.HasPrefix(name, "ini/quest/"):
		if QuestSource(root) != SourceShare {
			return nil, false, nil
		}
		return questBuiltRows(root, strings.TrimPrefix(name, "ini/quest/"))
	case name == "ini/text/tasktext.xml":
		if EffectiveSource(root, TaskTextDir) != SourceShare || TaskTextDirOf(root) == "" {
			return nil, false, nil
		}
		entries, _, _, err := ReadTaskText(root)
		if err != nil {
			return nil, true, err
		}
		return TextRows(entries), true, nil
	case name == "ini/npc/npcconfig/worldnpc/npctask_convoynpc.xml":
		if EffectiveSource(root, NPCConfigDir) != SourceShare {
			return nil, false, nil
		}
		rows, err := singleTableIDRows(root, "worldnpc/npctask_convoynpc.txt")
		return rows, true, err
	}
	return nil, false, nil
}

// questBuiltRows ini/Quest 下的产物路径 → 任务域行。不是任务域产物的路径
// （如黑金原档 Convoy/EventSummon.xml）返回 ok=false，调用方照旧读 XML。
func questBuiltRows(root, rel string) ([]map[string]string, bool, error) {
	// <域>/<域>.xml；根级散表是 <名>.xml 单段。
	if head, tail, isDomain := strings.Cut(rel, "/"); isDomain {
		domain, known := QuestDomainOf(head)
		if !known || !strings.EqualFold(tail, domain.Server+".xml") {
			return nil, false, nil
		}
		var texts map[string]string
		if domain.Client == "task" {
			entries, _, _, err := ReadTaskText(root)
			if err != nil {
				return nil, true, err
			}
			texts = entries
		}
		rows, _, err := ReadQuestDomain(root, domain, texts)
		return rows, true, err
	}
	for _, table := range QuestRootTables {
		if strings.EqualFold(rel, table.Output) {
			rows, err := ReadQuestRootTable(root, table.Source)
			return rows, true, err
		}
	}
	return nil, false, nil
}

// singleTableIDRows 一张 NPC 模板表的 ID 行（消费方只要 ID 集）。
func singleTableIDRows(root, rel string) ([]map[string]string, error) {
	path := filepath.Join(root, filepath.FromSlash(NPCConfigDir), filepath.FromSlash(rel))
	table, err := ReadTemplateTable(path)
	if err != nil {
		return nil, err
	}
	idCol := table.Column("ID")
	rows := make([]map[string]string, 0, len(table.Rows))
	for _, cells := range table.Rows {
		if idCol >= 0 && idCol < len(cells) && cells[idCol] != "" {
			rows = append(rows, map[string]string{"ID": cells[idCol]})
		}
	}
	return rows, nil
}
