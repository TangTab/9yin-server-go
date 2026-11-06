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
	sex := byte(0) // 目前抓到的创建角色流量走的是男性路径。
	return roleVisual{
		sex:       sex,
		photo:     get(1, `icon\players\boy01_create.png`),
		face:      get(2, "333333333333333333333333333333333333333333X38\x01"),
		hair:      get(4, `obj\char\b_hair\b_hair1`),
		cloth:     equipmentModels.resolve(get(5, "cloth_b0001"), sex),
		pants:     equipmentModels.resolve(get(6, "pants_b0001"), sex),
		shoes:     equipmentModels.resolve(get(7, "shoes_b0001"), sex),
		actionSet: "0h",
	}
}

var equipmentModels = loadEquipmentResolver()
