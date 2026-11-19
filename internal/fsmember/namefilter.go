package fsmember

// NameFilter 复刻 CNameFilter + GameMember::CheckName（§3.1）：
// HasInvalidCharW（非法字符门）+ Filter（违禁词表）双口径。
type NameFilter struct {
	invalidChars map[rune]bool
	words        map[string]bool
}

// NewNameFilter 建过滤器（非法字符集对齐常规命名门：控制字符与
// 分隔/空白/符号位；官方表为代码内静态，此取保守集）。
func NewNameFilter() *NameFilter {
	f := &NameFilter{
		invalidChars: make(map[rune]bool),
		words:        make(map[string]bool),
	}
	for r := rune(0); r < ' '; r++ {
		f.invalidChars[r] = true
	}
	for _, r := range " \t\r\n\x00\"'`~!@#$%^&*()+={}[]|\\:;<>,.?/" {
		f.invalidChars[r] = true
	}
	return f
}

// Load 复刻 LoadNameFilter(filter.txt)：每行一个违禁词（UTF-8/GBK
// 词面；Go 侧宿主转码后注入）。
func (f *NameFilter) Load(words []string) {
	for _, w := range words {
		if w != "" {
			f.words[w] = true
		}
	}
}

// HasInvalidChar 复刻 HasInvalidCharW：命中非法字符即真。
func (f *NameFilter) HasInvalidChar(name string) bool {
	for _, r := range name {
		if f.invalidChars[r] {
			return true
		}
	}
	return false
}

// ContainsWord 复刻 Filter：全名包含违禁词即真（子串口径）。
func (f *NameFilter) ContainsWord(name string) bool {
	if len(f.words) == 0 {
		return false
	}
	for w := range f.words {
		if contains(name, w) {
			return true
		}
	}
	return false
}

// CheckName 复刻 GameMember::CheckName(wchar*)：非法字符或违禁词
// 均拒绝。
func (f *NameFilter) CheckName(name string) bool {
	return !f.HasInvalidChar(name) && !f.ContainsWord(name)
}

func contains(s, sub string) bool {
	if len(sub) == 0 || len(sub) > len(s) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ParseFilterText 解析 filter.txt 文本（每行一词，CRLF/空行容忍）。
func ParseFilterText(data []byte) []string {
	var out []string
	for line := range eachLine(string(data)) {
		line = trimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }
