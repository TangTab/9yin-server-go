// taskgen 是任务目录快照生成工具（独立 CLI 形态）。
//
// 逻辑复刻自 official-2433-b-line 分支 cmd/protocol-probe/task_catalog.go
// 的 openTaskCatalogAt/rebuildTaskSnapshot 派生链，适配点：
//   - 资源访问由资源仓合并视图（resourceAccessor）退化为直接文件系统
//     读取（b-line 侧 walkResourceTree/readResourceBytes 本就有该回退路径）
//   - 工程根定位收窄为 -root flag > NINEYIN_SERVER_ROOT 环境变量 >
//     向上查找含 resources/modern/share 的目录 > 工作目录
//
// 数据源（均在工程 resources/ 下）：
//   - 任务文本：text/chineses/stringtask_normal/*.idres（TID 前缀白名单
//     见 internal/taskdata/text.go）
//   - NPC 绑定：share/task/task/storytasklist/*.txt +
//     share/task/qianjiu_tasknpc.ini 的 [TaskNpc] 段
//   - 任务链：share/task/task/task_drama_cite.ini
//
// 快照让 protocol-probe 类消费方启动避开 500+ 个 stringtask idres 与
// storytasklist 表的全量解析；本工具只负责生成/校验快照文件。
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

const serverRootEnvironment = "NINEYIN_SERVER_ROOT"

func main() {
	log.SetFlags(0)
	log.SetPrefix("taskgen: ")

	root := flag.String("root", defaultProjectRoot(), "工程根目录（resources/ 所在地）")
	out := flag.String("out", "", "快照输出路径（默认 <root>/artifacts/local/tasks.json）")
	force := flag.Bool("force", false, "忽略现有快照，强制从资源重扫")
	flag.Parse()

	if strings.TrimSpace(*root) == "" {
		log.Fatalf("empty project root")
	}
	snapshotPath := *out
	if snapshotPath == "" {
		snapshotPath = filepath.Join(*root, "artifacts", "local", "tasks.json")
	}

	if !*force {
		if data, err := os.ReadFile(snapshotPath); err == nil {
			if catalog, parseErr := taskdata.ParseSnapshot(data); parseErr == nil {
				log.Printf("snapshot ready tasks=%d bindings=%d bound_npcs=%d path=%s (use -force to rebuild)",
					catalog.Count(), catalog.BindingCount(), catalog.BoundNPCCount(), snapshotPath)
				return
			} else {
				log.Printf("snapshot unusable %s: %v; re-deriving from resources", snapshotPath, parseErr)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			log.Printf("read snapshot %s: %v; re-deriving from resources", snapshotPath, err)
		}
	}

	catalog, stats, err := rebuildTaskSnapshot(*root, snapshotPath)
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("catalog ready derived tasks=%d bindings=%d bound_npcs=%d snapshot_bytes=%d path=%s",
		stats.Tasks, stats.Bindings, stats.BoundNPCs, stats.Bytes, snapshotPath)
	_ = catalog
}

// taskSnapshotStats 汇总一次派生的结果，供日志回执。
type taskSnapshotStats struct {
	Tasks     int
	Bindings  int
	BoundNPCs int
	Bytes     int
}

// defaultTaskPaths 是三类数据源在工程根下的固定相对位置。
func defaultTaskPaths(root string) (textDir, storyListDir, npcIniPath, dramaCitePath string) {
	modern := filepath.Join(root, "resources", "modern")
	textDir = filepath.Join(modern, "text", "chineses", "stringtask_normal")
	storyListDir = filepath.Join(modern, "share", "task", "task", "storytasklist")
	npcIniPath = filepath.Join(modern, "share", "task", "qianjiu_tasknpc.ini")
	dramaCitePath = filepath.Join(modern, "share", "task", "task", "task_drama_cite.ini")
	return
}

// rebuildTaskSnapshot 从客户端资源重新派生任务目录并原子覆写快照。
func rebuildTaskSnapshot(root, snapshotPath string) (*taskdata.Catalog, taskSnapshotStats, error) {
	catalog, err := deriveTaskCatalog(root)
	if err != nil {
		return nil, taskSnapshotStats{}, err
	}
	data, err := taskdata.MarshalSnapshot(catalog.SnapshotAt(time.Now().UTC()))
	if err != nil {
		return nil, taskSnapshotStats{}, err
	}
	if err := persistTaskSnapshot(snapshotPath, data); err != nil {
		// 目录已在内存，快照写失败只影响下次启动的速度。
		return catalog, taskSnapshotStats{Tasks: catalog.Count(), Bindings: catalog.BindingCount(), BoundNPCs: catalog.BoundNPCCount()}, fmt.Errorf("persist task snapshot: %w", err)
	}
	return catalog, taskSnapshotStats{
		Tasks:     catalog.Count(),
		Bindings:  catalog.BindingCount(),
		BoundNPCs: catalog.BoundNPCCount(),
		Bytes:     len(data),
	}, nil
}

// deriveTaskCatalog 扫描资源并组装目录。文本与绑定目录缺失属致命
// 错误（返回 error）；单文件的解析失败记日志跳过。
func deriveTaskCatalog(root string) (*taskdata.Catalog, error) {
	textDir, storyListDir, npcIniPath, dramaCitePath := defaultTaskPaths(root)

	texts := make(map[int]*taskdata.Task)
	textFiles := 0
	if err := filepath.WalkDir(textDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".idres") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		taskdata.CollectTaskText(raw, texts)
		textFiles++
		return nil
	}); err != nil {
		return nil, fmt.Errorf("scan task text %s: %w", textDir, err)
	}
	if textFiles == 0 {
		return nil, fmt.Errorf("no *.idres found under %s", textDir)
	}
	tasks := make(map[int]taskdata.Task, len(texts))
	for id, task := range texts {
		tasks[id] = *task
	}

	var bindings []taskdata.Binding
	if err := filepath.WalkDir(storyListDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".txt") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		fileBindings, parseErr := taskdata.LoadStoryTaskListFromBytes(raw, path)
		if parseErr != nil {
			log.Printf("skip storytasklist %s: %v", path, parseErr)
			return nil
		}
		bindings = append(bindings, fileBindings...)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("scan storytasklist %s: %w", storyListDir, err)
	}
	// 愆咎城任务 NPC 表是补充来源，缺失或损坏只记日志。
	if raw, err := os.ReadFile(npcIniPath); err == nil {
		if npcBindings, iniErr := taskdata.LoadTaskNPCIniFromBytes(raw, npcIniPath); iniErr == nil {
			bindings = append(bindings, npcBindings...)
		} else {
			log.Printf("skip task npc ini %s: %v", npcIniPath, iniErr)
		}
	} else {
		log.Printf("skip task npc ini %s: %v", npcIniPath, err)
	}
	if len(bindings) == 0 {
		return nil, fmt.Errorf("no NPC task bindings derived from %s", storyListDir)
	}

	var cites map[int]taskdata.DramaCite
	if raw, err := os.ReadFile(dramaCitePath); err == nil {
		cites = taskdata.LoadDramaCiteFromBytes(raw)
	} else {
		log.Printf("skip task drama cite %s: %v", dramaCitePath, err)
	}
	return taskdata.NewCatalog(tasks, bindings, cites), nil
}

// persistTaskSnapshot 原子写快照（临时文件 + Sync + Rename）。
func persistTaskSnapshot(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create task snapshot directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".tasks-*.tmp")
	if err != nil {
		return fmt.Errorf("create task snapshot temporary: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set task snapshot temporary permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write task snapshot temporary: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync task snapshot temporary: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close task snapshot temporary: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace task snapshot: %w", err)
	}
	committed = true
	return nil
}

// defaultProjectRoot 定位工程根：环境变量 > 向上查找 > 工作目录。
func defaultProjectRoot() string {
	if configured := strings.TrimSpace(os.Getenv(serverRootEnvironment)); configured != "" {
		return filepath.Clean(configured)
	}
	var starts []string
	if executable, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(executable))
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		starts = append(starts, workingDirectory)
	}
	for _, start := range starts {
		if root, ok := findProjectRoot(start); ok {
			return root
		}
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		return filepath.Clean(workingDirectory)
	}
	return "."
}

func findProjectRoot(start string) (string, bool) {
	for current := filepath.Clean(start); ; current = filepath.Dir(current) {
		if info, err := os.Stat(filepath.Join(current, "resources", "modern", "share")); err == nil && info.IsDir() {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
	}
}
