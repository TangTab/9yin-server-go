package res

import "strings"

// Normalize 规范化资源名：/ → \，去首尾分隔符。
func Normalize(name string) string {
	n := strings.ReplaceAll(name, "/", `\`)
	n = strings.Trim(n, `\`)
	return n
}

// NormalizeLower 规范化 + 小写（MPQ 名字不区分大小写）。
func NormalizeLower(name string) string {
	return strings.ToLower(Normalize(name))
}
