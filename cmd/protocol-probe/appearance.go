package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type itemModels struct {
	male   string
	female string
}

// itemStatic 是 equipment.ini 的物品静态展示字段（背包/纸娃娃行需要），
// 对齐 Java EquipmentModels.ItemStatic：[config].ColorLevel/Level/
// MaxBuildCount/EffectPackID/EquipType + ArtPack→itemartstatic.ini 的
// Photo。
type itemStatic struct {
	photo         string
	colorLevel    int32
	level         int32
	maxBuildCount int32
	effectPackID  int32
	equipType     string
}

// artifactPacks 是套装式物品（身份装/时装套装衣）的分件 ArtPack
// （equipment.ini [config].HatArtPack/PlantsArtPack/ShoesArtPack——
// 注意官方拼写就是 Plants；Java EquipmentResolver 同表）。
type artifactPacks struct {
	hat    string
	plants string
	shoes  string
}

type equipmentResolver struct {
	artByItem  map[string]string
	models     map[string]itemModels
	statics    map[string]itemStatic
	photos     map[string]string
	actionSets map[string]string
	packs      map[string]artifactPacks
}

func loadEquipmentResolver() *equipmentResolver {
	root := os.Getenv("NINEYIN_SHARE_ROOT")
	if root == "" {
		root = filepath.Join(defaultModernShareRoot, "item")
	}
	r := &equipmentResolver{
		artByItem:  map[string]string{},
		models:     map[string]itemModels{},
		statics:    map[string]itemStatic{},
		photos:     map[string]string{},
		actionSets: map[string]string{},
		packs:      map[string]artifactPacks{},
	}
	staticFields := map[string]*itemStatic{}
	parseIntFallback := func(value string, fallback int32) int32 {
		if value == "" {
			return fallback
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fallback
		}
		return int32(parsed)
	}
	parseINI(filepath.Join(root, "equipment.ini"), func(section, key, value string) {
		if key == "ArtPack" {
			r.artByItem[section] = value
		}
		packs := r.packs[section]
		switch key {
		case "HatArtPack":
			packs.hat = value
		case "PlantsArtPack":
			packs.plants = value
		case "ShoesArtPack":
			packs.shoes = value
		}
		r.packs[section] = packs
		fields, ok := staticFields[section]
		if !ok {
			fields = &itemStatic{}
			staticFields[section] = fields
		}
		switch key {
		case "ColorLevel":
			fields.colorLevel = parseIntFallback(value, 1)
		case "Level":
			fields.level = parseIntFallback(value, 0)
		case "MaxBuildCount":
			fields.maxBuildCount = parseIntFallback(value, 2)
		case "EffectPackID":
			fields.effectPackID = parseIntFallback(value, 0)
		case "EquipType":
			fields.equipType = value
		}
	})
	for section, fields := range staticFields {
		r.statics[strings.ToLower(section)] = *fields
	}
	parseINI(filepath.Join(root, "itemartstatic.ini"), func(section, key, value string) {
		models := r.models[section]
		switch key {
		case "MaleModel":
			models.male = value
		case "FemaleModel":
			models.female = value
		case "Photo":
			r.photos[section] = value
		case "ActionSet":
			r.actionSets[section] = value
		}
		r.models[section] = models
	})
	// Photo 依赖 equipment.ini.ArtPack → itemartstatic.ini 的跨表联结，
	// 两表都解析完成后回填一次。
	for section := range staticFields {
		if art := r.artByItem[section]; art != "" {
			lower := strings.ToLower(section)
			updated := r.statics[lower]
			updated.photo = r.photos[art]
			r.statics[lower] = updated
		}
	}
	return r
}

// itemStatic 返回物品静态展示字段；未知配置返回 nil。
func (r *equipmentResolver) itemStatic(configID string) *itemStatic {
	if configID == "" {
		return nil
	}
	if stat, ok := r.statics[strings.ToLower(configID)]; ok {
		return &stat
	}
	return nil
}

// resolveActionSet 返回 itemartstatic [ArtPack].ActionSet（武器持械
// 动画组）；表缺配置返回空串由调用方按 ItemType 回退。
func (r *equipmentResolver) resolveActionSet(configID string) string {
	return r.actionSets[r.artByItem[configID]]
}

func parseINI(path string, consume func(section, key, value string)) {
	f, err := openResourceReader(path)
	if err != nil {
		return
	}
	defer f.Close()
	section := ""
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			continue
		}
		if section == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			consume(section, strings.TrimSpace(key), strings.TrimSpace(value))
		}
	}
}

func (r *equipmentResolver) resolve(config string, sex byte) string {
	if config == "" || strings.Contains(config, `\`) {
		return config
	}
	art := r.artByItem[config]
	models := r.models[art]
	if sex == 1 && models.female != "" {
		return models.female
	}
	if models.male != "" {
		return models.male
	}
	return config
}

// resolveComponent 套装分件解析（Java EquipmentResolver.resolvePack）：
// 配置空/已是模型路径 → "";分件 ArtPack 空或 "-1" 哨兵（该件缺席）→ "";
// 按性别取 MaleModel/FemaleModel（无表值空串——与 resolve 不同,不回退
// 配置名）。
func (r *equipmentResolver) resolveComponent(config string, art string, sex byte) string {
	if config == "" || strings.Contains(config, `\`) {
		return ""
	}
	if art == "" || art == "-1" {
		return ""
	}
	models := r.models[art]
	if sex == 1 && models.female != "" {
		return models.female
	}
	return models.male
}

// resolveHat/resolvePants/resolveShoes 身份装槽 16 整套与套装式时装
// (FashionCoat)自带的帽/裤/鞋分件模型。
func (r *equipmentResolver) resolveHat(config string, sex byte) string {
	return r.resolveComponent(config, r.packs[config].hat, sex)
}

func (r *equipmentResolver) resolvePants(config string, sex byte) string {
	return r.resolveComponent(config, r.packs[config].plants, sex)
}

func (r *equipmentResolver) resolveShoes(config string, sex byte) string {
	return r.resolveComponent(config, r.packs[config].shoes, sex)
}

type roleVisual struct {
	sex                            byte
	photo, face, hair              string
	cloth, pants, shoes, actionSet string
}

func resolveRoleVisual(appearance []string) roleVisual {
	get := func(index int, fallback string) string {
		if index >= 0 && index < len(appearance) && appearance[index] != "" {
			return appearance[index]
		}
		return fallback
	}
	sex := inferSex(appearance)
	photoDefault := `icon\players\boy01_create.png`
	hairDefault := `obj\char\b_hair\b_hair1`
	clothDefault, pantsDefault, shoesDefault := "cloth_b0001", "pants_b0001", "shoes_b0001"
	if sex == 1 {
		photoDefault = `icon\players\girl01_create.png`
		hairDefault = `obj\char\g_hair\g_hair1`
		clothDefault, pantsDefault, shoesDefault = "cloth_g0001", "pants_g0001", "shoes_g0001"
	}
	return roleVisual{
		sex:       sex,
		photo:     get(1, photoDefault),
		face:      get(2, "333333333333333333333333333333333333333333X38\x01"),
		hair:      get(4, hairDefault),
		cloth:     equipmentModels.resolve(get(5, clothDefault), sex),
		pants:     equipmentModels.resolve(get(6, pantsDefault), sex),
		shoes:     equipmentModels.resolve(get(7, shoesDefault), sex),
		actionSet: "0h",
	}
}

// inferSex 性别推断，对齐 Java RoleVisual.inferSex。DB 无独立性别列，
// 男女只体现在 role_appearances.appearance 串前缀（photo=girl|boy、
// 路径段 g_|b_、衣裤鞋裸配置 _g0xxx|_b0xxx），线上实证
// role 665=girl01/g_hair9/cloth_g0003、role 618=b_* 全路径。
// 优先级：photo > hair > 衣裤鞋；全空/无证据 → 0（男，与历史行为一致）。
func inferSex(appearance []string) byte {
	value := func(index int) string {
		if index >= 0 && index < len(appearance) {
			return appearance[index]
		}
		return ""
	}
	if byPhoto := photoSex(value(1)); byPhoto >= 0 {
		return byte(byPhoto)
	}
	if byHair := pathSex(value(4)); byHair >= 0 {
		return byte(byHair)
	}
	for index := 5; index <= 7; index++ {
		part := value(index)
		byCloth := pathSex(part)
		if byCloth < 0 && (strings.HasPrefix(part, "cloth_") || strings.HasPrefix(part, "pants_") || strings.HasPrefix(part, "shoes_")) {
			switch {
			case strings.Contains(part, "_g"):
				byCloth = 1
			case strings.Contains(part, "_b"):
				byCloth = 0
			}
		}
		if byCloth >= 0 {
			return byte(byCloth)
		}
	}
	return 0
}

// photoSex 按 photo=icon\players\boy01_create.png / girl01_create.png 判断。
func photoSex(photo string) int {
	lower := strings.ToLower(photo)
	switch {
	case strings.Contains(lower, "girl"):
		return 1
	case strings.Contains(lower, "boy"):
		return 0
	}
	return -1
}

// pathSex 按模型路径段判断：obj\char\g_hair\g_hair9（末段前缀 g_/b_）；
// 路径中段（obj\char\b_festival_1_106\b_cloth_1_106）任一段命中即判；
// 裸配置名 cloth_g0003 不含段，由调用方兜底。
func pathSex(value string) int {
	lower := strings.ToLower(value)
	if lower == "" {
		return -1
	}
	last := lower
	if slash := strings.LastIndex(lower, `\`); slash >= 0 {
		last = lower[slash+1:]
	}
	switch {
	case strings.HasPrefix(last, "g_"):
		return 1
	case strings.HasPrefix(last, "b_"):
		return 0
	}
	for _, segment := range strings.Split(lower, `\`) {
		switch {
		case strings.HasPrefix(segment, "g_"):
			return 1
		case strings.HasPrefix(segment, "b_"):
			return 0
		}
	}
	return -1
}

var equipmentModels = loadEquipmentResolver()
