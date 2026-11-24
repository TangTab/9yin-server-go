package res

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq"
)

// 编号多层资源仓。读取顺序固定为
//
//	根下散文件 > <前缀>N.rep > <前缀>N-1.rep > … > <前缀>001.rep
//
// 第一个命中生效。资源名即数据根下的相对路径（\ 分隔、不分大小写），
// 与散文件布局一致，所以同一套相对路径在"纯散文件"与"打包后"两种
// 数据根下都能用。层文件名见 layer.go：测试仓 b001.rep…，正式服
// data001.rep…，参考资源根（黑金原档等）h001.rep…。
//
// 生效时机：散文件实时生效（不进读缓存）；层文件在 Reload 时生效。
// 实例并发安全；Reload 整体原子换出全部层句柄。

// Options 配置资源仓的打开方式。
type Options struct {
	// Prefix 指定层文件族（PrefixTest / PrefixData）。空 = 自动探测：
	// 根下只存在一族时取之，两族并存报错，一族都没有即纯散文件仓。
	Prefix string
}

// ResFile 是合并视图中的一个文件。
type ResFile struct {
	Name  string
	Size  int64
	Layer string // "local" 或层名（如 "b002"）
}

// LayerInfo 是一个层文件的概览。
type LayerInfo struct {
	Number   int
	Name     string // 层名（b002 / data001）
	File     string // 文件名
	Path     string // 绝对路径
	Entries  int    // 归档条目数（含内部文件）
	Deleted  int    // 登记删除的名字数
	Size     int64  // 文件大小
	ModTime  time.Time
	Warnings []string
}

// layer 是一个打开中的层。
type layer struct {
	number  int
	label   string
	file    string
	archive *mpq.Archive
	deleted map[string]bool // 登记删除的名字（规范化小写）
}

// layerSet 持有一次打开得到的全部层（Reload 时整体原子换出）。
type layerSet struct {
	prefix string
	layers []*layer // 按层号升序
}

func (set *layerSet) close() {
	for _, l := range set.layers {
		l.archive.Close()
	}
}

// Store 是一个打开的资源仓。
type Store struct {
	dir     string // 绝对化数据根
	prefix  string // Options 指定的前缀；"" = 每次打开时自动探测
	below   int    // 只含层号小于该值的层（构建去重用；0 = 不限）
	noLoose bool   // 不含散文件层（构建去重用）
	set     atomic.Pointer[layerSet]
	idx     atomic.Pointer[fsIndex]
}

// Open 打开数据根，层文件族自动探测（见 Options）。
func Open(dir string) (*Store, error) {
	return openStore(dir, "", 0, false)
}

// OpenWith 打开数据根并显式指定层文件族。
func OpenWith(dir string, opt Options) (*Store, error) {
	prefix := canonicalPrefix(opt.Prefix)
	if prefix != "" && !IsPrefix(prefix) {
		return nil, fmt.Errorf("res: 无效的层文件前缀 %q（应为 %s 之一）",
			opt.Prefix, strings.Join(Families(), " / "))
	}
	return openStore(dir, prefix, 0, false)
}

func openStore(dir, prefix string, below int, noLoose bool) (*Store, error) {
	s := &Store{prefix: prefix, below: below, noLoose: noLoose}
	// 统一绝对化：相对路径（如 "."）normalize 后可能成为空路径，
	// 使散文件层的前缀校验失效。
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	s.dir = filepath.Clean(root)
	if err := s.reopen(); err != nil {
		return nil, err
	}
	return s, nil
}

// reopen 按磁盘现状重开全部层，原子换出旧句柄。
func (s *Store) reopen() error {
	set, err := openLayerSet(s.dir, s.prefix, s.below)
	if err != nil {
		return err
	}
	if old := s.set.Swap(set); old != nil {
		old.close()
	}
	s.idx.Store(nil)
	return nil
}

// index 返回名称索引（层集合换出后自动重建）。
func (s *Store) index() *fsIndex {
	set := s.set.Load()
	if idx := s.idx.Load(); idx != nil && idx.set == set {
		return idx
	}
	idx := buildIndex(set)
	s.idx.Store(idx)
	return idx
}

func openLayerSet(root, prefix string, below int) (*layerSet, error) {
	families, err := scanLayers(root)
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		// 自动探测：恰好一族时取之；多族并存是配置错误，不猜。
		var present []string
		for _, family := range familyOrder {
			if len(families[family]) > 0 {
				present = append(present, family)
			}
		}
		if len(present) > 1 {
			return nil, fmt.Errorf("res: 资源根 %s 并存多族层文件（%s），须指定前缀",
				root, strings.Join(present, " / "))
		}
		if len(present) == 1 {
			prefix = present[0]
		}
	}

	set := &layerSet{prefix: prefix}
	fail := func(err error) (*layerSet, error) {
		set.close()
		return nil, err
	}
	seen := map[int]string{}
	for _, f := range families[prefix] {
		if f.number < 1 || f.number > maxLayerCount {
			return fail(fmt.Errorf("res: 层号须在 1–%d 之间：%s",
				maxLayerCount, filepath.Join(root, f.name)))
		}
		if prev, dup := seen[f.number]; dup {
			return fail(fmt.Errorf("res: 层号 %d 重复：%s、%s", f.number, prev, f.name))
		}
		seen[f.number] = f.name
		if below > 0 && f.number >= below {
			continue
		}
		archive, err := mpq.Open(filepath.Join(root, f.name))
		if err != nil {
			return fail(fmt.Errorf("res: 层 %s 打开失败：%w", f.name, err))
		}
		deleted, err := loadDeleted(archive, f.name)
		if err != nil {
			archive.Close()
			return fail(err)
		}
		for key := range deleted {
			if archive.HasFile(key) {
				archive.Close()
				return fail(fmt.Errorf("res: 层 %s 对 %s 既收录又登记删除", f.name, key))
			}
		}
		set.layers = append(set.layers, &layer{
			number:  f.number,
			label:   f.label(),
			file:    f.name,
			archive: archive,
			deleted: deleted,
		})
	}
	return set, nil
}

// loadDeleted 读取层内登记的删除名单（(deleted)，每行一个名字）。
func loadDeleted(archive *mpq.Archive, file string) (map[string]bool, error) {
	if !archive.HasFile(deletedFile) {
		return nil, nil
	}
	raw, err := archive.ReadFile(deletedFile)
	if err != nil {
		return nil, fmt.Errorf("res: 层 %s 的 %s 不可读：%w", file, deletedFile, err)
	}
	deleted := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		if name := NormalizeLower(strings.TrimRight(line, "\r")); name != "" {
			deleted[name] = true
		}
	}
	return deleted, nil
}

// Dir 返回绝对化数据根。
func (s *Store) Dir() string { return s.dir }

// Prefix 返回实际使用的层文件族（纯散文件仓为空串）。
func (s *Store) Prefix() string { return s.set.Load().prefix }

// Reload 热重载：按磁盘现状重开全部层，原子换出句柄。
func (s *Store) Reload() error { return s.reopen() }

// Close 释放全部层句柄。
func (s *Store) Close() {
	if set := s.set.Load(); set != nil {
		set.close()
	}
}

// Lookup 返回名字对应文件的大小与所在层；不存在（含被登记删除）时
// ok=false。
func (s *Store) Lookup(name string) (size int64, layer string, ok bool) {
	norm := Normalize(name)
	if norm == "" || isInternalName(norm) {
		return 0, "", false
	}
	if !s.noLoose {
		if host, ok := s.hostPathOf(norm); ok {
			if st, err := os.Stat(host); err == nil && !st.IsDir() {
				return st.Size(), "local", true
			}
		}
	}
	key := strings.ToLower(norm)
	set := s.set.Load()
	for i := len(set.layers) - 1; i >= 0; i-- {
		l := set.layers[i]
		if l.deleted[key] {
			return 0, "", false
		}
		if entry, found := l.archive.FindFile(norm); found && entry.Exists() {
			return int64(entry.Size), l.label, true
		}
	}
	return 0, "", false
}

// Layer 返回名字所在层（"local" 或层名）；不存在时返回空串。
func (s *Store) Layer(name string) string {
	_, layer, ok := s.Lookup(name)
	if !ok {
		return ""
	}
	return layer
}

// Exists 报告名字是否存在。
func (s *Store) Exists(name string) bool {
	_, _, ok := s.Lookup(name)
	return ok
}

// Read 读取文件内容；不存在时返回 *fs.PathError（Err 为 fs.ErrNotExist），
// errors.Is(err, fs.ErrNotExist) 与 os.IsNotExist(err) 都可用。
// 返回的切片归调用方所有。
func (s *Store) Read(name string) ([]byte, error) {
	data, _, err := s.read(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// 用 PathError 而不是 %w 包装：os.IsNotExist 只拆 PathError。
			return nil, &fs.PathError{Op: "open", Path: Normalize(name), Err: fs.ErrNotExist}
		}
		return nil, err
	}
	return data, nil
}

// read 是统一读取路径：散文件 > 高层 > 低层；命中时报告所在层。
func (s *Store) read(name string) (data []byte, layer string, err error) {
	norm := Normalize(name)
	if norm == "" || isInternalName(norm) {
		return nil, "", fs.ErrNotExist
	}
	if !s.noLoose {
		if local, ok := s.localFileOf(norm); ok {
			data, err := os.ReadFile(local)
			if err == nil {
				return data, "local", nil
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, "", err
			}
		}
	}
	key := strings.ToLower(norm)
	set := s.set.Load()
	for i := len(set.layers) - 1; i >= 0; i-- {
		l := set.layers[i]
		if l.deleted[key] {
			break
		}
		if l.archive.HasFile(norm) {
			data, err := l.archive.ReadFile(norm)
			return data, l.label, err
		}
	}
	return nil, "", fs.ErrNotExist
}

// localFileOf 返回名字对应的散文件路径；文件不存在、本视图不含散文件层
// 或名字越界（路径穿越）时 ok=false。
func (s *Store) localFileOf(normalized string) (string, bool) {
	if s.noLoose {
		return "", false
	}
	candidate, ok := s.hostPathOf(normalized)
	if !ok {
		return "", false
	}
	if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
		return candidate, true
	}
	return "", false
}

// hostPathOf 把规范化名字换算成数据根下的主机路径；越界时 ok=false。
func (s *Store) hostPathOf(normalized string) (string, bool) {
	normalized = Normalize(normalized)
	if normalized == "" || normalized == "." {
		return "", false
	}
	candidate := filepath.Join(s.dir, filepath.FromSlash(strings.ReplaceAll(normalized, `\`, "/")))
	// 拒绝 .. / 绝对路径穿越。
	rel, err := filepath.Rel(s.dir, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return candidate, true
}

// List 返回合并视图的全量清单：高层覆盖低层，散文件覆盖一切，被登记
// 删除的名字不在其中。按名字（不分大小写）升序。
func (s *Store) List() ([]ResFile, error) {
	idx := s.index()
	out := make([]ResFile, 0, len(idx.keys))
	if !s.noLoose {
		if err := s.appendLocal(&out); err != nil {
			return nil, err
		}
	}
	local := make(map[string]bool, len(out))
	for _, f := range out {
		local[NormalizeLower(f.Name)] = true
	}
	for i, key := range idx.keys {
		if local[key] {
			continue
		}
		out = append(out, ResFile{Name: idx.names[i], Size: idx.sizes[i], Layer: idx.layers[i]})
	}
	sort.Slice(out, func(i, j int) bool {
		ki, kj := NormalizeLower(out[i].Name), NormalizeLower(out[j].Name)
		if ki != kj {
			return ki < kj
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// appendLocal 把数据根下的散文件追加进清单；层容器不是资源。
// 不可读的子目录跳过而不拖垮整份清单。
func (s *Store) appendLocal(out *[]ResFile) error {
	return filepath.WalkDir(s.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == s.dir {
				return fmt.Errorf("res: 散文件枚举失败：%s（%w）", s.dir, err)
			}
			return nil
		}
		if d.IsDir() || isLayerFile(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(s.dir, path)
		if err != nil {
			return nil
		}
		size := int64(0)
		if info, err := d.Info(); err == nil {
			size = info.Size()
		}
		*out = append(*out, ResFile{
			Name:  Normalize(filepath.ToSlash(rel)),
			Size:  size,
			Layer: "local",
		})
		return nil
	})
}

// Layers 返回各层概览（层号升序）。
func (s *Store) Layers() []LayerInfo {
	set := s.set.Load()
	out := make([]LayerInfo, 0, len(set.layers))
	for _, l := range set.layers {
		info := LayerInfo{
			Number:   l.number,
			Name:     l.label,
			File:     l.file,
			Path:     filepath.Join(s.dir, l.file),
			Entries:  len(l.archive.Entries()),
			Deleted:  len(l.deleted),
			Warnings: l.archive.Warnings(),
		}
		if st, err := os.Stat(info.Path); err == nil {
			info.Size, info.ModTime = st.Size(), st.ModTime()
		}
		out = append(out, info)
	}
	return out
}

// Warnings 返回打开各层时的告警之和（如 v4 MD5 不匹配）。
func (s *Store) Warnings() []string {
	var out []string
	for _, l := range s.set.Load().layers {
		out = append(out, l.archive.Warnings()...)
	}
	return out
}
