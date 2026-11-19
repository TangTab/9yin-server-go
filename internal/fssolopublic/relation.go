package fssolopublic

import (
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fspublic"
)

// Relation 复刻 CRelation（relation.cpp，SoloPublic+0x130）：
// 好友关系表 + SNS 动态流（feed）。装载链：store 0x2D→0x22 关系表、
// 0x3A→0x23 动态流（LoadFeed），feed 存活期配置经 0x30 回写 store。
type Relation struct {
	mu        sync.RWMutex
	relations []relationEntry // relation_t
	feeds     []*Feed         // feed_t（时间序）
	feedUsers map[string]int  // 账号 → feeds 下标（内含缓存计数）
	levels    int             // feed 等级数（SetFeedLevels）
	liveTime  []float64       // 每级存活期（秒；0x30 上报）
	uidNext   int
}

// relation_t：{name_a, name_b, type, liveTime}（0x22 archive 四字段）。
type relationEntry struct {
	A, B      string
	Type      int32
	LiveSecs  float64
	LiveUntil time.Time
}

// Feed 是 feed_t + 评论链（NewFeed/InnerAddComment 建立；
// LoadFeed 从 store 装载）。
type Feed struct {
	ID        string // 官方 NewFeed 返回的 feed id（buf guid）
	Owner     string // 发布者账号（char*）
	Receiver  string // 接收者账号
	Key       string
	Kind      int32
	Level     int32
	Time      int32  // 发布时刻（unix 秒）
	Text      string // wstr 正文
	GoodCount int32
	BadCount  int32
	Comments  []FeedComment
	Desc      string // wstr 附加描述（SetFeedDesc）
	liveSecs  float64
	expireAt  time.Time
}

// FeedComment 是 InnerAddComment 追加的评论项。
type FeedComment struct {
	Owner string
	Key   string
	Kind  int32
	Time  int32
	Text  string // wstr
}

func NewRelation() *Relation {
	return &Relation{feedUsers: map[string]int{}, levels: 3}
}

// SetFeedLevels / GetFeedLevels / SetFeedLiveTime / GetFeedLiveTime 官方
// feed 配置面（0x30 上报 {ServerId; double×levels}）。
func (r *Relation) SetFeedLevels(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n < 0 {
		n = 0
	}
	r.levels = n
	if len(r.liveTime) < n {
		r.liveTime = append(r.liveTime, make([]float64, n-len(r.liveTime))...)
	} else {
		r.liveTime = r.liveTime[:n]
	}
}

func (r *Relation) GetFeedLevels() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.levels
}

func (r *Relation) SetFeedLiveTime(level int, secs float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for len(r.liveTime) <= level {
		r.liveTime = append(r.liveTime, 0)
	}
	r.liveTime[level] = secs
}

func (r *Relation) GetFeedLiveTime(level int) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if level < 0 || level >= len(r.liveTime) {
		return 0
	}
	return r.liveTime[level]
}

// AddRelation 官方 AddRelation(a, b, type, liveSecs)（0x22 装载路径）。
func (r *Relation) AddRelation(a, b string, typ int32, liveSecs float64) bool {
	if a == "" || b == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.relations {
		if r.relations[i].A == a && r.relations[i].B == b {
			r.relations[i].Type = typ
			r.relations[i].LiveSecs = liveSecs
			r.relations[i].LiveUntil = time.Now().Add(time.Duration(liveSecs * float64(time.Second)))
			return true
		}
	}
	r.relations = append(r.relations, relationEntry{A: a, B: b, Type: typ, LiveSecs: liveSecs,
		LiveUntil: time.Now().Add(time.Duration(liveSecs * float64(time.Second)))})
	return true
}

// RemoveRelation 官方 RemoveRelation(a, b)。
func (r *Relation) RemoveRelation(a, b string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.relations {
		if r.relations[i].A == a && r.relations[i].B == b {
			r.relations = append(r.relations[:i], r.relations[i+1:]...)
			return true
		}
	}
	return false
}

// FindRelation 官方 FindRelation(a, b)。
func (r *Relation) FindRelation(a, b string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := range r.relations {
		if r.relations[i].A == a && r.relations[i].B == b {
			return true
		}
	}
	return false
}

// GetRelationCount / GetRelationList 官方查询面（按 a 列举）。
func (r *Relation) GetRelationCount(a string, typ int32) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for i := range r.relations {
		if r.relations[i].A == a && (typ == 0 || r.relations[i].Type == typ) {
			n++
		}
	}
	return n
}

func (r *Relation) GetRelationList(a string, typ int32, limit int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for i := range r.relations {
		if r.relations[i].A == a && (typ == 0 || r.relations[i].Type == typ) {
			out = append(out, r.relations[i].B)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}

// NewFeed 官方 NewFeed(owner, receiver, kind, level, time, text, desc)：
// 返回 feed id（guid 风格）。
func (r *Relation) NewFeed(owner, receiver string, kind, level int32, ts int32, text string, key string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uidNext++
	id := fspublic.CreateGUID()
	live := 0.0
	if level >= 0 && int(level) < len(r.liveTime) {
		live = r.liveTime[level]
	}
	f := &Feed{ID: id, Owner: owner, Receiver: receiver,
		Kind: kind, Level: level, Time: ts, Text: text, liveSecs: live}
	if live > 0 {
		f.expireAt = time.Now().Add(time.Duration(live * float64(time.Second)))
	}
	r.feeds = append(r.feeds, f)
	return id
}

// LoadFeed 官方 LoadFeed（0x23 装载路径 12 字段）。
func (r *Relation) LoadFeed(id, owner, receiver string, kind, level, ts int32,
	text string, good, bad int32, desc string, liveSecs float64) bool {
	if id == "" || owner == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.feeds {
		if f.ID == id {
			return true // 幂等
		}
	}
	f := &Feed{ID: id, Owner: owner, Receiver: receiver,
		Kind: kind, Level: level, Time: ts, Text: text,
		GoodCount: good, BadCount: bad, Desc: desc, liveSecs: liveSecs}
	if liveSecs > 0 {
		f.expireAt = time.Now().Add(time.Duration(liveSecs * float64(time.Second)))
	}
	r.feeds = append(r.feeds, f)
	return true
}

// FindFeed / AddComment / IncGood / IncBad / SetDesc 官方动态操作面。
func (r *Relation) FindFeed(id string) (*Feed, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, f := range r.feeds {
		if f.ID == id {
			cp := *f
			cp.Comments = append([]FeedComment(nil), f.Comments...)
			return &cp, true
		}
	}
	return nil, false
}

func (r *Relation) AddComment(id string, c FeedComment) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.feeds {
		if f.ID == id {
			f.Comments = append(f.Comments, c)
			return true
		}
	}
	return false
}

func (r *Relation) IncFeedGood(id string) bool { return r.incCount(id, +1, 0) }
func (r *Relation) IncFeedBad(id string) bool  { return r.incCount(id, 0, +1) }
func (r *Relation) GetFeedGoodBad(id string) (int32, int32, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, f := range r.feeds {
		if f.ID == id {
			return f.GoodCount, f.BadCount, true
		}
	}
	return 0, 0, false
}

func (r *Relation) incCount(id string, good, bad int32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.feeds {
		if f.ID == id {
			f.GoodCount += good
			f.BadCount += bad
			return true
		}
	}
	return false
}

func (r *Relation) SetFeedDesc(id, desc string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.feeds {
		if f.ID == id {
			f.Desc = desc
			return true
		}
	}
	return false
}

// SyncRecentFeeds 官方 SyncRecentFeeds(account, kind, limit, liveSecs, key)：
// 取 receiver=account 的最近动态（未过期优先，时间倒序）。
func (r *Relation) SyncRecentFeeds(account string, kind int32, limit int) []*Feed {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := time.Now()
	var out []*Feed
	for i := len(r.feeds) - 1; i >= 0; i-- {
		f := r.feeds[i]
		if f.Receiver != account && f.Owner != account {
			continue
		}
		if kind != 0 && f.Kind != kind {
			continue
		}
		if !f.expireAt.IsZero() && now.After(f.expireAt) {
			continue
		}
		cp := *f
		cp.Comments = append([]FeedComment(nil), f.Comments...)
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// ExpiredFeeds 周期清理（存活期到点）——官方 feed 过期由 liveTime 语义
// 隐含（GetFeedLiveTime 上报 store 侧同步淘汰），Go 版在查询侧过滤 +
// 此显式清扫。
func (r *Relation) ExpiredFeeds(now time.Time) []*Feed {
	r.mu.Lock()
	defer r.mu.Unlock()
	var dead []*Feed
	live := r.feeds[:0]
	for _, f := range r.feeds {
		if !f.expireAt.IsZero() && now.After(f.expireAt) {
			dead = append(dead, f)
			continue
		}
		live = append(live, f)
	}
	r.feeds = live
	return dead
}

// FeedCount / RelationCount 呈现用。
func (r *Relation) FeedCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.feeds)
}

func (r *Relation) RelationCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.relations)
}

// EncodeRelationArchive 按 0x22 应答载荷的 archive 段编码关系表
// （ReadString×2 + ReadInt + ReadDouble 每条）。
func (r *Relation) EncodeRelationArchive(a *fspublic.StoreArchive) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := range r.relations {
		if err := a.WriteString(r.relations[i].A); err != nil {
			return err
		}
		if err := a.WriteString(r.relations[i].B); err != nil {
			return err
		}
		if err := a.WriteInt(r.relations[i].Type); err != nil {
			return err
		}
		if err := a.WriteDouble(r.relations[i].LiveSecs); err != nil {
			return err
		}
	}
	return nil
}
