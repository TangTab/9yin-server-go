package main

// 资源读取统一入口：把散落的 os.Open / os.ReadFile / 目录枚举
// 收敛到 MPQ 三层资源仓（本地散文件 > res-patch.rep > res-base.rep）
// 之上，并带读缓存——同一文件只解压 / 读盘一次。
//
// 规则：
//   - 路径位于 <工程根>\resources\… 之下 → 经资源仓读取（可命中
//     res-patch.rep / res-base.rep，无需本地解包）；
//   - 资源仓不可用（缺 res-base.rep，或 NINEYIN_RESOURCE_STORE=off）
//     或路径在资源根之外（测试夹具、data/、自定义 share 根等）
//     → 回退直接文件系统。
//
// 环境变量：
//   - NINEYIN_RESOURCE_STORE=off      强制禁用资源仓（排查问题用）
//   - NINEYIN_RESOURCE_CACHE_MB=<n>   字节缓存上限（MiB，缺省 256）

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/mpq/res"
)

type resourceAccessor struct {
	root  string
	store *res.CachedStore // nil = 纯 os 回退模式
	fsys  *res.StoreFS
}

var (
	resourceOnce   sync.Once
	resourceAccess *resourceAccessor
)

// resourceAccessorOf 惰性打开全局资源仓（进程内一次）。
func resourceAccessorOf() *resourceAccessor {
	resourceOnce.Do(func() {
		resourceAccess = openResourceAccessor()
	})
	return resourceAccess
}

func openResourceAccessor() *resourceAccessor {
	access := &resourceAccessor{root: locateRuntimeProjectRoot()}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("NINEYIN_RESOURCE_STORE")), "off") {
		log.Printf("resource store disabled by NINEYIN_RESOURCE_STORE=off; falling back to direct file access")
		return access
	}
	cacheOpt := res.CacheOptions{}
	if raw := strings.TrimSpace(os.Getenv("NINEYIN_RESOURCE_CACHE_MB")); raw != "" {
		if mb, err := strconv.ParseInt(raw, 10, 64); err == nil && mb > 0 {
			cacheOpt.MaxBytes = mb << 20
		} else {
			log.Printf("invalid NINEYIN_RESOURCE_CACHE_MB %q ignored", raw)
		}
	}
	store, err := res.OpenCached(access.root, cacheOpt)
	if err != nil {
		// 资源仓缺层（如精简部署 / CI）不致命：整体回退 os。
		log.Printf("resource store unavailable (%v); reading resources via direct file access", err)
		return access
	}
	access.store = store
	access.fsys = store.FS()
	files, listErr := store.List()
	summary := "未知"
	if listErr == nil {
		summary = fmt.Sprintf("%d", len(files))
	}
	patch := "无"
	if store.HasPatch() {
		patch = fmt.Sprintf("%d 条目（版本 %s）", store.PatchCount(), store.PatchVersion())
	}
	log.Printf("resource store ready root=%s base=%d条目 patch=%s merged=%s cache<=%dMiB", access.root, store.BaseCount(), patch, summary, cacheLimitMB(cacheOpt))
	return access
}

func cacheLimitMB(opt res.CacheOptions) int64 {
	if opt.MaxBytes > 0 {
		return opt.MaxBytes >> 20
	}
	return res.DefaultCacheMaxBytes >> 20
}

// resourceStoreName 把主机路径换成仓内 fs 名（resources/…，/ 分隔）；
// 不在资源根之下时 ok=false。
func (a *resourceAccessor) resourceStoreName(path string) (string, bool) {
	if a.store == nil {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(a.root, abs)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "/") {
		return "", false
	}
	if !strings.HasPrefix(strings.ToLower(rel), "resources/") {
		return "", false
	}
	return rel, true
}

// readResourceBytes 读整个资源文件；回退 os.ReadFile。
func readResourceBytes(path string) ([]byte, error) {
	a := resourceAccessorOf()
	if name, ok := a.resourceStoreName(path); ok {
		return a.store.Read(name)
	}
	return os.ReadFile(path)
}

// openResourceReader 打开资源文件为流（扫描器逐行消费）；回退 os.Open。
func openResourceReader(path string) (io.ReadCloser, error) {
	a := resourceAccessorOf()
	if name, ok := a.resourceStoreName(path); ok {
		data, err := a.store.Read(name)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return os.Open(path)
}

// statResourcePath 取文件/目录信息；缺失时错误满足 os.ErrNotExist。
func statResourcePath(path string) (os.FileInfo, error) {
	a := resourceAccessorOf()
	if name, ok := a.resourceStoreName(path); ok {
		return a.fsys.Stat(name)
	}
	return os.Stat(path)
}

// readResourceDirEntries 枚举目录直接子项；回退 os.ReadDir。
func readResourceDirEntries(dir string) ([]os.DirEntry, error) {
	a := resourceAccessorOf()
	if name, ok := a.resourceStoreName(dir); ok {
		entries, err := a.fsys.ReadDir(name)
		if err != nil {
			return nil, err
		}
		out := make([]os.DirEntry, len(entries))
		for i, e := range entries {
			out[i] = &virtualDirEntry{name: e.Name(), dir: e.IsDir()}
		}
		return out, nil
	}
	return os.ReadDir(dir)
}

// walkResourceTree 等价 filepath.WalkDir，但树来自资源仓合并视图
// （本地 > patch > base）；回调收到的是主机风格路径。回退 filepath.WalkDir。
func walkResourceTree(root string, fn fs.WalkDirFunc) error {
	a := resourceAccessorOf()
	if name, ok := a.resourceStoreName(root); ok {
		info, err := a.fsys.Stat(name)
		if err != nil {
			return fn(root, nil, err)
		}
		if !info.IsDir() {
			return fn(root, &virtualDirEntry{name: info.Name()}, nil)
		}
		return walkResourceChildren(a, root, fn)
	}
	return filepath.WalkDir(root, fn)
}

func walkResourceChildren(a *resourceAccessor, dir string, fn fs.WalkDirFunc) error {
	name, _ := a.resourceStoreName(dir)
	entries, err := a.fsys.ReadDir(name)
	if err != nil {
		return fn(dir, nil, err)
	}
	for _, entry := range entries {
		hostPath := filepath.Join(dir, entry.Name())
		virtual := &virtualDirEntry{name: entry.Name(), dir: entry.IsDir()}
		if err := fn(hostPath, virtual, nil); err != nil {
			if errors.Is(err, filepath.SkipDir) {
				continue
			}
			return err
		}
		if entry.IsDir() {
			if err := walkResourceChildren(a, hostPath, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

// virtualDirEntry 让资源仓合成的枚举结果满足 os.DirEntry。
type virtualDirEntry struct {
	name string
	dir  bool
}

func (e *virtualDirEntry) Name() string { return e.name }
func (e *virtualDirEntry) IsDir() bool  { return e.dir }

func (e *virtualDirEntry) Type() fs.FileMode {
	if e.dir {
		return fs.ModeDir
	}
	return 0
}

func (e *virtualDirEntry) Info() (fs.FileInfo, error) {
	return virtualFileInfo{name: e.name, dir: e.dir}, nil
}

type virtualFileInfo struct {
	name string
	dir  bool
}

func (i virtualFileInfo) Name() string       { return i.name }
func (i virtualFileInfo) Size() int64        { return 0 }
func (i virtualFileInfo) Mode() fs.FileMode  { return 0444 }
func (i virtualFileInfo) ModTime() time.Time { return time.Time{} }
func (i virtualFileInfo) IsDir() bool        { return i.dir }
func (i virtualFileInfo) Sys() any           { return nil }
