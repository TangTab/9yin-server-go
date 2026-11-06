package res

import (
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

// StoreFS 把 CachedStore 的三层合并视图适配成 fs.FS：
//
//	fsys := store.FS()
//	data, _ := fs.ReadFile(fsys, "resources/modern/share/trade/shop.ini")
//	fs.WalkDir(fsys, "resources/modern/share/npc/npcconfig", walk)
//
// 名字用 / 分隔、大小写不敏感（内部规范化）。目录视图来自合并
// 清单的名称索引：某名字只要存在以其为前缀的更长名字即视为
// 目录。索引按 CachedStore 的重载代次自动失效重建；Reload 之外
// 新落盘的本地文件不会自动出现。
type StoreFS struct {
	store *CachedStore

	mu   sync.Mutex
	gen  int64    // 建索引时的代次
	keys []string // 规范化小写、反斜杠分隔、排序后的全量名字
}

// FS 返回该资源仓的 fs.FS 视图；同一 StoreFS 实例共享索引。
func (c *CachedStore) FS() *StoreFS {
	return &StoreFS{store: c}
}

// index 返回名称索引（必要时重建）。
func (s *StoreFS) index() ([]string, error) {
	gen := s.store.Generation()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys != nil && s.gen == gen {
		return s.keys, nil
	}
	files, err := s.store.List()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(files))
	for _, f := range files {
		keys = append(keys, NormalizeLower(f.Name))
	}
	sort.Strings(keys)
	s.keys, s.gen = keys, gen
	return keys, nil
}

// storeName 将 fs 名字（/ 分隔）转为仓内名字（\ 分隔，已小写）。
func storeName(fsName string) string {
	return strings.ToLower(strings.ReplaceAll(strings.Trim(fsName, "/"), "/", `\`))
}

// lookup 在索引中定位名字：文件须确切命中，目录看前缀。
func (s *StoreFS) lookup(fsName string) (isFile, isDir bool, err error) {
	if !fs.ValidPath(fsName) {
		return false, false, fs.ErrInvalid
	}
	if fsName == "." {
		return false, true, nil
	}
	keys, err := s.index()
	if err != nil {
		return false, false, err
	}
	key := storeName(fsName)
	i := sort.SearchStrings(keys, key)
	if i < len(keys) && keys[i] == key {
		return true, false, nil
	}
	prefix := key + `\`
	if j := sort.SearchStrings(keys, prefix); j < len(keys) && strings.HasPrefix(keys[j], prefix) {
		return false, true, nil
	}
	return false, false, fs.ErrNotExist
}

// -- 文件 / 目录信息合成 ------------------------------------------------------

type storeFSFile struct {
	name string // 仓内名字（\ 分隔）
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

// storeFileInfo 是合成 FileInfo：目录的 size/时间戳无意义，取零值。
type storeFileInfo struct {
	name string
	size int64
	dir  bool
}

func (i storeFileInfo) Name() string       { return i.name }
func (i storeFileInfo) Size() int64        { return i.size }
func (i storeFileInfo) Mode() fs.FileMode  { return 0444 }
func (i storeFileInfo) ModTime() time.Time { return time.Time{} }
func (i storeFileInfo) IsDir() bool        { return i.dir }
func (i storeFileInfo) Sys() any           { return nil }

// storeDirEntry 是合成 DirEntry；size 未知时为 0，需要时走 Info。
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
	return &storeFSFile{name: storeName(fsName), data: data}, nil
}

// ReadFile 实现 fs.ReadFileFS。
func (s *StoreFS) ReadFile(fsName string) ([]byte, error) {
	isFile, _, err := s.lookup(fsName)
	if err != nil {
		return nil, opErr("read", fsName, err)
	}
	if !isFile {
		return nil, opErr("read", fsName, fs.ErrInvalid)
	}
	data, err := s.store.Read(fsName)
	if err != nil {
		return nil, opErr("read", fsName, err)
	}
	return data, nil
}

// Stat 实现 fs.StatFS。
func (s *StoreFS) Stat(fsName string) (fs.FileInfo, error) {
	_, isDir, err := s.lookup(fsName)
	if err != nil {
		return nil, opErr("stat", fsName, err)
	}
	if isDir {
		return storeFileInfo{name: path.Base(fsName), dir: true}, nil
	}
	data, err := s.store.Read(fsName)
	if err != nil {
		return nil, opErr("stat", fsName, err)
	}
	return storeFileInfo{name: path.Base(fsName), size: int64(len(data))}, nil
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

// ReadDir 实现 fs.ReadDirFS：合并清单中该目录的直接子项（按名排序）。
func (s *StoreFS) ReadDir(fsName string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(fsName) {
		return nil, opErr("readdir", fsName, fs.ErrInvalid)
	}
	keys, err := s.index()
	if err != nil {
		return nil, err
	}
	prefix := ""
	if fsName != "." {
		prefix = storeName(fsName) + `\`
	}
	seen := map[string]storeDirEntry{}
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := key[len(prefix):]
		if rest == "" {
			continue
		}
		if back := strings.IndexByte(rest, '\\'); back >= 0 {
			// 中间段是子目录；只登记目录名，不下钻。
			name := rest[:back]
			if e, ok := seen[name]; !ok || !e.dir {
				seen[name] = storeDirEntry{name: name, dir: true}
			}
			continue
		}
		seen[rest] = storeDirEntry{name: rest}
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
