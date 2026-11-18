// structgen：struct XML → Go 组件定义代码生成器。
//
// 输入：testdata/struct/ 下的对象属性定义 XML（源自原版 Res/struct/）：
//
//	<properties>  属性（name/type/public/private/save/realtime/desc）
//	<records>     表格（name/cols/maxrows/save/public/private + column type/desc）
//	<internals>   引擎内部属性覆写（与 properties 同构）
//	<includes>    跨文件组合引用（如 struct\rule\fight.xml）
//
// 输出（internal/component/，只放生成物）：
//   - <组>_gen.go：组件结构体 + ComponentMeta（属性元数据）
//   - meta_gen.go：FieldMeta/ColumnMeta/RecordMeta/ComponentMeta 类型 + All/ByName 注册表
//
// 产出被三处共用：sync（推送哪些字段）、persist（落库哪些字段）、proto（属性号）。
// 规格变更一律改 XML 后重跑生成，不手改产物。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	in := flag.String("in", "testdata/struct", "struct XML 输入目录")
	out := flag.String("out", "internal/component", "生成产物目录")
	flag.Parse()

	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "structgen:", err)
		os.Exit(1)
	}
}

func run(in, out string) error {
	// 1. 收集并解析全部 XML
	var objs []*ObjectDef
	var warn []string
	err := filepath.WalkDir(in, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".xml") {
			return err
		}
		rel, _ := filepath.Rel(in, path)
		rel = filepath.ToSlash(rel)
		o, perr := parseObject(path)
		if perr != nil {
			return fmt.Errorf("解析失败 %s: %w", rel, perr)
		}
		o.File = rel
		if i := strings.IndexByte(rel, '/'); i >= 0 {
			o.Group = rel[:i]
		} else {
			o.Group = ""
		}
		objs = append(objs, o)
		return nil
	})
	if err != nil {
		return err
	}
	if len(objs) == 0 {
		return fmt.Errorf("输入目录 %s 无 XML 文件", in)
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].File < objs[j].File })
	assignGoNames(objs, &warn)

	// 2. includes 存在性校验
	files := map[string]bool{}
	for _, o := range objs {
		files[o.File] = true
	}
	for _, o := range objs {
		for _, inc := range o.Includes {
			if !files[inc] {
				warn = append(warn, fmt.Sprintf("%s: include 目标 %s 不存在", o.File, inc))
			}
		}
	}

	// 3. 按组聚合
	groups := map[string][]*ObjectDef{}
	for _, o := range objs {
		groups[o.Group] = append(groups[o.Group], o)
	}
	groupNames := make([]string, 0, len(groups))
	for g := range groups {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)

	// 4. 清理旧生成物，避免删除 struct 后残留
	entries, _ := os.ReadDir(out)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "_gen.go") {
			if err := os.Remove(filepath.Join(out, e.Name())); err != nil {
				return err
			}
		}
	}

	// 5. 生成组文件
	var stats struct{ props, records int }
	for _, g := range groupNames {
		src, err := genGroupFile(groups[g], &warn)
		if err != nil {
			return err
		}
		name := g + "_gen.go"
		if g == "" {
			name = "root_gen.go"
		}
		if err := os.WriteFile(filepath.Join(out, name), src, 0o644); err != nil {
			return err
		}
		for _, o := range groups[g] {
			stats.props += len(o.Internals) + len(o.Props)
			stats.records += len(o.Records)
		}
	}

	// 6. 元数据类型 + 注册表
	metaSrc, err := genMetaFile(objs)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "meta_gen.go"), metaSrc, 0o644); err != nil {
		return err
	}

	// 7. 报告
	fmt.Printf("structgen: %d 个对象、%d 组、%d 属性、%d 表格 → %s\n",
		len(objs), len(groups), stats.props, stats.records, out)
	for _, w := range warn {
		fmt.Println("  警告:", w)
	}
	return nil
}
