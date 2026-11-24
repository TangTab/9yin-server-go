// item：九阴设计态物品表 → 服务端 ini/Item XML 构建器（黑金物品配置
// 格式）。tools/datagen 的 item 构建器：输入输出路径由统一入口解析后经
// Options 传入，本包不做路径探测。
//
// 数据链与技能/NPC 同款（docs/official/技能服务端XML构建链与m211受理链.md
// /NPC服务端XML构建链与黑金样板.md）：「离线构建服务端 XML → 服务端只读
// 构建产物」。黑金运行时读 ini/Item/*.xml（dump_config.xml 的道具转储
// 清单：ini/Item/Equipment.xml、ItemUse.xml、Material.xml、Weapon.xml…），
// share/item/*.ini 是设计态表；九阴只有设计态表随客户端发布，服务端 XML
// 须重建。
//
// 黑金样板（参考资源根下的 ini/Item，37 档）的模型逐层对应到九阴产物：
//
//	黑金 path_config.xml    模板登记表（ID/Type/Path）→ path_config.xml
//	黑金 Equipment/Weapon/  物品大类分档 → 同构 11 档（九阴 105,085 件按
//	    ItemUse/Rune/…          script/EquipType/ViewID 归类，见 mapping.go）
//	黑金 ItemArtStatic.xml  静态包（ID="-1"/"0" 列序/列宽头 + 数据行，
//	    item_logic_pack.xml     物品行以 ArtPack/LogicPack 整型引用）→ 9 个
//	    …                       静态包，表头两行照留（黑金已发布形态）
//	黑金 ItemTypeList.xml   物品类型树（ID/FatherType/Desc）→ 由客户端
//	                            lua64/share/itemtype_define.lua 生成
//
// 输入（Options.In = 客户端资源解压根下的 share/item）：
//
//   - 模板：tool_item.ini / equipment.ini / tool_item_hw.ini / giftbox.ini
//   - 静态包：item_func_static / item_logical_static / itemartstatic /
//     item_use_skill_static / item_learn_skill_static / item_task_static /
//     item_ingredient_static / item_prop_modify_static / item_seed_static
//
// share/item 下其余表（fuse_*/life_*/exchangeitem 等合成、生活、兑换、精炼
// 域）不在物品模板范围，逐档记入报告 skipped。该版资源没有的模板表（如
// 客户端 share/item 无 giftbox.ini）记入警告并跳过，不中断其余档的构建。
//
// 输出（服务端方言 <Object><Property .../>，UTF-8）+ JSON 审计报告（每档
// 行数/script 分布/列出处、源表脏数据、静态包引用完整性、使用链统计）+
// Verify 回读校验（config.LoadObjectFile：行数、ID 序、首行属性）。
//
// 属性词汇：源表键名即九阴运行时词汇，缺省原样直通；按三处真档给出每个
// 输出列出处并写入报告——
//
//	heijin  黑金 ini/Item 样本行属性名（Options.Heijin，只读不动）
//	struct  struct/item/*.xml（含 includes）属性名
//	define  客户端 lua64/share/item_static_data_define.lua 的静态包列名
//	native  仅见于九阴源表
//	rename  审定改名：`@记录键`（@CompensateItemRec 等）去 @ 成属性名，
//	        多行以 ; 连接（黑金 RandSerial="1,3;2,6" 同款行串方言）
//
// 出处分类只入报告、不改输出：九阴官服/客户端按源表键名取物品属性
// （form_bag_func.lua QueryProp("FuncPack")、useitem_new.lua 同族
// nx_query_property(item,"Skill1")），黑金词汇（Desc/Only/CantSell…）
// 不得据其改名。
//
// 源表脏数据处理（全部入报告）：[-1]/[0] 表头元数据行在模板档丢弃（黑金
// 模板档无表头）、静态包照留；`[//说明]` 注释节与空节（运营备用位
// Operation_item_NNNN 等）跳过；非法 XML 属性名的键丢弃；节内重复键输出
// 原名_2 序号；空值不写。
package item

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/config"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/tools/datagen/genutil"
)

// 文件登记名（相对输出根）。
const (
	filePathConfig  = "path_config.xml"
	fileItemTypeLst = "ItemTypeList.xml"
)

// Options 一次物品构建的输入输出（路径由 tools/datagen 解析好传入）。
type Options struct {
	In             string // 设计态物品表根：客户端资源根下的 share/item
	Out            string // 服务端 ini/Item 输出根
	Heijin         string // 黑金参考资源根（h001.rep 所在目录）：ini/Item 样本与 struct/item schema，列出处判定用，只读
	LuaDefine      string // 客户端 lua64/share/item_static_data_define.lua（列出处判定）
	ItemTypeDefine string // 客户端 lua64/share/itemtype_define.lua（ItemTypeList 源；空串则不产出该档）
	ReportPath     string // 审计报告 JSON
	Verify         bool   // 生成后用 config.LoadObjectFile 回读校验行数、ID 序与首行属性
}

// Run 构建物品 XML：落盘 + 报告 + 控制台摘要。失败时也先打印已装填的
// 摘要，便于先看问题再报错。
func Run(opt Options) error {
	heijinDir := filepath.Join(opt.Heijin, "ini", "Item")
	structRoot := filepath.Join(opt.Heijin, "struct")
	report, err := run(opt.In, opt.Out, heijinDir, structRoot, opt.LuaDefine, opt.ItemTypeDefine, opt.ReportPath, opt.Verify)
	if report != nil {
		printSummary(report)
	}
	return err
}

// ---- 报告 ----

type refReport struct {
	Attr       string   `json:"attr"`
	Pack       string   `json:"pack"`
	Refs       int      `json:"refs"`
	Resolved   int      `json:"resolved"`
	Unresolved int      `json:"unresolved"`
	Sample     []string `json:"unresolved_sample,omitempty"`
}

type skippedReport struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

type useChainReport struct {
	WithFuncPack      int          `json:"with_funcpack"`       // 带 FuncPack（可使用）的物品行
	FuncPackResolved  int          `json:"funcpack_resolved"`   // 其功能包在 item_func_static 有行
	FuncPackLuaScript int          `json:"funcpack_lua_script"` // 功能包带 LuaScript（官服脚本）
	FuncPackNoPrepare int          `json:"funcpack_no_prepare"` // 功能包无 PrepareTime（瞬时）
	LearnSkillItems   int          `json:"learn_skill_items"`   // 带 LearnSkillPack 的行
	UseSkillItems     int          `json:"use_skill_items"`     // 带 UseSkillPack 的行
	BoxItems          int          `json:"box_items"`           // 带 DropID 的行
	CapitalItems      int          `json:"capital_items"`       // 带 CapitalType/CapitalValue 的行
	UsableByScript    []countEntry `json:"usable_by_script"`    // 带 FuncPack 行的 script 分布
}

type verifyReport struct {
	Files    int      `json:"files"`
	Failures []string `json:"failures,omitempty"`
}

type sourceReport struct {
	Source          string   `json:"source"`
	Sections        int      `json:"sections"`
	Rows            int      `json:"rows"`
	CommentSections []string `json:"comment_sections,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

type genReport struct {
	Input       string `json:"input"`
	Out         string `json:"out"`
	ReportPath  string `json:"report_path,omitempty"`
	Heijin      string `json:"heijin,omitempty"`
	ItemTypeSrc string `json:"itemtype_src,omitempty"`
	Vocabulary  struct {
		Struct           string `json:"struct"`
		Heijin           string `json:"heijin,omitempty"`
		Define           string `json:"define"`
		StructProperties int    `json:"struct_properties"`
		HeijinAttrs      int    `json:"heijin_attrs"`
		DefineNames      int    `json:"define_names"`
	} `json:"vocabulary"`
	Sources   []sourceReport  `json:"sources"`
	Files     []fileReport    `json:"files"`
	Refs      []refReport     `json:"refs"`
	UseChain  *useChainReport `json:"use_chain,omitempty"`
	Skipped   []skippedReport `json:"skipped"`
	FilesOut  int             `json:"files_out"`
	TotalRows int             `json:"total_rows"`
	Warnings  []string        `json:"warnings,omitempty"`
	Verify    *verifyReport   `json:"verify,omitempty"`
}

// run 构建 + 落盘 + 报告（控制台输出留给 printSummary；失败时也带回
// 已装填的报告，便于先看摘要再报错）。
func run(in, out, heijin, structRoot, luaDefine, itemTypeDefine, reportPath string, verify bool) (*genReport, error) {
	if info, err := res.Stat(in); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("物品表目录不可用 %s", in)
	}
	report := &genReport{Input: in, Out: out, ReportPath: reportPath}
	vocab := &vocabulary{}
	if heijin != "" {
		if attrs := loadHeiJinAttrs(heijin); len(attrs) > 0 {
			vocab.heijinAttrs = attrs
			report.Heijin, report.Vocabulary.Heijin = heijin, heijin
			report.Vocabulary.HeijinAttrs = len(attrs)
		} else {
			report.Warnings = append(report.Warnings, fmt.Sprintf("黑金样本缺席或无行 %s（列出处不含 heijin）", heijin))
		}
	}
	if props, err := loadStructProperties(structRoot); err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("struct schema 未装载，列出处退化: %v", err))
	} else {
		vocab.structProps = props
		report.Vocabulary.Struct = structRoot
		report.Vocabulary.StructProperties = len(props)
	}
	if names := loadDefineNames(luaDefine); len(names) > 0 {
		vocab.defineNames = names
		report.Vocabulary.Define = luaDefine
		report.Vocabulary.DefineNames = len(names)
	} else {
		report.Warnings = append(report.Warnings, fmt.Sprintf("lua 静态包列名文件缺席 %s", luaDefine))
	}

	// ---- 模板：源表 → 分档累加器 ----
	builders := map[string]*kindBuilder{}
	var order []string
	for _, spec := range kindSpecs {
		builders[spec.File] = newKindBuilder(spec)
		order = append(order, spec.File)
	}
	converted := map[string]bool{}
	for _, source := range templateSources {
		path := filepath.Join(in, source)
		raw, err := res.ReadFile(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// 该版资源没有这张模板表（如客户端 share/item 无 giftbox.ini）：
				// 记入警告与报告，其余档照常构建。
				report.Warnings = append(report.Warnings, fmt.Sprintf("模板源缺席，已跳过 %s", path))
				report.Skipped = append(report.Skipped, skippedReport{File: source, Reason: "模板源缺席（该版资源无此表）"})
				continue
			}
			return report, fmt.Errorf("读源表 %s: %w", path, err)
		}
		converted[source] = true
		t := parseINI(raw, source)
		sr := sourceReport{Source: path, Sections: len(t.Sections), Warnings: t.Warnings}
		for _, section := range t.Sections {
			if section.Name == "-1" || section.Name == "0" {
				continue // 表头元数据：模板档不保留（黑金模板档无表头）
			}
			if strings.HasPrefix(section.Name, "//") {
				sr.CommentSections = append(sr.CommentSections, section.Name)
				continue
			}
			row := map[string]string{}
			for i, key := range section.Keys {
				row[key] = section.Vals[i]
			}
			builder := builders[kindOf(row)]
			builder.addSource(source)
			builder.addSection(section)
			sr.Rows++
		}
		report.Sources = append(report.Sources, sr)
	}

	var written []*builtFile
	var kindFiles []*builtFile
	for _, name := range order {
		if len(builders[name].rows) == 0 {
			continue // 空档不产出、不登记（黑金 path_config 只登实有档）
		}
		file, fr := builders[name].finish(vocab)
		kindFiles = append(kindFiles, file)
		written = append(written, file)
		report.Files = append(report.Files, *fr)
	}

	// ---- 静态包 ----
	packIDs := map[string]map[string]bool{}               // Ref 键 → 行 ID 集
	packRows := map[string]map[string]map[string]string{} // Ref 键 → 行 ID → 属性
	for _, spec := range packSpecs {
		path := filepath.Join(in, spec.Source)
		raw, err := res.ReadFile(path)
		if err != nil {
			return report, fmt.Errorf("读静态包 %s: %w", path, err)
		}
		converted[spec.Source] = true
		file, fr, rows := buildPack(parseINI(raw, spec.Source), spec, vocab)
		packIDs[spec.Ref] = map[string]bool{}
		for _, id := range file.IDs {
			if id != "-1" && id != "0" {
				packIDs[spec.Ref][id] = true
			}
		}
		packRows[spec.Ref] = rows
		written = append(written, file)
		report.Files = append(report.Files, *fr)
	}

	// ---- 引用完整性 + 使用链统计（全量物品行过一遍） ----
	var allRows []*itemRow
	for _, name := range order {
		allRows = append(allRows, builders[name].rows...)
	}
	report.Refs = auditRefs(allRows, packIDs)
	report.UseChain = auditUseChain(allRows, packRows)

	// ---- 类型树 + 登记表 ----
	if itemTypeDefine != "" {
		raw, err := res.ReadFile(itemTypeDefine)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("itemtype_define 读失败，不产出 %s: %v", fileItemTypeLst, err))
		} else {
			file := buildItemTypeList(parseItemTypeList(string(raw)))
			written = append(written, file)
			report.ItemTypeSrc = itemTypeDefine
			report.Files = append(report.Files, fileReport{
				Title: "物品类型总表", Source: itemTypeDefine, Output: file.RelPath, Rows: len(file.IDs),
				Note: "ID/FatherType/Desc；父类型=最内层包含区间的下界，根 55555",
			})
		}
	} else {
		report.Warnings = append(report.Warnings, "itemtype_define.lua 缺席（-itemtype），未产出 "+fileItemTypeLst)
	}
	pathConfig := buildPathConfig(kindFiles)
	written = append(written, pathConfig)
	report.Files = append(report.Files, fileReport{
		Title: "模板登记表", Output: pathConfig.RelPath, Rows: len(pathConfig.IDs),
		Note: "黑金 path_config.xml 同构：ID/Type/Path",
	})

	// ---- 未转换清单 ----
	entries, err := res.ReadDir(in)
	if err != nil {
		return report, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name = strings.TrimSuffix(name, string(filepath.Separator))
			if !converted[name] {
				report.Skipped = append(report.Skipped, skippedReport{name, skipReason(name)})
			}
			continue
		}
		if converted[name] {
			continue
		}
		report.Skipped = append(report.Skipped, skippedReport{name, skipReason(name)})
	}

	// ---- 落盘（行尾跟随数据根里同名现有文件，见 genutil）----
	for _, file := range written {
		path := filepath.Join(out, filepath.FromSlash(file.RelPath))
		if err := genutil.WriteFile(path, file.Data); err != nil {
			return report, err
		}
		report.FilesOut++
		report.TotalRows += len(file.IDs)
	}
	if verify {
		report.Verify = verifyWritten(out, written)
	}
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return report, err
	}
	blob, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return report, err
	}
	if err := os.WriteFile(reportPath, append(blob, '\n'), 0o644); err != nil {
		return report, err
	}
	if report.Verify != nil && len(report.Verify.Failures) > 0 {
		return report, fmt.Errorf("verify 失败 %d 个文件", len(report.Verify.Failures))
	}
	return report, nil
}

// printSummary 控制台摘要（报告 JSON 的同名子集；verify 结论在此打印，
// 构建期不打印——单测走 run 静默）。
func printSummary(report *genReport) {
	fmt.Printf("item: %d 文件 %d 行 → %s\n", report.FilesOut, report.TotalRows, report.Out)
	kinds := map[string]int{}
	for _, fr := range report.Files {
		for _, column := range fr.Columns {
			kinds[column.Kind]++
		}
	}
	fmt.Printf("item: 列出处 heijin=%d struct=%d define=%d native=%d rename=%d\n",
		kinds[kindHeijin], kinds[kindStruct], kinds[kindDefine], kinds[kindNative], kinds[kindRename])
	for _, fr := range report.Files {
		fmt.Printf("  %-24s %6d 行  %s\n", fr.Output, fr.Rows, fr.Title)
		if len(fr.RecordKeys) > 0 {
			fmt.Printf("      @记录键: %v\n", fr.RecordKeys)
		}
		if len(fr.InvalidKeys) > 0 {
			fmt.Printf("      非法属性名键丢弃: %v\n", fr.InvalidKeys)
		}
		if fr.DuplicateIDs > 0 {
			fmt.Printf("      重复 ID %d 个（行全保留，装载首行胜出）\n", fr.DuplicateIDs)
		}
		if fr.EmptySections > 0 {
			fmt.Printf("      空节跳过 %d 个\n", fr.EmptySections)
		}
	}
	for _, ref := range report.Refs {
		fmt.Printf("  引用 %-15s → %-30s %6d 引用 / 未解析 %d\n", ref.Attr, ref.Pack, ref.Refs, ref.Unresolved)
	}
	if uc := report.UseChain; uc != nil {
		fmt.Printf("  使用链: 带FuncPack %d（功能包有行 %d：带LuaScript %d / 无读条 %d）LearnSkillPack %d UseSkillPack %d DropID %d Capital %d\n",
			uc.WithFuncPack, uc.FuncPackResolved, uc.FuncPackLuaScript, uc.FuncPackNoPrepare,
			uc.LearnSkillItems, uc.UseSkillItems, uc.BoxItems, uc.CapitalItems)
	}
	fmt.Printf("  未转换表 %d 个（见报告 skipped）\n", len(report.Skipped))
	for _, w := range report.Warnings {
		fmt.Println("  警告:", w)
	}
	fmt.Printf("item: 报告 %s\n", report.ReportPath)
	if report.Verify != nil {
		for _, failure := range report.Verify.Failures {
			fmt.Fprintln(os.Stderr, "item: verify 失败:", failure)
		}
		if len(report.Verify.Failures) == 0 {
			fmt.Printf("item: verify 通过（%d 个文件经 config.LoadObjectFile 回读一致）\n", report.Verify.Files)
		}
	}
}

// auditRefs 静态包引用完整性：物品行 Ref 键（非空且非 0）→ 包内是否有行。
func auditRefs(rows []*itemRow, packIDs map[string]map[string]bool) []refReport {
	var reports []refReport
	for _, spec := range packSpecs {
		ids := packIDs[spec.Ref]
		ref := refReport{Attr: spec.Ref, Pack: spec.Output}
		seen := map[string]bool{}
		for _, row := range rows {
			value := row.values[spec.Ref]
			if value == "" || value == "0" {
				continue
			}
			ref.Refs++
			if ids[value] {
				ref.Resolved++
			} else {
				ref.Unresolved++
				if len(ref.Sample) < 8 && !seen[value] {
					seen[value] = true
					ref.Sample = append(ref.Sample, value)
				}
			}
		}
		reports = append(reports, ref)
	}
	return reports
}

// auditUseChain 使用链统计（msg31 受理面的数据底盘）。
func auditUseChain(rows []*itemRow, packRows map[string]map[string]map[string]string) *useChainReport {
	uc := &useChainReport{}
	byScript := map[string]int{}
	funcs := packRows["FuncPack"]
	for _, row := range rows {
		v := row.values
		if fp := v["FuncPack"]; fp != "" && fp != "0" {
			uc.WithFuncPack++
			byScript[v["script"]]++
			if pack, ok := funcs[fp]; ok {
				uc.FuncPackResolved++
				if pack["LuaScript"] != "" {
					uc.FuncPackLuaScript++
				}
				if pack["PrepareTime"] == "" || pack["PrepareTime"] == "0" {
					uc.FuncPackNoPrepare++
				}
			}
		}
		if x := v["LearnSkillPack"]; x != "" && x != "0" {
			uc.LearnSkillItems++
		}
		if x := v["UseSkillPack"]; x != "" && x != "0" {
			uc.UseSkillItems++
		}
		if v["DropID"] != "" {
			uc.BoxItems++
		}
		if v["CapitalType"] != "" || v["CapitalValue"] != "" {
			uc.CapitalItems++
		}
	}
	uc.UsableByScript = topCounts(byScript, 0)
	return uc
}

// verifyWritten 用服务端装载器回读全部生成物：行数、ID 序、首行非空属性。
func verifyWritten(out string, files []*builtFile) *verifyReport {
	result := &verifyReport{}
	for _, file := range files {
		result.Files++
		path := filepath.Join(out, filepath.FromSlash(file.RelPath))
		doc, err := config.LoadObjectFile(path)
		if err != nil {
			result.Failures = append(result.Failures, fmt.Sprintf("%s: %v", file.RelPath, err))
			continue
		}
		if len(doc.Rows) != len(file.IDs) {
			result.Failures = append(result.Failures, fmt.Sprintf("%s: 行数 %d != 期望 %d", file.RelPath, len(doc.Rows), len(file.IDs)))
			continue
		}
		bad := ""
		for i, id := range file.IDs {
			if got := doc.Rows[i]["ID"]; got != id {
				bad = fmt.Sprintf("第 %d 行 ID %q != 期望 %q", i+1, got, id)
				break
			}
		}
		for name, value := range file.First {
			if doc.Rows[0][name] != value {
				bad = fmt.Sprintf("首行属性 %s=%q != 期望 %q", name, doc.Rows[0][name], value)
				break
			}
		}
		if bad != "" {
			result.Failures = append(result.Failures, fmt.Sprintf("%s: %s", file.RelPath, bad))
		}
	}
	return result
}

// ---- 词汇出处 ----

var (
	propertyPattern = regexp.MustCompile(`<property\s+name\s*=\s*"?([A-Za-z_][A-Za-z0-9_]*)"?`)
	includePattern  = regexp.MustCompile(`<path\s+name\s*=\s*"?([^"\s>]+)"?`)
	commentPattern  = regexp.MustCompile(`(?s)<!--.*?-->`)
	heijinAttrLine  = regexp.MustCompile(`(?s)<Property\b[^>]*?/>`)
	heijinAttrName  = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*=\s*"`)
	definePattern   = regexp.MustCompile(`(?m)^\s*Item_([A-Za-z0-9_]+)\s*=\s*\d+\s*$`)
)

// 词汇出处分类。
const (
	kindHeijin = "heijin" // 黑金 ini/Item 样本行属性名
	kindStruct = "struct" // struct/item 服务端 schema 属性名
	kindDefine = "define" // lua 静态包列名
	kindNative = "native" // 仅见于九阴源表
	kindRename = "rename" // 审定改名
)

type vocabulary struct {
	structProps map[string]bool
	heijinAttrs map[string]bool
	defineNames map[string]bool
}

// classify 列出处（heijin > struct > define > native）。
func (v *vocabulary) classify(attr string) string {
	switch {
	case v.heijinAttrs[attr]:
		return kindHeijin
	case v.structProps[attr]:
		return kindStruct
	case v.defineNames[attr]:
		return kindDefine
	}
	return kindNative
}

// loadStructProperties struct/item/*.xml 全部属性名的并集（逐档跟随
// includes——路径相对 struct 根的上一级，官方 ResourcePath 口径；先剥 XML
// 注释，注释掉的 property 不进词汇）。
func loadStructProperties(structRoot string) (map[string]bool, error) {
	props := map[string]bool{}
	base := filepath.Dir(structRoot)
	loaded := map[string]bool{}
	var walk func(rel string) error
	walk = func(rel string) error {
		if loaded[rel] {
			return nil
		}
		loaded[rel] = true
		raw, err := res.ReadFile(filepath.Join(base, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		text := commentPattern.ReplaceAllString(string(toUTF8(raw)), "")
		for _, m := range propertyPattern.FindAllStringSubmatch(text, -1) {
			props[m[1]] = true
		}
		for _, m := range includePattern.FindAllStringSubmatch(text, -1) {
			if err := walk(strings.ReplaceAll(m[1], `\`, "/")); err != nil {
				return err
			}
		}
		return nil
	}
	dir := filepath.Join(structRoot, "item")
	entries, err := res.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
			continue
		}
		if err := walk("struct/item/" + entry.Name()); err != nil {
			return nil, err
		}
	}
	if len(props) == 0 {
		return nil, fmt.Errorf("struct/item 无属性")
	}
	return props, nil
}

// loadHeiJinAttrs 黑金 ini/Item 样本全部 <Property/> 行的属性名并集。
func loadHeiJinAttrs(dir string) map[string]bool {
	attrs := map[string]bool{}
	entries, err := res.ReadDir(dir)
	if err != nil {
		return attrs
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
			continue
		}
		raw, err := res.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		text := commentPattern.ReplaceAllString(string(toUTF8(raw)), "")
		for _, element := range heijinAttrLine.FindAllString(text, -1) {
			for _, m := range heijinAttrName.FindAllStringSubmatch(element, -1) {
				attrs[m[1]] = true
			}
		}
	}
	return attrs
}

// loadDefineNames lua 静态包列名：`Item_<包>_<列> = 序号` 取末段（LogicPack
// 的两段式 `Item_BindType` 亦然）。
func loadDefineNames(path string) map[string]bool {
	names := map[string]bool{}
	raw, err := res.ReadFile(path)
	if err != nil {
		return names
	}
	for _, m := range definePattern.FindAllStringSubmatch(string(raw), -1) {
		if i := strings.LastIndexByte(m[1], '_'); i >= 0 {
			names[m[1][i+1:]] = true
		} else {
			names[m[1]] = true
		}
	}
	return names
}
