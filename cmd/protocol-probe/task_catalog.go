package main

// 任务目录装载：优先读 data/tasks.json 快照（首次派生后原子落盘），
// 快照缺失/损坏/版本过期时从客户端资源重扫并写回。快照让启动避开
// 500+ 个 stringtask idres 与 storytasklist 表的全量解析。
//
// 数据源（均在工程 resources/ 下，经统一资源仓读取）：
//   - 任务文本：text/chineses/stringtask_normal/*.idres（TID 前缀白名单
//     见 internal/taskdata/text.go；stringtask_special 的 FID 前缀属于
//     子任务文本，接取阶段再引入）
//   - NPC 绑定：share/task/task/storytasklist/*.txt +
//     share/task/qianjiu_tasknpc.ini 的 [TaskNpc] 段
//   - 任务链：share/task/task/task_drama_cite.ini

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/taskdata"
)

var (
	defaultTaskTextDir       = runtimeProjectPath("resources", "modern", "text", "chineses", "stringtask_normal")
	defaultStoryTaskListDir  = runtimeProjectPath("resources", "modern", "share", "task", "task", "storytasklist")
	defaultTaskNPCIniPath    = runtimeProjectPath("resources", "modern", "share", "task", "qianjiu_tasknpc.ini")
	defaultTaskDramaCitePath = runtimeProjectPath("resources", "modern", "share", "task", "task", "task_drama_cite.ini")
)

// taskCatalogPath 是任务快照的固定落盘位置。
func taskCatalogPath() string { return runtimeProjectPath("data", "tasks.json") }

// taskCatalog 是进程级任务目录：与资源仓、Npc_Func 同类的静态配置，
// main 在任何连接进入前装配一次，之后只读。nil 的查询语义由
// taskdata.Catalog 提供（返回空结果），测试可直接替换成夹具目录。
var taskCatalog *taskdata.Catalog

// openTaskCatalog 装载任务目录（快照位于 data/tasks.json）。
func openTaskCatalog() *taskdata.Catalog { return openTaskCatalogAt(taskCatalogPath()) }

// openTaskCatalogAt 装载任务目录。快照不可用则派生并写回；派生失败
// 不致命（与 Npc_Func 同策略）——返回空目录，NPC 不暴露任务绑定。
func openTaskCatalogAt(snapshotPath string) *taskdata.Catalog {
	path := snapshotPath
	if data, err := os.ReadFile(path); err == nil {
		if catalog, parseErr := taskdata.ParseSnapshot(data); parseErr == nil {
			log.Printf("task catalog ready from snapshot tasks=%d bindings=%d bound_npcs=%d path=%s",
				catalog.Count(), catalog.BindingCount(), catalog.BoundNPCCount(), path)
			return catalog
		} else {
			log.Printf("task snapshot unusable %s: %v; re-deriving from resources", path, parseErr)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Printf("read task snapshot %s: %v; re-deriving from resources", path, err)
	}
	catalog, stats, err := rebuildTaskSnapshot()
	if err != nil {
		log.Printf("task catalog unavailable: %v; NPCs will expose no task bindings", err)
		return taskdata.NewCatalog(nil, nil, nil)
	}
	log.Printf("task catalog ready derived tasks=%d bindings=%d bound_npcs=%d snapshot_bytes=%d path=%s",
		stats.Tasks, stats.Bindings, stats.BoundNPCs, stats.Bytes, path)
	return catalog
}

// taskSnapshotStats 汇总一次派生的结果，供启动日志与 GM 回执。
type taskSnapshotStats struct {
	Tasks     int
	Bindings  int
	BoundNPCs int
	Bytes     int
}

// rebuildTaskSnapshot 从客户端资源重新派生任务目录并原子覆写快照。
// GM 面板的"重建任务快照"也走这里；新目录重启后生效（不热换运行中
// 场景持有的指针）。
func rebuildTaskSnapshot() (*taskdata.Catalog, taskSnapshotStats, error) {
	catalog, err := deriveTaskCatalog()
	if err != nil {
		return nil, taskSnapshotStats{}, err
	}
	data, err := taskdata.MarshalSnapshot(catalog.SnapshotAt(time.Now().UTC()))
	if err != nil {
		return nil, taskSnapshotStats{}, err
	}
	path := taskCatalogPath()
	if err := persistTaskSnapshot(path, data); err != nil {
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
// 错误（返回 error，由调用方决定降级）；单文件的解析失败记日志跳过。
func deriveTaskCatalog() (*taskdata.Catalog, error) {
	texts := make(map[int]*taskdata.Task)
	textFiles := 0
	if err := walkResourceTree(defaultTaskTextDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".idres") {
			return nil
		}
		raw, readErr := readResourceBytes(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		taskdata.CollectTaskText(raw, texts)
		textFiles++
		return nil
	}); err != nil {
		return nil, fmt.Errorf("scan task text %s: %w", defaultTaskTextDir, err)
	}
	if textFiles == 0 {
		return nil, fmt.Errorf("no *.idres found under %s", defaultTaskTextDir)
	}
	tasks := make(map[int]taskdata.Task, len(texts))
	for id, task := range texts {
		tasks[id] = *task
	}

	var bindings []taskdata.Binding
	if err := walkResourceTree(defaultStoryTaskListDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".txt") {
			return nil
		}
		raw, readErr := readResourceBytes(path)
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
		return nil, fmt.Errorf("scan storytasklist %s: %w", defaultStoryTaskListDir, err)
	}
	// 愆咎城任务 NPC 表是补充来源，缺失或损坏只记日志。
	if raw, err := readResourceBytes(defaultTaskNPCIniPath); err == nil {
		if npcBindings, iniErr := taskdata.LoadTaskNPCIniFromBytes(raw, defaultTaskNPCIniPath); iniErr == nil {
			bindings = append(bindings, npcBindings...)
		} else {
			log.Printf("skip task npc ini %s: %v", defaultTaskNPCIniPath, iniErr)
		}
	} else {
		log.Printf("skip task npc ini %s: %v", defaultTaskNPCIniPath, err)
	}
	if len(bindings) == 0 {
		return nil, fmt.Errorf("no NPC task bindings derived from %s", defaultStoryTaskListDir)
	}

	var cites map[int]taskdata.DramaCite
	if raw, err := readResourceBytes(defaultTaskDramaCitePath); err == nil {
		cites = taskdata.LoadDramaCiteFromBytes(raw)
	} else {
		log.Printf("skip task drama cite %s: %v", defaultTaskDramaCitePath, err)
	}
	return taskdata.NewCatalog(tasks, bindings, cites), nil
}

// persistTaskSnapshot 原子写快照（临时文件 + Sync + Rename），
// 与 faculty 存档同一模板。
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
