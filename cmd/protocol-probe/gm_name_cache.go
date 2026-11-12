// Package main · gm_name_cache.go —— game_items 中文名全量内存缓存
// （设计文档 §5.1 S1，治「加载数据卡服务器」的主因）。
//
// 旧路径：每个目录请求（时装/卡牌/发型）都对 game_items 发 64 条/批、
// 批间串行的 SQL；面板页一开就是几十条查询 × 3 端点并发，与游戏循环
// 抢同一个库。新路径：boot 后台一次性全表载入内存 map（数万行、几 MB），
// 目录端点从此零 DB 查询；名称表变动靠代数（generation）推动 ETag 失效。
//
// fail-soft：装载完成前/放弃后，hub.itemNames 回落原批查路径（见 main.go
// 接线），行为与旧版完全一致；DB 不可用按间隔重试直到成功。
package main

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"
)

// gmNameRowLimit 内存化行数上限（>此值放弃，回落批查+缓存模式；
// 设计文档 §10 风险表：极端大表不该把内存吃穿）。
const gmNameRowLimit = 500_000

// gmNameRetryInterval DB 不可用时的重试间隔。
const gmNameRetryInterval = 60 * time.Second

// gmNameQueryTimeout 单次全表装载的超时。
const gmNameQueryTimeout = 30 * time.Second

type gmNameCache struct {
	mu      sync.RWMutex
	byID    map[string]string
	gen     uint64 // 0=未装载；每次成功装载 +1（ETag 代数）
	loaded  bool
	gaveUp  bool
	loadErr error
	rows    int
}

// gmItemNames 包级实例。测试不经 main() 直接用包级状态：不启动后台
// 装载即 gen=0、未就绪 → hub.itemNames 走注入的兜底路径。
var gmItemNames = newGMNameCache()

func newGMNameCache() *gmNameCache {
	return &gmNameCache{}
}

// start 起后台装载循环：立即装一次，失败按 interval 重试；成功即停
// （名称表变动频率极低，重启进程刷新，不引入定时重载）。
// db 为 nil（无 DB 部署）时记一条日志后直接放弃。
func (cache *gmNameCache) start(db *sql.DB, interval time.Duration) {
	if db == nil {
		log.Printf("GM name cache: no database, catalogs fall back to ConfigID names")
		return
	}
	go func() {
		for {
			err := cache.load(db)
			if err == nil {
				cache.mu.RLock()
				rows, gaveUp := cache.rows, cache.gaveUp
				cache.mu.RUnlock()
				if gaveUp {
					log.Printf("GM name cache: table exceeds %d rows, memory mode disabled", gmNameRowLimit)
				} else {
					log.Printf("GM name cache: %d item names loaded into memory", rows)
				}
				return
			}
			log.Printf("GM name cache load failed (retry in %s): %v", interval, err)
			time.Sleep(interval)
		}
	}()
}

// load 全表装载。行数超限时置 gaveUp 并返回 nil（停止重试）。
func (cache *gmNameCache) load(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), gmNameQueryTimeout)
	defer cancel()
	rows, err := db.QueryContext(ctx, "SELECT item_id, item_name FROM game_items")
	if err != nil {
		cache.mu.Lock()
		cache.loadErr = err
		cache.mu.Unlock()
		return err
	}
	defer rows.Close()
	names := make(map[string]string, 1024)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			cache.mu.Lock()
			cache.loadErr = err
			cache.mu.Unlock()
			return err
		}
		if name != "" {
			names[id] = name
		}
		if len(names) > gmNameRowLimit {
			cache.mu.Lock()
			cache.gaveUp = true
			cache.loadErr = nil
			cache.mu.Unlock()
			rows.Close()
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		cache.mu.Lock()
		cache.loadErr = err
		cache.mu.Unlock()
		return err
	}
	cache.mu.Lock()
	cache.byID = names
	cache.rows = len(names)
	cache.loaded = true
	cache.loadErr = nil
	cache.gen++
	cache.mu.Unlock()
	return nil
}

// lookup 内存就绪时按 ID 批量取名（结果只含命中的键；未命中键由调用方
// 回落 ConfigID，与旧 attachNames 语义一致）。ready=false 表示内存化
// 未完成或已放弃，调用方应回落旧批查路径。
func (cache *gmNameCache) lookup(configIDs []string) (names map[string]string, ready bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	if !cache.loaded || cache.gaveUp {
		return nil, false
	}
	names = make(map[string]string, len(configIDs))
	for _, id := range configIDs {
		if name, ok := cache.byID[id]; ok && strings.TrimSpace(name) != "" {
			names[id] = name
		}
	}
	return names, true
}

// generation 名称代数（ETag 用）。每次成功装载 +1，未装载为 0。
func (cache *gmNameCache) generation() uint64 {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.gen
}

// stats 供日志/排障读取。
func (cache *gmNameCache) stats() (rows int, loaded, gaveUp bool, lastErr error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.rows, cache.loaded, cache.gaveUp, cache.loadErr
}
