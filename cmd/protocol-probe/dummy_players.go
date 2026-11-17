package main

// 假人系统（dummy players，自 9yin-go-server2 移植；数据源按 server1 方向
// 改为 MySQL 权威存档 migrations/0021_dummy_players_*.sql.gz）。
//
// 实现路线（与 server2 同源，官服抓包报告《周围玩家数据模型显示抓取报告》）：
//   - 复用 0x0D 玩家归档（Type=2）+ 0x1F ServerLocation，属性下标按名经
//     0x09 协商表解析（fork-only 行按名缺席整组跳过，与 playerActor.addObjectAs
//     同一约定）；
//   - 假人在场景 populate 时**全量注册进共享 sceneRun**（display-only：
//     EntityPlayer 类实体、无 Vitals/Brain，被战斗/仇恨/技能选靶天然排除），
//     每连接视口按「预算准入」决定各自可见子集（dummy_scene.go）；
//   - 移动走真人成员广播方言：步起 0x20 serverMoving（server_moving.go），
//     到达 0x1F serverLocation；
//   - 配置仅来自 MySQL（server_dummy_players + game_cards），DB 编辑经
//     COUNT 探测缓存过期生效；表空/缺库 = 无假人（fail-soft）。

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// dummyObjectIDBase 假人对象 ID 段基址。取 ≥0x01000000 的独立段：forkNPCID
// 对该段直通不再加 0x11000000 前缀；与 NPC 段（0x11000002..）、远程成员
// wireID（无前缀低位）、玩家本体（playerObjectID∈0x11xxxxxx）均不碰撞。
const dummyObjectIDBase uint32 = 0x0E000000

// dummyObjectIDSpan 假人段容量上界（单场景匹配量远小于此）。
const dummyObjectIDSpan uint32 = 0x00F00000

func isDummyObjectID(id uint32) bool {
	return id >= dummyObjectIDBase && id < dummyObjectIDBase+dummyObjectIDSpan
}

// ---- 配置结构（与 migrations/0021 的列一一对应；JSON 列反序列化用 tag）----

type dummyPosition struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Z      float32 `json:"z"`
	Orient float32 `json:"orient"`
}

type dummyAppearanceCfg struct {
	Photo        string `json:"photo"`
	Face         string `json:"face"`
	Hair         string `json:"hair"`
	Hat          string `json:"hat"`
	Cloth        string `json:"cloth"`
	Pants        string `json:"pants"`
	Shoes        string `json:"shoes"`
	Weapon       string `json:"weapon"`
	WeaponName   string `json:"weaponName"`
	WeaponMode   string `json:"weaponMode"`
	ActionSet    string `json:"actionSet"`
	WeaponConfig string `json:"weaponConfig"`
}

type dummyPlayer struct {
	Name string `json:"name"`
	Sex  byte   `json:"sex"` // 0=男 1=女
	// School factionProperties 语义：原始门派/force_*/newschool_*。
	School string `json:"school"`
	Level  int    `json:"level"`
	// TitleID 称号 ID（0=无）。
	TitleID int `json:"titleId"`
	// PowerLevel 实力等级（0-160）：归档映射 LevelTitle 称号（roleListTitle，
	// 与选人界面同链路）与 StrengthNextStage 阶段（strengthStageInfo）。
	PowerLevel int `json:"powerLevel"`
	// NeiGong 成名内功名（StrengthNeiGongName），可空。
	NeiGong string `json:"neiGong"`
	// Character 性格标签（0正常 1侠 2恶 3狂 4邪）：按官服公式反推一致的
	// Justice/Evil/FreeBase 数值并下发 CharacterFlag/CharacterValue。
	Character int `json:"character"`
	// Card 风物志卡编号（game_cards.card_id）。>0 时按卡 ModelData 覆盖
	// Cloth/Hat/Pants/Shoes/Weapon 部件并下发 ShowEquipType=2（时装档）。
	Card int32 `json:"card"`
	// Patrol 巡逻路点（世界坐标）：≥2 个时假人沿路点往返（步起 0x20）。
	Patrol []dummyPosition `json:"patrol"`
	// PatrolRandom 随机人流巡逻：每到一站从场景官方 NPC 坐标池随机挑
	// 下一个目的地（"随机 NPC 起点到随机 NPC 终点"），无限循环；固定
	// 路点优先级更高。
	PatrolRandom bool `json:"patrolRandom"`
	// MoveMode 巡逻移动方式："walk"=步行 2.6m/s，""/"run"=跑步 6m/s；
	// yufeng=true 恒为御风疾跑 11m/s。
	MoveMode string `json:"moveMode"`
	// Yufeng 御风状态：归档附加 buf_ride_yufeng（StaticData=10293）。
	Yufeng bool `json:"yufeng"`
	// NeiGongBuff 内功 buff 档（neigong.ini QName，如 ng_wd_001）：归档按
	// 真玩家装备内功同一条链路填充 BufferInfo 槽 4 + BufferListStr。
	NeiGongBuff string `json:"neiGongBuff"`
	// Scene 过滤：匹配场景 Config（ini\scene\xxx）或 Resource（xxx）；
	// 空 = 所有场景。
	Scene  string             `json:"scene"`
	Action string             `json:"action"` // 出生动作（State），默认 stand
	Pos    dummyPosition      `json:"position"`
	Look   dummyAppearanceCfg `json:"appearance"`
}

func (d dummyPlayer) matchesScene(sceneConfig, sceneResource string) bool {
	scene := strings.TrimSpace(d.Scene)
	if scene == "" {
		return true
	}
	return strings.EqualFold(scene, sceneConfig) || strings.EqualFold(scene, sceneResource)
}

// ---- MySQL 装载（gmNameCache 同款 fail-soft 后台装载 + COUNT 探测热更）----

// dummyCardEntry 是 game_cards 的假人侧投影（外观部件覆盖只读
// male/female_model_data）。
type dummyCardEntry struct {
	Name            string
	FemaleModelData string
	MaleModelData   string
}

type dummyStoreState struct {
	mu      sync.RWMutex
	dummies []dummyPlayer
	cards   map[int32]dummyCardEntry
	loaded  bool
	rows    int64
	loadErr error
}

var installedDummyStore = &dummyStoreState{}

// dummyStoreQueryTimeout 单次全表装载的超时（3.2 万行 + 1 千卡）。
const dummyStoreQueryTimeout = 60 * time.Second

// dummyStoreProbeInterval 装载成功后的行数探测周期：COUNT(*) 变化即全量
// 重读——DB 侧编辑（改坐标/场景/外观）最多一个周期后生效。
const dummyStoreProbeInterval = 5 * time.Minute

// startDummyStore 起后台装载循环：立即装一次（含 game_cards），失败按
// 间隔重试；成功后周期 COUNT 探测驱动热更。db 为 nil 时记日志放弃。
func startDummyStore(db *sql.DB, interval time.Duration) {
	if db == nil {
		log.Printf("dummy players: no database; crowd dummies disabled")
		return
	}
	go func() {
		for {
			if err := installedDummyStore.load(db); err != nil {
				installedDummyStore.mu.Lock()
				installedDummyStore.loadErr = err
				installedDummyStore.mu.Unlock()
				log.Printf("dummy players: load failed (retry in %s): %v", interval, err)
				time.Sleep(interval)
				continue
			}
			installedDummyStore.mu.RLock()
			rows := installedDummyStore.rows
			installedDummyStore.mu.RUnlock()
			log.Printf("dummy players: %d row(s) loaded from MySQL (cards=%d)", rows, len(dummyCardCatalogSnapshot()))
			break
		}
		go installedDummyStore.probeLoop(db)
	}()
}

func (s *dummyStoreState) load(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), dummyStoreQueryTimeout)
	defer cancel()
	dummies, err := s.loadDummies(ctx, db)
	if err != nil {
		return err
	}
	cards, err := s.loadCards(ctx, db)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.dummies = dummies
	s.cards = cards
	s.rows = int64(len(dummies))
	s.loaded = true
	s.loadErr = nil
	s.mu.Unlock()
	return nil
}

func (s *dummyStoreState) loadDummies(ctx context.Context, db *sql.DB) ([]dummyPlayer, error) {
	rows, err := db.QueryContext(ctx, `SELECT name, sex, school, level, power_level, character_type,
		scene, action, neigong_buff, position, appearance, title_id,
		COALESCE(nei_gong, ''), card, patrol, patrol_random, COALESCE(move_mode, ''), yufeng
		FROM server_dummy_players ORDER BY dummy_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	dummies := make([]dummyPlayer, 0, 1024)
	for rows.Next() {
		var (
			d            dummyPlayer
			position     sql.NullString
			appearance   sql.NullString
			patrol       sql.NullString
			characterTy  int
			patrolRandom int
			yufeng       int
		)
		if err := rows.Scan(&d.Name, &d.Sex, &d.School, &d.Level, &d.PowerLevel, &characterTy,
			&d.Scene, &d.Action, &d.NeiGongBuff, &position, &appearance, &d.TitleID, &d.NeiGong,
			&d.Card, &patrol, &patrolRandom, &d.MoveMode, &yufeng); err != nil {
			return nil, err
		}
		d.Character = characterTy
		d.PatrolRandom = patrolRandom != 0
		d.Yufeng = yufeng != 0
		if position.Valid && position.String != "" {
			_ = json.Unmarshal([]byte(position.String), &d.Pos)
		}
		if appearance.Valid && appearance.String != "" {
			_ = json.Unmarshal([]byte(appearance.String), &d.Look)
		}
		if patrol.Valid && patrol.String != "" && patrol.String != "null" {
			_ = json.Unmarshal([]byte(patrol.String), &d.Patrol)
		}
		dummies = append(dummies, d)
	}
	return dummies, rows.Err()
}

func (s *dummyStoreState) loadCards(ctx context.Context, db *sql.DB) (map[int32]dummyCardEntry, error) {
	rows, err := db.QueryContext(ctx, `SELECT card_id, name, female_model_data, male_model_data FROM game_cards`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := make(map[int32]dummyCardEntry, 128)
	for rows.Next() {
		var (
			id   int32
			card dummyCardEntry
		)
		if err := rows.Scan(&id, &card.Name, &card.FemaleModelData, &card.MaleModelData); err != nil {
			return nil, err
		}
		cards[id] = card
	}
	return cards, rows.Err()
}

// probeLoop 周期 COUNT(*) 探测：行数变化即全量重读（保持热更语义）。
func (s *dummyStoreState) probeLoop(db *sql.DB) {
	ticker := time.NewTicker(dummyStoreProbeInterval)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var count int64
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM server_dummy_players").Scan(&count)
		cancel()
		if err != nil {
			continue
		}
		s.mu.RLock()
		previous := s.rows
		changed := s.rows != count
		s.mu.RUnlock()
		if !changed {
			continue
		}
		if err := s.load(db); err != nil {
			log.Printf("dummy players: reload failed: %v", err)
			continue
		}
		log.Printf("dummy players: row count %d → %d, reloaded", previous, count)
	}
}

// dummyPlayersSnapshot 返回已装载的假人配置（未装载/装载失败返回 nil）。
func dummyPlayersSnapshot() []dummyPlayer {
	installedDummyStore.mu.RLock()
	defer installedDummyStore.mu.RUnlock()
	if !installedDummyStore.loaded {
		return nil
	}
	return installedDummyStore.dummies
}

func dummyCardCatalogSnapshot() map[int32]dummyCardEntry {
	installedDummyStore.mu.RLock()
	defer installedDummyStore.mu.RUnlock()
	return installedDummyStore.cards
}

// ---- 外观（默认值与 resolveRoleVisual 同源；服装经 equipmentModels 按性别解析）----

type dummyVisual struct {
	sex                            byte
	photo, face, hair, hat         string
	cloth, pants, shoes, actionSet string
	weapon                         string
}

// defaultDummyFace 与 resolveRoleVisual 的兜底脸型一致（appearance.go）。
const defaultDummyFace = "333333333333333333333333333333333333333333X38\x01"

func (d dummyPlayer) dummyVisual() dummyVisual {
	sex := d.Sex
	if sex > 1 {
		sex = 0
	}
	photo := d.Look.Photo
	if photo == "" {
		if sex == 1 {
			photo = `icon\players\girl01_create.png`
		} else {
			photo = `icon\players\boy01_create.png`
		}
	}
	face := d.Look.Face
	if face == "" {
		face = defaultDummyFace
	}
	hair := d.Look.Hair
	if hair == "" {
		// 发型按性别取默认（女款 g_hair；男款在女模型上不渲染）。
		if sex == 1 {
			hair = `obj\char\g_hair\g_hair1`
		} else {
			hair = `obj\char\b_hair\b_hair1`
		}
	}
	cloth := d.Look.Cloth
	if cloth == "" {
		if sex == 1 {
			cloth = "cloth_g0001"
		} else {
			cloth = "cloth_b0001"
		}
	}
	pants := d.Look.Pants
	if pants == "" {
		if sex == 1 {
			pants = "pants_g0001"
		} else {
			pants = "pants_b0001"
		}
	}
	shoes := d.Look.Shoes
	if shoes == "" {
		if sex == 1 {
			shoes = "shoes_g0001"
		} else {
			shoes = "shoes_b0001"
		}
	}
	visual := dummyVisual{
		sex:       sex,
		photo:     photo,
		face:      face,
		hair:      hair,
		hat:       equipmentModels.resolve(d.Look.Hat, sex),
		cloth:     equipmentModels.resolve(cloth, sex),
		pants:     equipmentModels.resolve(pants, sex),
		shoes:     equipmentModels.resolve(shoes, sex),
		actionSet: d.Look.ActionSet,
		weapon:    d.Look.Weapon,
	}
	if visual.actionSet == "" {
		visual.actionSet = "0h"
	}
	// weaponConfig：道具 configID 自动解析武器模型 + 动作集（equipment.ini
	// → itemartstatic.ini，与真玩家 weaponVisualOf 同表）。
	if config := strings.TrimSpace(d.Look.WeaponConfig); config != "" {
		weaponName := equipmentModels.resolve(config, sex)
		if visual.weapon == "" {
			visual.weapon = weaponName
		}
		if resolvedSet := equipmentModels.resolveActionSet(config); resolvedSet != "" && d.Look.ActionSet == "" {
			visual.actionSet = resolvedSet
		}
	}
	// 武器资源名原样下发裸名（server2 v12 定案：客户端按风物志武器名解析，
	// 归一化为完整 idres 路径会让武器整只消失）；配置显式 WeaponName 优先。
	if visual.weapon == "" {
		visual.weapon = d.Look.WeaponName
	}
	return visual
}

// applyCardOverrides 把假人配置的风物志卡（game_cards）外观部件覆盖到
// visual 上，返回应下发的 ShowEquipType（有卡命中=2 时装档，无卡/未命中=0）。
// 卡声明了该部位才覆盖，未声明部位保持下层外观。
func (d dummyPlayer) applyCardOverrides(visual *dummyVisual) int32 {
	if d.Card <= 0 {
		return 0
	}
	slots, ok := dummyCardModelSlots(d.Card, visual.sex)
	if !ok {
		log.Printf("dummy players: card %d missing from game_cards or no model data; keeping base look", d.Card)
		return 0
	}
	if c, ok := slots["Cloth"]; ok {
		visual.cloth = c
	}
	if h, ok := slots["Hat"]; ok {
		visual.hat = h
	}
	if p, ok := slots["Pants"]; ok {
		visual.pants = p
	}
	if s, ok := slots["Shoes"]; ok {
		visual.shoes = s
	}
	if w, ok := slots["Weapon"]; ok {
		visual.weapon = w
		// 卡武器到手必须同步动作集：否则 actionSet 停留 "0h"（徒手动集），
		// 客户端手里有武器却播徒手动作。
		visual.actionSet = weaponActionSetFor(deriveWeaponMode(w))
	}
	return showEquipTypeShizhuang
}

// showEquipTypeShizhuang 与真玩家时装档同值（pushGearVisual 卡牌覆盖链）。
const showEquipTypeShizhuang int32 = 2

// dummyCardModelSlots 解析卡 ModelData（"Part,model;Part2,model2"）为部件
// → 模型名（server2 cardCatalog.modelSlotsFor 同构，数据源 game_cards）。
func dummyCardModelSlots(cardID int32, sex byte) (map[string]string, bool) {
	cards := dummyCardCatalogSnapshot()
	if cards == nil {
		return nil, false
	}
	card, exists := cards[cardID]
	if !exists {
		return nil, false
	}
	data := card.MaleModelData
	if sex == 1 {
		data = card.FemaleModelData
	}
	slots := make(map[string]string)
	for _, seg := range strings.Split(data, ";") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		comma := strings.IndexByte(seg, ',')
		if comma < 0 {
			continue
		}
		part := strings.TrimSpace(seg[:comma])
		model := strings.TrimSpace(seg[comma+1:])
		if part == "" || model == "" {
			continue
		}
		slots[part] = model
	}
	if len(slots) == 0 {
		return nil, false
	}
	return slots, true
}

// deriveWeaponMode 按武器资源名推断持械 mode（server2 card_player.go 同表）：
// 单手族 *_1_1 等；未知返回空串。
func deriveWeaponMode(weapon string) string {
	base := weapon
	if slash := strings.LastIndexAny(base, `\/`); slash >= 0 {
		base = base[slash+1:]
	}
	switch {
	case strings.HasPrefix(base, "sword"), strings.HasPrefix(base, "blade"), strings.HasPrefix(base, "knife"):
		return "sword1_1"
	case strings.HasPrefix(base, "dsword"), strings.HasPrefix(base, "lstuff"), strings.HasPrefix(base, "stuff"), strings.HasPrefix(base, "dknife"), strings.HasPrefix(base, "dax"), strings.HasPrefix(base, "bow"):
		return "dsword1_1"
	default:
		return ""
	}
}

// weaponActionSetFor 武器 mode → 动作集手数（server2 同表）：单手=1h，
// 双手长柄/重剑/弓=2h，未知/徒手=0h。
func weaponActionSetFor(mode string) string {
	switch mode {
	case "sword1_1", "blade1_1", "knife1_1":
		return "1h"
	case "dsword1_1", "lstuff1_1", "stuff1_1", "dknife1_1", "dax1_1", "bow1_1":
		return "2h"
	default:
		return "0h"
	}
}

// ---- 侠恶狂邪称号（官服公式，server2 原样移植）----
//
// 官服判定规则（2026-09-12 抓包对照）：
//   ① 侠义值+恶行值+桀骜值 > 100 才出行为标签，否则不显示（泛泛之辈）。
//   ② 侠/恶判定：侠恶条件 = 侠义值+恶行值 > 桀骜值×2 且 大值/小值 > 2。
//      满足时取大者定阵营，称号档按计算值=|侠义值-恶行值|。
//   ③ 侠恶条件不满足时判狂邪：实际狂邪值=(侠义值+恶行值)/2+桀骜值定档，
//      侠义值≥恶行值显示"狂"，否则"邪"。
//   ④ 称号档（计算值区间，官服同表）：
//      100-199 江湖小侠/心狠手辣/年少轻狂/亦正亦邪
//      200-449 行侠好义/作恶多端/放浪形骸/目带邪光
//      450-899 侠肝义胆/穷凶极恶/狂动一方/邪气逼人
//      900-1799 江湖豪侠/十恶不赦/狂放不羁/邪名一时
//      1800-3599 侠名远播/凶神恶煞/江湖狂人/邪魔歪道
//      3600-7199 仁义无双/无恶不作/天地狂客/天弃鬼邪
//      7200+ 侠之大者/恶贯满盈/逍遥游仙/在世修罗
//   ⑤ 单项数值上限 50000。

type dummyAlignmentTier struct{ min, max int32 }

var dummyAlignmentTiers = [...]dummyAlignmentTier{
	{100, 199},
	{200, 449},
	{450, 899},
	{900, 1799},
	{1800, 3599},
	{3600, 7199},
	{7200, 49998}, // 50000 单项上限内留出对方值的空间
}

// dummyAlignmentTierWeights 低档多、高档稀有（官服观感：满街"江湖小侠"）。
var dummyAlignmentTierWeights = [...]int{40, 25, 15, 10, 5, 3, 2}

// dummyAlignmentValues 按假人名字哈希**确定性**生成 (侠义, 恶行, 桀骜, 计算值)：
// 同一假人每次进场景数值不变。character: 0无 1侠 2恶 3狂 4邪。
func dummyAlignmentValues(name string, character int) (justice, evil, freeBase, charValue int32) {
	if character <= 0 {
		// 无标签：三围总和 ≤ 100（官服阈值①之内）。
		return int32(dummyRandIntn(41)), int32(dummyRandIntn(41)), int32(dummyRandIntn(20)), 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	rng := rand.New(rand.NewSource(int64(h.Sum32())))
	total := 0
	for _, w := range dummyAlignmentTierWeights {
		total += w
	}
	pick := rng.Intn(total)
	tierIdx := 0
	for i, w := range dummyAlignmentTierWeights {
		if pick < w {
			tierIdx = i
			break
		}
		pick -= w
	}
	tier := dummyAlignmentTiers[tierIdx]
	value := tier.min + rng.Int31n(tier.max-tier.min+1)
	switch character {
	case 1, 2: // 侠/恶：满足侠恶条件，大值=小值+计算值
		upper := value - 1
		if cap := int32(50000) - value - 1; cap < upper {
			upper = cap
		}
		if upper < 1 {
			upper = 1
		}
		small := 1 + rng.Int31n(upper)
		if character == 1 {
			justice, evil = value+small, small
		} else {
			justice, evil = small, value+small
		}
		freeBase = rng.Int31n((justice + evil) / 2)
		charValue = value
	case 3, 4: // 狂/邪：侠=恶 → 大值/小值=1≤2 → 判狂邪
		e := rng.Int31n(value + 1)
		if character == 3 {
			justice, evil = e, e // 侠义≥恶行 → 狂
		} else {
			justice, evil = e, e+1 // 侠义<恶行 → 邪
		}
		freeBase = value - e
		charValue = value
	}
	return justice, evil, freeBase, charValue
}

func dummyRandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	return rand.Intn(n)
}

// ---- 实力阶段（官服表，server2 原样移植；称号走 roleListTitle）----

var dummyStrengthStages = []struct {
	min  int32
	name string
}{
	{0, "不堪一击"},
	{6, "初学乍练"},
	{11, "初窥门径"},
	{16, "略有小成"},
	{21, "驾轻就熟"},
	{26, "融会贯通"},
	{31, "炉火纯青"},
	{36, "出类拔萃"},
	{41, "神乎其技"},
	{46, "出神入化"},
	{51, "傲视群雄"},
	{56, "登峰造极"},
	{61, "无与伦比"},
	{66, "所向披靡"},
	{71, "一代宗师"},
	{76, "神功盖世"},
	{81, "举世无双"},
	{86, "惊世骇俗"},
	{91, "撼天动地"},
	{96, "震古铄今"},
	{101, "超凡入圣"},
	{106, "威镇寰宇"},
	{111, "空前绝后"},
	{116, "天人合一"},
	{121, "深藏不露"},
	{136, "深不可测"},
	{151, "返璞归真"},
}

// dummyStrengthStageInfo 按实力点数返回下一阶段名与还差点数（已封顶返回
// 「暂未开放」/0）。
func dummyStrengthStageInfo(points int32) (nextName string, pointsToNext int32) {
	idx := 0
	for i, s := range dummyStrengthStages {
		if points >= s.min {
			idx = i
		}
	}
	if idx+1 < len(dummyStrengthStages) {
		next := dummyStrengthStages[idx+1]
		return next.name, next.min - points
	}
	return "暂未开放", 0
}

// ---- buff 槽（内功槽 4 + 御风槽 24；线型与真玩家 bufferInfoLocked 同构）----

// dummyYufengBuffSlot 御风状态占用的 BufferInfo 槽位（道具 buff 动态池
// 11..24 的末位，不与常驻槽冲突）。
const dummyYufengBuffSlot uint16 = 24

// 御风载体 buff（server2 big_qinggong/yufeng_wind 同值）。
const (
	yufengWindBuffConfigID   = "buf_ride_yufeng"
	yufengWindBuffStaticData = uint32(10293)
)

// dummyBuffLease 假人 buff 的静态租期：归档每次注册时重建 payload，24h
// 远超一次在线会话，无需到期管理。
const dummyBuffLease = 24 * time.Hour

// dummyBuffSlots 返回假人归档的 buff 槽内容（slot → BufferInfo 串）与
// BufferListStr。格式与真玩家 bufferInfoLocked/bufferListStringLocked 逐字段
// 同构：BufferInfo="staticData,obj,owner,level,expiry1601Ms,1"（全 HEX，
// seg5 为 1601 纪元毫秒），BufferListStr="configID,local-slot,level,
// obj-owner,expiryUnixMs,1,0|"。内功挂槽 4（level=3 与真玩家一致），
// 御风挂槽 24。
func dummyBuffSlots(objectID uint32, dummy dummyPlayer, now time.Time) (map[uint16]string, string) {
	slots := map[uint16]string{}
	var list strings.Builder
	expires := now.UTC().Add(dummyBuffLease)
	addBuff := func(slot uint16, configID string, staticData uint32, level int32) {
		slots[slot] = fmt.Sprintf("%X,%X,%X,%X,%X,%X", staticData, objectID, playerOwnerID, level, epoch1601Ms(expires), 1)
		fmt.Fprintf(&list, "%s,local-%d,%d,%d-%d,%d,1,0|", configID, slot, level, objectID, playerOwnerID, expires.UnixMilli())
	}
	if entry, ok := loadDummyNeiGongBuffPool()[strings.TrimSpace(dummy.NeiGongBuff)]; ok {
		addBuff(innerPowerBuffSlot, entry.BuffID, entry.StaticData, 3)
	}
	if dummy.Yufeng {
		addBuff(dummyYufengBuffSlot, yufengWindBuffConfigID, yufengWindBuffStaticData, 1)
	}
	return slots, list.String()
}

// ---- 0x0D Type=2 玩家归档 ----

// dummyAddObject 编码 0x0D Type=2 玩家归档。属性下标按名经 0x09 协商表
// 解析，fork-only 行（PosiX 族等）按名缺席整组跳过——与
// playerActor.addObjectAs（真玩家出生归档）同一约定。
func dummyAddObject(objectID uint32, dummy dummyPlayer, now time.Time) []byte {
	visual := dummy.dummyVisual()
	show := dummy.applyCardOverrides(&visual)
	// 头部通道（与真玩家 Hat 规则一致）：Hat 属性优先带帽模；无帽时带
	// 发型网格（现网已验证行为），Hair 属性相应清空。
	hairValue := visual.hair
	hatValue := visual.hat
	if hatValue != "" {
		hairValue = ""
	} else {
		hatValue = visual.hair
	}
	name := strings.TrimSpace(dummy.Name)
	if name == "" {
		name = "路人"
	}
	action := strings.TrimSpace(dummy.Action)
	if action == "" {
		action = "stand"
	}
	level := dummy.Level
	if level <= 0 {
		level = 1
	}
	// 移动配速（v15 分层）：御风 11 / 步行 2.6 / 跑步 6 / 站桩取真人默认。
	moveSpeed := float32(3.9)
	walkSpeed := float32(1.28)
	runSpeed := float32(3.9)
	if dummyMoves(dummy) {
		speed := dummyMoveSpeed(dummy)
		moveSpeed, runSpeed = speed, speed
		if dummy.MoveMode == "walk" {
			walkSpeed = speed
		}
	}

	msg := make([]byte, 0x3f, 0x180)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	binary.LittleEndian.PutUint32(msg[5:], playerOwnerID)
	binary.LittleEndian.PutUint32(msg[0x09:], math.Float32bits(dummy.Pos.X))
	binary.LittleEndian.PutUint32(msg[0x0D:], math.Float32bits(dummy.Pos.Y))
	binary.LittleEndian.PutUint32(msg[0x11:], math.Float32bits(dummy.Pos.Z))
	binary.LittleEndian.PutUint32(msg[0x15:], math.Float32bits(dummy.Pos.Orient))

	count := uint16(0)
	addByte := func(index uint16, value byte) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = append(msg, value)
		count++
	}
	addInt := func(index uint16, value int32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, uint32(value))
		count++
	}
	addFloat := func(index uint16, value float32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(value))
		count++
	}
	addString := func(index uint16, value string) {
		msg = appendStringProperty(msg, index, value)
		count++
	}
	addOptional := func(name, value string) bool {
		property, ok := clientdata.PlayerProperty(name)
		if !ok {
			return false
		}
		addString(property.Index, value)
		return true
	}

	addByte(clientdata.MustPlayerProperty("Type").Index, 2)               // Type = TYPE_PLAYER
	addByte(clientdata.MustPlayerProperty("Sex").Index, visual.sex)       // Sex
	addString(clientdata.MustPlayerProperty("Photo").Index, visual.photo) // Photo
	addByte(clientdata.MustPlayerProperty("Camp").Index, 0)               // Camp
	if race, ok := clientdata.SceneProperty("Race"); ok {                 // fork-only
		addInt(race.Index, 0)
	}
	addString(clientdata.MustPlayerProperty("Job").Index, "")          // Job
	addByte(clientdata.MustPlayerProperty("Level").Index, byte(level)) // Level
	if posiX, ok := clientdata.SceneProperty("PosiX"); ok {            // fork-only 坐标族
		addFloat(posiX.Index, dummy.Pos.X)
		addFloat(clientdata.MustSceneProperty("PosiY").Index, dummy.Pos.Y)
		addFloat(clientdata.MustSceneProperty("PosiZ").Index, dummy.Pos.Z)
		addFloat(clientdata.MustSceneProperty("Orient").Index, dummy.Pos.Orient)
	}
	addString(clientdata.MustPlayerProperty("Hair").Index, hairValue)             // Hair
	addString(clientdata.MustPlayerProperty("Face").Index, visual.face)           // Face
	addString(clientdata.MustPlayerProperty("Cloth").Index, visual.cloth)         // Cloth
	addString(clientdata.MustPlayerProperty("Pants").Index, visual.pants)         // Pants
	addString(clientdata.MustPlayerProperty("Shoes").Index, visual.shoes)         // Shoes
	addString(clientdata.MustPlayerProperty("ActionSet").Index, visual.actionSet) // ActionSet
	if visual.weapon != "" {
		addString(clientdata.MustPlayerProperty("Weapon").Index, visual.weapon) // Weapon（裸模型名）
	}
	addInt(propShowEquipType, show)
	addString(clientdata.MustPlayerProperty("State").Index, action)    // State：出生动作
	if floating, ok := clientdata.SceneProperty("FloatingState"); ok { // fork-only
		addInt(floating.Index, 0)
	}
	// 和平态（LS_NORMAL）：武器渲染在背部挂件位；LogicState=2 会进入战斗
	// 戒备姿势（观感差，server2 用户实测定案勿改回）。
	addByte(clientdata.MustPlayerProperty("LogicState").Index, 0)
	addFloat(clientdata.MustPlayerProperty("JumpSpeed").Index, 8.0)
	addFloat(clientdata.MustPlayerProperty("MoveSpeed").Index, moveSpeed)
	addFloat(clientdata.MustPlayerProperty("WalkSpeed").Index, walkSpeed)
	addFloat(clientdata.MustPlayerProperty("RunSpeed").Index, runSpeed)
	addInt(clientdata.MustPlayerProperty("SpeedRatio").Index, 0)
	if cantJump, ok := clientdata.SceneProperty("CantJump"); ok { // fork-only
		addByte(cantJump.Index, 0)
	}
	addInt(propHPRatio, 100)
	addInt(propMPRatio, 100)
	addInt(clientdata.MustPlayerProperty("MaxHP").Index, 1000)
	addInt(clientdata.MustPlayerProperty("MaxMP").Index, 100)
	addInt(propHP, 1000)
	addInt(propMP, 100)
	// Buff 链路：内功常驻 buff（槽 4）+ 御风状态（槽 24，可选）。
	slots, bufferList := dummyBuffSlots(objectID, dummy, now)
	addOptional("BufferListStr", bufferList)
	for _, slot := range nativeBuffSlots {
		property, ok := clientdata.PlayerProperty(bufferInfoPropertyName(slot))
		if !ok {
			continue
		}
		addString(property.Index, slots[slot])
	}
	// 门派/势力（School/Force/NewSchool）：factionProperties 同链路。
	for _, property := range factionProperties(dummy.School) {
		msg = appendNPCProperty(msg, property)
		count++
	}
	// 实力链路：PowerLevel + LevelTitle 称号（roleListTitle，选人界面同表）
	// + Strength 面板族（官方表缺席整组跳过）。
	powerLevel := int32(dummy.PowerLevel)
	if powerLevel < 0 {
		powerLevel = 0
	}
	if powerLevel > 160 {
		powerLevel = 160
	}
	nextStageName, pointsToNext := dummyStrengthStageInfo(powerLevel)
	addInt(propPowerLevel, powerLevel)
	addString(propLevelTitle, roleListTitle(int(powerLevel)))
	strength := []clientdata.IndexedProperty{
		{Name: "StrengthNeiGongName", Value: clientdata.StringValue(dummy.NeiGong)},
		{Name: "StrengthNeiGongLevel", Value: clientdata.Int32Value(powerLevel)},
		{Name: "StrengthNeiGongPoints", Value: clientdata.Int32Value(powerLevel)},
		{Name: "StrengthNeiGongCult", Value: clientdata.StringValue("")},
		{Name: "StrengthMartialCult", Value: clientdata.StringValue("")},
		{Name: "StrengthMeridianTotalLevel", Value: clientdata.Int32Value(0)},
		{Name: "StrengthNextStage", Value: clientdata.StringValue(nextStageName)},
		{Name: "StrengthNextStagePoints", Value: clientdata.Int32Value(pointsToNext)},
	}
	for i := range strength {
		if property, ok := clientdata.PlayerProperty(strength[i].Name); ok {
			strength[i].Index = property.Index
			msg = appendNPCProperty(msg, strength[i])
			count++
		}
	}
	// 性格标签（侠/恶/狂/邪）：数值按名字哈希确定性生成，与标签保持内部
	// 一致防面板穿帮。
	character := dummy.Character
	if character < 0 || character > 4 {
		character = 0
	}
	justice, evil, freeBase, charValue := dummyAlignmentValues(dummy.Name, character)
	addInt(propCharacterFlag, int32(character))
	addInt(propCharacterValue, charValue)
	addInt(propJustice, justice)
	addInt(propEvil, evil)
	addInt(propFreeBase, freeBase)
	addInt(propTitleID, int32(dummy.TitleID))
	addString(clientdata.MustPlayerProperty("Hat").Index, hatValue)                        // Hat
	msg = appendWideStringProperty(msg, clientdata.MustPlayerProperty("Name").Index, name) // Name (widestr)
	count++
	msg = appendObjectProperty(msg, clientdata.MustPlayerProperty("LastObject").Index, 0, 0) // LastObject 占位
	count++
	addFloat(clientdata.MustPlayerProperty("CollideRadius").Index, 0.9)
	addFloat(clientdata.MustPlayerProperty("ActionSpeed").Index, 1.0)
	binary.LittleEndian.PutUint16(msg[0x3d:], count)
	return msg
}

// ---- 移动配速 ----

const (
	dummyPatrolSpeed   float32 = 6.0  // 跑步
	dummyWalkSpeed     float32 = 2.6  // 步行
	dummyQingGongSpeed float32 = 11.0 // 御风疾跑
)

// dummyMoveSpeed 返回假人巡逻配速：yufeng=御风疾跑，moveMode=walk 步行，
// 默认跑步。
func dummyMoveSpeed(dummy dummyPlayer) float32 {
	switch {
	case dummy.Yufeng:
		return dummyQingGongSpeed
	case dummy.MoveMode == "walk":
		return dummyWalkSpeed
	default:
		return dummyPatrolSpeed
	}
}

// dummyMoves 报告该假人是否为移动假人（固定路点或随机人流）。
func dummyMoves(dummy dummyPlayer) bool {
	return len(dummy.Patrol) >= 2 || dummy.PatrolRandom
}

// dummyPatrolPointsFromCatalog 抽取场景官方 NPC 坐标作为巡逻假人的随机
// 目的地池（"NPC 到 NPC"人流路线的坐标来源）。
func dummyPatrolPointsFromCatalog(catalog []npcSpawn) []worldcore.Transform {
	if len(catalog) == 0 {
		return nil
	}
	points := make([]worldcore.Transform, 0, len(catalog))
	for _, npc := range catalog {
		points = append(points, worldcore.Transform{X: npc.x, Y: npc.y, Z: npc.z})
	}
	return points
}
