package res

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq"
)

// fsIndex 是来自层文件的存活条目的名称索引：高层覆盖低层，被更高层
// 登记删除的名字不在其中。散文件不进索引——它们在枚举时实时合并。
//
// 挂在 Store 上、按层集合换出自动失效（Reload 之外层文件不会变）。
type fsIndex struct {
	set    *layerSet
	keys   []string // 规范化小写、反斜杠分隔、升序
	names  []string // 与 keys 平行：原样大小写的名字
	sizes  []int64
	layers []string
	dirs   map[string]string // 目录键（小写全名）→ 原样大小写的目录名
}

func buildIndex(set *layerSet) *fsIndex {
	idx := &fsIndex{set: set, dirs: map[string]string{}}
	live := map[string]bool{}
	dead := map[string]bool{}
	for i := len(set.layers) - 1; i >= 0; i-- {
		l := set.layers[i]
		for _, e := range l.archive.Entries() {
			key := NormalizeLower(e.Name)
			if e.Name == "" || !e.Exists() || isInternalName(e.Name) || live[key] || dead[key] {
				continue
			}
			live[key] = true
			idx.keys = append(idx.keys, key)
			idx.names = append(idx.names, e.Name)
			idx.sizes = append(idx.sizes, int64(e.Size))
			idx.layers = append(idx.layers, l.label)
		}
		// 本层的删除名单只对更低的层生效，收完本层条目后再并入。
		for key := range l.deleted {
			dead[key] = true
		}
	}

	order := make([]int, len(idx.keys))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool { return idx.keys[order[a]] < idx.keys[order[b]] })
	keys := make([]string, len(order))
	names := make([]string, len(order))
	sizes := make([]int64, len(order))
	layers := make([]string, len(order))
	for i, o := range order {
		keys[i], names[i], sizes[i], layers[i] = idx.keys[o], idx.names[o], idx.sizes[o], idx.layers[o]
	}
	idx.keys, idx.names, idx.sizes, idx.layers = keys, names, sizes, layers

	for _, name := range names {
		parts := strings.Split(name, `\`)
		var key, display string
		for _, p := range parts[:len(parts)-1] {
			if key == "" {
				key, display = strings.ToLower(p), p
			} else {
				key, display = key+`\`+strings.ToLower(p), display+`\`+p
			}
			if _, ok := idx.dirs[key]; !ok {
				idx.dirs[key] = display
			}
		}
	}
	return idx
}

// isInternalName 报告名字是否为归档内部文件（不进合并视图）。
func isInternalName(name string) bool {
	if strings.EqualFold(name, deletedFile) {
		return true
	}
	for _, f := range mpq.SpecialFiles {
		if strings.EqualFold(name, f) {
			return true
		}
	}
	return false
}

// StoreFS 把资源仓的合并视图适配成 fs.FS：
//
//	fsys := store.FS()
//	data, _ := fs.ReadFile(fsys, "ini/Scene/born03/npc_creator.xml")
//	fs.WalkDir(fsys, "ini/Npc/NpcConfig", walk)
//
// 名字用 / 分隔、大小写不敏感（内部规范化），相对数据根。条目名保留
// 归档内的原样大小写；目录视图由名称索引合成。散文件与散目录在每次
// 访问时实时合并（新落盘的散文件立即可见）；层文件的内容要 Reload
// 之后才更新。
type StoreFS struct {
	store *Store
}

// FS 返回该资源仓的 fs.FS 视图。
func (s *Store) FS() *StoreFS { return &StoreFS{store: s} }

// FS 返回底层资源仓的 fs.FS 视图。
func (c *CachedStore) FS() *StoreFS { return &StoreFS{store: c.store} }

// index 返回底层资源仓的名称索引。
func (s *StoreFS) index() *fsIndex { return s.store.index() }

// storeName 将 fs 名字（/ 或 \ 分隔）转为索引键（\ 分隔、小写）。
func storeName(fsName string) string {
	return strings.ToLower(strings.ReplaceAll(strings.Trim(fsName, "/"), "/", `\`))
}

// lookup 定位名字：文件须确切命中，目录看前缀。散文件 / 散目录优先。
func (s *StoreFS) lookup(fsName string) (isFile, isDir bool, err error) {
	if !fs.ValidPath(fsName) {
		return false, false, fs.ErrInvalid
	}
	if fsName == "." {
		return false, true, nil
	}
	if st, ok := s.localStat(fsName); ok {
		return !st.IsDir(), st.IsDir(), nil
	}
	x := s.index()
	key := storeName(fsName)
	if i := sort.SearchStrings(x.keys, key); i < len(x.keys) && x.keys[i] == key {
		return true, false, nil
	}
	prefix := key + `\`
	if j := sort.SearchStrings(x.keys, prefix); j < len(x.keys) && strings.HasPrefix(x.keys[j], prefix) {
		return false, true, nil
	}
	return false, false, fs.ErrNotExist
}

// localStat 返回名字对应散文件 / 散目录的真实信息；磁盘上没有时 ok=false。
func (s *StoreFS) localStat(fsName string) (os.FileInfo, bool) {
	if s.store.noLoose {
		return nil, false
	}
	host, ok := s.store.hostPathOf(fsName)
	if !ok {
		return nil, false
	}
	st, err := os.Stat(host)
	if err != nil {
		return nil, false
	}
	return st, true
}

// -- 文件 / 目录信息合成 ------------------------------------------------------

type storeFSFile struct {
	name string
	data []byte
	off  int64
}

func (f *storeFSFile) Stat() (fs.FileInfo, error) {
	return storeFileInfo{name: baseName(f.name), size: int64(len(f.data))}, nil
}

func (f *storeFSFile) Read(p []byte) (int, error) {
	if f.off >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.off:])
	f.off += int64(n)
	return n, nil
}

func (f *storeFSFile) Close() error { return nil }

// storeFileInfo 是合成 FileInfo：层内文件的时间戳无意义，取零值。
type storeFileInfo struct {
	name string
	size int64
	dir  bool
}

func (i storeFileInfo) Name() string { return i.name }
func (i storeFileInfo) Size() int64  { return i.size }
func (i storeFileInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}
func (i storeFileInfo) ModTime() time.Time { return time.Time{} }
func (i storeFileInfo) IsDir() bool        { return i.dir }
func (i storeFileInfo) Sys() any           { return nil }

// storeDirEntry 是合成 DirEntry；大小取自名称索引。
type storeDirEntry struct {
	name string
	dir  bool
	size int64
}

func (e storeDirEntry) Name() string { return e.name }
func (e storeDirEntry) IsDir() bool  { return e.dir }

func (e storeDirEntry) Type() fs.FileMode {
	if e.dir {
		return fs.ModeDir
	}
	return 0
}

func (e storeDirEntry) Info() (fs.FileInfo, error) {
	return storeFileInfo{name: e.name, size: e.size, dir: e.dir}, nil
}

// baseName 取名字最后一段（兼容 \ 与 /）。
func baseName(name string) string {
	if i := strings.LastIndexAny(name, `\/`); i >= 0 {
		return name[i+1:]
	}
	return name
}

// -- fs.FS / fs.ReadFileFS / fs.StatFS / fs.ReadDirFS ------------------------

// Open 实现 fs.FS；目录返回可 ReadDir 的句柄。
func (s *StoreFS) Open(fsName string) (fs.File, error) {
	_, isDir, err := s.lookup(fsName)
	if err != nil {
		return nil, opErr("open", fsName, err)
	}
	if isDir {
		return &storeDirFile{fsys: s, dir: fsName}, nil
	}
	data, err := s.store.Read(fsName)
	if err != nil {
		return nil, opErr("open", fsName, err)
	}
	return &storeFSFile{name: Normalize(fsName), data: data}, nil
}

// ReadFile 实现 fs.ReadFileFS。
func (s *StoreFS) ReadFile(fsName string) ([]byte, error) {
	isFile, _, err := s.lookup(fsName)
	if err != nil {
		return nil, opErr("open", fsName, err)
	}
	if !isFile {
		return nil, opErr("read", fsName, fs.ErrInvalid)
	}
	data, err := s.store.Read(fsName)
	if err != nil {
		return nil, opErr("open", fsName, err)
	}
	return data, nil
}

// Stat 实现 fs.StatFS（只查名称索引与磁盘，不读文件内容）。
func (s *StoreFS) Stat(fsName string) (fs.FileInfo, error) {
	if !fs.ValidPath(fsName) {
		return nil, opErr("stat", fsName, fs.ErrInvalid)
	}
	if fsName == "." {
		return storeFileInfo{name: baseName(s.store.dir), dir: true}, nil
	}
	if st, ok := s.localStat(fsName); ok {
		return st, nil
	}
	_, isDir, err := s.lookup(fsName)
	if err != nil {
		return nil, opErr("stat", fsName, err)
	}
	x := s.index()
	key := storeName(fsName)
	if isDir {
		return storeFileInfo{name: baseName(x.dirs[key]), dir: true}, nil
	}
	i := sort.SearchStrings(x.keys, key)
	if i >= len(x.keys) || x.keys[i] != key {
		return nil, opErr("stat", fsName, fs.ErrNotExist)
	}
	return storeFileInfo{name: baseName(x.names[i]), size: x.sizes[i]}, nil
}

type storeDirFile struct {
	fsys    *StoreFS
	dir     string
	entries []fs.DirEntry
	read    bool
}

func (f *storeDirFile) Stat() (fs.FileInfo, error) {
	return storeFileInfo{name: path.Base(f.dir), dir: true}, nil
}

func (f *storeDirFile) Read([]byte) (int, error) { return 0, fs.ErrInvalid }

func (f *storeDirFile) Close() error { return nil }

func (f *storeDirFile) ReadDir(int) ([]fs.DirEntry, error) {
	if f.read {
		return nil, io.EOF
	}
	f.read = true
	return f.fsys.ReadDir(f.dir)
}

// ReadDir 实现 fs.ReadDirFS：名称索引中该目录的直接子项，实时并入磁盘上
// 的散文件 / 散目录（同名以散文件为准），按名字排序（同 os.ReadDir）。
func (s *StoreFS) ReadDir(fsName string) ([]fs.DirEntry, error) {
	isFile, _, err := s.lookup(fsName)
	if err != nil {
		return nil, opErr("open", fsName, err)
	}
	if isFile {
		return nil, opErr("readdirent", fsName, errNotDir)
	}

	x := s.index()
	prefix := ""
	if fsName != "." {
		prefix = storeName(fsName) + `\`
	}
	seen := map[string]fs.DirEntry{}
	lo := sort.SearchStrings(x.keys, prefix)
	for i := lo; i < len(x.keys) && strings.HasPrefix(x.keys[i], prefix); i++ {
		rest := x.keys[i][len(prefix):]
		if rest == "" {
			continue
		}
		if b := strings.IndexByte(rest, '\\'); b >= 0 {
			// 中间段是子目录：只登记目录名，不下钻。
			if _, ok := seen[rest[:b]]; !ok {
				seen[rest[:b]] = storeDirEntry{name: baseName(x.dirs[prefix+rest[:b]]), dir: true}
			}
			continue
		}
		seen[rest] = storeDirEntry{name: baseName(x.names[i]), size: x.sizes[i]}
	}

	if !s.store.noLoose {
		host := s.store.dir
		if fsName != "." {
			host, _ = s.store.hostPathOf(fsName)
		}
		if disk, err := os.ReadDir(host); err == nil {
			for _, e := range disk {
				if isLayerFile(e.Name()) {
					continue
				}
				seen[strings.ToLower(e.Name())] = e
			}
		}
	}

	entries := make([]fs.DirEntry, 0, len(seen))
	for _, e := range seen {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func opErr(op, name string, err error) error {
	return &fs.PathError{Op: op, Path: name, Err: err}
}

// errNotDir 在对文件做目录枚举时返回。不用 syscall.ENOTDIR：Windows 上
// 它是 ENOENT 的别名，会把"不是目录"误判成"不存在"。
var errNotDir = errors.New("not a directory")
