package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// 经脉修炼/激活（Java JingMaiCustomHandler/JingMaiMessages/LoginFlow
// 授予块全链对齐；docs/fork-p2-shop-title-jingmai.md）。
//
// 已学模型（Java PlayerMeridian）：starter 14 条（starterJingMaiIDs，
// F 盘 52795 抓包 View45 行序）进世界即学、每条 1 级；View45 只发已学
// 行，目录其余条目（逆脉/金脉）不发。修炼闸=已学（未学静默拒）。
//
// 150 = handleJingMaiCultivateCustom：[150, i32 sub, ...]
//
//	sub=1 设修炼位（values[2] 经脉 configID 字符串；空=停修）。已学
//	     闸拒绝未学 id（Java「未学经脉静默拒」同语义）；首次开修且无
//	     激活行时自动补 active_jingmai_rec（Java setCultivating）。
//	     sub=2 真气刷新（重推面板帧）；其余子令不受理。
//
// 161 = handleJingMaiActiveCustom：[161, i32 sub(1=激活/2=去活), str id]
//
//	（fork 反汇编：v1∈{1,2} 且 v2 为非空窄串），
//	active_jingmai_rec 记录行增删。
//
// 数值模型（Java MeridianCatalog 同源）：750 真气/周天（升级，也是
// View45 TotalFillValue 显示口径=每周天所需而非历史累计）、修炼速度
// 43 真气/分钟、每日真气上限 ZhenQiDayValue=500（ZQPLValue=今日已耗）。
// MaxLevel = jingmai_static.ini 的 MaxVarPropNo-MinVarPropNo+1（丹田 180、
// 正经 216、跷脉 225）。
const (
	clientCustomJingMaiCultivate int32 = 150
	clientCustomJingMaiActive    int32 = 161

	jingMaiSubSetCultivation int32 = 1
	jingMaiSubRefreshZhenQi  int32 = 2
	jingMaiSubActivate       int32 = 1
	jingMaiSubDeactivate     int32 = 2

	// View45 经脉容器（View43 内功 / View46 轻功同族）。
	viewportJingMai uint16 = 45

	zhenQiPerWeek     = 750 // 周天（升 1 级所需真气）
	zhenQiPerMinute   = 43  // 修炼速度（真气/分钟）
	zhenQiDayValueMax = 500 // ZhenQiDayValue 缺省（Java 同值）
	// jingMaiCombatZhenQiRatio 战斗真气折算比（Java MeridianCatalog.
	// COMBAT_ZHENQI_RATIO；zhenqi.ini damage_ratio=0.1——伤害合计×0.1
	// 向下取整保底 1 折真气）。
	jingMaiCombatZhenQiRatio = 0.1
	propCurJingMai           = 849 // string 当前修炼经脉
	propLastJingMai          = 850 // string 上次修炼经脉
	propJingMaiTotLevel      = 819
	propJMActCount           = 851 // 可激活条数（恒 1）
	propZhenQiDayValue       = 852
	propZQPLValue            = 855 // 今日真气
	propZQActlValue          = 856 // 恒 0
	propZQUnUsedValue        = 857 // 恒 0

	propJMStaticData = 2237
	propJMItemType   = 1889
	propJMLevel      = 1465 // byte
	propJMMaxLevel   = 2083
	propCurFill      = 764
	propTotalFill    = 765

	jingMaiDBQueryTimeout = 5 * time.Second
)

var (
	defaultJingMaiINIPath       = filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai.ini")
	defaultJingMaiStaticINIPath = filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai_static.ini")
)

// starterJingMaiIDs Java MeridianCatalog.STARTER_MERIDIANS 同表：F 盘
// 52795 抓包 View45 的 14 行授予顺序（丹田 + 十二正经 + 阴跷·隐/阳跷·隐）。
// 目录里其余条目（逆脉 _ni / 金脉 _jin 等）默认未学，不发 View45 行、
// 修炼闸拒绝（Java setCultivating「未学经脉静默拒」同语义）。
var starterJingMaiIDs = []string{
	"jm_dantian",
	"jm_zuyangming",
	"jm_zushaoyin",
	"jm_shoutaiyin",
	"jm_shoutaiyang",
	"jm_zutaiyin",
	"jm_zushaoyang",
	"jm_zujueyin",
	"jm_shoushaoyang",
	"jm_shouyangming",
	"jm_zutaiyang",
	"jm_shoushaoyin",
	"jm_yinqiao_wei",
	"jm_yangqiao_wei",
}

// ---- 目录（jingmai.ini + jingmai_static.ini，GBK 文件只读 ASCII 字段） ----

type jingMaiDef struct {
	configID   string
	itemType   int32 // 1003
	staticData int32
	maxLevel   int32 // jingmai_static.ini 区间宽
}

// globalJingMaiCatalog 是目录的包级缓存（main.handle 每连接装载成功后
// 刷新；战斗真气入账等无连接上下文的路径取这里）。
var globalJingMaiCatalog *jingMaiCatalog

type jingMaiCatalog struct {
	order []jingMaiDef
	byID  map[string]int // configID → order 下标
}

// loadJingMaiCatalog 解析经脉目录。jingmai.ini 小节名=经脉 configID
// （ItemType/StaticData）；jingmai_static.ini 以 StaticData 为小节名给
// Min/MaxVarPropNo 区间。
func loadJingMaiCatalog(jingMaiPath, staticPath string) (*jingMaiCatalog, error) {
	staticRanges := make(map[int32][2]int64)
	if file, err := openResourceReader(staticPath); err == nil {
		defer file.Close()
		section := int64(-1 << 62) // 未进入任何小节
		scanner := bufio.NewScanner(file)
		var minVar, maxVar int64
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				if section != -1<<62 && maxVar >= minVar {
					staticRanges[int32(section)] = [2]int64{minVar, maxVar}
				}
				parsed, parseErr := strconv.ParseInt(strings.Trim(line, "[]"), 10, 32)
				if parseErr != nil {
					section, minVar, maxVar = -1<<62, 0, -1
					continue
				}
				section, minVar, maxVar = parsed, 0, -1
				continue
			}
			key, value, found := strings.Cut(line, "=")
			if !found {
				continue
			}
			switch strings.TrimSpace(key) {
			case "MinVarPropNo":
				minVar, _ = strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			case "MaxVarPropNo":
				maxVar, _ = strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("jingmai catalog: read static %s: %v", staticPath, err)
		}
		if section != -1<<62 && maxVar >= minVar {
			staticRanges[int32(section)] = [2]int64{minVar, maxVar}
		}
	} else {
		log.Printf("jingmai catalog: static ranges unavailable (%s): %v", staticPath, err)
	}

	catalog := &jingMaiCatalog{byID: make(map[string]int)}
	file, err := openResourceReader(jingMaiPath)
	if err != nil {
		return nil, fmt.Errorf("open jingmai catalog %s: %w", jingMaiPath, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var current *jingMaiDef
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.Trim(line, "[]")
			catalog.order = append(catalog.order, jingMaiDef{configID: name, itemType: 1003})
			catalog.byID[name] = len(catalog.order) - 1
			current = &catalog.order[len(catalog.order)-1]
			continue
		}
		if current == nil {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "ItemType":
			parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
			current.itemType = int32(parsed)
		case "StaticData":
			parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
			current.staticData = int32(parsed)
			if bounds, ok := staticRanges[int32(parsed)]; ok {
				current.maxLevel = int32(bounds[1]-bounds[0]) + 1
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read jingmai catalog %s: %w", jingMaiPath, err)
	}
	if len(catalog.order) == 0 {
		return nil, fmt.Errorf("jingmai catalog %s has no sections", jingMaiPath)
	}
	for index := range catalog.order {
		if catalog.order[index].maxLevel <= 0 {
			catalog.order[index].maxLevel = 1
		}
	}
	return catalog, nil
}

func (catalog *jingMaiCatalog) def(configID string) (jingMaiDef, bool) {
	if catalog == nil {
		return jingMaiDef{}, false
	}
	index, ok := catalog.byID[configID]
	if !ok {
		return jingMaiDef{}, false
	}
	return catalog.order[index], true
}

// starterOrder starter 表里目录仍认识的条目（授予序保持 starter 顺序；
// 目录缺条自动跳过）。
func (catalog *jingMaiCatalog) starterOrder() []string {
	if catalog == nil {
		return nil
	}
	known := make([]string, 0, len(starterJingMaiIDs))
	for _, id := range starterJingMaiIDs {
		if _, ok := catalog.byID[id]; ok {
			known = append(known, id)
		}
	}
	return known
}

// ---- 会话状态（挂 playerActor.jingmai；锁纪律 p.mu） ----

type jingMaiProgressRow struct {
	Level int32 `json:"level"`
	Fill  int32 `json:"fill"`  // 当前周天内真气（0..749）
	Total int32 `json:"total"` // 历史累计真气（TotalFillValue 展示）
}

type jingMaiZhenQiPool struct {
	Day  string `json:"day"`  // 本地日期 YYYY-MM-DD
	Used int32  `json:"used"` // 今日已耗（ZQPLValue）
}

// jingMaiDailyCap 每日真气硬闸（GM set_zhenqi_cap；Java PlayerMeridian
// capEnabled/capValue 同构）：开启后当日已获真气（ZQPLValue）达到阈值
// 即拦截入账（被动修炼/打怪折算同一闸，超额笔按剩余额度截断、零帧
// 静默），ZhenQiDayValue 属性随开关显示阈值/回落官方缺省 500。
// 默认关闭（私服节奏不限量，docs/jingmai-cd-qinggong.md §二）。
type jingMaiDailyCap struct {
	Enabled bool  `json:"enabled"`
	Value   int32 `json:"value"` // 每日阈值 ≥1；关闭时仍保存，重开沿用
}

type jingMaiState struct {
	cur        string
	last       string
	learned    []string // 已学经脉授予序（= View45 行序；Java meridians LinkedHashMap）
	activated  []string
	progress   map[string]jingMaiProgressRow
	zhenQi     jingMaiZhenQiPool
	cap        jingMaiDailyCap
	lastTickAt time.Time
}

type jingMaiSnapshot struct {
	Cur       string                        `json:"cur"`
	Last      string                        `json:"last"`
	Learned   []string                      `json:"learned,omitempty"`
	Activated []string                      `json:"activated"`
	Progress  map[string]jingMaiProgressRow `json:"progress"`
	ZhenQi    jingMaiZhenQiPool             `json:"zhenQi"`
	Cap       jingMaiDailyCap               `json:"cap"`
}

func (p *playerActor) jingMaiSnapshotOf() jingMaiSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshot := jingMaiSnapshot{
		Cur: p.jingmai.cur, Last: p.jingmai.last,
		Learned:   append([]string(nil), p.jingmai.learned...),
		Activated: append([]string(nil), p.jingmai.activated...),
		ZhenQi:    p.jingmai.zhenQi,
		Cap:       p.jingmai.cap,
	}
	if len(p.jingmai.progress) > 0 {
		snapshot.Progress = make(map[string]jingMaiProgressRow, len(p.jingmai.progress))
		for id, row := range p.jingmai.progress {
			snapshot.Progress[id] = row
		}
	}
	return snapshot
}

// restoreJingMai 登录装载快照并重置结算锚点（首 tick 从 now 起算，不追账）。
// 旧档无 learned 节点时回填：starter 授予序 + 快照进度里多出的经脉
// （键序稳定），与 Java 快照 meridians 列表同语义。
func (p *playerActor) restoreJingMai(snapshot jingMaiSnapshot, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jingmai.cur = snapshot.Cur
	p.jingmai.last = snapshot.Last
	p.jingmai.learned = append([]string(nil), snapshot.Learned...)
	if p.jingmai.learned == nil {
		seen := make(map[string]bool, len(starterJingMaiIDs))
		for _, id := range starterJingMaiIDs {
			seen[id] = true
			p.jingmai.learned = append(p.jingmai.learned, id)
		}
		extra := make([]string, 0)
		for id := range snapshot.Progress {
			if !seen[id] {
				extra = append(extra, id)
			}
		}
		sort.Strings(extra)
		p.jingmai.learned = append(p.jingmai.learned, extra...)
	}
	p.jingmai.activated = append([]string(nil), snapshot.Activated...)
	if snapshot.Progress != nil {
		p.jingmai.progress = make(map[string]jingMaiProgressRow, len(snapshot.Progress))
		for id, row := range snapshot.Progress {
			p.jingmai.progress[id] = row
		}
	} else {
		p.jingmai.progress = make(map[string]jingMaiProgressRow)
	}
	p.jingmai.zhenQi = snapshot.ZhenQi
	// 旧档无 cap 节点（零值关闸）：阈值回落官方缺省 500，重开沿用。
	p.jingmai.cap = snapshot.Cap
	if p.jingmai.cap.Value < 1 {
		p.jingmai.cap.Value = zhenQiDayValueMax
	}
	p.jingmai.lastTickAt = now
}

// ensureJingMaiLearned 首次进世界种子（Java PlayerMeridian.create）：
// starter 14 条按授予序入学，每条 1 级 0 真气；已有 learned 幂等直返。
func (p *playerActor) ensureJingMaiLearned(catalog *jingMaiCatalog) {
	if catalog == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.jingmai.learned) > 0 {
		return
	}
	if p.jingmai.progress == nil {
		p.jingmai.progress = make(map[string]jingMaiProgressRow)
	}
	for _, id := range catalog.starterOrder() {
		p.jingmai.learned = append(p.jingmai.learned, id)
		if _, ok := p.jingmai.progress[id]; !ok {
			p.jingmai.progress[id] = jingMaiProgressRow{Level: 1}
		}
	}
}

// jingMaiRowLocked 已学序行号（1 起；0=未学）。须持 p.mu。
func (p *playerActor) jingMaiRowLocked(id string) int {
	for row, existing := range p.jingmai.learned {
		if existing == id {
			return row + 1
		}
	}
	return 0
}

func (p *playerActor) jingMaiCur() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.jingmai.cur
}

// hasLearnedJingMai 修炼闸查询（Java「未学经脉静默拒」）。
func (p *playerActor) hasLearnedJingMai(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.jingMaiRowLocked(id) > 0
}

// setCurJingMai 修炼位迁移：设置修炼时 cur/last 同值、停止修炼时 cur
// 清空而 last 保留最后一条。客户端显示契约（form_main_player.
// set_jingmaiinfo_tips / form_wuxue_jingmai_infomation.get_jingmai_item）：
// 「修炼经脉/修炼周天/修炼进度」tips 用 LastJingMai 定位 View45 行取
// Level/CurFillValue/TotalFillValue，浮窗「继续修炼」按钮亦读
// LastJingMai 重发 150/1——修炼中 last 必须与 cur 同值；旧「cur→last
// 迁移」语义下首修后 last 空、客户端 get_jingmai_item 返回 nil，四项
// 全不显示（2026-11-08 经脉 tips 无数据根因）。
func (p *playerActor) setCurJingMai(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if id != "" {
		p.jingmai.cur = id
		p.jingmai.last = id
	} else {
		p.jingmai.cur = ""
	}
	if p.jingmai.lastTickAt.IsZero() {
		p.jingmai.lastTickAt = time.Now()
	}
}

func (p *playerActor) activateJingMai(id string) (added bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, existing := range p.jingmai.activated {
		if existing == id {
			return false
		}
	}
	p.jingmai.activated = append(p.jingmai.activated, id)
	return true
}

// deactivateJingMai 摘除激活行；返回 0 基行号供 0x12 清行。
func (p *playerActor) deactivateJingMai(id string) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for row, existing := range p.jingmai.activated {
		if existing == id {
			p.jingmai.activated = append(p.jingmai.activated[:row], p.jingmai.activated[row+1:]...)
			return row, true
		}
	}
	return 0, false
}

// setJingMaiDailyCap GM set_zhenqi_cap：是否开启每日硬闸 + 每日阈值
//（阈值 ≥1 才受理；关闭时阈值仍保存，重开沿用——Java setDailyCap 同
// 语义）。ZhenQiDayValue 口径随开关即时变化，调用方负责补发属性帧。
func (p *playerActor) setJingMaiDailyCap(enabled bool, value int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if value >= 1 {
		p.jingmai.cap.Value = value
	}
	if p.jingmai.cap.Value < 1 {
		p.jingmai.cap.Value = zhenQiDayValueMax
	}
	p.jingmai.cap.Enabled = enabled
}

// zhenQiDayValueLocked 玩家属性 ZhenQiDayValue 口径：开闸显示阈值，
// 关闸回落官方缺省 500（Java zhenQiDayValue 同构）。须持 p.mu。
func (p *playerActor) zhenQiDayValueLocked() int32 {
	if p.jingmai.cap.Enabled && p.jingmai.cap.Value >= 1 {
		return p.jingmai.cap.Value
	}
	return zhenQiDayValueMax
}

// setJingMaiLevel GM set_jingmai_tier：直设周天等级（钳 1..目录上限，
// fill 归零走新周天起点——Java setLevel 同构）。返回受理前旧等级、
// 新行与 View45 行号；未学该经脉返回 ok=false（调用方先校验目录收录）。
func (p *playerActor) setJingMaiLevel(catalog *jingMaiCatalog, id string, level int32) (old int32, row jingMaiProgressRow, slot int, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot = p.jingMaiRowLocked(id)
	if slot == 0 {
		return 0, jingMaiProgressRow{}, 0, false
	}
	target := level
	if def, found := catalog.def(id); found && def.maxLevel > 0 {
		target = min(max(level, 1), def.maxLevel)
	} else if target < 1 {
		target = 1
	}
	if p.jingmai.progress == nil {
		p.jingmai.progress = make(map[string]jingMaiProgressRow)
	}
	row = p.jingmai.progress[id]
	old = row.Level
	row.Level = target
	row.Fill = 0
	p.jingmai.progress[id] = row
	return old, row, slot, true
}

// jingMaiPropsLocked 玩家经脉属性 8 件套（Java JingMaiMessages.playerProps
// 同字段集：849/850 string + 819/851/852/855/856/857 i32；addObject 与
// stateProperties 复用）。须持 p.mu。
func (p *playerActor) jingMaiPropsLocked() []clientdata.IndexedProperty {
	totalLevel := int32(0)
	for _, id := range p.jingmai.learned {
		totalLevel += p.jingmai.progress[id].Level
	}
	return []clientdata.IndexedProperty{
		{Index: propCurJingMai, Name: "CurJingMai", Value: clientdata.StringValue(p.jingmai.cur)},
		{Index: propLastJingMai, Name: "LastJingMai", Value: clientdata.StringValue(p.jingmai.last)},
		{Index: propJingMaiTotLevel, Name: "JingMaiTotalLevel", Value: clientdata.Int32Value(totalLevel)},
		{Index: propJMActCount, Name: "jmActCount", Value: clientdata.Int32Value(1)},
		{Index: propZhenQiDayValue, Name: "ZhenQiDayValue", Value: clientdata.Int32Value(p.zhenQiDayValueLocked())},
		{Index: propZQPLValue, Name: "ZQPLValue", Value: clientdata.Int32Value(p.jingmai.zhenQi.Used)},
		{Index: propZQActlValue, Name: "ZQActlValue", Value: clientdata.Int32Value(0)},
		{Index: propZQUnUsedValue, Name: "ZQUnUsedValue", Value: clientdata.Int32Value(0)},
	}
}

// jingMaiPropsFrame 玩家经脉属性帧（Java JingMaiMessages.playerPropsFrame）。
func (p *playerActor) jingMaiPropsFrame() ([]byte, error) {
	p.mu.Lock()
	properties := p.jingMaiPropsLocked()
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, properties)
}

// jingMaiViewRowFrame View45 行帧（Java meridianRowFrame 同构 6+1 属性；
// TotalFillValue=每周天所需真气 750，非历史累计——Java totalFillPerLevel）。
func jingMaiViewRowFrame(row jingMaiProgressRow, def jingMaiDef, slot uint16) ([]byte, error) {
	properties := []serverViewProperty{
		viewString(1440, def.configID),
		viewInt(propJMStaticData, def.staticData),
		viewInt(propJMItemType, def.itemType),
		viewByte(propJMLevel, byte(row.Level)),
		viewInt(propJMMaxLevel, def.maxLevel),
		viewInt(propTotalFill, zhenQiPerWeek),
	}
	if row.Fill > 0 {
		properties = append(properties, viewInt(propCurFill, row.Fill))
	}
	return serverViewAdd(viewportJingMai, slot, properties)
}

// jingMaiViewFrames View45 容器 + 已学行（Java viewFrames：行序=learned
// 授予序，容量=已学数；目录其余条目不下发）。slot 与 learned 序一致（1 起）。
func (p *playerActor) jingMaiViewFrames(catalog *jingMaiCatalog) ([][]byte, error) {
	if catalog == nil {
		return nil, nil
	}
	p.mu.Lock()
	learned := append([]string(nil), p.jingmai.learned...)
	progress := make(map[string]jingMaiProgressRow, len(p.jingmai.progress))
	for id, row := range p.jingmai.progress {
		progress[id] = row
	}
	p.mu.Unlock()
	if len(learned) == 0 {
		return nil, nil
	}
	frames := [][]byte{serverCreateView(serverViewSpec{ID: viewportJingMai, Capacity: uint16(len(learned))})}
	for index, id := range learned {
		def, ok := catalog.def(id)
		if !ok {
			// Java definition==null 兜底：ConfigID 照发，其余 0/默认。
			def = jingMaiDef{configID: id, itemType: 1003}
		}
		frame, err := jingMaiViewRowFrame(progress[id], def, uint16(index+1))
		if err != nil {
			return nil, fmt.Errorf("encode jingmai view row %s: %w", id, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// jingMaiViewUpdateFrame 行级增量帧（Java progressFrame/levelUpFrame）：
// 0x10 isView=1 obj=45 owner=行号；升级帧加 Level+TotalFillValue（750）。
func jingMaiViewUpdateFrame(slot uint16, row jingMaiProgressRow, levelUp bool) ([]byte, error) {
	properties := []clientdata.IndexedProperty{
		{Index: propCurFill, Name: "CurFillValue", Value: clientdata.Int32Value(row.Fill)},
	}
	if levelUp {
		properties = append(properties,
			clientdata.IndexedProperty{Index: propJMLevel, Name: "Level", Value: clientdata.ByteValue(byte(row.Level))},
			clientdata.IndexedProperty{Index: propTotalFill, Name: "TotalFillValue", Value: clientdata.Int32Value(zhenQiPerWeek)},
		)
	}
	return sceneObjectProperties(uint32(viewportJingMai), uint32(slot), 1, properties)
}

// jingMaiCultivateTick 1s ticker 的结算体：整分钟 +43 真气/分钟进当前
// 修炼经脉（每日硬闸开闸时按剩余额度截断，关闸不限量），满 750 周天
// 升级；变化才产帧。返回待写帧（升级帧 levelUp=true 带 Level+
// TotalFillValue，否则单 CurFillValue）。
func (p *playerActor) jingMaiCultivateTick(catalog *jingMaiCatalog, now time.Time) [][]byte {
	if catalog == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	state := &p.jingmai
	if state.lastTickAt.IsZero() {
		state.lastTickAt = now
		return nil
	}
	today := now.Format("2006-01-02")
	if state.zhenQi.Day != today {
		state.zhenQi.Day = today
		state.zhenQi.Used = 0
	}
	minutes := int64(now.Sub(state.lastTickAt) / time.Minute)
	if minutes <= 0 {
		return nil
	}
	state.lastTickAt = state.lastTickAt.Add(time.Duration(minutes) * time.Minute)
	if state.cur == "" {
		return nil
	}
	return p.applyZhenQiLocked(catalog, int64(zhenQiPerMinute)*minutes)
}

// jingMaiCombatFeed 战斗真气入账（Java PlayerMeridian.addCombatZhenQi：
// 伤害合计 × zhenqi.ini damage_ratio=0.1，向下取整保底 1）。命中结算后
// 按整招累计调用，返回 View45 行帧（CurFillValue/升级帧）。变化才产帧。
func (p *playerActor) jingMaiCombatFeed(damage int32) [][]byte {
	if damage <= 0 || globalJingMaiCatalog == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	state := &p.jingmai
	if state.cur == "" {
		return nil
	}
	now := time.Now()
	today := now.Format("2006-01-02")
	if state.zhenQi.Day != today {
		state.zhenQi.Day = today
		state.zhenQi.Used = 0
	}
	accrue := int64(math.Max(1, math.Floor(float64(damage)*jingMaiCombatZhenQiRatio)))
	return p.applyZhenQiLocked(globalJingMaiCatalog, accrue)
}

// applyZhenQiLocked 真气入账公共体（Java applyZhenQi 同构；被动 tick 与
// 战斗折算同池同闸）：每日硬闸开闸时按剩余额度截断（额度耗尽零帧
// 静默，关闸不限量——ZhenQiDayValue 仅展示口径）；溢出连升封顶
// maxLevel；变化才产帧。须持 p.mu。
func (p *playerActor) applyZhenQiLocked(catalog *jingMaiCatalog, accrue int64) [][]byte {
	state := &p.jingmai
	if accrue <= 0 {
		return nil
	}
	if state.cap.Enabled {
		headroom := int64(state.cap.Value) - int64(state.zhenQi.Used)
		if headroom <= 0 {
			return nil // 当日额度已用尽：硬闸拦截（零帧静默）
		}
		if accrue > headroom {
			accrue = headroom
		}
	}
	slot := p.jingMaiRowLocked(state.cur)
	if slot == 0 {
		return nil // 未学经脉不结算（150/1 已闸；旧档防御）
	}
	def, ok := catalog.def(state.cur)
	if !ok || def.maxLevel <= 0 {
		return nil
	}
	row := state.progress[state.cur]
	var frames [][]byte
	consumed := int64(0)
	for accrue > 0 && row.Level < def.maxLevel {
		need := int64(zhenQiPerWeek - row.Fill)
		if accrue < need {
			row.Fill += int32(accrue)
			consumed += accrue
			accrue = 0
			if frame, err := jingMaiViewUpdateFrame(uint16(slot), row, false); err == nil {
				frames = append(frames, frame)
			}
			break
		}
		accrue -= need
		consumed += need
		row.Fill = 0
		row.Level++
		if frame, err := jingMaiViewUpdateFrame(uint16(slot), row, true); err == nil {
			frames = append(frames, frame)
		}
	}
	if consumed <= 0 {
		return nil
	}
	row.Total += int32(consumed)
	if state.progress == nil {
		state.progress = make(map[string]jingMaiProgressRow)
	}
	state.progress[state.cur] = row
	state.zhenQi.Used += int32(consumed)
	// 尾随玩家经脉属性帧（Java meridianTickOnce/SkillCastHandler 同构）：
	// ZQPLValue（今日已获）与 JingMaiTotalLevel（升级后）随入账刷新，
	// 客户端 tips「经脉总等级」/浮窗今日真气条才有增量更新。锁内纯编码。
	if propsFrame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, p.jingMaiPropsLocked()); err == nil {
		frames = append(frames, propsFrame)
	} else {
		log.Printf("jingmai: encode props frame after accrual: %v", err)
	}
	return frames
}

// ---- 持久化（role_meridians，migrations/0004；缺表降级） ----

type jingMaiDBStore struct {
	db *sql.DB
}

func newJingMaiDBStore(db *sql.DB) *jingMaiDBStore {
	return &jingMaiDBStore{db: db}
}

func (store *jingMaiDBStore) Load(roleID role.RoleID) (jingMaiSnapshot, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return jingMaiSnapshot{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), jingMaiDBQueryTimeout)
	defer cancel()
	var raw []byte
	err := store.db.QueryRowContext(ctx,
		`SELECT snapshot FROM role_meridians WHERE role_id = ?`, roleID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return jingMaiSnapshot{}, false
	}
	if err != nil {
		log.Printf("jingmai store: load role=%d: %v", roleID, err)
		return jingMaiSnapshot{}, false
	}
	var snapshot jingMaiSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		log.Printf("jingmai store: decode role=%d snapshot: %v", roleID, err)
		return jingMaiSnapshot{}, false
	}
	return snapshot, true
}

func (store *jingMaiDBStore) Save(roleID role.RoleID, value jingMaiSnapshot) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("jingmai store: no database")
	}
	if roleID == 0 {
		return errors.New("jingmai store: zero role id")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode jingmai snapshot: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), jingMaiDBQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx, `INSERT INTO role_meridians (role_id, snapshot, version, updated_at)
VALUES (?, ?, 1, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE snapshot = VALUES(snapshot), version = version + 1, updated_at = UTC_TIMESTAMP(6)`,
		roleID, raw); err != nil {
		return fmt.Errorf("persist jingmai role=%d: %w", roleID, err)
	}
	return nil
}

// ---- handlers ----

// handleJingMaiCultivateCustom msg150（sub/值校验照 fork :288/:294/:305）。
func handleJingMaiCultivateCustom(link sceneMessageConnection, player *playerActor, catalog *jingMaiCatalog, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomJingMaiCultivate {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject jingmai cultivate before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) < 2 {
		log.Printf("%s: reject malformed JINGMAI_WUJI values=%v", remote, custom.Values)
		return true, nil
	}
	sub, ok := custom.Values[1].exactInt32()
	if !ok {
		log.Printf("%s: reject malformed JINGMAI_WUJI values=%v", remote, custom.Values)
		return true, nil
	}
	switch sub {
	case jingMaiSubRefreshZhenQi:
		frame, err := player.jingMaiPropsFrame()
		if err != nil {
			return true, fmt.Errorf("encode jingmai refresh: %w", err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return true, fmt.Errorf("write jingmai refresh: %w", err)
		}
		return true, nil
	case jingMaiSubSetCultivation:
		if len(custom.Values) < 3 || (custom.Values[2].Type != 6 && custom.Values[2].Type != 7) {
			log.Printf("%s: reject malformed JINGMAI_WUJI values=%v", remote, custom.Values)
			return true, nil
		}
		id := custom.Values[2].Text
		if id != "" && !player.hasLearnedJingMai(id) {
			// Java「未学经脉静默拒」：客户端不弹错，服务端只记日志。
			log.Printf("%s: reject cultivate unlearned jingmai id=%q", remote, id)
			return true, nil
		}
		previous := player.jingMaiCur()
		player.setCurJingMai(id)
		if id != "" {
			log.Printf("%s: set jingmai cultivation id=%s", remote, id)
		} else {
			log.Printf("%s: cleared jingmai cultivation", remote)
		}
		// Java 首修自动激活：此前未修炼且无激活行 → 补 active_jingmai_rec
		//（激活位=当前修炼经脉，jmActCount=1）。
		if previous == "" && id != "" && len(player.jingMaiActivatedSnapshot()) == 0 {
			if player.activateJingMai(id) {
				if err := link.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, recordActiveJingMai, id)); err != nil {
					return true, fmt.Errorf("write active_jingmai_rec: %w", err)
				}
				log.Printf("%s: activated jingmai id=%s via active_jingmai_rec", remote, id)
			}
		}
		frame, err := player.jingMaiPropsFrame()
		if err != nil {
			return true, fmt.Errorf("encode jingmai cultivate update: %w", err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return true, fmt.Errorf("write jingmai cultivate update: %w", err)
		}
		return true, nil
	default:
		log.Printf("%s: unhandled jingmai_wuji sub=%d values=%v", remote, sub, custom.Values)
		return true, nil
	}
}

// handleJingMaiActiveCustom msg161（fork :226-264；增删 active_jingmai_rec
// 记录行后回面板帧）。
func handleJingMaiActiveCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomJingMaiActive {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject active jingmai before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) < 3 {
		log.Printf("%s: reject malformed ACTIVE_JINGMAI values=%v", remote, custom.Values)
		return true, nil
	}
	sub, subOK := custom.Values[1].exactInt32()
	id := ""
	if (custom.Values[2].Type != 6 && custom.Values[2].Type != 7) || custom.Values[2].Text == "" {
		id = ""
	} else {
		id = custom.Values[2].Text
	}
	if !subOK || (sub != jingMaiSubActivate && sub != jingMaiSubDeactivate) || id == "" {
		log.Printf("%s: reject malformed ACTIVE_JINGMAI values=%v", remote, custom.Values)
		return true, nil
	}
	switch sub {
	case jingMaiSubActivate:
		if player.activateJingMai(id) {
			if err := link.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, recordActiveJingMai, id)); err != nil {
				return true, fmt.Errorf("write active_jingmai_rec: %w", err)
			}
			log.Printf("%s: activated jingmai id=%s via active_jingmai_rec", remote, id)
		} else {
			log.Printf("%s: active jingmai already present id=%s", remote, id)
		}
	case jingMaiSubDeactivate:
		if row, removed := player.deactivateJingMai(id); removed {
			if err := link.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordActiveJingMai, uint16(row))); err != nil {
				return true, fmt.Errorf("remove active_jingmai_rec: %w", err)
			}
			log.Printf("%s: deactivated jingmai id=%s via active_jingmai_rec", remote, id)
		} else {
			log.Printf("%s: active jingmai not present id=%s", remote, id)
		}
	}
	frame, err := player.jingMaiPropsFrame()
	if err != nil {
		return true, fmt.Errorf("encode jingmai vitality update: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("write jingmai vitality update: %w", err)
	}
	return true, nil
}

// pushJingMaiFrames 进世界授予/重放（Java LoginFlow meridian 授予块：
// ClientReady 后 46/43/40 同位）。首次进世界先种 starter 14 条 1 级；
// 0x0C 重入时玩家对象已清空，View45 与记录行随本函数整体重放。
func pushJingMaiFrames(link sceneMessageConnection, player *playerActor, catalog *jingMaiCatalog) error {
	player.ensureJingMaiLearned(catalog)
	frames, err := player.jingMaiViewFrames(catalog)
	if err != nil {
		return fmt.Errorf("encode jingmai view: %w", err)
	}
	for _, frame := range frames {
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write jingmai view: %w", err)
		}
	}
	// Java 授予块：cur 非空即补 active 行（noteMeridianActivated 同位）。
	cur := player.jingMaiCur()
	activated := player.jingMaiActivatedSnapshot()
	if cur != "" && !containsString(activated, cur) {
		player.activateJingMai(cur)
		activated = append(activated, cur)
	}
	for _, id := range activated {
		if err := link.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, recordActiveJingMai, id)); err != nil {
			return fmt.Errorf("replay active_jingmai_rec %s: %w", id, err)
		}
	}
	if len(activated) > 0 {
		log.Printf("replayed %d active_jingmai_rec rows", len(activated))
	}
	props, err := player.jingMaiPropsFrame()
	if err != nil {
		return fmt.Errorf("encode jingmai props: %w", err)
	}
	if err := link.WriteFrame(props); err != nil {
		return fmt.Errorf("write jingmai props: %w", err)
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (p *playerActor) jingMaiActivatedSnapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.jingmai.activated...)
}
