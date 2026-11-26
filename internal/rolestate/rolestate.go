package rolestate

// Package rolestate 建号初始状态：类型 + XML 资产 + 构建器。
// nx_roles.role_state_json 的 JSON 投影；fssolostore 持久化与
// entergen 进场生成共用本包单一事实源。
//
// 数据源（go:embed，运行时零路径依赖）：
//   - roledata/RoleBaseProp.xml      官方口径五维/资源初值/NewbieSkill/InitTask
//     （ID = 出生地×10 + 性别；docs/official/九阴建角初始数据归纳.md）
//   - roledata/StarterSet.xml        starter 授予集（b-line 常量提取，
//     tools/gen_starter_set.py 生成：25 武学/18+51 轻功/任务门/204 采集/6 内功）
//   - roledata/birth_locations.ini   出生点坐标（testdata/location.ini 镜像）
//
// 数值权威 = 客户端资源 ini（wuxuelevelinfo/neigong/skill_new）；本表为
// 无资源目录时的建号兜底口径（与 b-line newPlayerProgress 同源）。

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed roledata/RoleBaseProp.xml roledata/StarterSet.xml roledata/birth_locations.ini
var roledataFS embed.FS

// RoleState 建号初始状态（JSON 落 nx_roles.role_state_json）。
// Phase 4 进场原生生成（entergen）从本状态取值。
type RoleState struct {
	Sex                int            `json:"sex"`
	Scene              string         `json:"scene"`                // 场景 config（ini\scene\born03_YanYuZhuang）
	Resource           string         `json:"resource"`             // 场景 resource（born03）
	Appearance         []string       `json:"appearance,omitempty"` // 建角自定义外观串（save_para1 逗号拆；[1]photo [2]face(捏脸二进制) [4]hair——渲染自定义数据）
	Position           [4]float64     `json:"position"`             // x,y,z,orient（location.ini 方言）
	Str                int            `json:"str"`
	Dex                int            `json:"dex"`
	Ing                int            `json:"ing"`
	Spi                int            `json:"spi"`
	Sta                int            `json:"sta"`
	HP                 int            `json:"hp"`
	MP                 int            `json:"mp"`
	SP                 int            `json:"sp"`
	Parry              int            `json:"parry"`
	NewbieSkill        uint32         `json:"newbieSkill"`
	InitTask           uint32         `json:"initTask"`
	BaishiTask         uint32         `json:"baishiTask"`
	Skills             []StarterSkill `json:"skills"`
	QingGongCategories []string       `json:"qinggongCategories"`
	QingGongSkills     []string       `json:"qinggongSkills"`
	TaskAccepted       []uint32       `json:"taskAccepted"`
	TaskCompleted      []uint32       `json:"taskCompleted"`
	CollectSkills      []string       `json:"collectSkills"`
	NeiGong            []NeiGongBook  `json:"neigong"`
	CurNeiGong         string         `json:"curNeiGong"`
	FacultySeed        int64          `json:"facultySeed"`
	// Money 碎银（GM/经济测试位；对拍 2433 线 CapitalType1，0x10 属性
	// 帧按名寻址下发）。官银/黄金（CapitalType0/2）留多币种扩展。
	Money int64 `json:"money,omitempty"`
	// LiveGrooves 历练累计（玩家属性 LiveGroove_1..5，下标 1..5；任务完成
	// 奖励 prize_base 的 LiveGroove/LiveGrooveCount 累加进来）。0＝无。
	LiveGrooves [6]int32 `json:"liveGrooves"`
	// Reputes 声望记录（Repute_Record 行：名称、数值；任务完成奖励累加）。
	Reputes []Repute `json:"reputes,omitempty"`

	// ---- 自定义配置域（fork 客户端本地不落盘，登录全靠服务端回推；
	// 参考端 cmd/protocol-probe custom_config.go/shortcut_records.go 同源
	// 规则，随 0x04 存档链持久化到 role_state_json）----

	// Customizing C2S 107 引擎个性配置整串（S2C 705 回推）。二进制串
	// （字节值即配置项取值，含 ≥0x80）——必须走 BinaryString：普通
	// string 经 encoding/json 会被清洗成合法 UTF-8，高字节全换成 U+FFFD，
	// 回推客户端即画面配置错位（2026-11-23 重登卡顿根因）。
	Customizing BinaryString `json:"customizing,omitempty"`
	// Keybind/1/2 键位主串与方案一/二（C2S 195 scheme 0/1/2 → 属性
	// 198/199/200 type7；0x10 单属性帧带尾空格、705 不涉）。
	Keybind  string `json:"keybind,omitempty"`
	Keybind1 string `json:"keybind1,omitempty"`
	Keybind2 string `json:"keybind2,omitempty"`
	// OperateMode 操作模式（C2S 99 → 属性 223；0=3D WASD/1=2.5D 点击）。
	// Set 区分「从未设置」（不下发，客户端保持本地默认）与显式 0。
	OperateMode    int32 `json:"operateMode,omitempty"`
	OperateModeSet bool  `json:"operateModeSet,omitempty"`
	// NeiGongBind 内功栏绑定串（C2S 199 → 属性 201；25 格 "v;"+尾分号）。
	NeiGongBind string `json:"neiGongBind,omitempty"`
	// Shortcuts 快捷栏基线（C2S 200/201 编辑 → shortcut_rec 0x11 行重放）。
	// Set=true 即玩家已编辑（空表也是基线）：starter 打坐条目不复活。
	Shortcuts    []ShortcutRow `json:"shortcuts,omitempty"`
	ShortcutsSet bool          `json:"shortcutsSet,omitempty"`

	// ---- 进度域（进场下发、随 0x04 存档链持久化的玩家级累积）----

	// OpenMaps 开图位图（24×u32=768 场景位；scenes.ini 全局段号 i*32+j）。
	// nil=无档（首登回落外服基线位图）；进场叠加当前场景位后回写——
	// 2433 线无玩家级开图持久化（基线+当前场景每次重派生），本仓自建。
	OpenMaps []uint32 `json:"openMaps,omitempty"`

	// TaskTracks 流程接取的任务（NPC 任务链接取/步骤推进/提交受理维护）：
	// 接取时刻、追踪标志、当前步序与各步进度。区别于建号种子的
	// TaskAccepted / TaskCompleted 功能门行——那些不代表玩家接过任务。
	// 客户端追踪栏只显示追踪标志为 1 的已接行，目标行（Task_Record）也
	// 只给这些任务下发（官服引导抓包方言；见 tasktrack.go）。
	TaskTracks []TaskTrack `json:"taskTracks,omitempty"`

	// GameStep 新手引导阶段（客户端 SwitchManager 主步：官服登录 m414
	// ["gamestep", 主步, 5]，新手副本黑幕「初出茅庐／再战江湖」点选后 C2S
	// m972 回传同一值）：0＝未定；4＝初出茅庐（单机引导）；5＝再战江湖
	// （跳过引导、直接入江湖）。旧档缺省 0——只有身在新手副本且引导链
	// 未开始的角色才处于待选态（fssoloworld/guide.go）。
	GameStep int `json:"gameStep,omitempty"`
	// HelpStepID 新手帮助步骤链（freshmanhelp.ini 节名：客户端 C2S m950 上报
	// 下一步，服务端回写同名属性）。空＝未入链（档案取外服常量）。
	HelpStepID string `json:"helpStepID,omitempty"`
	// RelivePositon 复活点串（"<场景>,<x>,<y>,<z>,<朝向>,<键>"；跳过引导
	// 进江湖时落到鸡鸣驿）。空＝未设（档案取外服常量）。
	RelivePositon string `json:"relivePositon,omitempty"`

	// ---- 物品域（2433 线 bagRow/equipRow 口径建模；下发帧/C2S 操作链
	// （30 移动穿戴/31 使用/34 丢弃/36 整理）后续接入）----

	BagItems []BagItem   `json:"bagItems,omitempty"` // 背包行（分类页 slot 1..90）
	Equips   []EquipItem `json:"equips,omitempty"`   // 装备行（纸娃娃 View1 槽位 1..52）
	// EquipsSeeded 初始布衣种子闸（omitempty 下「功能上线前旧档」与
	//「玩家主动全脱」的 Equips 都缺席——建号/补建置位一次，此后全脱
	// 不再复穿）。
	EquipsSeeded bool `json:"equipsSeeded,omitempty"`

	// ---- 邮件域（2433 线 letterEntry/Java Letter 精简口径；行帧=记录表
	// RecvLetterRec 0x11 行，entergen/mailrecords.go 编码）----

	Letters []Letter `json:"letters,omitempty"` // 收件箱（GM 发信/读信/领附件维护）
}

// Letter 一封邮件（2433 线 letterEntry 同源）：Serial 是读信(353)/领附件
// (359)回执键；Type 101=系统邮件（LETTER_SYSTEM_POST_USER，101..199 客户端
// 归系统页签，发件人空串显示「系统邮件」）；Gold/Silver 官方方言是数字
// 窄串；Appendix 为附件 XML（AppendixXML 格式，领取后清空置已读）。
type Letter struct {
	Serial    string `json:"serial"`
	Type      int32  `json:"type"`
	Sender    string `json:"sender,omitempty"`
	SenderUID string `json:"senderUID,omitempty"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Gold      string `json:"gold,omitempty"`
	Silver    string `json:"silver,omitempty"`
	Appendix  string `json:"appendix,omitempty"`
	DateUnix  int64  `json:"dateUnix"`
	Read      bool   `json:"read,omitempty"`
}

// Repute 一条声望记录（Repute_Record 行：名称、数值）。
type Repute struct {
	Name  string `json:"name"`
	Count int32  `json:"count"`
}

// BagItem 背包行（2433 线 bagRow 精简口径）：ViewID 分类页 1 道具/2 装备/
// 3 材料/4 任务；Amount 堆叠数（客户端按 ConfigID 聚合显示）；Seq 行身份
// （跨表移动携带）、UID 实例稳定身份（随机属性/强化词缀键控位）。
type BagItem struct {
	Slot       int    `json:"slot"`
	ConfigID   string `json:"configID"`
	ItemType   int    `json:"itemType,omitempty"`
	Amount     int    `json:"amount,omitempty"`
	ViewID     int    `json:"viewID,omitempty"`
	Seq        int64  `json:"seq,omitempty"`
	UID        string `json:"uid,omitempty"`
	EquipType  string `json:"equipType,omitempty"`  // 装备行必发真值（渲染闸）
	ColorLevel int    `json:"colorLevel,omitempty"` // 缺省 1
	Photo      string `json:"photo,omitempty"`      // 图标（可缺省）
	// MaxAmount 堆叠上限（入包时取自物品模板行；0=未记——行帧回落旧口径，
	// 叠堆不设限）。
	MaxAmount int `json:"maxAmount,omitempty"`
}

// EquipItem 装备行（2433 线 equipRow 精简口径）：Slot=身体槽位（hat=1/
// cloth=3/pants=4/wrist=6/leg=7/shoes=8/weapon=22/treasure=25..29——
// equipbox.ini [EquipType] 槽位表）；词缀/随机属性后续按 UID 扩展。
type EquipItem struct {
	Slot      int    `json:"slot"`
	ConfigID  string `json:"configID"`
	ItemType  int    `json:"itemType,omitempty"`
	EquipType string `json:"equipType,omitempty"`
	Seq       int64  `json:"seq,omitempty"`
	UID       string `json:"uid,omitempty"`
}

// bagSlotMax 背包分类页容量（90 格，2433 线 freeBagSlot 口径）。
const bagSlotMax = 90

// StarterEquips 初始布衣三件（官服初始号 view1 真机口径：穿着进场，
// cloth/pants/shoes (b|g)0001，ItemType 141/142/143、PhyDef 每件 +6）。
// UniqueID 官方形状 {roleUID}-001-{unix}-{序}，跨穿脱/存档稳定。
func StarterEquips(sex int, roleUID string, nowUnix int64) []EquipItem {
	suffix := "b"
	if sex == 2 {
		suffix = "g"
	}
	rows := []struct {
		slot      int
		part      string
		itemType  int
		equipType string
	}{
		{3, "cloth", 141, "Cloth"},
		{4, "pants", 142, "Pants"},
		{8, "shoes", 143, "Shoes"},
	}
	equips := make([]EquipItem, 0, len(rows))
	for i, row := range rows {
		equips = append(equips, EquipItem{
			Slot:      row.slot,
			ConfigID:  row.part + "_" + suffix + "0001",
			ItemType:  row.itemType,
			EquipType: row.equipType,
			Seq:       int64(i + 1),
			UID:       fmt.Sprintf("%s-001-%d-%04d", roleUID, nowUnix, i+1),
		})
	}
	return equips
}

// NextItemSeq 物品行身份分配（max+1，0 起步；入包/穿戴跨表移动携带）。
func (s *RoleState) NextItemSeq() int64 {
	var max int64
	for _, item := range s.BagItems {
		if item.Seq > max {
			max = item.Seq
		}
	}
	for _, item := range s.Equips {
		if item.Seq > max {
			max = item.Seq
		}
	}
	return max + 1
}

// FreeBagSlot 分类页首个空格（1..90；全满 0）。
func (s *RoleState) FreeBagSlot(viewID int) int {
	taken := map[int]bool{}
	for _, item := range s.BagItems {
		if item.ViewID == viewID {
			taken[item.Slot] = true
		}
	}
	for slot := 1; slot <= bagSlotMax; slot++ {
		if !taken[slot] {
			return slot
		}
	}
	return 0
}

// FreeBagSlots 分类页空格数（入包容量预检用）。
func (s *RoleState) FreeBagSlots(viewID int) int {
	taken := 0
	for _, item := range s.BagItems {
		if item.ViewID == viewID && item.Slot >= 1 && item.Slot <= bagSlotMax {
			taken++
		}
	}
	return max(bagSlotMax-taken, 0)
}

// ShortcutRow shortcut_rec 一行：可见栏位索引、类型（"skill"/"item"…）
// 与 config ID（栏位索引非记录行号——客户端 C2S 201 传的也是栏位索引）。
type ShortcutRow struct {
	Index int32  `json:"index"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
}

// StarterSitcrossShortcut starter 打坐条目（Ctrl+0 槽位=索引 9，避开
// 初始战斗两槽）：客户端把 zs_default_01 识别为原生坐下/起立切换，
// 走 CustomSend 240 而非 211 战斗路线——不是 FuncManager 记录。
var StarterSitcrossShortcut = ShortcutRow{Index: 9, Kind: "skill", ID: "zs_default_01"}

// EffectiveShortcuts 快捷栏生效表：无玩家编辑基线（ShortcutsSet=false）
// 且索引 9 空缺时补 starter 打坐条目；有基线以编辑结果为准（删除即
// 删除，不复活）。帧生成与编辑应答（整表重建方言）共用本口径。
func (s RoleState) EffectiveShortcuts() []ShortcutRow {
	rows := append([]ShortcutRow(nil), s.Shortcuts...)
	if s.ShortcutsSet {
		return rows
	}
	for _, row := range rows {
		if row.Index == StarterSitcrossShortcut.Index {
			return rows
		}
	}
	return append(rows, StarterSitcrossShortcut)
}

// StarterSkill 初始武学行（View40 技能行 + 0x28 档案技能区共用）。
type StarterSkill struct {
	ConfigID   string  `json:"configID"`
	StaticData int     `json:"staticData"`
	ItemType   int     `json:"itemType"`
	Level      int     `json:"level"`
	MaxLevel   int     `json:"maxLevel"`
	Total      int     `json:"total,omitempty"`
	PauseTime  float64 `json:"pauseTime,omitempty"`
}

// NeiGongBook 初始内功书（View43 行 + 修为/五维加成来源）。
type NeiGongBook struct {
	ConfigID       string `json:"configID"`
	StaticData     int    `json:"staticData"`
	ItemType       int    `json:"itemType"`
	Level          int    `json:"level"`
	MaxLevel       int    `json:"maxLevel"`
	Total          int    `json:"total"`
	Power          int    `json:"power"`
	MaxPower       int    `json:"maxPower"`
	BuffID         string `json:"buffID,omitempty"`
	BuffStaticData int    `json:"buffStaticData,omitempty"`
	BuffLevel      int    `json:"buffLevel,omitempty"`
	StrAdd         int    `json:"strAdd,omitempty"`
	StaAdd         int    `json:"staAdd,omitempty"`
	DexAdd         int    `json:"dexAdd,omitempty"`
	IngAdd         int    `json:"ingAdd,omitempty"`
	SpiAdd         int    `json:"spiAdd,omitempty"`
	MaxHPAdd       int    `json:"maxHPAdd,omitempty"`
	MaxMPAdd       int    `json:"maxMPAdd,omitempty"`
	MaxParryAdd    int    `json:"maxParryAdd,omitempty"`
	MinMagicDefAdd int    `json:"minMagicDefAdd,omitempty"`
}

// starterMoney 建号碎银初值（2433 线 flag starter-silver 默认 99999 的
// 测试口径；官方建号货币基线未逆向，先对齐 fork 线便于对拍）。
const starterMoney = 99999

// ---- RoleBaseProp.xml ----

type roleBasePropXML struct {
	Property []roleBasePropRow `xml:"Property"`
}

type roleBasePropRow struct {
	ID          string `xml:"ID,attr"`
	Scene       string `xml:"Scene,attr"`
	Sex         string `xml:"Sex,attr"`
	Str         string `xml:"Str,attr"`
	Dex         string `xml:"Dex,attr"`
	Ing         string `xml:"Ing,attr"`
	Spi         string `xml:"Spi,attr"`
	Sta         string `xml:"Sta,attr"`
	HP          string `xml:"HP,attr"`
	MP          string `xml:"MP,attr"`
	SP          string `xml:"SP,attr"`
	Parry       string `xml:"Parry,attr"`
	NewbieSkill string `xml:"NewbieSkill,attr"`
	InitTask    string `xml:"InitTask,attr"`
	BaishiTask  string `xml:"BaishiTask,attr"`
}

// ---- StarterSet.xml ----

type starterSetXML struct {
	Skills struct {
		Skill []struct {
			ConfigID   string `xml:"ConfigID,attr"`
			StaticData string `xml:"StaticData,attr"`
			ItemType   string `xml:"ItemType,attr"`
			Level      string `xml:"Level,attr"`
			MaxLevel   string `xml:"MaxLevel,attr"`
			Total      string `xml:"Total,attr"`
			PauseTime  string `xml:"PauseTime,attr"`
		} `xml:"Skill"`
	} `xml:"StarterSkills"`
	QingGong struct {
		Category struct {
			IDs string `xml:"IDs,attr"`
		} `xml:"Category"`
		Instance struct {
			IDs string `xml:"IDs,attr"`
		} `xml:"Instance"`
	} `xml:"QingGong"`
	TaskGates struct {
		Unlock struct {
			FunctionUnlock string `xml:"FunctionUnlock,attr"`
			FacultyAlt     string `xml:"FacultyAlt,attr"`
			FacultyActAlt  string `xml:"FacultyActAlt,attr"`
			OpenGate       string `xml:"OpenGate,attr"`
		} `xml:"Unlock"`
		Guide struct {
			First string `xml:"First,attr"`
			Last  string `xml:"Last,attr"`
		} `xml:"Guide"`
		SchoolLead struct {
			First string `xml:"First,attr"`
			Last  string `xml:"Last,attr"`
		} `xml:"SchoolLead"`
		Act struct {
			First string `xml:"First,attr"`
			Last  string `xml:"Last,attr"`
		} `xml:"Act"`
		FuncGate struct {
			IDs string `xml:"IDs,attr"`
		} `xml:"FuncGate"`
	} `xml:"TaskGates"`
	CollectSkills struct {
		IDs string `xml:"IDs,attr"`
	} `xml:"CollectSkills"`
	NeiGong struct {
		Book []struct {
			ConfigID       string `xml:"ConfigID,attr"`
			StaticData     string `xml:"StaticData,attr"`
			ItemType       string `xml:"ItemType,attr"`
			Level          string `xml:"Level,attr"`
			MaxLevel       string `xml:"MaxLevel,attr"`
			Total          string `xml:"Total,attr"`
			Power          string `xml:"Power,attr"`
			MaxPower       string `xml:"MaxPower,attr"`
			BuffID         string `xml:"buffID,attr"`
			BuffStaticData string `xml:"buffStaticData,attr"`
			BuffLevel      string `xml:"buffLevel,attr"`
			StrAdd         string `xml:"strAdd,attr"`
			StaAdd         string `xml:"staAdd,attr"`
			DexAdd         string `xml:"dexAdd,attr"`
			IngAdd         string `xml:"ingAdd,attr"`
			SpiAdd         string `xml:"spiAdd,attr"`
			MaxHPAdd       string `xml:"maxHPAdd,attr"`
			MaxMPAdd       string `xml:"maxMPAdd,attr"`
			MaxParryAdd    string `xml:"maxParryAdd,attr"`
			MinMagicDefAdd string `xml:"minMagicDefAdd,attr"`
		} `xml:"Book"`
	} `xml:"NeiGong"`
}

func atoi(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func atou32(s string) uint32 {
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	return uint32(v)
}

func atof(s string) float64 {
	// PauseTime 生成器写的是 Go 表达式残余（如 43 * 1000.0 / 30）——
	// gen_starter_set 已折算；容错裸数。
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func splitIDs(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitU32(s string) []uint32 {
	parts := splitIDs(s)
	out := make([]uint32, 0, len(parts))
	for _, p := range parts {
		out = append(out, atou32(p))
	}
	return out
}

// roleStateData 三份内嵌资产的进程内单次装载结果。
type roleStateData struct {
	base    []roleBasePropRow
	starter starterSetXML
	birth   map[string]string
}

// loadRoleStateData 装载三份内嵌资产（进程内单次）。
var roleStateAssets = func() (out roleStateData) {
	raw, err := roledataFS.ReadFile("roledata/RoleBaseProp.xml")
	if err != nil {
		panic("fssolostore: roledata/RoleBaseProp.xml missing: " + err.Error())
	}
	var bx roleBasePropXML
	if err := xml.Unmarshal(raw, &bx); err != nil {
		panic("fssolostore: RoleBaseProp.xml parse: " + err.Error())
	}
	out.base = bx.Property

	raw, err = roledataFS.ReadFile("roledata/StarterSet.xml")
	if err != nil {
		panic("fssolostore: roledata/StarterSet.xml missing: " + err.Error())
	}
	var sx starterSetXML
	if err := xml.Unmarshal(raw, &sx); err != nil {
		panic("fssolostore: StarterSet.xml parse: " + err.Error())
	}
	out.starter = sx

	raw, err = roledataFS.ReadFile("roledata/birth_locations.ini")
	if err != nil {
		panic("fssolostore: birth_locations.ini missing: " + err.Error())
	}
	out.birth = map[string]string{}
	section := ""
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}
		if i := strings.Index(line, "="); i > 0 && strings.HasPrefix(section, "BornCJ") {
			out.birth[section+"."+line[:i]] = strings.TrimSpace(line[i+1:])
		}
	}
	return out
}()

// InferSexFromPara 从建角外观串推断性别（world 侧 roles.go inferSex 同口径：
// photo girl/boy > 发型 g_/b_；缺证据默认男=1）。
func InferSexFromPara(para string) int {
	fields := strings.Split(para, ",")
	contains := func(sub string) bool { return strings.Contains(para, sub) }
	if len(fields) > 0 {
		photo := fields[0]
		if strings.Contains(photo, "girl") {
			return 2
		}
		if strings.Contains(photo, "boy") {
			return 1
		}
	}
	if contains("g_hair") || contains("_g_") {
		return 2
	}
	if contains("b_hair") || contains("_b_") {
		return 1
	}
	return 1
}

// DefaultBirthScene 默认出生场景：新手副本 born_single_01（官服新号首次
// 登录进的单机引导场景，2026-11-26 起；此前是烟雨庄 born03）。
const DefaultBirthScene = GuideScene

// GuideScene 新手副本（单机引导场景）的 resource。
const GuideScene = "born_single_01"

// GuideScenePath / GuideSceneLanding 新手副本的场景全名与建号落点。它不在
// location.ini 的 BornCJ 表里（单场景克隆态）；落点取引导起点 NPC
// funnpcborn003031（沙滩的赵子亟）旁偏移 2m——GM「新手引导」传送预设
// （fssoloworld/scenepresets.go）同此一值。
const GuideScenePath = `ini\scene\born_single_01`

var GuideSceneLanding = [4]float64{197.626, 4.746, -108.829, -2.746}

// birthStatSceneFallback 建号属性行的回落：场景在 RoleBaseProp.xml 没有
// 专属行时取哪个场景的行。新手副本不在官方建角表里，五维/资源与烟雨庄
// 新号同值。
var birthStatSceneFallback = map[string]string{GuideScene: "born03"}

// SceneOption location.ini BornCJ 节导出的一个可选场景目的地
// （GM 场景切换/传送门共用）：Key=场景 resource 短名（born03/city01…），
// Config=场景配置路径（ini\scene\…），Position=该节出生点坐标。
type SceneOption struct {
	Key      string
	Config   string
	Position [4]float64
}

// BirthSceneOptions 全部 BornCJ 出生态场景选项（按节序稳定输出；同
// resource 只保留首见节）。Key 推导与 BuildRoleState 的 Resource 同口径：
// Scene 路径末段截到首个 '_'（born03_YanYuZhuang → born03，无 '_' 保持
// 全名 clone001）。
func BirthSceneOptions() []SceneOption {
	birth := roleStateAssets.birth
	keys := make([]string, 0, len(birth))
	for key := range birth {
		if strings.HasSuffix(key, ".Scene") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys) // BornCJ 节名定宽数字序——首见节稳定
	seen := map[string]bool{}
	out := make([]SceneOption, 0, len(keys))
	for _, key := range keys {
		scenePath := birth[key]
		seg := scenePath
		if i := strings.LastIndexAny(scenePath, `\/`); i >= 0 {
			seg = scenePath[i+1:]
		}
		resourceKey := seg
		if i := strings.Index(seg, "_"); i > 0 {
			resourceKey = seg[:i]
		}
		if seen[resourceKey] {
			continue
		}
		seen[resourceKey] = true
		opt := SceneOption{Key: resourceKey, Config: scenePath}
		name := strings.TrimSuffix(key, ".Scene")
		if loc, ok := birth[name+".Location"]; ok {
			parts := strings.Split(loc, ",")
			if len(parts) == 4 {
				for i := 0; i < 4; i++ {
					opt.Position[i], _ = strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
				}
			}
		}
		out = append(out, opt)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// FindSceneOption 按 resource 短名查场景选项（大小写不敏感；缺席 false）。
func FindSceneOption(key string) (SceneOption, bool) {
	for _, opt := range BirthSceneOptions() {
		if strings.EqualFold(opt.Key, key) {
			return opt, true
		}
	}
	return SceneOption{}, false
}

// BuildRoleState 按（出生场景 resource, 性别, 外观串）构建建号初始状态。
// scene 传空回落 DefaultBirthScene；facultySeed 为修为种子（b-line
// facultySeed 口径，可配）。
func BuildRoleState(scene string, sex int, para string, facultySeed int64) (RoleState, error) {
	if scene == "" {
		scene = DefaultBirthScene
	}
	if sex == 0 {
		sex = InferSexFromPara(para)
	}
	base, starter, birth := roleStateAssets.base, roleStateAssets.starter, roleStateAssets.birth
	// 建角自定义外观（save_para1 逗号拆原样保留——发型/照片/捏脸属玩家
	// 自定义数据，进场渲染按槽位取，档内缺席回退默认）
	var appearance []string
	if para != "" {
		appearance = append(appearance, strings.Split(para, ",")...)
	}

	rowScene := scene
	if fallback, ok := birthStatSceneFallback[scene]; ok {
		rowScene = fallback
	}
	var row *roleBasePropRow
	for i := range base {
		if base[i].Scene == rowScene && base[i].Sex == strconv.Itoa(sex) {
			r := base[i]
			row = &r
			break
		}
	}
	if row == nil {
		return RoleState{}, fmt.Errorf("fssolostore: RoleBaseProp.xml has no row for scene=%s sex=%d", scene, sex)
	}

	state := RoleState{
		Sex:                sex,
		Appearance:         appearance,
		Str:                atoi(row.Str),
		Dex:                atoi(row.Dex),
		Ing:                atoi(row.Ing),
		Spi:                atoi(row.Spi),
		Sta:                atoi(row.Sta),
		HP:                 atoi(row.HP),
		MP:                 atoi(row.MP),
		SP:                 atoi(row.SP),
		Parry:              atoi(row.Parry),
		NewbieSkill:        atou32(row.NewbieSkill),
		InitTask:           atou32(row.InitTask),
		BaishiTask:         atou32(row.BaishiTask),
		QingGongCategories: splitIDs(starter.QingGong.Category.IDs),
		QingGongSkills:     splitIDs(starter.QingGong.Instance.IDs),
		CollectSkills:      splitIDs(starter.CollectSkills.IDs),
		FacultySeed:        facultySeed,
		// 碎银初值对拍 2433 线 starter-silver 测试口径（CapitalType1）。
		Money: starterMoney,
	}
	// 场景全名/坐标：location.ini BornCJ 节的 Scene 路径末段以 scene 为
	// 前缀（born03 → ini\scene\born03_YanYuZhuang）即命中。
	for key, val := range birth {
		if !strings.HasSuffix(key, ".Scene") {
			continue
		}
		seg := val
		if i := strings.LastIndexAny(val, `\/`); i >= 0 {
			seg = val[i+1:]
		}
		if strings.HasPrefix(strings.ToLower(seg), strings.ToLower(scene)+"_") ||
			strings.EqualFold(seg, scene) {
			name := strings.TrimSuffix(key, ".Scene")
			if loc, ok := birth[name+".Location"]; ok {
				parts := strings.Split(loc, ",")
				if len(parts) == 4 {
					for i := 0; i < 4; i++ {
						state.Position[i], _ = strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
					}
				}
			}
			state.Scene = val
			break
		}
	}
	switch {
	case state.Scene != "":
	case scene == GuideScene:
		state.Scene, state.Position = GuideScenePath, GuideSceneLanding
	default:
		state.Scene = `ini\scene\born03_YanYuZhuang`
	}
	// Resource = 地形资源键 = 参数 scene 本身（born03/school01/born_single_01
	// ——对应 map\map_ter_born\<born03> 等目录）。真机 0x0B 帧 Resource 实证
	// 值 "born03"；旧按 Scene 尾段切片得 "YanYuZhuang"，客户端开图即报
	// 「找不到地图相关的文件」。
	state.Resource = scene

	for _, sk := range starter.Skills.Skill {
		state.Skills = append(state.Skills, StarterSkill{
			ConfigID: sk.ConfigID, StaticData: atoi(sk.StaticData), ItemType: atoi(sk.ItemType),
			Level: atoi(sk.Level), MaxLevel: atoi(sk.MaxLevel),
			Total: atoi(sk.Total), PauseTime: atof(sk.PauseTime),
		})
	}
	for _, bk := range starter.NeiGong.Book {
		state.NeiGong = append(state.NeiGong, NeiGongBook{
			ConfigID: bk.ConfigID, StaticData: atoi(bk.StaticData), ItemType: atoi(bk.ItemType),
			Level: atoi(bk.Level), MaxLevel: atoi(bk.MaxLevel), Total: atoi(bk.Total),
			Power: atoi(bk.Power), MaxPower: atoi(bk.MaxPower),
			BuffID: bk.BuffID, BuffStaticData: atoi(bk.BuffStaticData), BuffLevel: atoi(bk.BuffLevel),
			StrAdd: atoi(bk.StrAdd), StaAdd: atoi(bk.StaAdd), DexAdd: atoi(bk.DexAdd),
			IngAdd: atoi(bk.IngAdd), SpiAdd: atoi(bk.SpiAdd),
			MaxHPAdd: atoi(bk.MaxHPAdd), MaxMPAdd: atoi(bk.MaxMPAdd),
			MaxParryAdd: atoi(bk.MaxParryAdd), MinMagicDefAdd: atoi(bk.MinMagicDefAdd),
		})
	}
	if len(state.NeiGong) > 0 {
		state.CurNeiGong = state.NeiGong[0].ConfigID
	}

	// 任务门：Task_Accepted 行集（guide + alt + schoolLead + act + actAlt）
	// 与 Task_Completed 位图集（unlock + openGate + funcGate）——b-line
	// facultyAcceptedTaskIDs/grantFunctionUnlock 同口径展开（SeedTaskIDs
	// 是同一集合的查询面，任务链推进用它区分种子门行与玩家进度）。
	seedAccepted, seedCompleted := SeedTaskIDs()
	state.TaskAccepted = append(state.TaskAccepted, seedAccepted...)
	state.TaskCompleted = append(state.TaskCompleted, seedCompleted...)

	return state, nil
}

// MarshalRoleState 序列化为 nx_roles.role_state_json 内容。
func MarshalRoleState(state RoleState) (string, error) {
	b, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UnmarshalRoleState 反解（LoadRole/回读）。
func UnmarshalRoleState(raw string) (RoleState, error) {
	var state RoleState
	if strings.TrimSpace(raw) == "" {
		return RoleState{}, nil
	}
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return RoleState{}, err
	}
	return state, nil
}

// BinaryString 按字节保真的串字段：JSON 里编码为 {"b64":"<base64>"}，
// 不经 encoding/json 对 string 的 UTF-8 清洗（非法字节会被换成 U+FFFD，
// 且一个字节膨胀成三个——定长字节配置串自此整体错位）。
type BinaryString string

type binaryStringJSON struct {
	B64 string `json:"b64"`
}

// replacementBytes 旧编码器清洗非法 UTF-8 时写入的 U+FFFD（线上的
// EF BF BD）。必须按字节子串匹配：ContainsRune 会把任何非法 UTF-8
// 字节解码成 U+FFFD，合法的高字节取值会被误判。
const replacementBytes = "\uFFFD"

// MarshalJSON base64 包装。
func (b BinaryString) MarshalJSON() ([]byte, error) {
	return json.Marshal(binaryStringJSON{B64: base64.StdEncoding.EncodeToString([]byte(b))})
}

// UnmarshalJSON 兼容两种形态：新档 {"b64":…}；旧档裸串（清洗前写下的
// 控制字符转义串）——仅当不含 U+FFFD 时采用，含则视为已被清洗的坏串
// 丢弃（宁可不恢复配置，也不把错位的配置回推客户端）。
func (b *BinaryString) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var legacy string
		if err := json.Unmarshal(data, &legacy); err != nil {
			return err
		}
		if strings.Contains(legacy, replacementBytes) {
			legacy = ""
		}
		*b = BinaryString(legacy)
		return nil
	}
	var wrapped binaryStringJSON
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(wrapped.B64)
	if err != nil {
		return fmt.Errorf("binary string base64: %w", err)
	}
	*b = BinaryString(raw)
	return nil
}

// ValidCustomizing 报告客户端 107 上传的配置串可入库：不含 U+FFFD。
// 客户端本地配置一旦被历史坏串污染，其转储会原样带着 U+FFFD 上传——
// 收下即把坏值固化进存档。
func ValidCustomizing(config string) bool {
	return !strings.Contains(config, replacementBytes)
}
