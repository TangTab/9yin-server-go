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
// 授予块全链对齐；docs/systems/fork-p2-shop-title-jingmai.md）。
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

	jingMaiDBQueryTimeout = 5 * time.Second
)

// 属性索引按名解析（0x09 表方言门控换表自动跟随；fork 位见各行注记）。
var (
	propCurJingMai      = clientdata.MustPlayerProperty("CurJingMai").Index        // string 当前修炼经脉（fork=849）
	propLastJingMai     = clientdata.MustPlayerProperty("LastJingMai").Index       // string 上次修炼经脉（fork=850）
	propJingMaiTotLevel = clientdata.MustPlayerProperty("JingMaiTotalLevel").Index // JingMaiTotalLevel（fork=819）
	propJMActCount      = clientdata.MustPlayerProperty("jmActCount").Index        // 可激活条数（恒 1）（fork=851）
	propZhenQiDayValue  = clientdata.MustPlayerProperty("ZhenQiDayValue").Index    // ZhenQiDayValue（fork=852）
	propZQPLValue       = clientdata.MustPlayerProperty("ZQPLValue").Index         // 今日真气（fork=855）
	propZQActlValue     = clientdata.MustPlayerProperty("ZQActlValue").Index       // 恒 0（fork=856）
	propZQUnUsedValue   = clientdata.MustPlayerProperty("ZQUnUsedValue").Index     // 恒 0（fork=857）
	propJMStaticData    = clientdata.MustPlayerProperty("StaticData").Index        // StaticData（fork=2237）
	propJMVarPropNo     = clientdata.MustPlayerProperty("VarPropNo").Index         // VarPropNo（fork=2239）
	propJMItemType      = clientdata.MustPlayerProperty("ItemType").Index          // ItemType（fork=1889）
	propJMLevel         = clientdata.MustPlayerProperty("Level").Index             // byte（fork=1465）
	propJMMaxLevel      = clientdata.MustPlayerProperty("MaxLevel").Index          // MaxLevel（fork=2083）
	propCurFill         = clientdata.MustPlayerProperty("CurFillValue").Index      // CurFillValue（fork=764）
	propTotalFill       = clientdata.MustPlayerProperty("TotalFillValue").Index    // TotalFillValue（fork=765）
)

var (
	defaultJingMaiINIPath       = filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai.ini")
	defaultJingMaiStaticINIPath = filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai_static.ini")
)

// starterJingMaiIDs 全套开通序（2026-11-15 官方式对齐：学坐忘功后开通
// 全套经脉）。前 14 条=F 盘 52795 抓包 View45 的官方新号授予序（丹田 +
// 11 正经 + 阴跷·隐/阳跷·隐）；追加 7 条补齐全套（手厥阴 + 冲/带/
// 阴维/阳维/阴跷/阳跷）。逆脉 _ni / 经筋 _jin 不自动开通（官方走心魔/
// 兑换玩法，本服无——GM open_xuewei 目录可及）。未学经脉修炼闸拒绝
// （Java setCultivating「未学经脉静默拒」同语义）。
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
	"jm_shoujueyin",
	"jm_chong",
	"jm_dai",
	"jm_yinwei",
	"jm_yangwei",
	"jm_yinqiao",
	"jm_yangqiao",
}

// ---- 目录（jingmai.ini + jingmai_static.ini，GBK 文件只读 ASCII 字段） ----

type jingMaiDef struct {
	configID     string
	itemType     int32 // 1003
	staticData   int32
	maxLevel     int32 // jingmai_static.ini 区间宽
	minVarPropNo int32 // 区间起点（VarPropNo=Min+Level-1；目录缺区间 0）
	// perLevel 官方逐层累积属性（jingmai_attrs.go attachJingMaiAttrs
	// 填充；缺表空 map=零加成降级）。xueWei 穴位地图有序表（UI ini）。
	perLevel map[int32]jingMaiAttrSet
	xueWei   []string
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
				current.minVarPropNo = int32(bounds[0])
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
	// 官方式数据链挂接（jingmai_attrs.go；best-effort，缺表降级）：
	// 逐层属性（proppack）、穴位地图（UI ini）、开穴规则/互斥懒加载。
	catalog.attachJingMaiAttrs()
	catalog.attachXueWeiOrder()
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
// 默认关闭（私服节奏不限量，docs/combat/jingmai-cd-qinggong.md §二）。
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
	// openedXueWei 已开穴位（View44 行序；内功 activerule 自动开 + GM
	// open_xuewei，只增不减）。actLimit 可激活经脉条数上限（官方初始 3，
	// GM set_jingmai_act_count 1..8；jmActCount 属性即此值）。
	openedXueWei []string
	actLimit     int32
}

type jingMaiSnapshot struct {
	Cur       string                        `json:"cur"`
	Last      string                        `json:"last"`
	Learned   []string                      `json:"learned,omitempty"`
	Activated []string                      `json:"activated"`
	Progress  map[string]jingMaiProgressRow `json:"progress"`
	ZhenQi    jingMaiZhenQiPool             `json:"zhenQi"`
	Cap       jingMaiDailyCap               `json:"cap"`
	// 官方式穴位/激活上限（2026-11-15；旧档缺字段=空穴/默认条数，回填
	// 语义见 restoreJingMai）。
	OpenedXueWei []string `json:"openedXueWei,omitempty"`
	ActLimit     int32    `json:"actLimit,omitempty"`
}

func (p *playerActor) jingMaiSnapshotOf() jingMaiSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshot := jingMaiSnapshot{
		Cur: p.jingmai.cur, Last: p.jingmai.last,
		Learned:      append([]string(nil), p.jingmai.learned...),
		Activated:    append([]string(nil), p.jingmai.activated...),
		ZhenQi:       p.jingmai.zhenQi,
		Cap:          p.jingmai.cap,
		OpenedXueWei: append([]string(nil), p.jingmai.openedXueWei...),
		ActLimit:     p.jingmai.actLimit,
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
	// 官方式扩展（旧档缺字段回填）：穴位空集随内功规则/重入再收敛；
	// actLimit 缺省官方初始 3 条。
	p.jingmai.openedXueWei = append([]string(nil), snapshot.OpenedXueWei...)
	p.jingmai.actLimit = snapshot.ActLimit
	if p.jingmai.actLimit < 1 {
		p.jingmai.actLimit = jingMaiActLimitDefault
	}
	if p.jingmai.actLimit > jingMaiActLimitMax {
		p.jingmai.actLimit = jingMaiActLimitMax
	}
	p.jingmai.lastTickAt = now
}

// ensureJingMaiLearned 进世界种子与补齐（Java PlayerMeridian.create +
// 2026-11-15 全套开通）：starter 全集按授予序入学，每条 1 级 0 真气；
// 老档已学的保留原行序，缺的追加（幂等并集——14 条旧档自动补成全套）。
func (p *playerActor) ensureJingMaiLearned(catalog *jingMaiCatalog) {
	if catalog == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	known := make(map[string]bool, len(p.jingmai.learned))
	for _, id := range p.jingmai.learned {
		known[id] = true
	}
	if p.jingmai.progress == nil {
		p.jingmai.progress = make(map[string]jingMaiProgressRow)
	}
	added := false
	for _, id := range catalog.starterOrder() {
		if known[id] {
			continue
		}
		p.jingmai.learned = append(p.jingmai.learned, id)
		if _, ok := p.jingmai.progress[id]; !ok {
			p.jingmai.progress[id] = jingMaiProgressRow{Level: 1}
		}
		added = true
	}
	if added {
		log.Printf("jingmai: supplemented full-set meridians to %d rows", len(p.jingmai.learned))
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

// jingMaiOpenedXueWeiCountLocked 该经脉已开穴位数（按穴位地图有序表
// 计数，未知穴不计）。须持 p.mu。
func (p *playerActor) jingMaiOpenedXueWeiCountLocked(def jingMaiDef) int32 {
	if len(def.xueWei) == 0 {
		return 0
	}
	opened := make(map[string]bool, len(p.jingmai.openedXueWei))
	for _, id := range p.jingmai.openedXueWei {
		opened[id] = true
	}
	count := int32(0)
	for _, xwID := range def.xueWei {
		if opened[xwID] {
			count++
		}
	}
	return count
}

// jingMaiCapLocked 该经脉当前周天上限（阶梯模型 jingmai_attrs.go；目录
// 缺条回落目录上限）。须持 p.mu。
func (p *playerActor) jingMaiCapLocked(catalog *jingMaiCatalog, id string) int32 {
	def, ok := catalog.def(id)
	if !ok || def.maxLevel <= 0 {
		return jingMaiInitialCap
	}
	return jingMaiCapFor(def.maxLevel, p.jingMaiOpenedXueWeiCountLocked(def), int32(len(def.xueWei)))
}

// jingMaiCapOf 锁外查询：该经脉当前周天上限（GM 回报/测试用）。
func (p *playerActor) jingMaiCapOf(catalog *jingMaiCatalog, id string) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.jingMaiCapLocked(catalog, id)
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
// （阈值 ≥1 才受理；关闭时阈值仍保存，重开沿用——Java setDailyCap 同
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

// setJingMaiLevel GM set_jingmai_tier：直设周天等级（钳 1..当前阶梯上限
// ——开穴不足时不再直通目录满上限，2026-11-15 官方式；fill 归零走新
// 周天起点——Java setLevel 同构）。返回受理前旧等级、新行与 View45 行号；
// 未学该经脉返回 ok=false（调用方先校验目录收录）。
func (p *playerActor) setJingMaiLevel(catalog *jingMaiCatalog, id string, level int32) (old int32, row jingMaiProgressRow, slot int, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot = p.jingMaiRowLocked(id)
	if slot == 0 {
		return 0, jingMaiProgressRow{}, 0, false
	}
	limit := jingMaiInitialCap
	if def, found := catalog.def(id); found && def.maxLevel > 0 {
		limit = p.jingMaiCapLocked(catalog, id)
	}
	target := min(max(level, 1), limit)
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

// recomputeJingMaiBonusLocked 重算激活经脉的官方逐层加成（activated∩
// learned，等级钳当前阶梯上限，Σ perLevel[level]——逐层累积值直取）写
// p.progress.meridianBonus（effectiveFive/derivedResourceAdds 等消费）。
// 变更后调用方负责 syncEquippedNeiGongResourcesLocked + 全量属性帧。
// 须持 p.mu。
func (p *playerActor) recomputeJingMaiBonusLocked(catalog *jingMaiCatalog) {
	var bonus jingMaiAttrSet
	if catalog != nil {
		learned := make(map[string]bool, len(p.jingmai.learned))
		for _, id := range p.jingmai.learned {
			learned[id] = true
		}
		for _, id := range p.jingmai.activated {
			if !learned[id] {
				continue
			}
			def, ok := catalog.def(id)
			if !ok || def.maxLevel <= 0 {
				continue
			}
			level := p.jingmai.progress[id].Level
			if cap := p.jingMaiCapLocked(catalog, id); level > cap {
				level = cap
			}
			if level < 1 {
				level = 1
			}
			bonus.add(def.attrSetForLevel(level))
		}
	}
	p.progress.meridianBonus = bonus
}

// refreshJingMaiBonus 锁外入口：重算经脉加成并同步推导血/蓝上限。调用
// 方随后推全量属性帧（playerActor.vitalUpdate）。
func (p *playerActor) refreshJingMaiBonus(catalog *jingMaiCatalog) {
	p.mu.Lock()
	p.recomputeJingMaiBonusLocked(catalog)
	p.syncEquippedNeiGongResourcesLocked()
	p.mu.Unlock()
}

// reEvaluateXueWeiLocked 按内功开穴规则收敛已开穴位（xuewei_activerule：
// 已学内功达层即开，只增不减）。返回新增穴数。须持 p.mu。
func (p *playerActor) reEvaluateXueWeiLocked() int {
	catalog := xueWeiCatalogOf()
	if catalog == nil || len(catalog.rules) == 0 {
		return 0
	}
	opened := make(map[string]bool, len(p.jingmai.openedXueWei))
	for _, id := range p.jingmai.openedXueWei {
		opened[id] = true
	}
	added := 0
	for i := range p.progress.books {
		book := &p.progress.books[i]
		for _, rule := range catalog.rules[book.configID] {
			if book.level >= rule.level && !opened[rule.xueWeiID] {
				opened[rule.xueWeiID] = true
				p.jingmai.openedXueWei = append(p.jingmai.openedXueWei, rule.xueWeiID)
				added++
			}
		}
	}
	return added
}

// jingMaiXueWeiReEval 内功层数变化后的开穴增量收敛（登录/重入整推路径
// 用 reEvaluateXueWeiLocked 静默收敛）：新穴 View44 追加行 + 受影响经脉
// View45 MaxLevel 上调帧。无新增零帧。锁内纯编码。
func (p *playerActor) jingMaiXueWeiReEval(catalog *jingMaiCatalog) ([][]byte, bool) {
	xw := xueWeiCatalogOf()
	if xw == nil || catalog == nil {
		return nil, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	capsBefore := make(map[string]int32, len(p.jingmai.learned))
	for _, id := range p.jingmai.learned {
		capsBefore[id] = p.jingMaiCapLocked(catalog, id)
	}
	opened := make(map[string]bool, len(p.jingmai.openedXueWei))
	for _, id := range p.jingmai.openedXueWei {
		opened[id] = true
	}
	var frames [][]byte
	added := false
	for i := range p.progress.books {
		book := &p.progress.books[i]
		for _, rule := range xw.rules[book.configID] {
			if book.level < rule.level || opened[rule.xueWeiID] {
				continue
			}
			opened[rule.xueWeiID] = true
			p.jingmai.openedXueWei = append(p.jingmai.openedXueWei, rule.xueWeiID)
			def := xw.defs[rule.xueWeiID]
			if def.configID == "" {
				def = xueWeiDef{configID: rule.xueWeiID}
			}
			if frame, err := jingMaiXueWeiRowFrame(def, uint16(len(p.jingmai.openedXueWei))); err == nil {
				frames = append(frames, frame)
			}
			added = true
		}
	}
	if !added {
		return nil, false
	}
	for row, id := range p.jingmai.learned {
		capAfter := p.jingMaiCapLocked(catalog, id)
		if capAfter > capsBefore[id] {
			if frame, err := jingMaiViewCapUpdateFrame(uint16(row+1), capAfter); err == nil {
				frames = append(frames, frame)
			}
		}
	}
	return frames, true
}

// setJingMaiActLimit GM set_jingmai_act_count：可激活经脉条数 1..8
// （官方初始 3）。返回生效值；jmActCount 属性随 8 件套下发，调用方补帧。
func (p *playerActor) setJingMaiActLimit(value int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if value < 1 {
		value = 1
	}
	if value > jingMaiActLimitMax {
		value = jingMaiActLimitMax
	}
	p.jingmai.actLimit = value
	return value
}

// gmOpenJingMaiXueWei GM open_xuewei：越内功规则直开穴位（xueWeiID=
// "all" 时按穴位地图全开该经脉）。返回增量帧（View44 追加行 + View45
// MaxLevel 上调帧）与实际新开的穴 id 列表；未学经脉/未知穴返回错误。
func (p *playerActor) gmOpenJingMaiXueWei(catalog *jingMaiCatalog, jingMaiID, xueWeiID string) ([][]byte, []string, error) {
	xw := xueWeiCatalogOf()
	if catalog == nil || xw == nil {
		return nil, nil, fmt.Errorf("jingmai catalog unavailable")
	}
	def, ok := catalog.def(jingMaiID)
	if !ok {
		return nil, nil, fmt.Errorf("unknown jingmai %q", jingMaiID)
	}
	p.mu.Lock()
	if p.jingMaiRowLocked(jingMaiID) == 0 {
		p.mu.Unlock()
		return nil, nil, fmt.Errorf("jingmai %q not learned", jingMaiID)
	}
	capBefore := p.jingMaiCapLocked(catalog, jingMaiID)
	opened := make(map[string]bool, len(p.jingmai.openedXueWei))
	for _, id := range p.jingmai.openedXueWei {
		opened[id] = true
	}
	defer p.mu.Unlock()
	targets := make([]string, 0, len(def.xueWei))
	if xueWeiID == "all" {
		targets = append(targets, def.xueWei...)
	} else {
		if _, known := xw.defs[xueWeiID]; !known {
			return nil, nil, fmt.Errorf("unknown xuewei %q", xueWeiID)
		}
		targets = append(targets, xueWeiID)
	}
	var frames [][]byte
	var fresh []string
	viewReady := len(p.jingmai.openedXueWei) > 0
	for _, target := range targets {
		if opened[target] {
			continue
		}
		opened[target] = true
		p.jingmai.openedXueWei = append(p.jingmai.openedXueWei, target)
		fresh = append(fresh, target)
		// 客户端已有 View44 容器时直接追加行；首开（容器未建）留给
		// 调用方走 jingMaiXueWeiViewFrames 整推。
		if viewReady {
			rowDef := xw.defs[target]
			if frame, err := jingMaiXueWeiRowFrame(rowDef, uint16(len(p.jingmai.openedXueWei))); err == nil {
				frames = append(frames, frame)
			}
		}
	}
	if len(fresh) == 0 {
		return nil, nil, nil
	}
	if row := p.jingMaiRowLocked(jingMaiID); row > 0 {
		if capAfter := p.jingMaiCapLocked(catalog, jingMaiID); capAfter > capBefore {
			if frame, err := jingMaiViewCapUpdateFrame(uint16(row), capAfter); err == nil {
				frames = append(frames, frame)
			}
		}
	}
	return frames, fresh, nil
}

// jingMaiPropsLocked 玩家经脉属性 8 件套（Java JingMaiMessages.playerProps
// 同字段集：849/850 string + 819/851/852/855/856/857 i32；addObject 与
// stateProperties 复用）。jmActCount=可激活条数上限（官方初始 3 → 九阴
// 志最多 8，本服 GM set_jingmai_act_count 调节；旧硬编码 1 是数据异常
// 根源之一——客户端 tooltip「最多可激活{@1}条」直接读它）。须持 p.mu。
func (p *playerActor) jingMaiPropsLocked() []clientdata.IndexedProperty {
	totalLevel := int32(0)
	for _, id := range p.jingmai.learned {
		totalLevel += p.jingmai.progress[id].Level
	}
	actLimit := p.jingmai.actLimit
	if actLimit < 1 {
		actLimit = jingMaiActLimitDefault
	}
	return []clientdata.IndexedProperty{
		{Index: propCurJingMai, Name: "CurJingMai", Value: clientdata.StringValue(p.jingmai.cur)},
		{Index: propLastJingMai, Name: "LastJingMai", Value: clientdata.StringValue(p.jingmai.last)},
		{Index: propJingMaiTotLevel, Name: "JingMaiTotalLevel", Value: clientdata.Int32Value(totalLevel)},
		{Index: propJMActCount, Name: "jmActCount", Value: clientdata.Int32Value(actLimit)},
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

// jingMaiAttributeFrame 经脉加成变化后的安全增量帧（0x10）：只带
// playerProgress.properties() 的养成面板列（五维/HP·MP Add/招架/内防
// 等），不带 LogicState/State/CurSkillID——战斗中重推全量 vitalUpdate
// 会洗掉冲锋受理簇的接近态（2026-11-09 会话实证），经脉结算只刷数值列。
func (p *playerActor) jingMaiAttributeFrame() ([]byte, error) {
	p.mu.Lock()
	properties := p.progress.properties()
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, properties)
}

// jingMaiViewRowFrame View45 行帧（Java meridianRowFrame 同构 6+1 属性；
// TotalFillValue=每周天所需真气 750，非历史累计——Java totalFillPerLevel）。
// displayMax=当前阶梯上限（开穴提升；2026-11-15 官方式，MaxLevel 属性
// 即客户端显示分母，不再直发目录满上限）。
func jingMaiViewRowFrame(row jingMaiProgressRow, def jingMaiDef, slot uint16, displayMax int32) ([]byte, error) {
	if displayMax <= 0 {
		displayMax = def.maxLevel
	}
	properties := []serverViewProperty{
		viewString(clientdata.MustPlayerProperty("ConfigID").Index, def.configID),
		viewInt(propJMStaticData, def.staticData),
		viewInt(propJMItemType, def.itemType),
		viewInt(propJMVarPropNo, def.minVarPropNo+int32(row.Level)-1),
		viewByte(propJMLevel, byte(row.Level)),
		viewInt(propJMMaxLevel, displayMax),
		viewInt(propTotalFill, zhenQiPerWeek),
	}
	if row.Fill > 0 {
		properties = append(properties, viewInt(propCurFill, row.Fill))
	}
	return serverViewAdd(viewportJingMai, slot, properties)
}

// recordPlugRec plug_record(419) 官方 447 schema 列型 [7,3,7]（窄串,
// i32, 窄串）——激活经脉的插栓登记（外服私服 c.pcapng 4.7s：View45 激活行
// 每行伴随一条，对象域 obj=45(视图) owner=行号）。op14 事务伴随帧本服
// 全局缺失（对照文档 §11.2-A），此处仅补记录行。
const recordPlugRec uint16 = 419

// pushJingMaiPlugRecords 激活经脉的 plug_record 伴随行整推（View45 行
// 序定位 slot；未学/目录缺失跳过）。m150 激活时由调用方即时补单行。
func pushJingMaiPlugRecords(link sceneMessageConnection, player *playerActor, catalog *jingMaiCatalog, activated []string) error {
	if catalog == nil || len(activated) == 0 {
		return nil
	}
	player.mu.Lock()
	learned := append([]string(nil), player.jingmai.learned...)
	progress := make(map[string]jingMaiProgressRow, len(player.jingmai.progress))
	for id, row := range player.jingmai.progress {
		progress[id] = row
	}
	player.mu.Unlock()
	slotOf := make(map[string]uint16, len(learned))
	for index, id := range learned {
		slotOf[id] = uint16(index + 1)
	}
	for _, id := range activated {
		slot, ok := slotOf[id]
		if !ok {
			continue
		}
		def, hasDef := catalog.def(id)
		if !hasDef {
			def = jingMaiDef{configID: id, itemType: 1003}
		}
		frame, err := jingMaiPlugRecFrame(def, progress[id], slot)
		if err != nil {
			return fmt.Errorf("encode plug_record %s: %w", id, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write plug_record %s: %w", id, err)
		}
	}
	return nil
}

// jingMaiPlugRecFrame 激活经脉的 plug_record 伴随行（obj=45 owner=slot，
// 列值按外服私服语义推断：[configID, 等级, 空]——外服私服行内容值未解出样本，
// 待真机校准）。
func jingMaiPlugRecFrame(def jingMaiDef, row jingMaiProgressRow, slot uint16) ([]byte, error) {
	return serverRecordAddCells(uint32(viewportJingMai), uint32(slot), recordPlugRec, []recordCell{
		recordString(def.configID),
		recordInt(uint32(row.Level)),
		recordString(""),
	})
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
	caps := make(map[string]int32, len(learned))
	for _, id := range learned {
		caps[id] = p.jingMaiCapLocked(catalog, id)
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
		frame, err := jingMaiViewRowFrame(progress[id], def, uint16(index+1), caps[id])
		if err != nil {
			return nil, fmt.Errorf("encode jingmai view row %s: %w", id, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// ---- 穴位 View44（官方式穴位系统；行存在=已开，客户端 GetLearnID_XueWei
// 按行判定 OPEN/CLOSE） ----

// viewportXueWei VIEWPORT_XUEWEI（share/view_define.lua:32）。
const viewportXueWei uint16 = 44

// jingMaiXueWeiRowFrame View44 行帧（[ConfigID, StaticData, ItemType=1004,
// Level=1]；形状与 View45 行同构，真机校验）。
func jingMaiXueWeiRowFrame(def xueWeiDef, slot uint16) ([]byte, error) {
	properties := []serverViewProperty{
		viewString(clientdata.MustPlayerProperty("ConfigID").Index, def.configID),
		viewInt(propJMStaticData, def.staticData),
		viewInt(propJMItemType, 1004),
		viewByte(propJMLevel, 1),
	}
	return serverViewAdd(viewportXueWei, slot, properties)
}

// jingMaiXueWeiViewFrames View44 容器 + 已开穴位全行（登录/0x0C 重入整推；
// 行序=state.openedXueWei 追加序，增量开穴时按 len+1 追加行帧）。
func (p *playerActor) jingMaiXueWeiViewFrames() ([][]byte, error) {
	catalog := xueWeiCatalogOf()
	if catalog == nil {
		return nil, nil
	}
	p.mu.Lock()
	opened := append([]string(nil), p.jingmai.openedXueWei...)
	p.mu.Unlock()
	if len(opened) == 0 {
		return nil, nil
	}
	capacity := uint16(catalog.total)
	if capacity < 1 {
		capacity = 1
	}
	frames := [][]byte{serverCreateView(serverViewSpec{ID: viewportXueWei, Capacity: capacity})}
	for index, xwID := range opened {
		def, ok := catalog.defs[xwID]
		if !ok {
			def = xueWeiDef{configID: xwID}
		}
		frame, err := jingMaiXueWeiRowFrame(def, uint16(index+1))
		if err != nil {
			return nil, fmt.Errorf("encode xuewei view row %s: %w", xwID, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// jingMaiViewCapUpdateFrame 阶梯上限变化帧（0x10 isView=1 obj=45 owner=行号，
// 只带 MaxLevel）。
func jingMaiViewCapUpdateFrame(slot uint16, displayMax int32) ([]byte, error) {
	return sceneObjectProperties(uint32(viewportJingMai), uint32(slot), 1, []clientdata.IndexedProperty{
		{Index: propJMMaxLevel, Name: "MaxLevel", Value: clientdata.Int32Value(displayMax)},
	})
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
// TotalFillValue，否则单 CurFillValue）；当前修炼经脉处于激活位且发生
// 升级时追加全量属性帧（官方逐层加成变化）。
func (p *playerActor) jingMaiCultivateTick(catalog *jingMaiCatalog, now time.Time) [][]byte {
	if catalog == nil {
		return nil
	}
	p.mu.Lock()
	state := &p.jingmai
	if state.lastTickAt.IsZero() {
		state.lastTickAt = now
		p.mu.Unlock()
		return nil
	}
	today := now.Format("2006-01-02")
	if state.zhenQi.Day != today {
		state.zhenQi.Day = today
		state.zhenQi.Used = 0
	}
	minutes := int64(now.Sub(state.lastTickAt) / time.Minute)
	if minutes <= 0 {
		p.mu.Unlock()
		return nil
	}
	state.lastTickAt = state.lastTickAt.Add(time.Duration(minutes) * time.Minute)
	if state.cur == "" {
		p.mu.Unlock()
		return nil
	}
	frames, leveledActivated := p.applyZhenQiLocked(catalog, int64(zhenQiPerMinute)*minutes)
	p.mu.Unlock()
	if leveledActivated {
		if attribute, err := p.jingMaiAttributeFrame(); err == nil {
			frames = append(frames, attribute)
		}
	}
	return frames
}

// jingMaiCombatFeed 战斗真气入账（Java PlayerMeridian.addCombatZhenQi：
// 伤害合计 × zhenqi.ini damage_ratio=0.1，向下取整保底 1）。命中结算后
// 按整招累计调用，返回 View45 行帧（CurFillValue/升级帧）。变化才产帧；
// 升级且激活位时追加全量属性帧（同 cultivateTick）。
func (p *playerActor) jingMaiCombatFeed(damage int32) [][]byte {
	if damage <= 0 || globalJingMaiCatalog == nil {
		return nil
	}
	p.mu.Lock()
	state := &p.jingmai
	if state.cur == "" {
		p.mu.Unlock()
		return nil
	}
	now := time.Now()
	today := now.Format("2006-01-02")
	if state.zhenQi.Day != today {
		state.zhenQi.Day = today
		state.zhenQi.Used = 0
	}
	accrue := int64(math.Max(1, math.Floor(float64(damage)*jingMaiCombatZhenQiRatio)))
	frames, leveledActivated := p.applyZhenQiLocked(globalJingMaiCatalog, accrue)
	p.mu.Unlock()
	if leveledActivated {
		if attribute, err := p.jingMaiAttributeFrame(); err == nil {
			frames = append(frames, attribute)
		}
	}
	return frames
}

// applyZhenQiLocked 真气入账公共体（Java applyZhenQi 同构；被动 tick 与
// 战斗折算同池同闸）：每日硬闸开闸时按剩余额度截断（额度耗尽零帧
// 静默，关闸不限量——ZhenQiDayValue 仅展示口径）；溢出连升封顶当前
// 阶梯上限（开穴不足不越 9/36/…档）；变化才产帧。第二返回值=当前
// 修炼经脉处于激活位且发生升级（调用方解锁后补全量属性帧）。须持
// p.mu。
func (p *playerActor) applyZhenQiLocked(catalog *jingMaiCatalog, accrue int64) ([][]byte, bool) {
	state := &p.jingmai
	if accrue <= 0 {
		return nil, false
	}
	if state.cap.Enabled {
		headroom := int64(state.cap.Value) - int64(state.zhenQi.Used)
		if headroom <= 0 {
			return nil, false // 当日额度已用尽：硬闸拦截（零帧静默）
		}
		if accrue > headroom {
			accrue = headroom
		}
	}
	slot := p.jingMaiRowLocked(state.cur)
	if slot == 0 {
		return nil, false // 未学经脉不结算（150/1 已闸；旧档防御）
	}
	def, ok := catalog.def(state.cur)
	if !ok || def.maxLevel <= 0 {
		return nil, false
	}
	cap := p.jingMaiCapLocked(catalog, state.cur)
	activated := false
	for _, id := range state.activated {
		if id == state.cur {
			activated = true
			break
		}
	}
	row := state.progress[state.cur]
	var frames [][]byte
	consumed := int64(0)
	leveled := false
	for accrue > 0 && row.Level < cap {
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
		leveled = true
		if frame, err := jingMaiViewUpdateFrame(uint16(slot), row, true); err == nil {
			frames = append(frames, frame)
		}
	}
	if consumed <= 0 {
		return nil, false
	}
	row.Total += int32(consumed)
	if state.progress == nil {
		state.progress = make(map[string]jingMaiProgressRow)
	}
	state.progress[state.cur] = row
	state.zhenQi.Used += int32(consumed)
	// 升级改官方逐层加成：激活位即时重算并同步推导血/蓝上限（全量
	// 属性帧由调用方解锁后追加）。
	if leveled && activated {
		p.recomputeJingMaiBonusLocked(catalog)
		p.syncEquippedNeiGongResourcesLocked()
	}
	// 尾随玩家经脉属性帧（Java meridianTickOnce/SkillCastHandler 同构）：
	// ZQPLValue（今日已获）与 JingMaiTotalLevel（升级后）随入账刷新，
	// 客户端 tips「经脉总等级」/浮窗今日真气条才有增量更新。锁内纯编码。
	if propsFrame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, p.jingMaiPropsLocked()); err == nil {
		frames = append(frames, propsFrame)
	} else {
		log.Printf("jingmai: encode props frame after accrual: %v", err)
	}
	return frames, leveled && activated
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
		//（激活位=当前修炼经脉）。官方式条数闸：余量不足则不自动占位
		//（2026-11-15；旧版无条件补行）。
		if previous == "" && id != "" && len(player.jingMaiActivatedSnapshot()) == 0 {
			if player.jingMaiActHeadroom() > 0 {
				if player.activateJingMai(id) {
					if err := player.rewriteJingMaiActiveRows(link); err != nil {
						return true, fmt.Errorf("rewrite active_jingmai_rec: %w", err)
					}
					player.refreshJingMaiBonus(globalJingMaiCatalog)
					log.Printf("%s: activated jingmai id=%s via active_jingmai_rec", remote, id)
				}
			} else {
				log.Printf("%s: skip auto activate jingmai id=%s (act limit reached)", remote, id)
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
		if containsString(player.jingMaiActivatedSnapshot(), id) {
			log.Printf("%s: active jingmai already present id=%s", remote, id)
			break
		}
		// 官方式互斥（jingmai_active_exclude.ini：跷↔跷·隐）：激活前先
		// 摘已在列的互斥经脉（腾出的激活位参与下方条数闸）。
		for _, excluded := range jingMaiExcludesOf(id) {
			if !containsString(player.jingMaiActivatedSnapshot(), excluded) {
				continue
			}
			if _, removed := player.deactivateJingMai(excluded); removed {
				log.Printf("%s: deactivated excluded jingmai id=%s (for %s)", remote, excluded, id)
			}
		}
		// 官方式条数闸：jmActCount 上限（初始 3，GM 可调 1..8）。客户端
		// 无对应错误键，服务端静默拒绝（与官服一致）。
		if player.jingMaiActHeadroom() <= 0 {
			log.Printf("%s: reject activate jingmai id=%s (act limit reached)", remote, id)
			break
		}
		if player.activateJingMai(id) {
			log.Printf("%s: activated jingmai id=%s via active_jingmai_rec", remote, id)
		}
	case jingMaiSubDeactivate:
		if _, removed := player.deactivateJingMai(id); removed {
			log.Printf("%s: deactivated jingmai id=%s via active_jingmai_rec", remote, id)
		} else {
			log.Printf("%s: active jingmai not present id=%s", remote, id)
		}
	}
	// active_jingmai_rec 整表重写（tersure 同模式）：单行加帧的源行 0 在
	// 引擎侧是「插到行 0」语义，与服务端追加序错位会让删行打错行——
	// 「关不掉/激活显示错乱」根因（2026-11-15 真机日志逐事件重建实证）。
	// 无论本次是否受理都整表收敛，客户端 UI 恒向服务端状态看齐。
	if err := player.rewriteJingMaiActiveRows(link); err != nil {
		return true, fmt.Errorf("rewrite active_jingmai_rec: %w", err)
	}
	// 官方式激活加成：激活集变化即重算逐层属性并同步推导血/蓝上限，
	// 全量属性帧随 8 件套一起下发（五维/HP/MP/招架/内防实时可见）。
	player.refreshJingMaiBonus(globalJingMaiCatalog)
	frame, err := player.jingMaiPropsFrame()
	if err != nil {
		return true, fmt.Errorf("encode jingmai vitality update: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("write jingmai vitality update: %w", err)
	}
	if attribute, attrErr := player.jingMaiAttributeFrame(); attrErr == nil {
		if err := link.WriteFrame(attribute); err != nil {
			return true, fmt.Errorf("write jingmai attribute frame: %w", err)
		}
	}
	return true, nil
}

// pushJingMaiFrames 进世界授予/重放（Java LoginFlow meridian 授予块：
// ClientReady 后 46/43/40 同位）。首次进世界种全套 starter 1 级（老档
// 14 条幂等补齐）；官方式穴位收敛（activerule 静默开穴）+ 激活加成重算
// 后整推：View44（已开穴位）→ View45（阶梯上限）→ plug_record →
// active_jingmai_rec → 8 件套。0x0C 重入时玩家对象已清空，随本函数
// 整体重放。
func pushJingMaiFrames(link sceneMessageConnection, player *playerActor, catalog *jingMaiCatalog) error {
	player.ensureJingMaiLearned(catalog)
	player.jingMaiPrepareLogin(catalog)
	xueWeiFrames, err := player.jingMaiXueWeiViewFrames()
	if err != nil {
		return fmt.Errorf("encode jingmai xuewei view: %w", err)
	}
	for _, frame := range xueWeiFrames {
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write jingmai xuewei view: %w", err)
		}
	}
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
	// 外服私服 4.7s 补齐：激活经脉每行伴随 plug_record#419（c.pcapng View45
	// 激活行逐行成对实证，此前零下发——对照文档 §11.1-14）。
	if plugErr := pushJingMaiPlugRecords(link, player, catalog, activated); plugErr != nil {
		return plugErr
	}
	// active_jingmai_rec 整表重写（单帧批量，官服登录样本同构；行数
	// 记账随 0x0C 重入复位）。
	if err := player.rewriteJingMaiActiveRows(link); err != nil {
		return fmt.Errorf("replay active_jingmai_rec: %w", err)
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

// resetJingMaiActiveRows 0x0C 重入时复位 active_jingmai_rec 行数记账
// （ServerExitScene 销毁玩家对象及其全部记录行，客户端表已空）。
func (p *playerActor) resetJingMaiActiveRows() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jingmaiActiveRowsSent = 0
}

// rewriteJingMaiActiveRows active_jingmai_rec 整表重写：按会话记账降序
// DelRow 清表（引擎删行后紧凑，高位先删避免错位），再单帧批量重加全量
// 激活集（官服 c.pcapng 登录批量帧同构）。客户端行序与内容解耦，
// FindRecordRow 按内容命中——增删/互斥/拒绝任一路径后统一收敛。
func (p *playerActor) rewriteJingMaiActiveRows(link sceneMessageConnection) error {
	p.mu.Lock()
	activated := append([]string(nil), p.jingmai.activated...)
	sent := p.jingmaiActiveRowsSent
	p.mu.Unlock()
	for row := sent - 1; row >= 0; row-- {
		if err := link.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordActiveJingMai, uint16(row))); err != nil {
			return fmt.Errorf("clear active_jingmai_rec row=%d: %w", row, err)
		}
	}
	if len(activated) > 0 {
		if err := link.WriteFrame(serverRecordAddStringRows(playerObjectID, playerOwnerID, recordActiveJingMai, activated)); err != nil {
			return fmt.Errorf("rewrite active_jingmai_rec rows=%d: %w", len(activated), err)
		}
	}
	p.mu.Lock()
	p.jingmaiActiveRowsSent = len(activated)
	p.mu.Unlock()
	return nil
}

// jingMaiActHeadroom 可激活余量（jmActCount 上限 − 已激活数，负数视 0）。
func (p *playerActor) jingMaiActHeadroom() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	limit := p.jingmai.actLimit
	if limit < 1 {
		limit = jingMaiActLimitDefault
	}
	headroom := int(limit) - len(p.jingmai.activated)
	if headroom < 0 {
		return 0
	}
	return headroom
}

// jingMaiExcludesOf 激活互斥查询（jingmai_active_exclude.ini；无表空集）。
func jingMaiExcludesOf(id string) []string {
	catalog := xueWeiCatalogOf()
	if catalog == nil {
		return nil
	}
	return catalog.excludes[id]
}

// jingMaiPrepareLogin 整推前的静默收敛（pushJingMaiFrames 专用）：内功
// activerule 开穴（只增不减）→ 激活加成重算 → 推导血/蓝上限同步。帧由
// 随后的整体重放承载，这里不产增量帧。
func (p *playerActor) jingMaiPrepareLogin(catalog *jingMaiCatalog) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if added := p.reEvaluateXueWeiLocked(); added > 0 {
		log.Printf("jingmai: auto-opened %d xuewei by neigong activerule", added)
	}
	p.recomputeJingMaiBonusLocked(catalog)
	p.syncEquippedNeiGongResourcesLocked()
}
