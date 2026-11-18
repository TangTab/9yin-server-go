package main

import (
	"fmt"
	"go/format"
	"strings"
)

const ecsImport = "github.com/lizhanyin/9yin-server-go/internal/ecs"

// 原版属性类型 → Go 类型。object → ecs.EntityID（generation 安全句柄）。
var goTypes = map[string]string{
	"byte":    "uint8",
	"word":    "uint16",
	"dword":   "uint32",
	"int":     "int32",
	"int64":   "int64",
	"float":   "float32",
	"double":  "float64",
	"string":  "string",
	"widestr": "string",
	"object":  "ecs.EntityID",
}

func goType(t string) (string, bool) {
	gt, ok := goTypes[strings.ToLower(strings.TrimSpace(t))]
	return gt, ok
}

// camel 分词转大驼峰；首字符为数字时加 P 前缀。
func camel(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == ' ' || r == '-' })
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(p)
		if r[0] >= '0' && r[0] <= '9' {
			b.WriteByte('P')
		}
		if r[0] >= 'a' && r[0] <= 'z' {
			r[0] = r[0] - 'a' + 'A'
		}
		b.WriteRune(r[0])
		b.WriteString(string(r[1:]))
	}
	return b.String()
}

// typeName 组件类型名：由 run() 按全路径分段驼峰预分配（含去重），
// 此处只读结果。如 rule/fight/flow_new.xml → RuleFightFlowNew。
func typeName(o *ObjectDef) string { return o.GoName }

// assignGoNames 为全部对象分配唯一类型名：路径各段驼峰拼接；
// 极端撞名（如 fight_new 与 fightNew）加 X2/X3 后缀并在 warn 记录。
func assignGoNames(objs []*ObjectDef, warn *[]string) {
	used := map[string]int{}
	for _, o := range objs {
		base := strings.TrimSuffix(o.File, ".xml")
		var b strings.Builder
		for _, seg := range strings.Split(base, "/") {
			b.WriteString(camel(seg))
		}
		n := b.String()
		if used[n] > 0 {
			used[n]++
			newName := fmt.Sprintf("%sX%d", n, used[n])
			*warn = append(*warn, fmt.Sprintf("类型名冲突：%s 与先前对象同为 %s，改为 %s", o.File, n, newName))
			n = newName
		} else {
			used[n] = 1
		}
		o.GoName = n
	}
}

// fieldInfo 结构体字段与元数据共用的命名结果（两处严格一致）。
type fieldInfo struct {
	Prop   PropDef
	GoName string
	GoType string
	Known  bool
}

// fieldInfos 统一计算字段名（含冲突重命名），供结构体与元数据生成共用。
func fieldInfos(o *ObjectDef, warn *[]string) []fieldInfo {
	all := make([]PropDef, 0, len(o.Internals)+len(o.Props))
	all = append(all, o.Internals...)
	all = append(all, o.Props...)

	used := map[string]int{}
	out := make([]fieldInfo, 0, len(all))
	for _, p := range all {
		gt, ok := goType(p.Type)
		if !ok {
			gt = "int64"
			*warn = append(*warn, fmt.Sprintf("%s: 属性 %s 未知类型 %q，按 int64 处理", o.File, p.Name, p.Type))
		}
		name := camel(p.Name)
		if used[name] > 0 {
			used[name]++
			newName := fmt.Sprintf("%sX%d", name, used[name])
			*warn = append(*warn, fmt.Sprintf("%s: 属性 %s 命名冲突，字段改为 %s", o.File, p.Name, newName))
			name = newName
		} else {
			used[name] = 1
		}
		out = append(out, fieldInfo{Prop: p, GoName: name, GoType: gt, Known: ok})
	}
	return out
}

// trueFlags 按固定顺序输出为 true 的标志（如 " Public: true, Save: true,"）。
func trueFlags(m map[string]bool) string {
	var parts []string
	for _, k := range []string{"Internal", "Public", "Private", "Save", "Realtime"} {
		if m[k] {
			parts = append(parts, k+": true")
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, ", ") + ","
}

// genGroupFile 生成一个组的组件文件（已 gofmt）。
func genGroupFile(objs []*ObjectDef, warn *[]string) ([]byte, error) {
	var b strings.Builder
	needsECS := false

	for _, o := range objs {
		fis := fieldInfos(o, warn)
		for _, fi := range fis {
			// 只有结构体字段真正声明 ecs.EntityID 才需要 import；
			// record 列的 GoType 只是元数据字符串，不构成引用。
			if fi.GoType == "ecs.EntityID" {
				needsECS = true
			}
		}

		tn := typeName(o)
		fmt.Fprintf(&b, "// %s 对应 %s（internals %d / properties %d / records %d）。\n",
			tn, o.File, len(o.Internals), len(o.Props), len(o.Records))
		fmt.Fprintf(&b, "type %s struct {\n", tn)
		for _, fi := range fis {
			fmt.Fprintf(&b, "\t%s %s\n", fi.GoName, fi.GoType)
		}
		b.WriteString("}\n\n")

		fmt.Fprintf(&b, "// %sMeta 属性/表格元数据（sync、persist、proto 共用）。\n", tn)
		fmt.Fprintf(&b, "var %sMeta = ComponentMeta{\n", tn)
		fmt.Fprintf(&b, "\tName: %q,\n\tFile: %q,\n", tn, o.File)
		if len(fis) > 0 {
			b.WriteString("\tFields: []FieldMeta{\n")
			for _, fi := range fis {
				flagStr := trueFlags(map[string]bool{
					"Internal": fi.Prop.Internal, "Public": fi.Prop.Public, "Private": fi.Prop.Private,
					"Save": fi.Prop.Save, "Realtime": fi.Prop.Realtime,
				})
				fmt.Fprintf(&b, "\t\t{GoName: %q, PropName: %q, PropType: %q, GoType: %q,%s Desc: %q},\n",
					fi.GoName, fi.Prop.Name, fi.Prop.Type, fi.GoType, flagStr, fi.Prop.Desc)
			}
			b.WriteString("\t},\n")
		}
		if len(o.Records) > 0 {
			b.WriteString("\tRecords: []RecordMeta{\n")
			for _, r := range o.Records {
				flagStr := trueFlags(map[string]bool{
					"Public": r.Public, "Private": r.Private, "Save": r.Save,
				})
				fmt.Fprintf(&b, "\t\t{Name: %q, Cols: %d, MaxRows: %d,%s Desc: %q", r.Name, r.Cols, r.MaxRows, flagStr, r.Desc)
				if len(r.Columns) > 0 {
					fmt.Fprintf(&b, ", Columns: []ColumnMeta{\n")
					for i, c := range r.Columns {
						gt, ok := goType(c.Type)
						if !ok {
							gt = "int64"
							*warn = append(*warn, fmt.Sprintf("%s: 表 %s 列 %d 未知类型 %q，按 int64 处理", o.File, r.Name, i, c.Type))
						}
						fmt.Fprintf(&b, "\t\t\t{Index: %d, PropType: %q, GoType: %q, Desc: %q},\n", i, c.Type, gt, c.Desc)
					}
					b.WriteString("\t\t}},\n")
				} else {
					b.WriteString("},\n")
				}
			}
			b.WriteString("\t},\n")
		}
		if len(o.Includes) > 0 {
			b.WriteString("\tIncludes: []string{\n")
			for _, inc := range o.Includes {
				fmt.Fprintf(&b, "\t\t%q,\n", inc)
			}
			b.WriteString("\t},\n")
		}
		b.WriteString("}\n\n")
	}

	var head strings.Builder
	head.WriteString("// Code generated by tools/structgen from testdata/struct; DO NOT EDIT.\n\n")
	head.WriteString("package component\n\n")
	if needsECS {
		fmt.Fprintf(&head, "import %q\n\n", ecsImport)
	}

	src, err := format.Source([]byte(head.String() + b.String()))
	if err != nil {
		return nil, fmt.Errorf("gofmt 失败（生成物语法错误）: %w\n--- 原始输出 ---\n%s", err, head.String()+b.String())
	}
	return src, nil
}

// genMetaFile 生成元数据类型与注册表文件。
func genMetaFile(objs []*ObjectDef) ([]byte, error) {
	var b strings.Builder
	b.WriteString("// Code generated by tools/structgen from testdata/struct; DO NOT EDIT.\n\n")
	b.WriteString("package component\n\n")

	b.WriteString(`// FieldMeta 单个属性字段的元数据。PropType 为原版声明类型（byte/word/int/
// int64/float/double/string/widestr/object/dword），GoType 为映射后的 Go 类型。
// Public/Private 决定客户端可见性，Save 决定是否落库，Realtime 决定实时推送。
type FieldMeta struct {
	GoName, PropName, PropType, GoType, Desc string
	Internal, Public, Private, Save, Realtime bool
}

// ColumnMeta 表格列元数据（列无名，身份是位置索引）。
type ColumnMeta struct {
	Index                int
	PropType, GoType, Desc string
}

// RecordMeta 表格（Record）元数据：行级增删改，需行操作粒度同步。
type RecordMeta struct {
	Name                  string
	Cols, MaxRows         int
	Public, Private, Save bool
	Desc                  string
	Columns               []ColumnMeta
}

// ComponentMeta 一个 struct XML 文件的完整元数据。
type ComponentMeta struct {
	Name, File string
	Fields     []FieldMeta
	Records    []RecordMeta
	Includes   []string
}

`)

	b.WriteString("// All 全部组件元数据，按（组, 文件）排序。\nvar All = []*ComponentMeta{\n")
	for _, o := range objs {
		fmt.Fprintf(&b, "\t&%sMeta,\n", typeName(o))
	}
	b.WriteString("}\n\n")

	b.WriteString(`// ByName 组件类型名索引。
var ByName = func() map[string]*ComponentMeta {
	m := make(map[string]*ComponentMeta, len(All))
	for _, c := range All {
		m[c.Name] = c
	}
	return m
}()
`)

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("gofmt 失败: %w", err)
	}
	return src, nil
}
