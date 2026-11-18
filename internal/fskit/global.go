package fskit

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Global 六个扩展函数（官方 Global.obj，extfunc 注册给脚本层）。
// 名字去掉 ext_ 前缀即脚本函数名（官方字符串表 0x100c06ec 起）。

// SplitString 官方 ext_split_string（0x1004f4f0）：
//   - sep 取 args[1] 首字符，空串 → ','；
//   - 分隔符逐位置 NUL，连续分隔符的空 token 跳过；
//   - 尾随分隔符产生一个空串 token（p != buf 判据）；
//   - 空输入零输出。
func SplitString(str, sep string) []string {
	c := byte(',')
	if sep != "" {
		c = sep[0]
	}
	buf := []byte(str)
	var out []string
	token := -1
	p := 0
	cstrAt := func(off int) string { // 分隔符已置 NUL，取 [off, NUL) 串
		e := off
		for e < len(buf) && buf[e] != 0 {
			e++
		}
		return string(buf[off:e])
	}
	for ; p < len(buf) && buf[p] != 0; p++ {
		if buf[p] == c {
			buf[p] = 0
			if token >= 0 {
				out = append(out, cstrAt(token))
				token = -1
			}
		} else if token < 0 {
			token = p
		}
	}
	if token >= 0 {
		out = append(out, cstrAt(token))
	} else if p != 0 {
		// 尾随分隔符：输出末尾空串（对齐官方 p != buf 分支）
		out = append(out, "")
	}
	return out
}

// SplitFilePath 官方 ext_split_file_path（0x1004f880）：GetFullPathNameA 语义：
// 归一化绝对路径后按最后一个分隔符切 [目录(含尾分隔符), 文件名]；
// 无文件名部分（路径以分隔符结尾）→ [全路径]；失败 → 空。
func SplitFilePath(path string) []string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil
	}
	// 官方 GetFullPathNameA 保留尾分隔符（filepath.Clean 会吞，须补回）
	trailing := strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\")
	abs = filepath.Clean(abs)
	if trailing && abs[len(abs)-1] != '\\' && abs[len(abs)-1] != '/' {
		abs += "\\"
	}
	if abs[len(abs)-1] == '\\' || abs[len(abs)-1] == '/' {
		return []string{abs}
	}
	i := strings.LastIndexAny(abs, "\\/")
	if i < 0 {
		return []string{abs}
	}
	return []string{abs[:i+1], abs[i+1:]}
}

// SplitFileName 官方 ext_split_file_name（0x1004fb30）：从尾向前找 '.'：
// 命中 → [base, ext]；遇 '\\' 或 '/' 先到 → [整名, ""]。".hidden" → ["", "hidden"]。
func SplitFileName(name string) []string {
	buf := []byte(name)
	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i] == '.' {
			return []string{string(buf[:i]), string(buf[i+1:])}
		}
		if buf[i] == '\\' || buf[i] == '/' {
			break
		}
	}
	return []string{name, ""}
}

// GetFileTime 官方 ext_get_file_time（0x1004fd90）：最后写时间，**UTC**
// （GetFileTime + FileTimeToSystemTime，无 FileTimeToLocalFileTime），
// 格式 "%04d-%02d-%02d %02d:%02d:%02d"；失败 → ""。
func GetFileTime(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	t := fi.ModTime().UTC()
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second())
}

// GetFullPath 官方 ext_get_full_path（0x10050080）：GetFullPathNameA → 全路径；
// 失败 → ""。
func GetFullPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return filepath.Clean(abs)
}

// GenUniqueName 官方 ext_gen_unique_name（0x10050200）：CoCreateGuid →
// "%08X%04X%04X" + 8×"%02X"（Data1/2/3 + Data4[0..7] 正序）= 32 位大写 hex
// 无连字符。Go 侧以 crypto/rand 构造同构 v4 GUID（CoCreateGuid 即 v4）。
func GenUniqueName() string {
	var g [16]byte
	if _, err := rand.Read(g[:]); err != nil {
		return ""
	}
	g[6] = (g[6] & 0x0f) | 0x40 // version 4
	g[8] = (g[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%08X%04X%04X%02X%02X%02X%02X%02X%02X%02X%02X",
		uint32(g[0])|uint32(g[1])<<8|uint32(g[2])<<16|uint32(g[3])<<24,
		uint16(g[4])|uint16(g[5])<<8,
		uint16(g[6])|uint16(g[7])<<8,
		g[8], g[9], g[10], g[11], g[12], g[13], g[14], g[15])
}

// FileTime 对齐官方 port_file_time_t{year,month,day,hour,minute,second}（UTC）。
type FileTime struct {
	Year, Month, Day, Hour, Minute, Second int
}

// GetFileTimeStruct 是 GetFileTime 的结构化版（官方 Port_GetFileTime 输出）。
func GetFileTimeStruct(path string) (FileTime, bool) {
	fi, err := os.Stat(path)
	if err != nil {
		return FileTime{}, false
	}
	t := fi.ModTime().UTC()
	return FileTime{t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second()}, true
}

var _ = time.Now // 保留 time 引用（结构化时间转换用）
