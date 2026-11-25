package skilldata

// damage.go 伤害计算包（Skill_DamageCalculate.xml）与公式（Skill_Formula.xml）
// 的只读联查。
//
// 数据链（docs/combat/combat-rewrite-hardcoded-audit.md「每击公式引擎」）：
//
//	技能逐级行 DamageCalculate（数值 ID）→ Skill_DamageCalculate 行的 r 列表
//	    r = 公式ID,伤害类型,参数…        （节内多条输出为 r/r_2…）
//	→ Skill_Formula 行（ID=公式号）：Src1..12/Prop1..12 给 P1..P12 的来源，
//	    Formula 为 P<n>/R[a,b]/MIN[..]/MAX[..]/#属性 表达式。
//
// 本包只做数据与转译，不做求值：求值在 Lua 引擎（internal/script 的
// combat/formula.lua，公式转译成闭包装载），回退面在 Go 战斗链。

import (
	"fmt"
	"strconv"
	"strings"
)

// 产物文件名（相对 ini/Skill）。
const (
	fileDamageCalculate = "Skill_DamageCalculate.xml"
	fileFormula         = "Skill_Formula.xml"
)

// 公式参数来源类别（formula.ini 的 Src 列）。0/1/5/6 有客户端
// tips_func_skill.lua 的 process_formula 佐证，其余按设计注释与取值分布
// 归纳（见 docs/official/战斗接入与技能Lua宿主.md §公式参数来源）。
const (
	SrcSelfProp     = 0  // 攻击者属性（Prop=属性名）
	SrcSkillContext = 1  // 技能上下文：HurtCount 击数 / CurTargetCount 目标数 / Level
	SrcTargetProp   = 2  // 目标属性（Prop=属性名）
	SrcSelfVital    = 3  // 攻击者属性（HP/MP/MaxHP 资源位，与 0 同义）
	SrcConstant     = 4  // 常量（Prop 本身即数值）
	SrcArg          = 5  // 伤害计算行的位置实参（数值）
	SrcArgRange     = 6  // 位置实参（"a;b" 区间，求值时取区间内随机数）
	SrcSelfBuff     = 7  // 攻击者身上 buff（Prop=ConfigID）的叠加层数
	SrcTargetBuff   = 8  // 目标身上 buff 的叠加层数
	SrcDistance     = 9  // 与目标的距离
	SrcNamedVar     = 10 // 具名变量（Prop=变量名，如药师技能等级）
	SrcSelfPropAlt  = 11 // 攻击者属性（Dot 伤害族，与 0 同义）
	SrcSelfMaxHP    = 12 // 攻击者 MaxHP
	SrcTargetMaxHP  = 13 // 目标 MaxHP
	SrcTargetHP     = 14 // 目标 HP
	SrcSelfPropAlt2 = 15 // 攻击者属性（与 0 同义）
)

// 伤害类型记号 → define.lua 的 DAMAGE_TYPE_*（外功/刚/柔/阴/阳）。
var damageTypeCodes = map[string]int{
	"PhyDamage": 0, "StiDamage": 1, "JujDamage": 2, "NegDamage": 3, "MasDamage": 4,
}

// DamageEntry 伤害计算包的一条 r 行。
type DamageEntry struct {
	Formula    int      // 公式号（Skill_Formula 行 ID）
	DamageType int      // DAMAGE_TYPE_*（未知记号按外功）
	TypeName   string   // 源记号（PhyDamage…）
	Args       []string // 位置实参（r 行第 3 段起，空段保留以对位 P<n>）
}

// FormulaParam 公式第 n 个参数（P<n>）的来源绑定。
type FormulaParam struct {
	Source int
	Prop   string
}

// Formula 一条伤害公式（Skill_Formula.xml 一行）。
type Formula struct {
	ID      int
	Params  [12]FormulaParam
	Expr    string // Formula 表达式原文
	Comment string // 设计注释
}

// DamageCalculate 伤害计算包的全部 r 行（按序）；包缺席返回 nil。
func (c *Catalog) DamageCalculate(id int) []DamageEntry {
	row, ok := c.table(fileDamageCalculate).lookup(strconv.Itoa(id))
	if !ok {
		return nil
	}
	var entries []DamageEntry
	// 首条键名 r，其后 r_2/r_3…（skillgen 的节内重复键输出）。
	key := "r"
	for n := 1; n <= 32 && row.has(key); n++ {
		fields := strings.Split(row.text(key), ",")
		if len(fields) >= 2 {
			formula, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
			typeName := strings.TrimSpace(fields[1])
			entries = append(entries, DamageEntry{
				Formula: formula, DamageType: damageTypeCodes[typeName], TypeName: typeName,
				Args: fields[2:],
			})
		}
		key = "r_" + strconv.Itoa(n+1)
	}
	return entries
}

// Formula 按公式号取公式（产物缺席或无此行返回 false）。
func (c *Catalog) Formula(id int) (Formula, bool) {
	row, ok := c.table(fileFormula).lookup(strconv.Itoa(id))
	if !ok {
		return Formula{}, false
	}
	f := Formula{ID: id, Expr: row.text("Formula"), Comment: row.text("Comment")}
	for n := 1; n <= 12; n++ {
		f.Params[n-1] = FormulaParam{Source: row.int("Src"+strconv.Itoa(n), 0), Prop: row.text("Prop" + strconv.Itoa(n))}
	}
	return f, len(f.Expr) > 0
}

// Formulas 全部公式（装载顺序）。
func (c *Catalog) Formulas() []Formula {
	var out []Formula
	for _, id := range c.table(fileFormula).ids() {
		if formula, ok := c.Formula(atoiDefault(id, 0)); ok {
			out = append(out, formula)
		}
	}
	return out
}

// LuaBody 把公式表达式转译为 Lua 函数体（求值环境 PV/R/Q/MIN/MAX 由引擎
// 注入）：
//
//	P<n>     → PV[n]            参数位（引擎按 Src/Prop 绑定取值）
//	R[a,b]   → R(a,b)           区间掷值
//	MIN[..]  → MIN(a,b,…)       MAX 同
//	#Prop    → Q("Prop")        直接引用（属性/计数）
//	裸标识字  → Q("标识字")       （CurTargetCount 等少数行）
//
// 全角括号/逗号归一化为半角。子集之外的记号返回错误——调用方把该公式记为
// 不支持，对应伤害走回退面。
func (f Formula) LuaBody() (string, error) { return Transpile(f.Expr) }

// Transpile 伤害公式表达式 → Lua 函数体（规则见 Formula.LuaBody）。
func Transpile(expr string) (string, error) {
	expr = strings.NewReplacer("（", "(", "）", ")", "，", ",", "［", "[", "］", "]").Replace(expr)
	var out strings.Builder
	i, n := 0, len(expr)
	call := func(name string, at int) (string, int, error) {
		// 闭括号按配对扫描（嵌套调用 MIN[..,MAX[..]] 的内层 ] 不是结尾）。
		end := matchBracket(expr, at-1)
		if end < 0 {
			return "", 0, fmt.Errorf("%s[ 未闭合 at %d", name, at-1)
		}
		var args []string
		for _, part := range splitTopLevel(expr[at:end], ',') {
			body, err := Transpile(strings.TrimSpace(part))
			if err != nil {
				return "", 0, err
			}
			args = append(args, body)
		}
		return name + "(" + strings.Join(args, ",") + ")", end + 1, nil
	}
	// depth 圆括号深度：配平校验。不配平的表达式转译即拒（源表脏数据，
	// 如 formula.ini [1116] 的原值；该行已在构建期按审定值修正）——否则
	// 要到引擎装载期编译才失败。
	depth := 0
	for i < n {
		c := expr[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case strings.IndexByte("()+-*/.", c) >= 0 || (c >= '0' && c <= '9'):
			switch c {
			case '(':
				depth++
			case ')':
				depth--
				if depth < 0 {
					return "", fmt.Errorf("圆括号不配平（多余的 )）at %d", i)
				}
			}
			out.WriteByte(c)
			i++
		case c == '#':
			j := i + 1
			for j < n && isIdentByte(expr[j]) {
				j++
			}
			if j == i+1 {
				return "", fmt.Errorf("# 后缺属性名 at %d", i)
			}
			out.WriteString("Q(" + luaQuote(expr[i+1:j]) + ")")
			i = j
		case isIdentStart(c):
			j := i + 1
			for j < n && isIdentByte(expr[j]) {
				j++
			}
			word := expr[i:j]
			upper := strings.ToUpper(word)
			switch {
			case j < n && expr[j] == '[' && (upper == "R" || upper == "MIN" || upper == "MAX"):
				text, next, err := call(upper, j+1)
				if err != nil {
					return "", err
				}
				out.WriteString(text)
				i = next
			case (c == 'P' || c == 'p') && j > i+1 && allDigits(expr[i+1:j]):
				index, _ := strconv.Atoi(expr[i+1 : j])
				if index < 1 || index > 12 {
					return "", fmt.Errorf("参数位 P%s 越界 at %d", expr[i+1:j], i)
				}
				out.WriteString("PV[" + expr[i+1:j] + "]")
				i = j
			case c == 'R' || c == 'r' || (upper == "MIN" || upper == "MAX"):
				return "", fmt.Errorf("%s 后缺 [ at %d", word, i)
			default:
				out.WriteString("Q(" + luaQuote(word) + ")")
				i = j
			}
		default:
			return "", fmt.Errorf("不支持的字符 %q at %d", string(c), i)
		}
	}
	if depth != 0 {
		return "", fmt.Errorf("圆括号不配平（%d 个 ( 未闭合）", depth)
	}
	body := out.String()
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("空表达式")
	}
	return body, nil
}

// matchBracket 与 open 处开括号配对的闭括号下标（圆括号与方括号共用
// 深度计数；无配对返回 -1）。
func matchBracket(expr string, open int) int {
	depth := 0
	for i := open; i < len(expr); i++ {
		switch expr[i] {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitTopLevel 按顶层逗号拆分（忽略圆括号/方括号内）。
func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

func luaQuote(s string) string { return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"` }

func isIdentStart(c byte) bool { return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isIdentByte(c byte) bool  { return isIdentStart(c) || c >= '0' && c <= '9' }
func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}
