// npc：九阴客户端 NPC 数据 → 服务端 ini 配置转换器。tools/datagen 的 npc
// 构建器：输入输出路径由统一入口解析后经 Options 传入，本包不做路径探测。
//
// 输入一（Options.Client = 客户端资源解压根）：
//   - share/npc/npcconfig/*.txt        76 张 GBK+TAB 模板表（3 行头）
//   - share/npc/npcconfig/npc_type.ini 表 → 服务端原路径清单（[file]/flie<N>）
//   - share/creator/npc_creator/<场景>/*.xml  场景摆放（静态+随机 creator）
//   - share/task/task/<域>/*.txt       任务域表（Task/StoryTaskList/
//     SubFunction/HuntList 等 15 域，任务 ID 与客户端文案键同体系）+ 根级散表
//
// 输入二（战斗 NPC 模板，同在客户端 share/task 下）：
//   - npctemplate*.tab（GBK TSV，1 行头；npctemplate.tab 主档优先，
//     npctemplate2.tab 等补充档按 glob 合并，同 ID 首档胜出）
//
// 输入三（文案，<client>/text/chineses；目录不存在则跳过）：
//   - stringtask_normal/ + stringtask_special/  任务串档（key=value）
//
// 输出（服务端方言 <Object><Property .../>，UTF-8，格式以服务端为准）：
//   - <Out>/NpcConfig/<组>/<表>.xml       NPC 模板（属性名按审定映射表
//     转服务端词汇，映射不到的列不进输出、只在报告记录）
//   - <ScenesOut>/<场景>/npc_creator.xml 每场景摆放（属性名保持 creator 源名）
//   - <QuestOut>/<域>/<域>.xml           任务域表（服务端词汇透传，不映射；
//     主表 Title 已按 TitleId 解析为客户端显示文本）
//   - <Out>/NpcTemplate.xml               战斗 NPC 模板
//   - <TextOut>/TaskText.xml             任务文案（ID=串键，Text=文本）
//   - <ReportPath>                        JSON 审计报告（四类列映射、
//     ID 冲突、孤儿摆放、任务域统计）
//
// 输出目录的规范位直接归九阴产物：黑金原档在参考资源根
// （testdata/references），不再做首跑备份。本项目是九阴服务端复刻而非黑金
// 复刻，黑金档仅作属性词汇对照留存。构建链与黑金样板分析见
// docs/official/NPC服务端XML构建链与黑金样板.md。
package npc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/config"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// Options 一次 NPC 构建的输入输出（路径由 tools/datagen 解析好传入）。
type Options struct {
	Client     string // 客户端资源解压根（含 share/npc/npcconfig、share/creator、share/task、text/chineses）
	Out        string // 服务端 ini/Npc 输出根：NpcConfig/<组>/<表>.xml + NpcTemplate.xml
	ScenesOut  string // 场景摆放 XML 输出根（ini/Scene）：<场景>/npc_creator.xml
	QuestOut   string // 任务域 XML 输出根（ini/Quest）
	TextOut    string // 任务文案 XML 输出根（ini/Text）
	ReportPath string // 审计报告 JSON
	Verify     bool   // 生成后用 config.LoadObjectFile 回读校验行数与 ID 集
}

// Run 构建 NPC 配置：模板表 / 场景摆放 / 任务域 / 战斗模板 / 文案 → 落盘 +
// 报告 + 控制台摘要。
func Run(opt Options) error {
	// 战斗 NPC 模板源同在客户端 share/task 下（npctemplate*.tab）。
	npctemplate := filepath.Join(opt.Client, "share", "task")
	return run(opt.Client, opt.Out, opt.ScenesOut, npctemplate, opt.QuestOut, opt.TextOut, opt.ReportPath, opt.Verify)
}

// ---- 审计报告 ----

type tableReport struct {
	Client     string   `json:"client"`
	Server     string   `json:"server"`
	Rows       int      `json:"rows"`
	Direct     []string `json:"direct,omitempty"`
	Rename     []string `json:"rename,omitempty"`
	TableRec   []string `json:"table_rec,omitempty"`
	Dropped    []string `json:"dropped,omitempty"`
	Duplicates []string `json:"duplicate_columns,omitempty"`
	Notes      []string `json:"notes,omitempty"`
}

type genReport struct {
	Modern      string        `json:"modern"`
	Tables      []tableReport `json:"tables"`
	TotalRows   int           `json:"total_rows"`
	UniqueIDs   int           `json:"unique_ids"`
	IDConflicts []string      `json:"id_conflicts,omitempty"`
	Creators    struct {
		Scenes       int      `json:"scenes"`
		Instances    int      `json:"instances"`
		OrphanCount  int      `json:"orphan_count"`
		OrphanSample []string `json:"orphan_sample,omitempty"`
	} `json:"creators"`
	Quest struct {
		Source            string         `json:"source,omitempty"`
		NpcTemplateSource string         `json:"npc_template_source,omitempty"`
		Domains           []domainReport `json:"domains,omitempty"`
		NpcTemplateRow    int            `json:"npc_template_rows"`
		TextRows          int            `json:"text_rows"`
		TextConflicts     int            `json:"text_conflicts"`
		NpcTalkRows       int            `json:"npc_talk_rows"`
		NpcTalkConflicts  int            `json:"npc_talk_conflicts"`
	} `json:"quest"`
	Warnings []string `json:"warnings,omitempty"`
}

func run(modern, outDir, scenesOut, npctemplateIn, questOut, textOut, reportPath string, verify bool) error {
	npcconfigDir := filepath.Join(modern, "share", "npc", "npcconfig")
	creatorDir := filepath.Join(modern, "share", "creator", "npc_creator")
	taskDir := filepath.Join(modern, "share", "task", "task")
	for _, dir := range []string{npcconfigDir, creatorDir, taskDir} {
		if info, err := res.Stat(dir); err != nil || !info.IsDir() {
			return fmt.Errorf("npc: 输入目录不可用 %s", dir)
		}
	}

	report := &genReport{Modern: modern}

	entries, warnings, err := loadManifest(npcconfigDir)
	if err != nil {
		return err
	}
	report.Warnings = append(report.Warnings, warnings...)

	// 模板表转换。
	allIDs := map[string]string{} // ID → 首个来源表（冲突审计）
	written := map[string]int{}   // 输出路径 → 期望行数（verify 用）
	var writtenOrder []string
	for _, entry := range entries {
		tr := tableReport{Client: entry.ClientPath, Server: filepath.ToSlash(entry.ServerRel)}
		path := filepath.Join(npcconfigDir, filepath.FromSlash(entry.ClientPath))
		raw, err := res.ReadFile(path)
		if err != nil {
			tr.Notes = append(tr.Notes, fmt.Sprintf("打开失败: %v", err))
			report.Tables = append(report.Tables, tr)
			continue
		}
		table, err := LoadNPCTable(bytes.NewReader(raw), entry.ClientPath)
		if err != nil {
			tr.Notes = append(tr.Notes, fmt.Sprintf("解析失败: %v", err))
			report.Tables = append(report.Tables, tr)
			continue
		}
		plan, dups := planTable(table)
		tr.Duplicates = dups
		first := table.firstIndex()
		for _, pc := range plan.Columns {
			switch pc.Kind {
			case "direct":
				tr.Direct = append(tr.Direct, pc.SourceName)
			case "rename":
				tr.Rename = append(tr.Rename, fmt.Sprintf("%s→%s", pc.SourceName, pc.OutAttr))
			case "tableRec":
				tr.TableRec = append(tr.TableRec, fmt.Sprintf("%s→%s", pc.SourceName, pc.OutAttr))
			default:
				tr.Dropped = append(tr.Dropped, droppedNote(table, pc))
			}
		}
		data := genNpcConfig(table, plan)
		outPath := filepath.Join(outDir, filepath.FromSlash(entry.ServerRel))
		if err := writeGenerated(outPath, data); err != nil {
			return err
		}
		// 行数与 ID 审计（与 genNpcConfig 的取舍保持一致：空 ID 行跳过）。
		rows := 0
		idIdx, hasID := first["ID"]
		for _, row := range table.Rows {
			if !hasID || idIdx >= len(row) || row[idIdx] == "" {
				continue
			}
			rows++
			id := row[idIdx]
			if source, ok := allIDs[id]; ok && source != entry.ClientPath {
				report.IDConflicts = append(report.IDConflicts, fmt.Sprintf("%s: %s 与 %s", id, source, entry.ClientPath))
			} else {
				allIDs[id] = entry.ClientPath
			}
		}
		tr.Rows = rows
		report.TotalRows += rows
		written[outPath] = rows
		writtenOrder = append(writtenOrder, outPath)
		report.Tables = append(report.Tables, tr)
	}
	report.UniqueIDs = len(allIDs)

	// 场景摆放转换 + 孤儿审计。
	sceneNames, err := listScenes(creatorDir)
	if err != nil {
		return err
	}
	for _, scene := range sceneNames {
		data, count, warnings, err := convertScene(filepath.Join(creatorDir, scene))
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("场景 %s: %v", scene, err))
			continue
		}
		for _, w := range warnings {
			report.Warnings = append(report.Warnings, fmt.Sprintf("场景 %s: %s", scene, w))
		}
		outPath := filepath.Join(scenesOut, scene, "npc_creator.xml")
		if err := writeGenerated(outPath, data); err != nil {
			return err
		}
		written[outPath] = count
		writtenOrder = append(writtenOrder, outPath)
		report.Creators.Scenes++
		report.Creators.Instances += count
	}
	for orphan := range collectOrphans(scenesOut, allIDs) {
		report.Creators.OrphanCount++
		if len(report.Creators.OrphanSample) < 50 {
			report.Creators.OrphanSample = append(report.Creators.OrphanSample, orphan)
		}
	}
	sort.Strings(report.Creators.OrphanSample)

	// 任务文案先收集：主表 Title 解析与 TaskText.xml 取同一份键值表。
	textRoot := filepath.Join(modern, "text", "chineses")
	textEntries := map[string]string{}
	textConflicts := 0
	hasTextRoot := false
	if info, err := res.Stat(textRoot); err == nil && info.IsDir() {
		hasTextRoot = true
		entries, _, conflicts, warnings, err := collectTaskText(textRoot)
		report.Warnings = append(report.Warnings, warnings...)
		if err != nil {
			return err
		}
		textEntries, textConflicts = entries, conflicts
	} else {
		fmt.Printf("npc: 文案目录不存在，跳过文案提取（%s）\n", textRoot)
	}

	// 客户端任务域（share/task；主表 Title 按 TitleId 解析文案）。
	taskRoot := filepath.Join(modern, "share", "task")
	report.Quest.Source = taskRoot
	domains, questWritten, warnings, err := convertQuest(taskRoot, textEntries, questOut)
	if err != nil {
		return err
	}
	report.Quest.Domains = domains
	report.Warnings = append(report.Warnings, warnings...)
	for path, rows := range questWritten {
		written[path] = rows
		writtenOrder = append(writtenOrder, path)
	}

	// 战斗 NPC 模板（-npctemplate 独立源：文件或所在目录；npctemplate*.tab
	// glob 合并——npctemplate2.tab 空壳/坏档 warning 跳过）。
	report.Quest.NpcTemplateSource = npctemplateIn
	if rows, outPath, warnings, err := convertNpcTemplate(npctemplateIn, outDir); err == nil {
		report.Quest.NpcTemplateRow = rows
		report.Warnings = append(report.Warnings, warnings...)
		written[outPath] = rows
		writtenOrder = append(writtenOrder, outPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("npc: npctemplate 源 %s: %w", npctemplateIn, err)
	} else {
		fmt.Printf("npc: 战斗模板源不存在，跳过（-npctemplate=%s）\n", npctemplateIn)
	}

	// 任务文案落盘（与主表 Title 同源）与 NPC 对话框散档
	// （string_npctalk/string_taskmenu/stringtalk → NpcTalk.xml）。
	if hasTextRoot {
		taskTextPath := filepath.Join(textOut, "TaskText.xml")
		if err := writeTextTable(taskTextPath, textEntries); err != nil {
			return err
		}
		report.Quest.TextRows = len(textEntries)
		report.Quest.TextConflicts = textConflicts
		written[taskTextPath] = len(textEntries)
		writtenOrder = append(writtenOrder, taskTextPath)
		talkRows, talkConflicts, talkWarnings, err := convertNpcTalk(textRoot, textOut)
		if err != nil {
			return err
		}
		report.Quest.NpcTalkRows = talkRows
		report.Quest.NpcTalkConflicts = talkConflicts
		report.Warnings = append(report.Warnings, talkWarnings...)
		npcTalkPath := filepath.Join(textOut, "NpcTalk.xml")
		written[npcTalkPath] = talkRows
		writtenOrder = append(writtenOrder, npcTalkPath)
	}

	// 报告落盘。
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return err
	}
	blob, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(reportPath, append(blob, '\n'), 0o644); err != nil {
		return err
	}

	// 摘要。
	fmt.Printf("npc: %d 表 %d 行（唯一 ID %d）→ %s\n", len(report.Tables), report.TotalRows, report.UniqueIDs, outDir)
	fmt.Printf("npc: %d 场景 %d 摆放实例 → %s（孤儿 %d）\n", report.Creators.Scenes, report.Creators.Instances, scenesOut, report.Creators.OrphanCount)
	if report.Quest.Source != "" {
		fmt.Printf("npc: 任务域 %d 档（Task/StoryTaskList/SubFunction/...）→ %s\n", len(report.Quest.Domains), questOut)
	}
	if report.Quest.NpcTemplateRow > 0 {
		fmt.Printf("npc: 战斗 NPC 模板 %d 行（源 %s）→ %s\n",
			report.Quest.NpcTemplateRow, report.Quest.NpcTemplateSource, filepath.Join(outDir, "NpcTemplate.xml"))
	}
	if report.Quest.TextRows > 0 {
		fmt.Printf("npc: 文案 %d 键 → %s\n", report.Quest.TextRows, textOut)
	}
	if report.Quest.NpcTalkRows > 0 {
		fmt.Printf("npc: NPC 对话框文案 %d 键 → %s\n", report.Quest.NpcTalkRows, filepath.Join(textOut, "NpcTalk.xml"))
	}
	fmt.Printf("npc: 报告 %s\n", reportPath)
	for _, w := range report.Warnings {
		fmt.Println("  警告:", w)
	}
	if len(report.IDConflicts) > 0 {
		fmt.Printf("  ID 冲突 %d 条（详见报告）\n", len(report.IDConflicts))
	}

	if verify {
		return verifyWritten(writtenOrder, written)
	}
	return nil
}

// listScenes 场景目录清单（字典序）。
func listScenes(creatorDir string) ([]string, error) {
	entries, err := res.ReadDir(creatorDir)
	if err != nil {
		return nil, fmt.Errorf("npc: 读场景根 %s: %w", creatorDir, err)
	}
	var scenes []string
	for _, e := range entries {
		if e.IsDir() {
			scenes = append(scenes, e.Name())
		}
	}
	sort.Strings(scenes)
	return scenes, nil
}

// collectOrphans 摆放 id 不在模板 ID 全集的孤儿（去重）。
func collectOrphans(scenesOut string, allIDs map[string]string) map[string]struct{} {
	orphans := map[string]struct{}{}
	_ = filepath.WalkDir(scenesOut, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".xml") {
			return err
		}
		doc, err := config.LoadObjectFile(path)
		if err != nil {
			return nil // 孤儿审计尽力而为
		}
		for _, row := range doc.Rows {
			id := row["id"]
			if id == "" {
				id = row["ID"]
			}
			if id != "" {
				if _, ok := allIDs[id]; !ok {
					orphans[id] = struct{}{}
				}
			}
		}
		return nil
	})
	return orphans
}

// verifyWritten 用服务端装载器回读全部生成物，校验行数。
func verifyWritten(order []string, expected map[string]int) error {
	bad := 0
	for _, path := range order {
		doc, err := config.LoadObjectFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "npc: verify %s: %v\n", path, err)
			bad++
			continue
		}
		if len(doc.Rows) != expected[path] {
			fmt.Fprintf(os.Stderr, "npc: verify %s: 行数 %d != 期望 %d\n", path, len(doc.Rows), expected[path])
			bad++
		}
	}
	if bad > 0 {
		return fmt.Errorf("npc: verify 失败 %d 个文件", bad)
	}
	fmt.Printf("npc: verify 通过（%d 个文件经 config.LoadObjectFile 回读一致）\n", len(order))
	return nil
}
