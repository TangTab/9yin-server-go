package sharedata

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func questDomain(t *testing.T, server string) QuestDomain {
	t.Helper()
	domain, ok := QuestDomainOf(server)
	if !ok {
		t.Fatalf("未知任务域 %s", server)
	}
	return domain
}

// 任务域合并：各档按文件名排序；列取并集、行按列键对齐（缺列留空）；空档
// 与错置档（列头没有域签名）跳过；行方言 = ID 置首、空 cell 省略；主表
// Title 按 TitleId 解析（留空回落 title_<ID>，解析不到省略）。
func TestReadQuestDomain(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "share", "task", "task", "task")
	write(t, filepath.Join(dir, "a.txt"), "str\tstr\tstr\tstr\tstr\tstr\nID\tTitle\tTitleId\tContextId\tAcceptNpc\tSubmitNpc\n"+
		"1\t备注一\ttitle_1\tcontext_1\tNpcA\tNpcB\n2\t备注二\t\tcontext_2\tNpcC\t\n")
	write(t, filepath.Join(dir, "b.txt"), gbk(t, "str\tstr\tstr\tstr\tstr\tstr\nID\tX\tAcceptNpc\tTitleId\tContextId\tSubmitNpc\n3\t中文值\tNpcD\t\t\t\n"))
	write(t, filepath.Join(dir, "misplaced.txt"), "str\tstr\nID\tFoo\n9\tbar\n")
	write(t, filepath.Join(dir, "empty.txt"), "")
	write(t, filepath.Join(dir, "ignore.ini"), "[x]\n")

	rows, audit, err := ReadQuestDomain(root, questDomain(t, "Task"), map[string]string{"title_1": "初入江湖"})
	if err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{
		{"ID": "1", "Title": "初入江湖", "TitleId": "title_1", "ContextId": "context_1", "AcceptNpc": "NpcA", "SubmitNpc": "NpcB"},
		{"ID": "2", "ContextId": "context_2", "AcceptNpc": "NpcC"}, // title_2 无文案：Title 省略
		{"ID": "3", "X": "中文值", "AcceptNpc": "NpcD"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("行：\n 得 %v\n 应 %v", rows, want)
	}
	if audit.Files != 2 || len(audit.Skipped) != 2 || audit.TitleText != 1 {
		t.Fatalf("audit=%+v（入表 2、跳过 空档+错置档、Title 解析 1）", audit)
	}

	// 域目录缺席：按缺席处理（fsgame 读不到即视为没有这一域）。
	if _, _, err := ReadQuestDomain(root, QuestDomain{Client: "nope", Server: "Nope"}, nil); !os.IsNotExist(err) {
		t.Fatalf("域目录缺席应报 not-exist，得 %v", err)
	}
}

// equalRows 行序列相等（nil 与空序列同视）。
func equalRows(a, b []map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// rowSetDiff 按键列比较两组行（不问顺序），给出差异摘要：只在一侧的键与
// 值不同的键各列前几个。
func rowSetDiff(a, b []map[string]string, key string) string {
	am, bm := map[string]map[string]string{}, map[string]map[string]string{}
	for _, row := range a {
		am[row[key]] = row
	}
	for _, row := range b {
		bm[row[key]] = row
	}
	var onlyA, onlyB, differ []string
	for k, x := range am {
		y, ok := bm[k]
		if !ok {
			onlyA = append(onlyA, k)
			continue
		}
		if !reflect.DeepEqual(x, y) {
			differ = append(differ, fmt.Sprintf("%s：直读 %q / xml %q", k, x["Text"], y["Text"]))
		}
	}
	for k := range bm {
		if _, ok := am[k]; !ok {
			onlyB = append(onlyB, k)
		}
	}
	sort.Strings(onlyA)
	sort.Strings(onlyB)
	sort.Strings(differ)
	return fmt.Sprintf("只在直读 %d 个 %v\n只在 xml %d 个 %v\n值不同 %d 个 %v",
		len(onlyA), clip(onlyA), len(onlyB), clip(onlyB), len(differ), clip(differ))
}

func clip(list []string) []string {
	if len(list) > 6 {
		return append(list[:6], "…")
	}
	return list
}

// 文案并表（先 normal 后 special、首见胜出）与 BuiltRows 改道：任务域 /
// 文案 / 护送 NPC 模板表走直读，非产物路径与黑金原档不动；来源跟随开关
// 与文案目录在否；NINEYIN_TASK_TEXT_DIR 可把文案目录指到数据根之外。
func TestBuiltRows(t *testing.T) {
	// 不继承外部环境的文案目录覆盖（否则读到的不是本例造的文案）。
	t.Setenv(taskTextEnv, "")
	root := t.TempDir()
	taskDir := filepath.Join(root, "share", "task", "task", "subfunction")
	write(t, filepath.Join(taskDir, "a.txt"), "str\tstr\tstr\nID\tType\tTypeID\n7\tstory\t482\n")
	write(t, filepath.Join(root, "share", "task", "task_config.txt"), "str\tstr\nID\tDesc\nk1\t字段\n")
	write(t, filepath.Join(root, "share", "npc", "npcconfig", "worldnpc", "npctask_convoynpc.txt"), "a\tb\ns\ts\nID\tName\nC1\t车夫\n")
	textDir := filepath.Join(root, "text", "chineses")
	write(t, filepath.Join(textDir, "stringtask_normal", "a.idres"), "title_1=初入江湖\n")
	write(t, filepath.Join(textDir, "stringtask_special", "b.idres"), "title_1=后来者\ntitle_2=第二\n")

	t.Setenv(sourceEnv, "")
	// 文案目录不在：任务域回落 XML（即使 share/task/task 在）。
	if _, _, _, err := ReadTaskText(filepath.Join(root, "absent")); err == nil {
		t.Fatal("文案目录缺席应报错")
	}
	entries, files, conflicts, err := ReadTaskText(root)
	if err != nil || files != 2 || conflicts != 1 || len(entries) != 2 || entries["title_1"] != "初入江湖" {
		t.Fatalf("文案并表：entries=%v files=%d conflicts=%d err=%v", entries, files, conflicts, err)
	}
	if QuestSource(root) != SourceShare {
		t.Fatalf("share/task/task 与文案都在应判 share 源，得 %v", QuestSource(root))
	}
	if TaskTextDirOf(root) != textDir {
		t.Fatalf("文案目录 %q，应为 %q", TaskTextDirOf(root), textDir)
	}

	sub, ok, err := BuiltRows(root, "ini/Quest/SubFunction/SubFunction.xml")
	if err != nil || !ok || !reflect.DeepEqual(sub, []map[string]string{{"ID": "7", "Type": "story", "TypeID": "482"}}) {
		t.Fatalf("SubFunction 直读：ok=%v err=%v rows=%v", ok, err, sub)
	}
	// 域目录缺席：改道成立但读不到（调用方按缺席处理）。
	if _, ok, err := BuiltRows(root, `ini\Quest\Task\Task.xml`); !ok || !os.IsNotExist(err) {
		t.Fatalf("Task 域目录缺席：ok=%v err=%v", ok, err)
	}
	// 域目录在但没有任何有效档：空表，不算错。
	if err := os.MkdirAll(filepath.Join(root, "share", "task", "task", "choose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if rows, ok, err := BuiltRows(root, "ini/Quest/Choose/Choose.xml"); err != nil || !ok || len(rows) != 0 {
		t.Fatalf("空域：ok=%v err=%v rows=%v", ok, err, rows)
	}
	cfg, ok, err := BuiltRows(root, "ini/Quest/task_config.xml")
	if err != nil || !ok || !reflect.DeepEqual(cfg, []map[string]string{{"ID": "k1", "Desc": "字段"}}) {
		t.Fatalf("根级表直读：ok=%v err=%v rows=%v", ok, err, cfg)
	}
	text, ok, err := BuiltRows(root, "ini/Text/TaskText.xml")
	if err != nil || !ok || !reflect.DeepEqual(text, []map[string]string{{"ID": "title_1", "Text": "初入江湖"}, {"ID": "title_2", "Text": "第二"}}) {
		t.Fatalf("文案直读：ok=%v err=%v rows=%v", ok, err, text)
	}
	convoy, ok, err := BuiltRows(root, "ini/Npc/NpcConfig/WorldNpc/NpcTask_ConvoyNpc.xml")
	if err != nil || !ok || !reflect.DeepEqual(convoy, []map[string]string{{"ID": "C1"}}) {
		t.Fatalf("护送 NPC 表直读：ok=%v err=%v rows=%v", ok, err, convoy)
	}
	// 不归直读管：黑金原档路径、别的产物、大小写与分隔符差异不影响识别。
	for _, rel := range []string{"ini/Quest/Convoy/EventSummon.xml", "ini/Quest/Unknown/x.xml", "ini/Npc/NpcTemplate.xml", "struct/npc/x.xml"} {
		if _, ok, err := BuiltRows(root, rel); ok || err != nil {
			t.Fatalf("%s 不应改道直读：ok=%v err=%v", rel, ok, err)
		}
	}

	// 显式 xml：全部照旧读产物。
	t.Setenv(sourceEnv, "xml")
	for _, rel := range []string{"ini/Quest/SubFunction/SubFunction.xml", "ini/Text/TaskText.xml", "ini/Npc/NpcConfig/WorldNpc/NpcTask_ConvoyNpc.xml"} {
		if _, ok, _ := BuiltRows(root, rel); ok {
			t.Fatalf("显式 xml 下 %s 不应改道", rel)
		}
	}

	// 文案另放：NINEYIN_TASK_TEXT_DIR 指到数据根之外。
	t.Setenv(sourceEnv, "")
	elsewhere := t.TempDir()
	write(t, filepath.Join(elsewhere, "stringtask_normal", "a.idres"), "title_1=别处\n")
	t.Setenv(taskTextEnv, elsewhere)
	if entries, _, _, _ := ReadTaskText(root); entries["title_1"] != "别处" {
		t.Fatalf("文案目录覆盖未生效：%v", entries)
	}
	t.Setenv(taskTextEnv, filepath.Join(root, "absent"))
	if QuestSource(root) != SourceXML {
		t.Fatal("文案目录覆盖指向不存在的目录时应回落 XML")
	}
}

// ---- 真档对拍 ------------------------------------------------------------

// parityRoot 对拍用的数据根：env NINEYIN_DATA_ROOT，否则仓内 testdata。
func parityRoot(t *testing.T) string {
	t.Helper()
	if v := strings.TrimSpace(os.Getenv("NINEYIN_DATA_ROOT")); v != "" {
		return v
	}
	return filepath.Join("..", "..", "testdata")
}

func requireBoth(t *testing.T, root, sharePath, xmlPath string) {
	t.Helper()
	_, shareErr := res.Stat(filepath.Join(root, filepath.FromSlash(sharePath)))
	_, xmlErr := res.Stat(filepath.Join(root, filepath.FromSlash(xmlPath)))
	if shareErr != nil || xmlErr != nil {
		t.Skipf("对拍需要两种来源同在：share %s=%v xml %s=%v", sharePath, shareErr, xmlPath, xmlErr)
	}
}

// propRows 解析 <Object><Property k="v"/>（gb 系声明按 GB18030 解码）。
func propRows(t *testing.T, path string) []map[string]string {
	t.Helper()
	data, err := res.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = func(charset string, in io.Reader) (io.Reader, error) {
		switch strings.ToLower(charset) {
		case "gb2312", "gbk", "gb18030":
			return simplifiedchinese.GB18030.NewDecoder().Reader(in), nil
		}
		return nil, fmt.Errorf("不支持的字符集 %q", charset)
	}
	var rows []map[string]string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("%s: %v", path, err)
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
	return rows
}

// TestQuestDomainParity 全部任务域 + 根级散表：直读行与 XML 产物逐行
// 相同（含行序——SubFunction 的步骤序靠它）。主表的 Title 两边都抹掉再比
// （直读侧是否带文案见 TestQuestTextParity）。
func TestQuestDomainParity(t *testing.T) {
	root := parityRoot(t)
	requireBoth(t, root, QuestDomainDir, "ini/Quest")

	total := 0
	for _, domain := range QuestDomains {
		rows, audit, err := ReadQuestDomain(root, domain, nil)
		if err != nil {
			t.Fatalf("%s: %v", domain.Server, err)
		}
		xmlPath := filepath.Join(root, "ini", "Quest", domain.Server, domain.Server+".xml")
		product := propRows(t, xmlPath)
		if len(rows) != len(product) {
			t.Fatalf("%s 行数不同：直读 %d / xml %d（入表 %d 档，跳过 %d：%v）",
				domain.Server, len(rows), len(product), audit.Files, len(audit.Skipped), audit.Skipped)
		}
		for i := range rows {
			if domain.Client == "task" {
				delete(rows[i], "Title")
				delete(product[i], "Title")
			}
			if !reflect.DeepEqual(rows[i], product[i]) {
				t.Fatalf("%s 第 %d 行不同：\n 直读 %v\n xml  %v", domain.Server, i, rows[i], product[i])
			}
		}
		total += len(rows)
	}
	for _, table := range QuestRootTables {
		if _, err := res.Stat(filepath.Join(root, filepath.FromSlash("share/task"), filepath.FromSlash(table.Source))); err != nil {
			continue
		}
		rows, err := ReadQuestRootTable(root, table.Source)
		if err != nil {
			t.Fatalf("%s: %v", table.Source, err)
		}
		product := propRows(t, filepath.Join(root, "ini", "Quest", table.Output))
		if !equalRows(rows, product) {
			t.Fatalf("根级表 %s 不同：直读 %d 行 / xml %d 行", table.Source, len(rows), len(product))
		}
		total += len(rows)
	}
	t.Logf("任务域对拍一致：%d 域 + 根级散表，共 %d 行", len(QuestDomains), total)
}

// TestQuestTextParity 任务文案与带文案的主表 Title：文案目录可用（数据根
// 自带 text/chineses，或 NINEYIN_TASK_TEXT_DIR 指定）时，直读的文案行、
// 主表 Title 与 XML 产物逐行相同；不可用时跳过。
func TestQuestTextParity(t *testing.T) {
	root := parityRoot(t)
	requireBoth(t, root, QuestDomainDir, "ini/Quest")
	t.Setenv(sourceEnv, "")
	if TaskTextDirOf(root) == "" {
		t.Skip("文案目录不可用（数据根无 text/chineses，也未设 NINEYIN_TASK_TEXT_DIR）")
	}

	entries, files, _, err := ReadTaskText(root)
	if err != nil {
		t.Fatal(err)
	}
	product := propRows(t, filepath.Join(root, "ini", "Text", "TaskText.xml"))
	if rows := TextRows(entries); !equalRows(rows, product) {
		t.Fatalf("任务文案不同：直读 %d 键（%d 档）/ xml %d 行\n%s", len(rows), files, len(product), rowSetDiff(rows, product, "ID"))
	}

	task, audit, err := ReadQuestDomain(root, questDomain(t, "Task"), entries)
	if err != nil {
		t.Fatal(err)
	}
	product = propRows(t, filepath.Join(root, "ini", "Quest", "Task", "Task.xml"))
	if len(task) != len(product) {
		t.Fatalf("主表行数不同：直读 %d / xml %d", len(task), len(product))
	}
	differ := 0
	for i := range task {
		if !reflect.DeepEqual(task[i], product[i]) {
			differ++
			if differ <= 5 {
				t.Logf("第 %d 行不同：直读 Title=%q / xml Title=%q（ID %s）", i, task[i]["Title"], product[i]["Title"], task[i]["ID"])
			}
		}
	}
	if differ > 0 {
		t.Fatalf("主表 %d 行不同（共 %d 行，直读解析到文案 %d 行）", differ, len(task), audit.TitleText)
	}
	t.Logf("文案对拍一致：%d 键（%d 档）；主表 %d 行，Title 解析 %d 行", len(entries), files, len(task), audit.TitleText)
}
