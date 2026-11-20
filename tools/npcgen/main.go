// npcgen：九阴客户端 NPC 数据 → 服务端 ini 配置转换器。
//
// 输入一（-in，默认 NINEYIN_MODERN 环境变量 > 仓库兄弟目录
// ..\9yin_server-go\resources\modern）：
//   - share/npc/npcconfig/*.txt        76 张 GBK+TAB 模板表（3 行头）
//   - share/npc/npcconfig/npc_type.ini 表 → 服务端原路径清单（[file]/flie<N>）
//   - share/creator/npc_creator/<场景>/*.xml  场景摆放（静态+随机 creator）
//
// 输入二（-tasks，默认 NINEYIN_TASKS 环境变量 > 仓库兄弟目录
// ..\九阴真经ol\task；目录不存在则跳过）：
//   - Task/<域>/<前缀>_*.txt   BGserverCJ 任务域表（HuntList 杀怪/
//     CollectList 采集/Convoy 押镖/Prize 奖励/StoryTaskList 剧情等，
//     同域分档合并去重）
//   - task002.txt / task_config.txt / CharEffect.txt / npctemplate.tab
//
// 输出（服务端方言 <Object><Property .../>，UTF-8，格式以服务端为准）：
//   - <out>/NpcConfig/<组>/<表>.xml       NPC 模板（属性名按审定映射表
//     转服务端词汇，映射不到的列不进输出、只在报告记录）
//   - <scenes-out>/<场景>/npc_creator.xml 每场景摆放（属性名保持 creator 源名）
//   - <quest-out>/<域>/<域>.xml           任务域表（服务端词汇透传，不映射）
//   - <out>/NpcTemplate.xml               战斗 NPC 模板（BGserverCJ 侧）
//   - <report>                            JSON 审计报告（四类列映射、
//     ID 冲突、孤儿摆放、任务域统计）
//
// 原 NpcConfig/NpcConfigAuto 黑金档首跑前整体移入 <out>/heijin_backup/。
// 本项目是九阴服务端复刻而非黑金复刻，黑金档仅作属性词汇对照留存。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/config"
)

// modernRootEnvironment 现代客户端提取资源根的环境变量。
const modernRootEnvironment = "NINEYIN_MODERN"

func main() {
	in := flag.String("in", defaultModernRoot(), "现代客户端提取资源根（含 share/npc/npcconfig）")
	out := flag.String("out", filepath.Join(repoTestData(), "ini", "Npc"), "服务端 ini/Npc 输出根")
	scenesOut := flag.String("scenes-out", filepath.Join(repoTestData(), "ini", "scene"), "场景摆放 XML 输出根")
	tasksIn := flag.String("tasks", defaultTasksRoot(), "BGserverCJ 任务数据根（含 Task\\ 与 npctemplate.tab；目录不存在则跳过任务转换）")
	questOut := flag.String("quest-out", filepath.Join(repoTestData(), "ini", "Quest"), "任务域 XML 输出根")
	reportPath := flag.String("report", filepath.Join(repoRoot(), "artifacts", "local", "npcgen-report.json"), "审计报告 JSON 输出")
	verify := flag.Bool("verify", false, "生成后用 config.LoadObjectFile 回读校验行数与 ID 集")
	flag.Parse()
	if err := run(*in, *out, *scenesOut, *tasksIn, *questOut, *reportPath, *verify); err != nil {
		fmt.Fprintln(os.Stderr, "npcgen:", err)
		os.Exit(1)
	}
}

// defaultTasksRoot BGserverCJ 任务数据根：flag > NINEYIN_TASKS 环境变量 >
// 仓库兄弟目录 九阴真经ol\task。
func defaultTasksRoot() string {
	if configured := strings.TrimSpace(os.Getenv("NINEYIN_TASKS")); configured != "" {
		return filepath.Clean(configured)
	}
	return filepath.Join(filepath.Dir(repoRoot()), "九阴真经ol", "task")
}

// defaultModernRoot 输入根解析：flag > NINEYIN_MODERN > 仓库兄弟目录
// 9yin_server-go/resources/modern（九阴客户端提取资源所在工程）。
func defaultModernRoot() string {
	if configured := strings.TrimSpace(os.Getenv(modernRootEnvironment)); configured != "" {
		return filepath.Clean(configured)
	}
	return filepath.Join(filepath.Dir(repoRoot()), "9yin_server-go", "resources", "modern")
}

// repoRoot 向上定位仓库根（go.mod 所在层）。
func repoRoot() string {
	var starts []string
	if executable, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(executable))
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		starts = append(starts, workingDirectory)
	}
	for _, start := range starts {
		for current := filepath.Clean(start); ; current = filepath.Dir(current) {
			if info, err := os.Stat(filepath.Join(current, "go.mod")); err == nil && !info.IsDir() {
				return current
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		return filepath.Clean(workingDirectory)
	}
	return "."
}

func repoTestData() string { return filepath.Join(repoRoot(), "testdata") }

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
	Backup      string        `json:"backup"`
	Tables      []tableReport `json:"tables"`
	TotalRows   int           `json:"total_rows"`
	UniqueIDs   int           `json:"unique_ids"`
	IDConflicts []string      `json:"id_conflicts,omitempty"`
	Creators    struct {
		Scenes      int      `json:"scenes"`
		Instances   int      `json:"instances"`
		OrphanCount int      `json:"orphan_count"`
		OrphanSample []string `json:"orphan_sample,omitempty"`
	} `json:"creators"`
	Quest struct {
		Source         string         `json:"source,omitempty"`
		Domains        []domainReport `json:"domains,omitempty"`
		NpcTemplateRow int            `json:"npc_template_rows"`
	} `json:"quest"`
	Warnings []string `json:"warnings,omitempty"`
}

func run(modern, outDir, scenesOut, tasksIn, questOut, reportPath string, verify bool) error {
	npcconfigDir := filepath.Join(modern, "share", "npc", "npcconfig")
	creatorDir := filepath.Join(modern, "share", "creator", "npc_creator")
	for _, dir := range []string{npcconfigDir, creatorDir} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return fmt.Errorf("npcgen: 输入目录不可用 %s（-in=%s）", dir, modern)
		}
	}

	report := &genReport{Modern: modern}
	backupNote, err := backupHeiJin(outDir)
	if err != nil {
		return err
	}
	report.Backup = backupNote

	entries, warnings, err := loadManifest(npcconfigDir)
	if err != nil {
		return err
	}
	report.Warnings = append(report.Warnings, warnings...)

	// 模板表转换。
	allIDs := map[string]string{} // ID → 首个来源表（冲突审计）
	written := map[string]int{}    // 输出路径 → 期望行数（verify 用）
	var writtenOrder []string
	for _, entry := range entries {
		tr := tableReport{Client: entry.ClientPath, Server: filepath.ToSlash(entry.ServerRel)}
		path := filepath.Join(npcconfigDir, filepath.FromSlash(entry.ClientPath))
		file, err := os.Open(path)
		if err != nil {
			tr.Notes = append(tr.Notes, fmt.Sprintf("打开失败: %v", err))
			report.Tables = append(report.Tables, tr)
			continue
		}
		table, err := LoadNPCTable(file, entry.ClientPath)
		file.Close()
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

	// BGserverCJ 任务域 + 战斗 NPC 模板（源缺失则跳过，不致命）。
	if info, err := os.Stat(tasksIn); err == nil && info.IsDir() {
		report.Quest.Source = tasksIn
		domains, questWritten, warnings, err := convertQuest(tasksIn, questOut)
		if err != nil {
			return err
		}
		report.Quest.Domains = domains
		report.Warnings = append(report.Warnings, warnings...)
		for path, rows := range questWritten {
			written[path] = rows
			writtenOrder = append(writtenOrder, path)
		}
		if rows, outPath, err := convertNpcTemplate(tasksIn, outDir); err == nil {
			report.Quest.NpcTemplateRow = rows
			written[outPath] = rows
			writtenOrder = append(writtenOrder, outPath)
		} else if !os.IsNotExist(err) {
			report.Warnings = append(report.Warnings, fmt.Sprintf("npctemplate.tab: %v", err))
		}
	} else {
		fmt.Printf("npcgen: 任务数据根不存在，跳过任务转换（-tasks=%s）\n", tasksIn)
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
	fmt.Printf("npcgen: %d 表 %d 行（唯一 ID %d）→ %s\n", len(report.Tables), report.TotalRows, report.UniqueIDs, outDir)
	fmt.Printf("npcgen: %d 场景 %d 摆放实例 → %s（孤儿 %d）\n", report.Creators.Scenes, report.Creators.Instances, scenesOut, report.Creators.OrphanCount)
	if report.Quest.Source != "" {
		fmt.Printf("npcgen: 任务域 %d 档（HuntList/CollectList/...）+ 战斗 NPC 模板 %d 行 → %s\n",
			len(report.Quest.Domains), report.Quest.NpcTemplateRow, questOut)
	}
	fmt.Printf("npcgen: %s；报告 %s\n", report.Backup, reportPath)
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
	entries, err := os.ReadDir(creatorDir)
	if err != nil {
		return nil, fmt.Errorf("npcgen: 读场景根 %s: %w", creatorDir, err)
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
			fmt.Fprintf(os.Stderr, "npcgen: verify %s: %v\n", path, err)
			bad++
			continue
		}
		if len(doc.Rows) != expected[path] {
			fmt.Fprintf(os.Stderr, "npcgen: verify %s: 行数 %d != 期望 %d\n", path, len(doc.Rows), expected[path])
			bad++
		}
	}
	if bad > 0 {
		return fmt.Errorf("npcgen: verify 失败 %d 个文件", bad)
	}
	fmt.Printf("npcgen: verify 通过（%d 个文件经 config.LoadObjectFile 回读一致）\n", len(order))
	return nil
}
