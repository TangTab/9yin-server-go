// Package combatcalc 把 formula.ini 的伤害公式求值移入 Lua：宿主装载
// scripts/module/combat/formula.lua 引擎并注入转译后的公式闭包，战斗链按
// damage_calculate 的 r 行逐击调用 M.calc_hit。
//
// 与 internal/neigong 同款错误策略（战斗热路径收窄爆炸半径）：装载期坏
// 公式跳过（记 unsupported，调用该公式的技能回退 legacy 面板×倍率路径）；
// 运行期引擎出错只禁用整引擎一次 WARN，之后全部调用回退 legacy——服务器
// 永不因脚本层缺失停摆。
package combatcalc

import (
	"fmt"
	"strings"
)

// transpileFormula 把 formula.ini 的 Formula 表达式转译为 Lua 函数体
// （不含 "return " 前缀与 " end" 外壳，由宿主包成完整函数源）。
//
// 语法（客户端 tips_func_skill.lua 的 process_formula 实证子集）：
//   - 数字字面量、+ - * / ( ) 与空白原样保留；
//   - P<n>      → PV[n]（参数位，由引擎按 Src/Prop 绑定解析）；
//   - R[a,b]    → R(a,b)（区间掷值；a/b 可为数字或 P 引用，递归转译）；
//   - #Prop     → Q("Prop")（直接属性引用，如 #CurTargetCount）；
//   - 全角 （） 归一化为半角（formula.ini [63] 实例）。
//
// 出现该子集之外的记号（未知字母词等）返回错误——调用方将该公式记为
// unsupported，对应技能走 legacy 掷骰。转译只重写原子，最终语法正确性
// 由装载期的 Lua parse/compile 把关（双保险）。
func transpileFormula(expr string) (string, error) {
	// 全角括号归一化（formula.ini [63] 实例）——多字节字符需在逐字节扫描前替换。
	expr = strings.NewReplacer("（", "(", "）", ")", "，", ",").Replace(expr)
	var out strings.Builder
	i := 0
	for i < len(expr) {
		c := expr[i]
		switch {
		case c == ' ' || c == '\t':
			out.WriteByte(c)
			i++
		case c == '(' || c == ')' || c == '+' || c == '-' || c == '*' || c == '/' ||
			(c >= '0' && c <= '9') || c == '.':
			out.WriteByte(c)
			i++
		case c == 'P' || c == 'p':
			j := i + 1
			for j < len(expr) && expr[j] >= '0' && expr[j] <= '9' {
				j++
			}
			if j == i+1 {
				return "", fmt.Errorf("P 后缺参数序号 at %d", i)
			}
			out.WriteString("PV[" + expr[i+1:j] + "]")
			i = j
		case c == 'R' || c == 'r':
			if i+1 >= len(expr) || expr[i+1] != '[' {
				return "", fmt.Errorf("R 后缺 [ at %d", i)
			}
			end := strings.IndexByte(expr[i+2:], ']')
			if end < 0 {
				return "", fmt.Errorf("R[ 未闭合 at %d", i)
			}
			inner := expr[i+2 : i+2+end]
			args := strings.SplitN(inner, ",", 2)
			if len(args) != 2 {
				return "", fmt.Errorf("R[...] 参数不足 at %d", i)
			}
			left, err := transpileFormula(strings.TrimSpace(args[0]))
			if err != nil {
				return "", err
			}
			right, err := transpileFormula(strings.TrimSpace(args[1]))
			if err != nil {
				return "", err
			}
			out.WriteString("R(" + left + "," + right + ")")
			i = i + 2 + end + 1
		case c == '#':
			j := i + 1
			for j < len(expr) && (isIdentChar(expr[j])) {
				j++
			}
			if j == i+1 {
				return "", fmt.Errorf("# 后缺属性名 at %d", i)
			}
			out.WriteString(`Q("` + expr[i+1:j] + `")`)
			i = j
		default:
			return "", fmt.Errorf("不支持的字符 %q at %d", string(c), i)
		}
	}
	body := out.String()
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("空表达式")
	}
	return body, nil
}

func isIdentChar(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// closureSource 包出完整 Lua 函数源：宿主 Parse→Compile→Run 取回闭包，
// 经 install_formulas 注入引擎。
func closureSource(body string) string {
	return "return function(PV, R, Q) return " + body + " end"
}
