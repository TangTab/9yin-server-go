package res

import (
	"container/list"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// CachedStore 在 Store 之上加两级读缓存，供服务端常驻使用：
//
//   - 字节缓存：Read 命中后按 LRU 保留原始内容，重复读取不再
//     重复解压 MPQ / 读盘；
//   - 解析缓存：Load[T] 把「读字节 + parse」整体缓存，解析一次
//     到处复用（解析大 ini 往往比解压更贵）。
//
// 并发安全；同一文件的并发首读只做一次（其余等待结果）。
// Reload 之后所有缓存失效。Read / Load 返回的数据是缓存内部
// 对象，调用方只能读、不能改。
type CachedStore struct {
	store *Store

	maxBytes   int64        // 字节缓存总上限；单文件超限则只透传不缓存
	maxParsed  int          // 解析缓存条目上限
	generation atomic.Int64 // Reload 计数，FS 适配器据此失效名称索引

	mu        sync.Mutex
	bytes     map[string]*list.Element // key: NormalizeLower(name)
	byteLRU   *list.List               // front = 最近使用；值 *byteEntry
	byteSize  int64
	parsed    map[parsedKey]*list.Element // 值 *parsedEntry
	parsedLRU *list.List
	parsedN   int
	flights   map[string]*flight // 进行中的读取（key 同 bytes）

	// 统计（在 mu 内累加；快照走 Stats）。
	stats CacheStats
}

// parsedKey 以「类型 + 名字」区分不同解析视图。
type parsedKey struct {
	typ  reflect.Type
	name string
}

// flight 是一次进行中的读取；并发同键读者等待它完成。
type flight struct {
	done chan struct{}
	data []byte
	err  error
}

type byteEntry struct {
	name string
	data []byte
}

type parsedEntry struct {
	key   parsedKey
	value any
}

// CacheStats 是缓存命中统计快照。
type CacheStats struct {
	ByteHits    int64 // 字节缓存命中
	ByteMisses  int64 // 字节缓存未命中（含被逐出后的再次读取）
	ParsedHits  int64 // 解析缓存命中
	ParsedMiss  int64 // 解析缓存未命中
	Evictions   int64 // LRU 逐出条数（字节 + 解析）
	CachedFiles int   // 当前字节缓存文件数
	CachedBytes int64 // 当前字节缓存总字节数
	ParsedKeys  int   // 当前解析缓存条目数
}

// 缓存缺省上限：256 MiB 字节 / 4096 条解析结果。服务端常驻
// 加载的现代 ini（buff_static ~11 MiB 等）远低于该额度。
const (
	DefaultCacheMaxBytes  = 256 << 20
	DefaultCacheMaxParsed = 4096
)

// CacheOptions 配置缓存上限；零值字段取缺省。
type CacheOptions struct {
	MaxBytes  int64 // 字节缓存总上限（<=0 → 缺省）
	MaxParsed int   // 解析缓存条目上限（<=0 → 缺省）
}

// OpenCached 打开资源根并包一层缓存（Open + CacheOptions）。
func OpenCached(dir string, opt CacheOptions) (*CachedStore, error) {
	store, err := Open(dir)
	if err != nil {
		return nil, err
	}
	return WrapCached(store, opt), nil
}

// WrapCached 为已打开的 Store 加缓存。注意：后续热重载必须走
// CachedStore.Reload（内部计数代次供 FS 适配器失效索引）。
func WrapCached(store *Store, opt CacheOptions) *CachedStore {
	if opt.MaxBytes <= 0 {
		opt.MaxBytes = DefaultCacheMaxBytes
	}
	if opt.MaxParsed <= 0 {
		opt.MaxParsed = DefaultCacheMaxParsed
	}
	return &CachedStore{
		store:     store,
		maxBytes:  opt.MaxBytes,
		maxParsed: opt.MaxParsed,
		bytes:     make(map[string]*list.Element),
		byteLRU:   list.New(),
		parsed:    make(map[parsedKey]*list.Element),
		parsedLRU: list.New(),
		flights:   make(map[string]*flight),
	}
}

// Store 返回底层资源仓（诊断用；直接使用会绕过缓存）。
func (c *CachedStore) Store() *Store { return c.store }

// Generation 返回当前重载代次；从 0 开始，每次 Reload +1。
func (c *CachedStore) Generation() int64 { return c.generation.Load() }

// Reload 热重载并清空两级缓存。
func (c *CachedStore) Reload() error {
	if err := c.store.Reload(); err != nil {
		return err
	}
	c.mu.Lock()
	c.bytes = make(map[string]*list.Element)
	c.byteLRU = list.New()
	c.byteSize = 0
	c.parsed = make(map[parsedKey]*list.Element)
	c.parsedLRU = list.New()
	c.parsedN = 0
	c.mu.Unlock()
	c.generation.Add(1)
	return nil
}

// Close 释放底层归档句柄（缓存随之作废）。
func (c *CachedStore) Close() { c.store.Close() }

// Read 读取文件内容；返回值是缓存内部切片，调用方不得修改。
func (c *CachedStore) Read(name string) ([]byte, error) {
	return c.readBytes(name)
}

// readBytes 统一读取路径：字节缓存 → 进行中读取 → 底层 Store。
func (c *CachedStore) readBytes(name string) ([]byte, error) {
	key := NormalizeLower(name)
	if key == "" {
		return nil, fmt.Errorf("res: 资源名为空")
	}

	c.mu.Lock()
	if elem, ok := c.bytes[key]; ok {
		c.byteLRU.MoveToFront(elem)
		data := elem.Value.(*byteEntry).data
		c.stats.ByteHits++
		c.mu.Unlock()
		return data, nil
	}
	// 已有同键读取进行中：等待它，不重复解压。
	if f, ok := c.flights[key]; ok {
		c.mu.Unlock()
		<-f.done
		return f.data, f.err
	}
	f := &flight{done: make(chan struct{})}
	c.flights[key] = f
	c.stats.ByteMisses++
	c.mu.Unlock()

	data, err := c.store.Read(name)
	f.data, f.err = data, err
	close(f.done)

	c.mu.Lock()
	delete(c.flights, key)
	if err == nil && int64(len(data)) <= c.maxBytes {
		c.rememberBytesLocked(key, data)
	}
	c.mu.Unlock()
	return data, err
}

// rememberBytesLocked 记入字节缓存并按 LRU 逐出超限条目（调用方持锁）。
func (c *CachedStore) rememberBytesLocked(key string, data []byte) {
	if elem, ok := c.bytes[key]; ok {
		c.byteSize -= int64(len(elem.Value.(*byteEntry).data))
		c.byteLRU.Remove(elem)
		delete(c.bytes, key)
	}
	c.bytes[key] = c.byteLRU.PushFront(&byteEntry{name: key, data: data})
	c.byteSize += int64(len(data))
	for c.byteSize > c.maxBytes && c.byteLRU.Len() > 1 {
		oldest := c.byteLRU.Back()
		entry := oldest.Value.(*byteEntry)
		c.byteSize -= int64(len(entry.data))
		c.byteLRU.Remove(oldest)
		delete(c.bytes, entry.name)
		c.stats.Evictions++
	}
}

// Exists 报告名字是否存在（墓碑视为不存在）。
func (c *CachedStore) Exists(name string) bool { return c.store.Exists(name) }

// Layer 返回名字所在层；与 Store.Layer 一致。
func (c *CachedStore) Layer(name string) string { return c.store.Layer(name) }

// List 返回三层合并后的全量清单。
func (c *CachedStore) List() ([]ResFile, error) { return c.store.List() }

// Load 读取并解析，结果按「类型 + 名字」缓存；value 必须视为只读。
//
// 典型用法：把大 ini 一次解析成表，多个子系统共享同一份结果，
// 「读 + 解析」都只发生一次。
func Load[T any](c *CachedStore, name string, parse func([]byte) (T, error)) (T, error) {
	var zero T
	if c == nil {
		return zero, fmt.Errorf("res: CachedStore 为 nil")
	}
	key := parsedKey{typ: reflect.TypeOf((*T)(nil)).Elem(), name: NormalizeLower(name)}
	if key.name == "" {
		return zero, fmt.Errorf("res: 资源名为空")
	}

	c.mu.Lock()
	if elem, ok := c.parsed[key]; ok {
		c.parsedLRU.MoveToFront(elem)
		value := elem.Value.(*parsedEntry).value.(T)
		c.stats.ParsedHits++
		c.mu.Unlock()
		return value, nil
	}
	// 解析缓存未命中但字节缓存命中：免解压，直接解析。
	if elem, ok := c.bytes[key.name]; ok {
		c.byteLRU.MoveToFront(elem)
		data := elem.Value.(*byteEntry).data
		c.stats.ParsedMiss++
		c.mu.Unlock()
		value, err := parse(data)
		if err != nil {
			return zero, err
		}
		c.rememberParsedLocked(key, value)
		return value, nil
	}
	c.stats.ParsedMiss++
	c.mu.Unlock()

	data, err := c.readBytes(name)
	if err != nil {
		return zero, err
	}
	value, err := parse(data)
	if err != nil {
		return zero, err
	}
	c.rememberParsedLocked(key, value)
	return value, nil
}

// rememberParsedLocked 记入解析缓存并按 LRU 逐出超限条目（调用方持锁）。
func (c *CachedStore) rememberParsedLocked(key parsedKey, value any) {
	if _, ok := c.parsed[key]; ok {
		return
	}
	c.parsed[key] = c.parsedLRU.PushFront(&parsedEntry{key: key, value: value})
	c.parsedN++
	for c.parsedN > c.maxParsed {
		oldest := c.parsedLRU.Back()
		entry := oldest.Value.(*parsedEntry)
		c.parsedLRU.Remove(oldest)
		delete(c.parsed, entry.key)
		c.parsedN--
		c.stats.Evictions++
	}
}

// Stats 返回缓存统计快照。
func (c *CachedStore) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStats{
		ByteHits:    c.stats.ByteHits,
		ByteMisses:  c.stats.ByteMisses,
		ParsedHits:  c.stats.ParsedHits,
		ParsedMiss:  c.stats.ParsedMiss,
		Evictions:   c.stats.Evictions,
		CachedFiles: len(c.bytes),
		CachedBytes: c.byteSize,
		ParsedKeys:  c.parsedN,
	}
}

// -- 诊断透传（语义同 Store 同名方法） ----------------------------------------

func (c *CachedStore) BaseCount() int       { return c.store.BaseCount() }
func (c *CachedStore) PatchCount() int      { return c.store.PatchCount() }
func (c *CachedStore) HasPatch() bool       { return c.store.HasPatch() }
func (c *CachedStore) BaseVersion() string  { return c.store.BaseVersion() }
func (c *CachedStore) PatchVersion() string { return c.store.PatchVersion() }
func (c *CachedStore) Warnings() []string   { return c.store.Warnings() }
