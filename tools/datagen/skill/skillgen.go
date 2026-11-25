// skill：九阴设计态技能表 → 服务端 ini/Skill XML 构建器（黑金模型）。
// tools/datagen 的 skill 构建器：输入输出路径由统一入口解析后经 Options
// 传入，本包不做路径探测。
//
// 数据链与 NPC 同款（docs/official/NPC服务端XML构建链与黑金样板.md）：
// 「离线构建服务端 XML → 服务端只读构建产物」。官服运行时本就读 XML
// （FsGameLogic SkillModule::LoadResource @0x387310：
// sprintf("ini/skill/skill_new_%d.xml", GetMemberId()) +
// ini/Skill/Skill_Normal_Attack.xml），share/skill/*.ini 是设计态表。
//
// 黑金样板（参考资源根下的 ini/Skill 7 个原档）的模型逐层对应到九阴
// 产物（<out>/ 下）：
//
//	黑金 skill_new.xml.xml   技能对象行 → Skill_New/<逻辑类>.xml（按 script
//	                                      逻辑类分档＝服务端对象类，对应
//	                                      「按玩法域分组」）
//	黑金 Skill_Config.xml    引用注册表 → 九阴包按数值 ID 列引用
//	                                     （Consume=1440 → Skill_Consume 行
//	                                       1440），无需路径注册表
//	黑金 skill_static.xml    静态包行   → skill_static.xml（ID=StaticData；九阴
//	                                     产物接管同名路径，见 mapping.go 注）
//	黑金 Skill_Main_*.xml    深层包     → Skill_VarProp/<逻辑类>.xml 逐级可变
//	                                     属性行，引用 Skill_Consume /
//	                                     Skill_TargetShape / Skill_HitShape /
//	                                     Skill_Flow / Skill_LockData /
//	                                     Skill_UseLimit / Skill_RushRange.xml
//
// 命中结算（2026-11-25 战斗接入）沿同一模型再加三件：
//
//	逐级行 DamageCalculate（数值 ID）→ Skill_DamageCalculate.xml（r= 列表）
//	    → Skill_Formula.xml（Src/Prop 参数来源 + Formula 表达式）
//	buff_new → Buff_New/<逻辑类>.xml → buff_static.xml（ID=StaticData）
//	    → Buff_VarProp/Buffer.xml（ID=VarPropNo）
//
// 规范位归九阴产物：黑金原档只作属性词汇对照与 internal/fsgame 的测试
// 夹具（fsgame readResource 规范位缺席时回退参考资源根取同名黑金样本），
// 不再拷进输出目录。本项目是九阴服务端复刻而非黑金复刻。
//
// 输入（Options.In = 客户端资源解压根下的 share/skill）：
//
//   - skill_new.ini / skill_static.ini / skill_{normal,lock}_varprop.ini
//   - skill_consume.ini / attack_targetshape.ini / attack_hitshape.ini
//   - skill_lock_data.ini / skill_use_limit.ini / rush_range.ini
//   - damage_calculate.ini / formula.ini（伤害计算包与公式）
//   - buff_new.ini / buff_static.ini / buff_varprop.ini（buff 族）
//
// 输入（Options.Rule = 客户端资源根下的 share/rule）：rule/flow_config.ini
// 流程包。
//
// 输出（服务端方言 <Object><Property .../>，UTF-8）+ JSON 审计报告（列出处、
// 丢弃列、表头元数据行、合并冲突标记、重复键/ID）+ Verify 回读校验
// （config.LoadObjectFile：行数、ID 序、首行属性）。
//
// 属性词汇：技能源表是服务端设计态表，列名本身即服务端词汇（不同于 NPC 的
// 客户端列）。工具按三处真档给每个输出列定出处并写入报告：
//
//	direct  struct/skill/skillbase.xml（含 includes）属性名
//	heijin  黑金 ini/Skill 样本属性名
//	define  客户端 lua64/share/skill_static_data_define.lua 的静态包列名
//	native  仅见于九阴源表的服务端原生列
//
// 出处分类只入报告、不改输出：九阴官服运行时词汇就是源表列名
// （lua/buf/buf_jn_lmsj_03.lua:31 `nx_query_property(cur_skill,
// "AConsumeMP")`），struct 的 AConsumeCurMP 等是黑金词汇，不得据其改名。
//
// 源表脏数据处理（全部入报告）：[-1] 列序头/[0] 列宽头丢弃；残留的 SVN
// 合并冲突标记行（<<<<<<< .mine / >>>>>>> .rNNN，单侧无 =======）跳过、
// 其间数据保留；非法 XML 属性名的键丢弃；节内重复键输出为 _2 序号。
package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/config"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/tools/datagen/genutil"
)

// heijinSamples 黑金 ini/Skill 样本（词汇出处之二；只读不动）。
var heijinSamples = []string{
	"Skill_Config.xml", "Skill_Normal_Attack.xml", "skill_new.xml.xml", "skill_static.xml",
	"Buff_Config.xml", "Buff_Modify.xml", "buff_static.xml",
}

// Options 一次技能构建的输入输出（路径由 tools/datagen 解析好传入）。
type Options struct {
	In         string // 设计态技能表根：客户端资源根下的 share/skill
	Rule       string // 流程包根：客户端资源根下的 share/rule（含 flow_config.ini）；空串取 In 的同级 rule 目录
	Out        string // 服务端 ini/Skill 输出根
	Heijin     string // 黑金参考资源根（h001.rep 所在目录）：ini/Skill 样本与 struct/skill schema，列出处判定用，只读
	LuaDefine  string // 客户端 lua64/share/skill_static_data_define.lua（列出处判定）
	ReportPath string // 审计报告 JSON
	Verify     bool   // 生成后用 config.LoadObjectFile 回读校验行数、ID 序与首行属性
}

// Run 构建技能 XML：落盘 + 报告 + 控制台摘要。
func Run(opt Options) error {
	if opt.Rule == "" {
		opt.Rule = filepath.Join(filepath.Dir(filepath.Clean(opt.In)), "rule")
	}
	heijinDir := filepath.Join(opt.Heijin, "ini", "Skill")
	structRoot := filepath.Join(opt.Heijin, "struct")
	return run(opt.In, opt.Rule, opt.Out, structRoot, heijinDir, opt.LuaDefine, opt.ReportPath, opt.Verify)
}

// ---- 报告 ----

type tableReport struct {
	Title           string         `json:"title"`
	Source          string         `json:"source"`
	Outputs         []string       `json:"outputs"`
	Rows            int            `json:"rows"`
	Columns         []columnReport `json:"columns"`
	Dropped         []columnReport `json:"dropped,omitempty"`
	HeadersDropped  []string       `json:"headers_dropped,omitempty"`
	ConflictMarkers []int          `json:"conflict_markers,omitempty"`
	InvalidKeys     []string       `json:"invalid_keys,omitempty"`
	DuplicateKeys   int            `json:"duplicate_keys"`
	DuplicateIDs    int            `json:"duplicate_ids"`
	Fixed           []fixReport    `json:"fixed,omitempty"`
	Note            string         `json:"note,omitempty"`
	Warnings        []string       `json:"warnings,omitempty"`
}

type verifyReport struct {
	Files    int      `json:"files"`
	Failures []string `json:"failures,omitempty"`
}

type genReport struct {
	Input      string `json:"input"`
	Rule       string `json:"rule"`
	Out        string `json:"out"`
	Vocabulary struct {
		Struct           string `json:"struct"`
		Heijin           string `json:"heijin"`
		Define           string `json:"define"`
		StructProperties int    `json:"struct_properties"`
		HeijinAttrs      int    `json:"heijin_attrs"`
		DefineNames      int    `json:"define_names"`
	} `json:"vocabulary"`
	Tables    []tableReport `json:"tables"`
	Files     int           `json:"files"`
	TotalRows int           `json:"total_rows"`
	Warnings  []string      `json:"warnings,omitempty"`
	Verify    *verifyReport `json:"verify,omitempty"`
}

func run(in, rule, out, structRoot, heijin, luaDefine, reportPath string, verify bool) error {
	if info, err := res.Stat(in); err != nil || !info.IsDir() {
		return fmt.Errorf("技能表目录不可用 %s", in)
	}
	report := &genReport{Input: in, Rule: rule, Out: out}

	vocab := &vocabulary{}
	attrs := loadHeijinAttrs(heijin)
	if props, err := loadStructProperties(structRoot); err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("struct schema 未装载，列出处退化为 heijin/define/native: %v", err))
	} else {
		vocab.structProps = props
		report.Vocabulary.Struct = structRoot
		report.Vocabulary.StructProperties = len(props)
	}
	if len(attrs) > 0 {
		vocab.heijinAttrs = attrs
		report.Vocabulary.Heijin = heijin
		report.Vocabulary.HeijinAttrs = len(attrs)
	} else {
		report.Warnings = append(report.Warnings, fmt.Sprintf("黑金样本缺席 %s", heijin))
	}
	if names := loadDefineNames(luaDefine); len(names) > 0 {
		vocab.defineNames = names
		report.Vocabulary.Define = luaDefine
		report.Vocabulary.DefineNames = len(names)
	} else {
		report.Warnings = append(report.Warnings, fmt.Sprintf("lua 静态包列名文件缺席或无可识别的列定义 %s", luaDefine))
	}

	var written []*builtFile
	for _, spec := range specs {
		source := filepath.Join(in, filepath.FromSlash(spec.Source))
		if spec.Rule {
			source = filepath.Join(rule, filepath.FromSlash(spec.Source))
		}
		raw, err := res.ReadFile(source)
		if err != nil {
			return fmt.Errorf("读源表 %s: %w", source, err)
		}
		build := convertTable(parseINI(raw, spec.Source), spec, vocab)
		tr := tableReport{
			Title: spec.Title, Source: source, Rows: build.Rows, Note: spec.Note,
			HeadersDropped: build.HeadersDropped, ConflictMarkers: build.ConflictMarkers,
			InvalidKeys: build.InvalidKeys, DuplicateKeys: build.DuplicateKeys,
			DuplicateIDs: build.DuplicateIDs, Dropped: build.Dropped, Fixed: build.Fixed, Warnings: build.Warnings,
		}
		for _, file := range build.Files {
			path := filepath.Join(out, filepath.FromSlash(file.RelPath))
			if err := genutil.WriteFile(path, file.Data); err != nil {
				return err
			}
			tr.Outputs = append(tr.Outputs, file.RelPath)
			if tr.Columns == nil {
				tr.Columns = file.Columns
			}
			written = append(written, file)
		}
		report.Tables = append(report.Tables, tr)
		report.Files += len(build.Files)
		report.TotalRows += build.Rows
	}
	if verify {
		report.Verify = verifyWritten(out, written)
	}

	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return err
	}
	blob, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(reportPath, append(blob, '\n'), 0o644); err != nil {
		return err
	}

	// 摘要。
	kinds := map[string]int{}
	for _, tr := range report.Tables {
		for _, column := range append(append([]columnReport{}, tr.Columns...), tr.Dropped...) {
			kinds[column.Kind]++
		}
	}
	fmt.Printf("skill: %d 表 %d 行 %d 文件 → %s\n", len(report.Tables), report.TotalRows, report.Files, out)
	fmt.Printf("skill: 列出处 direct=%d heijin=%d define=%d native=%d rename=%d dropped=%d\n",
		kinds[kindDirect], kinds[kindHeijin], kinds[kindDefine], kinds[kindNative], kinds[kindRename], kinds[kindDropped])
	for _, tr := range report.Tables {
		fmt.Printf("  %-40s %6d 行 → %s\n", tr.Title, tr.Rows, strings.Join(tr.Outputs, ", "))
		if len(tr.ConflictMarkers) > 0 {
			fmt.Printf("      合并冲突标记 %d 处（行 %v）\n", len(tr.ConflictMarkers), tr.ConflictMarkers)
		}
		if len(tr.HeadersDropped) > 0 {
			fmt.Printf("      表头元数据行丢弃 %v\n", tr.HeadersDropped)
		}
		if len(tr.InvalidKeys) > 0 {
			fmt.Printf("      非法属性名键丢弃 %d 个\n", len(tr.InvalidKeys))
		}
		if tr.DuplicateKeys > 0 {
			fmt.Printf("      节内重复键 %d 处\n", tr.DuplicateKeys)
		}
		if tr.DuplicateIDs > 0 {
			fmt.Printf("      重复节名 %d 个\n", tr.DuplicateIDs)
		}
		for _, fix := range tr.Fixed {
			fmt.Printf("      审定修正 [%s] %s：%q → %q（%s）\n", fix.ID, fix.Key, fix.From, fix.To, fix.Reason)
		}
	}
	for _, w := range report.Warnings {
		fmt.Println("  警告:", w)
	}
	fmt.Printf("skill: 报告 %s\n", reportPath)
	if report.Verify != nil {
		if len(report.Verify.Failures) > 0 {
			for _, failure := range report.Verify.Failures {
				fmt.Fprintln(os.Stderr, "skill: verify 失败:", failure)
			}
			return fmt.Errorf("skill: verify 失败 %d 个文件", len(report.Verify.Failures))
		}
		fmt.Printf("skill: verify 通过（%d 个文件经 config.LoadObjectFile 回读一致）\n", report.Verify.Files)
	}
	return nil
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
	// 属性名/包含路径：引号可缺省（structgen 同款容错——原版存在裸值写法）。
	propertyPattern = regexp.MustCompile(`<property\s+name\s*=\s*"?([A-Za-z_][A-Za-z0-9_]*)"?`)
	includePattern  = regexp.MustCompile(`<path\s+name\s*=\s*"?([^"\s>]+)"?`)
	// 静态包列名：`Skill_LauchType = 0;` 形态（客户端 lua64 版行尾无分号，
	// 服务端 lua 版有——两者都要认），以及注释模板段里的
	// `Skill_AConsumeMP;`（流程/消耗包列，无列序）。
	definePattern  = regexp.MustCompile(`(?m)^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\d+\s*;?`)
	definePackLine = regexp.MustCompile(`(?m)^\s*Skill_([A-Za-z_][A-Za-z0-9_]*);`)
	commentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)
)

// loadStructProperties struct/skill/skillbase.xml 及其 <includes> 递归的
// 属性名并集（includes 路径相对 struct 根的上一级，官方 ResourcePath 口径）。
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
		path := filepath.Join(base, filepath.FromSlash(rel))
		raw, err := res.ReadFile(path)
		if err != nil {
			return err
		}
		// 先剥 XML 注释：注释掉的 property/include（skillbase.xml 的
		// BreakTime、modifyhurt.xml 即此形态）不进词汇、不跟随。
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
	if err := walk("struct/skill/skillbase.xml"); err != nil {
		return nil, err
	}
	return props, nil
}

// loadHeijinAttrs 黑金 ini/Skill 样本全部行的属性名并集。
func loadHeijinAttrs(dir string) map[string]bool {
	attrs := map[string]bool{}
	for _, name := range heijinSamples {
		doc, err := config.LoadObjectFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		for _, row := range doc.Rows {
			for key := range row {
				attrs[key] = true
			}
		}
	}
	return attrs
}

// loadDefineNames lua 静态包列名（`Name = 列序` 行，行尾分号可有可无）。
//
// 返回的是属性名本身：定义文件里写成 `Skill_LauchType = 0`，而输出 XML 的
// 属性名是不带 `Skill_` 前缀的 `LauchType`——不去前缀的话这个词汇集永远
// 匹配不上（下方 `Skill_AConsumeMP;` 形态本就只取后段，两处口径一致）。
func loadDefineNames(path string) map[string]bool {
	names := map[string]bool{}
	raw, err := res.ReadFile(path)
	if err != nil {
		return names
	}
	text := string(toUTF8(raw))
	for _, m := range definePattern.FindAllStringSubmatch(text, -1) {
		names[strings.TrimPrefix(m[1], "Skill_")] = true
	}
	for _, m := range definePackLine.FindAllStringSubmatch(text, -1) {
		names[m[1]] = true
	}
	return names
}
