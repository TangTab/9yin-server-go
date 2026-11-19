package fssololog

import (
	"strconv"
	"strings"
)

// ParseLine 复刻官方 echohook.cpp 静态 parse_string + parse_token
// （§6.3）：
//
//   - parse_string：空格/Tab 且不在引号内切 token；'"' 翻转引号态且
//     引号字符本身保留进 token；行尾冲刷。
//   - parse_token：按序试型——整型（atoi）→ 浮点（atof）→ 字符串
//     （PERSISTID/宽串归并为字符串，见包文档取舍）。
//
// 例："setp solo_log AutoExit false" → ["setp","solo_log","AutoExit","false"]
// （"false" 试型失败落字符串；布尔语义由 setp 解释器处理）。
func ParseLine(line string) []any {
	tokens := parseString(line)
	args := make([]any, 0, len(tokens))
	for _, tok := range tokens {
		args = append(args, parseToken(tok))
	}
	return args
}

// parseString 官方 parse_string：引号保留、引号内不切分。
func parseString(line string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	for _, c := range line {
		if c == '"' {
			inQuote = !inQuote
			cur.WriteRune(c) // 引号字符本身保留进 token
			continue
		}
		if !inQuote && (c == ' ' || c == '\t') {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(c)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String()) // 行尾冲刷
	}
	return tokens
}

// parseToken 官方 parse_token 按序试型：int → float64 → string。
func parseToken(tok string) any {
	if n, err := strconv.Atoi(tok); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(tok, 64); err == nil {
		return f
	}
	return tok
}
