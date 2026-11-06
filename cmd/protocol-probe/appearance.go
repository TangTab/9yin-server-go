package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type itemModels struct {
	male   string
	female string
}

type equipmentResolver struct {
	artByItem map[string]string
	models    map[string]itemModels
}

func loadEquipmentResolver() *equipmentResolver {
	root := os.Getenv("NINEYIN_SHARE_ROOT")
	if root == "" {
		root = filepath.Join(defaultModernShareRoot, "item")
	}
	r := &equipmentResolver{artByItem: map[string]string{}, models: map[string]itemModels{}}
	parseINI(filepath.Join(root, "equipment.ini"), func(section, key, value string) {
		if key == "ArtPack" {
			r.artByItem[section] = value
		}
	})
	parseINI(filepath.Join(root, "itemartstatic.ini"), func(section, key, value string) {
		models := r.models[section]
		switch key {
		case "MaleModel":
			models.male = value
		case "FemaleModel":
			models.female = value
		}
		r.models[section] = models
	})
	return r
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
