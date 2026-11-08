package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 角色选择界面显示名解析，逐行为对齐 Java 参考实现
// nineyin/protocol/SceneNames.java + LoginMessages.roleListTitle/bookIntroKey。
// 客户端 form_stage_roles/form_roles.lua 对 rolePara 的消费方式：
//
//	f23 实力称号 → GetText("desc_"..f23)（desc_title001=不堪一击 … desc_title027=返璞归真）
//	f24 所在场景 → GetText(f24)（查 stringname 表；查不到就原样显示键，即"编码"）
//	f25 剧本标题 → GetText(f25)（introduction_qdz01_title=凤凰誓 等）
//
// 因此 f24 必须发真实中文场景名（或客户端认识的 scene_name_* 命名键），
// f23/f25 发客户端本地化键。数据源：
//
//	share/rule/scenes.ini（GBK，config/resource → 场景 id）
//	text/chineses/stringname.idres（UTF-8，键=ini\scene\xxx 小写，如
//	"ini\scene\school09_yihuagong=移花宫"、"city05=成都"）
var (
	modernScenesRulePath  = filepath.Join(defaultModernShareRoot, "rule", "scenes.ini")
	modernStringNamePath  = filepath.Join(defaultModernTextRoot, "chineses", "stringname.idres")
	fallbackSceneNameKey  = "scene_name_017"
	bookIntroCodeByNumber = map[string]string{
		"book1": "qdz",
		"book2": "yyz",
		"book4": "cd",
		"book5": "jm",
		"book8": "wmp",
	}
)

// roleListPowerLevel：一期种子 151，与参考服同种子玩家一致
// （Java LoginFlow.ROLE_LIST_POWER_LEVEL，2026-10-30 抓包 #0 校准 → title027）。
// 待实力等级接入角色存档后改为真实值。
const roleListPowerLevel = 151

type sceneNameTables struct {
	idByConfig   map[string]int
	idByResource map[string]int
	nameByKey    map[string]string
}

var (
	sceneNameTablesOnce  sync.Once
	sceneNameTablesCache *sceneNameTables
)

func sceneNameTablesOf() *sceneNameTables {
	sceneNameTablesOnce.Do(func() {
		sceneNameTablesCache = loadSceneNameTables()
	})
	return sceneNameTablesCache
}

// loadSceneNameTables 容错装载（Java ResourceCache 语义）：文件缺失/解析失败
// 得到空表，选角帧回退 scene_name_NNN 键，登录链路永不因此报错。
func loadSceneNameTables() *sceneNameTables {
	tables := &sceneNameTables{
		idByConfig:   make(map[string]int),
		idByResource: make(map[string]int),
		nameByKey:    make(map[string]string),
	}
	if err := loadScenesINI(modernScenesRulePath, tables); err != nil {
		log.Printf("scene name tables: scenes.ini unavailable: %v", err)
	}
	if err := loadStringNameIDRes(modernStringNamePath, tables); err != nil {
		log.Printf("scene name tables: stringname.idres unavailable: %v", err)
	}
	return tables
}

// loadScenesINI 解析 scenes.ini（GBK）：数字节名 = 场景 id，Config 值形如
// ini\scene\school09_YiHuaGong；全路径进 idByConfig，末段基名进 idByResource
// （同基名首见优先，Java putIfAbsent）。
func loadScenesINI(path string, tables *sceneNameTables) error {
	data, err := readResourceBytes(path)
	if err != nil {
		return err
	}
	if decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data); err == nil {
		data = decoded
	}
	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || section == "" || !strings.EqualFold(strings.TrimSpace(key), "Config") {
			continue
		}
		value = strings.TrimSpace(value)
		id, err := strconv.Atoi(section)
		if err != nil {
			continue
		}
		tables.idByConfig[strings.ToLower(value)] = id
		base := value
		if slash := strings.LastIndexAny(base, `\/`); slash >= 0 {
			base = base[slash+1:]
		}
		if _, exists := tables.idByResource[strings.ToLower(base)]; !exists {
			tables.idByResource[strings.ToLower(base)] = id
		}
	}
	return nil
}

// loadStringNameIDRes 解析 UTF-8 idres（key=value 行，BOM 容错）：键统一小写，
// 键重复取首值、空值跳过（Java ResourceCache.idresLowerCased）。
func loadStringNameIDRes(path string, tables *sceneNameTables) error {
	content, err := readResourceBytes(path)
	if err != nil {
		return err
	}
	for _, raw := range strings.Split(strings.TrimPrefix(string(content), "\ufeff"), "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(raw), "=")
		if !found {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if _, exists := tables.nameByKey[key]; !exists {
			tables.nameByKey[key] = value
		}
	}
	return nil
}

// sceneDisplayName 角色所在（下线）场景显示名：先查 stringname 的 config 键、
// 再查 resource 键，均未命中回退 scene_name_NNN（N=scenes.ini 场景 id，客户端
// 命名键可本地化），表缺失/未知场景回退 scene_name_017。
func sceneDisplayName(config, resource string) string {
	tables := sceneNameTablesOf()
	configLower := strings.ToLower(config)
	resourceLower := strings.ToLower(resource)
	if name := tables.nameByKey[configLower]; name != "" {
		return name
	}
	if name := tables.nameByKey[resourceLower]; name != "" {
		return name
	}
	if configLower != "" {
		if id, ok := tables.idByConfig[configLower]; ok {
			return fmt.Sprintf("scene_name_%03d", id)
		}
	}
	if resourceLower != "" {
		if id, ok := tables.idByResource[resourceLower]; ok {
			return fmt.Sprintf("scene_name_%03d", id)
		}
	}
	return fallbackSceneNameKey
}

// bookIntroKey 剧本标题本地化键；未知剧本回退 book 原值。
func bookIntroKey(book string) string {
	if code, ok := bookIntroCodeByNumber[book]; ok {
		return "introduction_" + code + "01_title"
	}
	return book
}

// roleListTitle 选人界面实力称号（f23）：等级 → title0NN 档位键，客户端按
// desc_title0NN 本地化。阶梯 = share/faculty/facultylevel.ini [config] 权威
// 表（0-240 共 32 阶）：0-135 每 5 点 title001-025（深藏不露）、136-240 每
// 15 点 title026-032（深不可测…搏日擒天）。旧实现 131-135 段误入 title026、
// >165 截断 title027，均与客户端表不符（盘点 §2.7/§2.8）。
func roleListTitle(powerLevel int) string {
	if powerLevel <= 5 {
		return "title001"
	}
	switch {
	case powerLevel <= 120:
		title := (powerLevel-1)/5 + 1
		if title > 24 {
			title = 24
		}
		return fmt.Sprintf("title%03d", title)
	case powerLevel <= 135:
		return "title025"
	default:
		// 136 起 15 点一阶：136-150=026 … 226-240=032，超出封顶 032。
		title := 26 + (powerLevel-136)/15
		if title > 32 {
			title = 32
		}
		return fmt.Sprintf("title%03d", title)
	}
}
