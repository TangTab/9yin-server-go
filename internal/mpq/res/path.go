package res

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 路径式访问入口：os.* / filepath.* 的等价替换，让"数据根下的相对路径"
// 在纯散文件与打包后（bNNN.rep / dataNNN.rep）两种布局下都能用。
//
// 规则：路径落在数据根之下——自身或任一级祖先目录里有层文件——即经
// 资源仓读取（散文件 > 高层 > 低层）；否则直读文件系统，行为与 os.*
// 一致。调用方不需要知道数据根在哪、有没有打包。
//
//	ReadFile(path)       ≈ os.ReadFile
//	Stat / Lstat(path)   ≈ os.Stat / os.Lstat
//	ReadDir(dir)         ≈ os.ReadDir
//	WalkDir(root, fn)    ≈ filepath.WalkDir
//	Glob(pattern)        ≈ filepath.Glob
//
// 散文件实时生效；层文件的内容要 ResetPathCache（重新打开）之后才更新。
// 同一数据根在进程内只打开一次，带读缓存。

// 目录判定与资源仓的进程内缓存。命中（是数据根）永久有效；未命中与
// 打开失败只短暂有效——层文件可能在进程运行期间才落盘。
const mountScanTTL = time.Second

type mountVerdict struct {
	root string // "" = 该目录及其祖先都不是数据根
	at   time.Time
}

type mountFailure struct {
	err error
	at  time.Time
}

type mount struct {
	store *CachedStore
	fsys  *StoreFS
}

var mountMu sync.Mutex

var (
	mountVerdicts = map[string]mountVerdict{} // 绝对目录 → 数据根判定
	mounts        = map[string]*mount{}       // 绝对数据根 → 已打开的资源仓
	mountFailures = map[string]mountFailure{}
)

// ResetPathCache 关闭全部路径式资源仓并清空目录判定缓存。层文件在本进程
// 运行期间被构建、替换或删除之后调用（BuildLayer / Delete 已内置）；测试
// 里也用它释放临时目录上的文件句柄（Windows 上占用着就无法清理）。
// 调用期间不得有其他 goroutine 在用这些入口。
func ResetPathCache() {
	mountMu.Lock()
	defer mountMu.Unlock()
	for _, m := range mounts {
		m.store.Close()
	}
	mountVerdicts = map[string]mountVerdict{}
	mounts = map[string]*mount{}
	mountFailures = map[string]mountFailure{}
}

// hasLayerFile 报告目录自身是否有层文件。
func hasLayerFile(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && isLayerFile(e.Name()) {
			return true
		}
	}
	return false
}

// findRoot 自 dir 起逐级向上找最近的数据根；不在任何数据根之下返回空串。
func findRoot(dir string) string {
	mountMu.Lock()
	defer mountMu.Unlock()
	now := time.Now()
	walked := []string{dir}
	for {
		if v, ok := mountVerdicts[dir]; ok {
			if v.root != "" || now.Sub(v.at) < mountScanTTL {
				// 沿途各目录沿用这次判定的扫描时刻，不滑动续期。
				for _, d := range walked {
					mountVerdicts[d] = mountVerdict{root: v.root, at: v.at}
				}
				return v.root
			}
		}
		if hasLayerFile(dir) {
			for _, d := range walked {
				mountVerdicts[d] = mountVerdict{root: dir, at: now}
			}
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			for _, d := range walked {
				mountVerdicts[d] = mountVerdict{at: now}
			}
			return ""
		}
		dir = parent
		walked = append(walked, dir)
	}
}

// openMount 返回数据根的进程内共享资源仓。
func openMount(root string) (*mount, error) {
	mountMu.Lock()
	defer mountMu.Unlock()
	if m := mounts[root]; m != nil {
		return m, nil
	}
	now := time.Now()
	if f, ok := mountFailures[root]; ok && now.Sub(f.at) < mountScanTTL {
		return nil, f.err
	}
	store, err := Open(root)
	if err != nil {
		mountFailures[root] = mountFailure{err: err, at: now}
		return nil, err
	}
	m := &mount{store: WrapCached(store, CacheOptions{})}
	m.fsys = m.store.FS()
	mounts[root] = m
	delete(mountFailures, root)
	return m, nil
}

// resolve 把主机路径换算成资源仓与仓内 fs 名字（/ 分隔，根为 "."）。
// 路径不在数据根之下时 m 为 nil（调用方直读文件系统）；数据根打不开
// 时返回错误。
func resolve(path string, asDir bool) (m *mount, name string, err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}
	abs = filepath.Clean(abs)
	start := abs
	if !asDir {
		start = filepath.Dir(abs)
	}
	root := findRoot(start)
	if root == "" {
		return nil, "", nil
	}
	if m, err = openMount(root); err != nil {
		return nil, "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return nil, "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, "", nil
	}
	return m, filepath.ToSlash(rel), nil
}

// -- 访问入口 ------------------------------------------------------------------

// ReadFile 读取整个文件；数据根之外的路径即 os.ReadFile。返回的切片归
// 调用方所有（资源仓读缓存里的内容不会被改写）。
func ReadFile(path string) ([]byte, error) {
	m, name, err := resolve(path, false)
	if err != nil {
		return nil, hostErr("open", path, err)
	}
	if m == nil {
		return os.ReadFile(path)
	}
	data, err := m.store.Read(name)
	if err != nil {
		return nil, hostErr("open", path, err)
	}
	return append([]byte{}, data...), nil
}

// ReadFileShared 同 ReadFile，但返回的切片可能是资源仓读缓存的内部对象：
// 调用方只能读、不能改。适合对同一大文件反复做小片段访问的场合（如按
// 偏移索引取属性行）——免掉每次整文件拷贝。数据根之外的路径即 os.ReadFile。
func ReadFileShared(path string) ([]byte, error) {
	m, name, err := resolve(path, false)
	if err != nil {
		return nil, hostErr("open", path, err)
	}
	if m == nil {
		return os.ReadFile(path)
	}
	data, err := m.store.Read(name)
	if err != nil {
		return nil, hostErr("open", path, err)
	}
	return data, nil
}

// Stat 取文件 / 目录信息；数据根之外的路径即 os.Stat。
func Stat(path string) (fs.FileInfo, error) { return statPath(path) }

// Lstat 同 Stat（资源仓内不区分符号链接）；数据根之外即 os.Lstat。
func Lstat(path string) (fs.FileInfo, error) {
	m, _, err := resolve(path, false)
	if err != nil {
		return nil, hostErr("stat", path, err)
	}
	if m == nil {
		return os.Lstat(path)
	}
	return statPath(path)
}

func statPath(path string) (fs.FileInfo, error) {
	m, name, err := resolve(path, false)
	if err != nil {
		return nil, hostErr("stat", path, err)
	}
	if m == nil {
		return os.Stat(path)
	}
	info, err := m.fsys.Stat(name)
	if err != nil {
		return nil, hostErr("stat", path, err)
	}
	return info, nil
}

// ReadDir 枚举目录的直接子项（按名字排序）；数据根之外即 os.ReadDir。
func ReadDir(dir string) ([]fs.DirEntry, error) {
	m, name, err := resolve(dir, true)
	if err != nil {
		return nil, hostErr("open", dir, err)
	}
	if m == nil {
		return os.ReadDir(dir)
	}
	entries, err := m.fsys.ReadDir(name)
	if err != nil {
		return nil, hostErr(readdirOp(err), dir, err)
	}
	return entries, nil
}

func readdirOp(err error) string {
	if errors.Is(err, errNotDir) {
		return "readdirent"
	}
	return "open"
}

// WalkDir 等价 filepath.WalkDir：树来自资源仓合并视图（散文件 > 高层 >
// 低层），回调收到主机风格路径；数据根之外即 filepath.WalkDir。
func WalkDir(root string, fn fs.WalkDirFunc) error {
	var err error
	m, name, resolveErr := resolve(root, true)
	switch {
	case resolveErr != nil:
		err = fn(root, nil, hostErr("lstat", root, resolveErr))
	case m == nil:
		return filepath.WalkDir(root, fn)
	default:
		var info fs.FileInfo
		if info, err = m.fsys.Stat(name); err != nil {
			err = fn(root, nil, hostErr("lstat", root, err))
			break
		}
		err = walkOne(m, root, name, fs.FileInfoToDirEntry(namedInfo{info, filepath.Base(root)}), fn)
	}
	if err == fs.SkipDir || err == fs.SkipAll {
		return nil
	}
	return err
}

// namedInfo 用给定名字包一层 FileInfo：根条目的名字取主机路径的末段，
// 与 filepath.WalkDir 一致。
type namedInfo struct {
	fs.FileInfo
	name string
}

func (i namedInfo) Name() string { return i.name }

// walkOne 深度优先遍历合并视图；SkipDir / SkipAll 语义同 filepath.WalkDir。
func walkOne(m *mount, host, name string, d fs.DirEntry, fn fs.WalkDirFunc) error {
	if err := fn(host, d, nil); err != nil || !d.IsDir() {
		if err == fs.SkipDir && d.IsDir() {
			err = nil
		}
		return err
	}
	entries, err := m.fsys.ReadDir(name)
	if err != nil {
		if err := fn(host, d, hostErr(readdirOp(err), host, err)); err != nil {
			if err == fs.SkipDir && d.IsDir() {
				return nil
			}
			return err
		}
	}
	for _, e := range entries {
		host1, name1 := filepath.Join(host, e.Name()), path.Join(name, e.Name())
		if err := walkOne(m, host1, name1, e, fn); err != nil {
			if err == fs.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

// Glob 等价 filepath.Glob（模式语法与大小写敏感性相同）：目录枚举走
// 资源仓合并视图。目录不可读时跳过，同 filepath.Glob。
func Glob(pattern string) ([]string, error) {
	if !globHasMeta(pattern) {
		if _, err := Lstat(pattern); err != nil {
			return nil, nil
		}
		return []string{pattern}, nil
	}
	dir, file := globSplit(pattern)
	dir = globCleanDir(dir)
	if !globHasMeta(dir) {
		return globInDir(dir, file, nil)
	}
	dirs, err := Glob(dir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, d := range dirs {
		if matches, err = globInDir(d, file, matches); err != nil {
			return nil, err
		}
	}
	return matches, nil
}

func globHasMeta(p string) bool { return strings.ContainsAny(p, "*?[") }

// globSplit 在最后一个路径分隔符处切开模式；没有分隔符时目录为空串。
func globSplit(pattern string) (dir, file string) {
	if i := strings.LastIndexAny(pattern, "/\\"); i >= 0 {
		return pattern[:i+1], pattern[i+1:]
	}
	return "", pattern
}

// globCleanDir 去掉目录部分的尾分隔符（卷根保留）。不去掉的话，带通配
// 的目录部分会在自身上无限递归。
func globCleanDir(dir string) string {
	rest := dir[len(filepath.VolumeName(dir)):]
	if strings.Trim(rest, "/\\") == "" {
		return dir
	}
	return strings.TrimRight(dir, "/\\")
}

func globInDir(dir, pattern string, matches []string) ([]string, error) {
	entries, err := ReadDir(dir)
	if err != nil {
		return matches, nil
	}
	for _, e := range entries {
		ok, err := filepath.Match(pattern, e.Name())
		if err != nil {
			return nil, err
		}
		if ok {
			matches = append(matches, filepath.Join(dir, e.Name()))
		}
	}
	return matches, nil
}

// hostErr 把资源仓错误包成 os 风格的 *fs.PathError（带主机路径）；不存在
// 时 Err 归一为 fs.ErrNotExist，errors.Is(err, fs.ErrNotExist) 与
// os.IsNotExist(err) 照常可用。
func hostErr(op, hostPath string, err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		err = pe.Err
	}
	if errors.Is(err, fs.ErrNotExist) {
		err = fs.ErrNotExist
	}
	return &fs.PathError{Op: op, Path: hostPath, Err: err}
}
