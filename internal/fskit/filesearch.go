package fskit

import (
	"path/filepath"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

const FileSearchTypeName = "FileSearch"

// FileSearch 复刻官方 fskit/filesearch.cpp 实体。
type FileSearch struct {
	path     string
	fileName string
	dirList  []string
	fileList []string
}

func NewFileSearch() *FileSearch { return &FileSearch{} }

// Path / FileName 属性 getter（官方 0x10050370/0x100503d0）。
func (fs *FileSearch) GetPath() string     { return fs.path }
func (fs *FileSearch) GetFileName() string { return fs.fileName }

// Port_FindFirst 通配方言（官方 0x1004e630）：
// filename 空 → "<dir>*.*"；含 '.' → "<dir><filename>"；否则 "<dir>*.<filename>"。
// Windows FindFirstFileA 语义：'*' 跨 '.' 任意匹配，'*.*' 匹配一切（含无扩展名），
// 匹配大小写不敏感。Go 侧按 ReadDir + 自实现匹配等价复刻（不依赖 8.3 短名兼容）。
func buildFindPattern(dir, fileName string) string {
	if fileName == "" {
		return dir + "*.*"
	}
	if strings.Contains(fileName, ".") {
		return dir + fileName
	}
	return dir + "*." + fileName
}

// matchWinPattern 复刻 FindFirstFileA 的 '*'/问号匹配（大小写不敏感；*.* 全过）。
func matchWinPattern(pattern, name string) bool {
	if pattern == "*.*" {
		return true
	}
	return matchWildFold(pattern, name)
}

func matchWildFold(p, n string) bool {
	// 经典双指针回溯；折叠用官方 tolower 表（ASCII 折叠、高位字节原样）
	ip, in := 0, 0
	star, mark := -1, 0
	for in < len(n) {
		if ip < len(p) && (p[ip] == '?' || tolowerTable[p[ip]] == tolowerTable[n[in]]) {
			ip++
			in++
		} else if ip < len(p) && p[ip] == '*' {
			star = ip
			ip++
			mark = in
		} else if star >= 0 {
			ip = star + 1
			mark++
			in = mark
		} else {
			return false
		}
	}
	for ip < len(p) && p[ip] == '*' {
		ip++
	}
	return ip == len(p)
}

// SearchDir 官方 0x1004e430：枚举 path 下目录（排除 "." ".."），存 dirList，
// 记 m_path，true。找不到（打不开/不存在）→ false（列表已清空）。
func (fs *FileSearch) SearchDir(path string) bool {
	fs.dirList = fs.dirList[:0]
	entries, ok := findEntries(path, "")
	if !ok {
		return false
	}
	for _, name := range entries {
		if name == "." || name == ".." {
			continue
		}
		if isDir(filepath.Join(path, name)) {
			fs.dirList = append(fs.dirList, name)
		}
	}
	fs.path = path
	return true
}

// SearchFile 官方 0x1004e920：同构，取文件（非目录即文件），存 fileList，
// 额外记 m_fileName。fileName 为 "" 时匹配全部（*.*）。
func (fs *FileSearch) SearchFile(path, fileName string) bool {
	fs.fileList = fs.fileList[:0]
	entries, ok := findEntries(path, fileName)
	if !ok {
		return false
	}
	for _, name := range entries {
		if name == "." || name == ".." {
			continue
		}
		if !isDir(filepath.Join(path, name)) {
			fs.fileList = append(fs.fileList, name)
		}
	}
	fs.path = path
	fs.fileName = fileName
	return true
}

// findEntries 官方 Port_FindFirst/FindNext：pattern 一次成型后枚举。
// 目录打不开 → (nil, false)（对齐 FindFirstFileA INVALID_HANDLE_VALUE）。
func findEntries(dir, fileName string) ([]string, bool) {
	pattern := buildFindPattern(dir, fileName)
	// pattern 的通配部分 = 去掉 dir 前缀后的尾段；dir 须以分隔符结尾才合法
	base := dir
	if base != "" && !strings.HasSuffix(base, "\\") && !strings.HasSuffix(base, "/") {
		// 官方 dir 由调用方保证尾分隔符；防御性截断 pattern 到最后分隔符
		if i := strings.LastIndexAny(pattern, "\\/"); i >= 0 {
			base = pattern[:i+1]
		}
	}
	wild := pattern[len(base):]
	ents, err := res.ReadDir(base)
	if err != nil {
		return nil, false
	}
	var out []string
	for _, e := range ents {
		if matchWinPattern(wild, e.Name()) {
			out = append(out, e.Name())
		}
	}
	return out, true
}

func isDir(p string) bool {
	fi, err := res.Stat(p)
	return err == nil && fi.IsDir()
}

func (fs *FileSearch) GetDirCount() int  { return len(fs.dirList) }
func (fs *FileSearch) GetFileCount() int { return len(fs.fileList) }

// GetDirList / GetFileList 官方 0x1004ebd0/0x1004ecf0：
// result.Clear() + Assign(容器, 0, count)——纯名字，无路径前缀。
func (fs *FileSearch) GetDirList(result *VarList) {
	result.Clear()
	for _, n := range fs.dirList {
		result.AddString(n)
	}
}

func (fs *FileSearch) GetFileList(result *VarList) {
	result.Clear()
	for _, n := range fs.fileList {
		result.AddString(n)
	}
}
