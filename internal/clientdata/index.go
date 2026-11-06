// Package clientdata 对从现代客户端包中解包出的文件建立索引。
//
// 它有意模拟 FxPackage 的虚拟名称规则，而不是套用
// filepath.Clean：ASCII 字母不区分大小写，但斜杠与
// 反斜杠仍是不同的字节。较早挂载的条目会遮蔽
// 较晚挂载中的同名条目。
package clientdata

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrNotFound = errors.New("clientdata: virtual file not found")

// Mount 描述一个已解包的资源目录树。Directory 必须是包虚拟根
// 对应的目录（对常见的解包器输出而言，就是
// <name>.package.files\res）。VirtualPrefix 会原样前置，不改变
// 其分隔符风格。挂载顺序即包加载顺序，因此也是
// 遮蔽优先级。
type Mount struct {
	Name          string
	Directory     string
	VirtualPrefix string
}

type Config struct {
	Mounts          []Mount
	FailOnRootError bool
}

// ExtractedPackageMount 适配当前解包器输出的目录。
// 解包器把包内容存放在物理上的 "res" 目录之下，而 "res" 本身
// 并不属于包的虚拟文件名。
func ExtractedPackageMount(name, extractionDirectory string) Mount {
	return Mount{Name: name, Directory: filepath.Join(extractionDirectory, "res")}
}

type DiagnosticKind string

const (
	DiagnosticRoot       DiagnosticKind = "root"
	DiagnosticWalk       DiagnosticKind = "walk"
	DiagnosticUnreadable DiagnosticKind = "unreadable"
	DiagnosticShadowed   DiagnosticKind = "shadowed"
)

// Diagnostic 表示一条非致命的索引观察记录，除非 FailOnRootError 将
// root 和 walk 失败升级为致命错误。这里保留错误文本而非 error
// 值，使 Index 保持不可变，且便于在工具中序列化。
type Diagnostic struct {
	Kind         DiagnosticKind
	Mount        string
	VirtualPath  string
	PhysicalPath string
	Error        string
}

type Source struct {
	Mount        string
	MountIndex   int
	PhysicalPath string
	RelativePath string
	Size         int64
	Readable     bool
	ReadError    string
}

// Entry 包含同一个引擎等价虚拟名称的全部来源。Sources 按
// 挂载优先级排序；Sources[0] 即生效来源。
type Entry struct {
	VirtualPath string
	Sources     []Source
}

func (e Entry) Winner() Source { return e.Sources[0] }
func (e Entry) Shadowed() bool { return len(e.Sources) > 1 }

type LookupResult struct {
	Entry               *Entry
	CaseFolded          bool
	SeparatorCandidates []string
}

// Found 报告是否符合严格的 FxPackage 风格匹配。SeparatorCandidates
// 仅用于诊断，绝不会令 Found 变为 true。
func (r LookupResult) Found() bool { return r.Entry != nil }

type Index struct {
	entries      map[string]*Entry
	ordered      []*Entry
	bySeparators map[string][]*Entry
	diagnostics  []Diagnostic
}

// Build 递归扫描已配置的挂载，不修改任何客户端文件。默认情况下，
// 缺失的根目录会记录为诊断信息，这样部分解包的结果仍可检查。
// 启动校验时可设置 FailOnRootError。
func Build(config Config) (*Index, error) {
	return build(config, probeReadable)
}

type readabilityProbe func(string) error

func build(config Config, probe readabilityProbe) (*Index, error) {
	if len(config.Mounts) == 0 {
		return nil, errors.New("clientdata: no mounts configured")
	}
	index := &Index{
		entries:      make(map[string]*Entry),
		bySeparators: make(map[string][]*Entry),
	}
	seenMounts := make(map[string]struct{}, len(config.Mounts))
	for mountIndex, mount := range config.Mounts {
		mount.Name = strings.TrimSpace(mount.Name)
		if mount.Name == "" {
			return nil, fmt.Errorf("clientdata: mount %d has no name", mountIndex)
		}
		if _, exists := seenMounts[mount.Name]; exists {
			return nil, fmt.Errorf("clientdata: duplicate mount name %q", mount.Name)
		}
		seenMounts[mount.Name] = struct{}{}
		if mount.Directory == "" {
			return nil, fmt.Errorf("clientdata: mount %q has no directory", mount.Name)
		}
		prefix, err := validatePrefix(mount.VirtualPrefix)
		if err != nil {
			return nil, fmt.Errorf("clientdata: mount %q: %w", mount.Name, err)
		}
		root, err := filepath.Abs(mount.Directory)
		if err != nil {
			return nil, fmt.Errorf("clientdata: mount %q: %w", mount.Name, err)
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			if err == nil {
				err = errors.New("not a directory")
			}
			diagnostic := Diagnostic{Kind: DiagnosticRoot, Mount: mount.Name, PhysicalPath: root, Error: err.Error()}
			index.diagnostics = append(index.diagnostics, diagnostic)
			if config.FailOnRootError {
				return nil, fmt.Errorf("clientdata: mount %q root %q: %w", mount.Name, root, err)
			}
			continue
		}

		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				index.diagnostics = append(index.diagnostics, Diagnostic{Kind: DiagnosticWalk, Mount: mount.Name, PhysicalPath: path, Error: walkErr.Error()})
				if config.FailOnRootError {
					return walkErr
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			info, infoErr := entry.Info()
			if infoErr != nil {
				index.diagnostics = append(index.diagnostics, Diagnostic{Kind: DiagnosticWalk, Mount: mount.Name, PhysicalPath: path, Error: infoErr.Error()})
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			relative, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			// 解包出的包名以 Windows 分隔符记录。后续不要对查找输入做
			// 归一化：斜杠不相等是有意为之。
			relative = strings.ReplaceAll(relative, "/", `\`)
			virtualPath := relative
			if prefix != "" {
				virtualPath = prefix + `\` + relative
			}
			source := Source{
				Mount:        mount.Name,
				MountIndex:   mountIndex,
				PhysicalPath: path,
				RelativePath: relative,
				Size:         info.Size(),
				Readable:     true,
			}
			if readErr := probe(path); readErr != nil {
				source.Readable = false
				source.ReadError = readErr.Error()
				index.diagnostics = append(index.diagnostics, Diagnostic{Kind: DiagnosticUnreadable, Mount: mount.Name, VirtualPath: virtualPath, PhysicalPath: path, Error: readErr.Error()})
			}
			key := foldASCII(virtualPath)
			if existing := index.entries[key]; existing != nil {
				existing.Sources = append(existing.Sources, source)
				index.diagnostics = append(index.diagnostics, Diagnostic{
					Kind: DiagnosticShadowed, Mount: mount.Name, VirtualPath: virtualPath, PhysicalPath: path,
					Error: fmt.Sprintf("shadowed by mount %q at %s", existing.Sources[0].Mount, existing.Sources[0].PhysicalPath),
				})
				return nil
			}
			entryValue := &Entry{VirtualPath: virtualPath, Sources: []Source{source}}
			index.entries[key] = entryValue
			index.ordered = append(index.ordered, entryValue)
			index.bySeparators[separatorDiagnosticKey(virtualPath)] = append(index.bySeparators[separatorDiagnosticKey(virtualPath)], entryValue)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("clientdata: scan mount %q: %w", mount.Name, err)
		}
	}
	sort.Slice(index.ordered, func(i, j int) bool {
		return foldASCII(index.ordered[i].VirtualPath) < foldASCII(index.ordered[j].VirtualPath)
	})
	return index, nil
}

func validatePrefix(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, `/`) || strings.HasPrefix(value, `\`) || strings.HasSuffix(value, `/`) || strings.HasSuffix(value, `\`) {
		return "", errors.New("virtual prefix must not start or end with a separator")
	}
	if strings.Contains(value, `/`) {
		return "", errors.New("virtual prefix must use backslashes")
	}
	return value, nil
}

func probeReadable(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	var one [1]byte
	_, readErr := file.Read(one[:])
	if errors.Is(readErr, io.EOF) {
		readErr = nil // 空的常规文件视为可读。
	}
	closeErr := file.Close()
	if readErr != nil {
		return readErr
	}
	return closeErr
}

func (index *Index) Lookup(virtualPath string) LookupResult {
	if index == nil || virtualPath == "" {
		return LookupResult{}
	}
	if entry := index.entries[foldASCII(virtualPath)]; entry != nil {
		return LookupResult{Entry: cloneEntry(entry), CaseFolded: virtualPath != entry.VirtualPath}
	}
	candidates := index.bySeparators[separatorDiagnosticKey(virtualPath)]
	result := LookupResult{SeparatorCandidates: make([]string, 0, len(candidates))}
	for _, candidate := range candidates {
		result.SeparatorCandidates = append(result.SeparatorCandidates, candidate.VirtualPath)
	}
	sort.Strings(result.SeparatorCandidates)
	return result
}

func (index *Index) Entry(virtualPath string) (Entry, bool) {
	result := index.Lookup(virtualPath)
	if result.Entry == nil {
		return Entry{}, false
	}
	return *result.Entry, true
}

func (index *Index) Entries() []Entry {
	if index == nil {
		return nil
	}
	entries := make([]Entry, len(index.ordered))
	for i, entry := range index.ordered {
		entries[i] = *cloneEntry(entry)
	}
	return entries
}

func (index *Index) Diagnostics() []Diagnostic {
	if index == nil {
		return nil
	}
	return append([]Diagnostic(nil), index.diagnostics...)
}

func (index *Index) ReadFile(virtualPath string) ([]byte, error) {
	entry, ok := index.Entry(virtualPath)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, virtualPath)
	}
	source := entry.Winner()
	if !source.Readable {
		return nil, fmt.Errorf("clientdata: %s from %s is unreadable: %s", virtualPath, source.Mount, source.ReadError)
	}
	data, err := os.ReadFile(source.PhysicalPath)
	if err != nil {
		return nil, fmt.Errorf("clientdata: read %s from %s: %w", virtualPath, source.Mount, err)
	}
	return data, nil
}

func cloneEntry(entry *Entry) *Entry {
	copyValue := *entry
	copyValue.Sources = append([]Source(nil), entry.Sources...)
	return &copyValue
}

func foldASCII(value string) string {
	buffer := []byte(value)
	for i, b := range buffer {
		if b >= 'A' && b <= 'Z' {
			buffer[i] = b + ('a' - 'A')
		}
	}
	return string(buffer)
}

func separatorDiagnosticKey(value string) string {
	buffer := []byte(foldASCII(value))
	for i, b := range buffer {
		if b == '/' {
			buffer[i] = '\\'
		}
	}
	return string(buffer)
}
