package res

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/local/9yin-go-server/internal/mpq"
)

// 三层资源读取：本地散文件 > res-patch.rep > res-base.rep，第一个命中生效。
// 实例并发安全；Reload 可热切换补丁（原子替换归档句柄）。
//
// 目录布局（dir 为资源根，统一绝对化）：
//
//	dir/res-base.rep        全量基础层（必需）
//	dir/res-patch.rep       覆盖层（可选，缺省或清单缺失时无此层）
//	dir/res-manifest.json   外置清单（可选；缺失时按缺省布局只用 base + 本地层）
//	dir/resources/...       本地散文件（名字与归档内一致，含 resources\ 前缀）

// ResFile 是合并视图中的一个文件。
type ResFile struct {
	Name  string
	Size  int64
	Layer string // "local" / "patch" / "base"
}

// layers 持有打开中的归档句柄（Reload 时整体原子换出）。
type layers struct {
	base  *mpq.Archive
	patch *mpq.Archive // nil = 无补丁层
}

func (l *layers) close() {
	l.base.Close()
	if l.patch != nil {
		l.patch.Close()
	}
}

// Store 是一个打开的三层资源仓。
type Store struct {
	dir       string
	localRoot string
	layers    atomic.Pointer[layers]
	manifest  atomic.Pointer[Manifest]
}

// Open 打开资源根目录。清单存在则按清单；否则按缺省布局
// （res-base.rep + resources/ 本地层，无补丁）。
func Open(dir string) (*Store, error) {
	// 统一绝对化：相对路径（如 "."）normalize 后可能成为空路径，
	// 使本地层的前缀校验失效。
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	root = filepath.Clean(root)

	manifest := DefaultManifest()
	if _, err := os.Stat(manifestPath(root)); err == nil {
		if manifest, err = LoadManifest(manifestPath(root)); err != nil {
			return nil, err
		}
	}
	l, err := openLayers(root, manifest)
	if err != nil {
		return nil, err
	}
	s := &Store{dir: root}
	s.localRoot = filepath.Join(root, localDirOf(manifest))
	s.layers.Store(l)
	s.manifest.Store(manifest)
	return s, nil
}

func localDirOf(m *Manifest) string {
	if m.LocalDir == "" {
		return DefaultLocalDir
	}
	return localDirName(m.LocalDir)
}

func openLayers(dir string, manifest *Manifest) (*layers, error) {
	basePath := filepath.Join(dir, manifest.BaseFile)
	if st, err := os.Stat(basePath); err != nil || st.IsDir() {
		return nil, fmt.Errorf("res: 基础层不存在：%s", basePath)
	}
	base, err := mpq.Open(basePath)
	if err != nil {
		return nil, err
	}
	var patch *mpq.Archive
	if manifest.PatchFile != "" {
		patchPath := filepath.Join(dir, manifest.PatchFile)
		if st, err := os.Stat(patchPath); err == nil && !st.IsDir() {
			if patch, err = mpq.Open(patchPath); err != nil {
				base.Close()
				return nil, err
			}
		}
	}
	return &layers{base: base, patch: patch}, nil
}

// Reload 热重载：按磁盘上最新的清单重开 base/patch，原子换出句柄。
func (s *Store) Reload() error {
	manifest := DefaultManifest()
	if _, err := os.Stat(manifestPath(s.dir)); err == nil {
		if manifest, err = LoadManifest(manifestPath(s.dir)); err != nil {
			return err
		}
	}
	next, err := openLayers(s.dir, manifest)
	if err != nil {
		return err
	}
	old := s.layers.Swap(next)
	s.manifest.Store(manifest)
	if old != nil {
		old.close()
	}
	return nil
}

// Close 释放归档句柄。
func (s *Store) Close() {
	if l := s.layers.Load(); l != nil {
		l.close()
	}
}

// Layer 返回名字所在层："local" / "patch" / "base"；
// 不存在（含墓碑）返回空串。
func (s *Store) Layer(name string) string {
	norm := Normalize(name)
	if s.isDeleted(norm) {
		return ""
	}
	if s.localFileOf(norm) != "" {
		return "local"
	}
	l := s.layers.Load()
	if l.patch != nil && l.patch.HasFile(norm) {
		return "patch"
	}
	if l.base.HasFile(norm) {
		return "base"
	}
	return ""
}

// Exists 报告名字是否存在（墓碑视为不存在）。
func (s *Store) Exists(name string) bool { return s.Layer(name) != "" }

// Read 读取文件内容；不存在（含墓碑）时报错。
func (s *Store) Read(name string) ([]byte, error) {
	norm := Normalize(name)
	if s.isDeleted(norm) {
		return nil, fmt.Errorf("res: 资源不存在（墓碑）：%s", name)
	}
	if local := s.localFileOf(norm); local != "" {
		return os.ReadFile(local)
	}
	l := s.layers.Load()
	if l.patch != nil && l.patch.HasFile(norm) {
		return l.patch.ReadFile(norm)
	}
	if l.base.HasFile(norm) {
		return l.base.ReadFile(norm)
	}
	return nil, fmt.Errorf("res: 资源不存在：%s", name)
}

func (s *Store) isDeleted(normalized string) bool {
	return s.manifest.Load().Deleted[strings.ToLower(normalized)]
}

// localFileOf 返回本地散文件路径；文件不存在或名字越界（路径穿越）时返回空串。
func (s *Store) localFileOf(normalized string) string {
	if normalized == "" {
		return ""
	}
	candidate := filepath.Join(s.dir, filepath.FromSlash(strings.ReplaceAll(normalized, `\`, "/")))
	// 拒绝 .. / 绝对路径穿越。
	rel, err := filepath.Rel(s.dir, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
		return candidate
	}
	return ""
}

// List 返回三层合并后的全量清单：patch 覆盖 base，本地覆盖 patch，墓碑移除。
// 顺序与参考实现一致（base 序、patch 序、本地序）。
func (s *Store) List() ([]ResFile, error) {
	l := s.layers.Load()
	index := make(map[string]int)
	var merged []ResFile
	upsert := func(name, layer string, size int64) {
		key := NormalizeLower(name)
		if i, ok := index[key]; ok {
			merged[i] = ResFile{Name: name, Size: size, Layer: layer}
			return
		}
		index[key] = len(merged)
		merged = append(merged, ResFile{Name: name, Size: size, Layer: layer})
	}

	for _, e := range l.base.Entries() {
		if e.Name == "" || !e.Exists() || strings.EqualFold(e.Name, "(listfile)") {
			continue
		}
		upsert(e.Name, "base", int64(e.Size))
	}
	if l.patch != nil {
		for _, e := range l.patch.Entries() {
			if e.Name == "" || !e.Exists() || strings.EqualFold(e.Name, "(listfile)") {
				continue
			}
			upsert(e.Name, "patch", int64(e.Size))
		}
	}
	// 本地层：只枚举清单 localDir 子树（名字与归档一致，含 localDir 前缀）。
	localDirName := localDirOf(s.manifest.Load())
	localTree := filepath.Join(s.dir, localDirName)
	if st, err := os.Stat(localTree); err == nil && st.IsDir() {
		err := filepath.WalkDir(localTree, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(localTree, path)
			if err != nil {
				return err
			}
			name := localDirName + "/" + filepath.ToSlash(rel)
			upsert(Normalize(name), "local", fileSizeOf(path))
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("res: 本地层枚举失败：%s（%v）", localTree, err)
		}
	}
	// 墓碑移除（键已统一小写，与合并键一致）。
	if deleted := s.manifest.Load().Deleted; len(deleted) > 0 {
		kept := merged[:0]
		for _, f := range merged {
			if !deleted[NormalizeLower(f.Name)] {
				kept = append(kept, f)
			}
		}
		merged = kept
	}
	return merged, nil
}

func fileSizeOf(path string) int64 {
	if st, err := os.Stat(path); err == nil {
		return st.Size()
	}
	return 0
}

// -- 诊断 ------------------------------------------------------------------

// BaseCount 返回 base 层条目数。
func (s *Store) BaseCount() int { return len(s.layers.Load().base.Entries()) }

// PatchCount 返回 patch 层条目数（无补丁层为 0）。
func (s *Store) PatchCount() int {
	l := s.layers.Load()
	if l.patch != nil {
		return len(l.patch.Entries())
	}
	return 0
}

// Warnings 返回打开告警（base 与 patch 之和）。
func (s *Store) Warnings() []string {
	l := s.layers.Load()
	out := append([]string(nil), l.base.Warnings()...)
	if l.patch != nil {
		out = append(out, l.patch.Warnings()...)
	}
	return out
}

// BaseVersion 返回清单声明的 base 版本。
func (s *Store) BaseVersion() string { return s.manifest.Load().BaseVersion }

// PatchVersion 返回清单声明的 patch 版本。
func (s *Store) PatchVersion() string { return s.manifest.Load().PatchVersion }

// HasPatch 报告是否有补丁层。
func (s *Store) HasPatch() bool { return s.layers.Load().patch != nil }
