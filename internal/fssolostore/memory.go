package fssolostore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// ErrUnsupported Generic 兜底未实现（Server 仅 trace）。
var ErrUnsupported = errors.New("fssolostore: unsupported store command")

// MemoryStore 是 Backend 的内存实现（对齐 fssologlobal LocalStore 模式）：
// 五通道语义内存化 + JSON 快照兜底；分周 suffix 退化为常量空串
// （无 DDL 面，PrepareSplitTables 返回原值生效即恒 ""）。
type MemoryStore struct {
	mu sync.Mutex

	manages []ManageEntry
	chunks  map[uint32]map[uint32][]byte // scene → chunkID → data
	roles   map[string]*memRole          // name → role
	byAcct  map[string][]string          // account@server → names

	pubTables map[string]map[string][]byte // table → name → save_data
	pubOrder  map[string][]string

	acctLogged map[string]int // dc_accounts.name → logged

	auctionAttr map[string][]byte
	auctionRec  map[string][]byte

	logs map[string][][]LogValue // 逻辑表 → 行

	suffix string
	path   string // 快照路径（空=不落盘）
}

type memRole struct {
	brief    RoleBrief
	meta     RoleMeta
	saveData []byte
	online   bool
	illegals int32
}

// NewMemoryStore 建内存库（可选快照路径）。
func NewMemoryStore(snapshotPath string) *MemoryStore {
	m := &MemoryStore{
		chunks:      map[uint32]map[uint32][]byte{},
		roles:       map[string]*memRole{},
		byAcct:      map[string][]string{},
		pubTables:   map[string]map[string][]byte{},
		pubOrder:    map[string][]string{},
		acctLogged:  map[string]int{},
		auctionAttr: map[string][]byte{},
		auctionRec:  map[string][]byte{},
		logs:        map[string][][]LogValue{},
		path:        snapshotPath,
	}
	m.loadSnapshot()
	return m
}

func acctKey(account string, serverID int) string { return account + "@" + itoa(serverID) }

// ---- Manage（nx_config）----

func (m *MemoryStore) ManageInfos() ([]ManageEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ManageEntry, len(m.manages))
	copy(out, m.manages)
	return out, nil
}

// AddManageEntry 测试/装配注入（官方 nx_config 种子数据）。
func (m *MemoryStore) AddManageEntry(e ManageEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.manages = append(m.manages, e)
}

// ---- Chunks（nx_chunks）----

func (m *MemoryStore) ChunkIDs(sceneID uint32) ([]uint32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ids []uint32
	for id := range m.chunks[sceneID] {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func (m *MemoryStore) LoadChunk(sceneID uint32, ids []uint32) (map[uint32][]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[uint32][]byte{}
	scene := m.chunks[sceneID]
	for _, id := range ids {
		if d, ok := scene[id]; ok {
			out[id] = append([]byte(nil), d...)
		}
	}
	return out, nil
}

func (m *MemoryStore) SaveChunk(sceneID uint32, id uint32, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.chunks[sceneID] == nil {
		m.chunks[sceneID] = map[uint32][]byte{}
	}
	m.chunks[sceneID][id] = append([]byte(nil), data...)
	return nil
}

func (m *MemoryStore) DeleteChunk(sceneID uint32, id uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.chunks[sceneID], id)
	return nil
}

// ---- Roles（nx_roles）----

func (m *MemoryStore) ResetOnline(serverID int) ([]RoleBrief, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []RoleBrief
	for _, r := range m.roles {
		if r.meta.ServerID == serverID || serverID == 0 {
			r.online = false
			out = append(out, r.brief)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *MemoryStore) SaveRole(name string, meta RoleMeta, saveData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.roles[name]
	if r == nil {
		r = &memRole{}
		m.roles[name] = r
	}
	r.brief = RoleBrief{Name: name, UID: meta.UID}
	r.meta = meta
	r.saveData = append([]byte(nil), saveData...)
	return nil
}

func (m *MemoryStore) LoadRole(account, name string, serverID int) ([]byte, RoleMeta, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.roles[name]
	if r == nil || r.brief.Deleted {
		return nil, RoleMeta{}, false, nil
	}
	if account != "" && r.meta.Account != account {
		return nil, RoleMeta{}, false, nil
	}
	if serverID != 0 && r.meta.ServerID != serverID {
		return nil, RoleMeta{}, false, nil
	}
	return append([]byte(nil), r.saveData...), r.meta, true, nil
}

func (m *MemoryStore) CreateRole(account, name, uid string, serverID int, meta RoleMeta, saveData []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, dup := m.roles[name]; dup {
		return false, nil // SELECT name FROM nx_roles WHERE name='%s'
	}
	meta.Account = account
	meta.ServerID = serverID
	meta.UID = uid
	r := &memRole{
		brief:    RoleBrief{Name: name, UID: uid, Index: meta.Index},
		meta:     meta,
		saveData: append([]byte(nil), saveData...),
	}
	m.roles[name] = r
	k := acctKey(account, serverID)
	m.byAcct[k] = append(m.byAcct[k], name)
	return true, nil
}

func (m *MemoryStore) SetOnline(name string, on bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r := m.roles[name]; r != nil {
		r.online = on
	}
	return nil
}

func (m *MemoryStore) QueryAccount(account string, serverID int) ([]RoleBrief, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []RoleBrief
	for _, name := range m.byAcct[acctKey(account, serverID)] {
		if r := m.roles[name]; r != nil && !r.brief.Deleted {
			out = append(out, r.brief)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out, nil
}

// ---- Public（公共空间表）----

func (m *MemoryStore) PubInfo() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for t := range m.pubTables {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, nil
}

func (m *MemoryStore) PubLoad(table string) (map[string][]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string][]byte{}
	for k, v := range m.pubTables[table] {
		out[k] = append([]byte(nil), v...)
	}
	return out, nil
}

func (m *MemoryStore) PubSave(table, name string, saveData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pubTables[table] == nil {
		m.pubTables[table] = map[string][]byte{}
	}
	if _, ok := m.pubTables[table][name]; !ok {
		m.pubOrder[table] = append(m.pubOrder[table], name)
	}
	m.pubTables[table][name] = append([]byte(nil), saveData...)
	return nil
}

func (m *MemoryStore) PubAdd(table, name, uid string, saveData []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pubTables[table] == nil {
		m.pubTables[table] = map[string][]byte{}
	}
	if _, dup := m.pubTables[table][name]; dup {
		return false, nil // SELECT name FROM %s WHERE name='%s'
	}
	m.pubOrder[table] = append(m.pubOrder[table], name)
	m.pubTables[table][name] = append([]byte(nil), saveData...)
	return true, nil
}

func (m *MemoryStore) PubRemove(table, name string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.pubTables[table][name]; !ok {
		return false, nil
	}
	delete(m.pubTables[table], name)
	for i, n := range m.pubOrder[table] {
		if n == name {
			m.pubOrder[table] = append(m.pubOrder[table][:i], m.pubOrder[table][i+1:]...)
			break
		}
	}
	return true, nil
}

// PubOrder 表内名序（测试/呈现）。
func (m *MemoryStore) PubOrder(table string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.pubOrder[table]...)
}

// ---- Auction（nx_auction PUB4 块）----

func (m *MemoryStore) AuctionLoad(kind AuctionKind) (map[string][]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	src := m.auctionAttr
	if kind == AuctionRec {
		src = m.auctionRec
	}
	out := make(map[string][]byte, len(src))
	for k, v := range src {
		out[k] = append([]byte(nil), v...)
	}
	return out, nil
}

func (m *MemoryStore) AuctionSave(kind AuctionKind, name string, saveData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if kind == AuctionRec {
		m.auctionRec[name] = append([]byte(nil), saveData...)
	} else {
		m.auctionAttr[name] = append([]byte(nil), saveData...)
	}
	return nil
}

// ---- Account（dc_accounts）----

func (m *MemoryStore) AccountLogon(name string, logged int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.acctLogged[name]
	if !ok {
		return false, nil // 帐号不存在（SELECT * WHERE name AND status=1）
	}
	if cur != 0 && cur != logged {
		return false, nil // (logged = 0 OR logged = %d)
	}
	m.acctLogged[name] = logged
	return true, nil
}

func (m *MemoryStore) AccountLogoff(name string, logged int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.acctLogged[name]
	if !ok || cur != logged {
		return false, nil // WHERE name='%s' AND logged=%d
	}
	m.acctLogged[name] = 0
	return true, nil
}

// AddAccount 测试/装配注入（官方 INSERT dc_accounts 面）。
func (m *MemoryStore) AddAccount(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acctLogged[name] = 0
}

// ---- Log ----

func (m *MemoryStore) InsertLog(table string, vals []LogValue) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs[table] = append(m.logs[table], append([]LogValue(nil), vals...))
	return nil
}

// LogRows 表行数（测试/呈现）。
func (m *MemoryStore) LogRows(table string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.logs[table])
}

func (m *MemoryStore) PrepareSplitTables(suffix string) (string, error) {
	return "", nil // 内存库无 DDL：suffix 恒空串
}

func (m *MemoryStore) SplitSuffix() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.suffix
}

func (m *MemoryStore) Generic(msg uint32, payload []byte) error { return ErrUnsupported }

func (m *MemoryStore) Ping() error  { return nil }
func (m *MemoryStore) Close() error { return m.saveSnapshot() }

// ---- JSON 快照 ----

type memSnapshot struct {
	Manages []ManageEntry                `json:"manages"`
	Chunks  map[string]map[string][]byte `json:"chunks"`
	Roles   map[string]*memRoleSnap      `json:"roles"`
	Pub     map[string]map[string][]byte `json:"pub"`
	Acct    map[string]int               `json:"accounts"`
	Attr    map[string][]byte            `json:"auction_attr"`
	Rec     map[string][]byte            `json:"auction_rec"`
	Logs    map[string][][]LogValue      `json:"logs"`
}

type memRoleSnap struct {
	Brief    RoleBrief `json:"brief"`
	Meta     RoleMeta  `json:"meta"`
	SaveData []byte    `json:"save_data"`
	Online   bool      `json:"online"`
}

func (m *MemoryStore) saveSnapshot() error {
	if m.path == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	snap := memSnapshot{
		Manages: m.manages,
		Chunks:  map[string]map[string][]byte{},
		Roles:   map[string]*memRoleSnap{},
		Pub:     m.pubTables,
		Acct:    m.acctLogged,
		Attr:    m.auctionAttr,
		Rec:     m.auctionRec,
		Logs:    m.logs,
	}
	for sc, mm := range m.chunks {
		t := map[string][]byte{}
		for id, d := range mm {
			t[utoa(uint64(id))] = d
		}
		snap.Chunks[utoa(uint64(sc))] = t
	}
	for n, r := range m.roles {
		snap.Roles[n] = &memRoleSnap{r.brief, r.meta, r.saveData, r.online}
	}
	data, err := json.MarshalIndent(snap, "", " ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(m.path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(m.path, data, 0o644)
}

func (m *MemoryStore) loadSnapshot() {
	if m.path == "" {
		return
	}
	data, err := os.ReadFile(m.path)
	if err != nil {
		return
	}
	var snap memSnapshot
	if json.Unmarshal(data, &snap) != nil {
		return
	}
	if snap.Manages != nil {
		m.manages = snap.Manages
	}
	for sc, mm := range snap.Chunks {
		var sceneID uint32
		fmt.Sscan(sc, &sceneID)
		t := map[uint32][]byte{}
		for id, d := range mm {
			var idv uint32
			fmt.Sscan(id, &idv)
			t[idv] = d
		}
		m.chunks[sceneID] = t
	}
	for n, r := range snap.Roles {
		if r != nil {
			m.roles[n] = &memRole{r.Brief, r.Meta, r.SaveData, r.Online, 0}
			k := acctKey(r.Meta.Account, r.Meta.ServerID)
			m.byAcct[k] = append(m.byAcct[k], n)
		}
	}
	if snap.Pub != nil {
		m.pubTables = snap.Pub
	}
	if snap.Acct != nil {
		m.acctLogged = snap.Acct
	}
	if snap.Attr != nil {
		m.auctionAttr = snap.Attr
	}
	if snap.Rec != nil {
		m.auctionRec = snap.Rec
	}
	if snap.Logs != nil {
		m.logs = snap.Logs
	}
}
