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
	"embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

//go:embed roledata/RoleBaseProp.xml roledata/StarterSet.xml roledata/birth_locations.ini
var roledataFS embed.FS

// RoleState 建号初始状态（JSON 落 nx_roles.role_state_json）。
// Phase 4 进场原生生成（entergen）从本状态取值。
type RoleState struct {
	Sex                int            `json:"sex"`
	Scene              string         `json:"scene"`    // 场景 config（ini\scene\born03_YanYuZhuang）
	Resource           string         `json:"resource"` // 场景 resource（born03）
	Position           [4]float64     `json:"position"` // x,y,z,orient（location.ini 方言）
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

// DefaultBirthScene 默认出生场景（官方默认烟雨庄 born03）。
const DefaultBirthScene = "born03"

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

	var row *roleBasePropRow
	for i := range base {
		if base[i].Scene == scene && base[i].Sex == strconv.Itoa(sex) {
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
	if state.Scene == "" {
		state.Scene = `ini\scene\born03_YanYuZhuang`
	}
	if i := strings.LastIndex(state.Scene, `_`); i >= 0 {
		state.Resource = state.Scene[i+1:]
	} else {
		state.Resource = scene
	}

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
	// facultyAcceptedTaskIDs/grantFunctionUnlock 同口径展开。
	g := starter.TaskGates
	for id := atou32(g.Guide.First); id != 0 && id <= atou32(g.Guide.Last); id++ {
		state.TaskAccepted = append(state.TaskAccepted, id)
	}
	if v := atou32(g.Unlock.FacultyAlt); v != 0 {
		state.TaskAccepted = append(state.TaskAccepted, v)
	}
	for id := atou32(g.SchoolLead.First); id != 0 && id <= atou32(g.SchoolLead.Last); id++ {
		state.TaskAccepted = append(state.TaskAccepted, id)
	}
	for id := atou32(g.Act.First); id != 0 && id <= atou32(g.Act.Last); id++ {
		state.TaskAccepted = append(state.TaskAccepted, id)
	}
	if v := atou32(g.Unlock.FacultyActAlt); v != 0 {
		state.TaskAccepted = append(state.TaskAccepted, v)
	}
	state.TaskCompleted = append(state.TaskCompleted,
		atou32(g.Unlock.FunctionUnlock), atou32(g.Unlock.OpenGate))
	state.TaskCompleted = append(state.TaskCompleted, splitU32(g.FuncGate.IDs)...)

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
