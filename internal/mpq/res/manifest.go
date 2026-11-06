// Package res 实现三层资源读取：本地散文件 > res-patch.rep > res-base.rep，
// 第一个命中生效；清单墓碑（deleted）三层皆视为不存在。
package res

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 缺省布局的文件与目录名。
const (
	DefaultBaseFile  = "res-base.rep"
	DefaultPatchFile = "res-patch.rep"
	DefaultLocalDir  = "resources"
	manifestFileName = "res-manifest.json"
)

// Manifest 描述外置于 MPQ 的三层资源布局 res-manifest.json：
//
//	{
//	  "base":  { "file": "res-base.rep",  "version": "0001" },
//	  "patch": { "file": "res-patch.rep", "version": "0003" },   // 可缺省：无补丁层
//	  "local": "resources",                                       // 本地散文件根
//	  "deleted": ["resources\\ini\\old.ini"]                      // 墓碑
//	}
//
// 版本号用于启动校验与热重载判断；deleted 表达 patch 链无法表达的删除。
type Manifest struct {
	BaseFile     string
	BaseVersion  string
	PatchFile    string // "" = 无补丁层
	PatchVersion string
	LocalDir     string          // 本地散文件根（相对清单目录）
	Deleted      map[string]bool // 墓碑集合（规范化小写）
}

type manifestJSON struct {
	Base *struct {
		File    string `json:"file"`
		Version string `json:"version"`
	} `json:"base"`
	Patch *struct {
		File    string `json:"file"`
		Version string `json:"version"`
	} `json:"patch"`
	Local   *string  `json:"local"`
	Deleted []string `json:"deleted"`
}

// DefaultManifest 返回无清单时的缺省布局：只有 base + 本地层。
func DefaultManifest() *Manifest {
	return &Manifest{
		BaseFile: DefaultBaseFile,
		LocalDir: DefaultLocalDir,
		Deleted:  map[string]bool{},
	}
}

// LoadManifest 读取并校验清单文件。
func LoadManifest(file string) (*Manifest, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("res: 清单不可读：%s（%v）", file, err)
	}
	var m manifestJSON
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("res: 清单不可读：%s（%v）", file, err)
	}
	if m.Base == nil || m.Base.File == "" {
		return nil, fmt.Errorf("res: 清单缺少 base 段：%s", file)
	}

	out := &Manifest{
		BaseFile:    m.Base.File,
		BaseVersion: m.Base.Version,
		LocalDir:    DefaultLocalDir,
		Deleted:     map[string]bool{},
	}
	if m.Patch != nil {
		out.PatchFile = m.Patch.File
		out.PatchVersion = m.Patch.Version
	}
	if m.Local != nil && *m.Local != "" {
		out.LocalDir = *m.Local
	}
	// 统一小写：MPQ 名字不区分大小写，查找与移除都以小写为准。
	for _, d := range m.Deleted {
		out.Deleted[NormalizeLower(d)] = true
	}
	return out, nil
}

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

// localDirName 将本地根目录名规范化为干净的相对路径（去 ./ 与空段，/ 分隔）。
func localDirName(localDir string) string {
	var parts []string
	for _, p := range strings.FieldsFunc(localDir, func(r rune) bool { return r == '/' || r == '\\' }) {
		if p != "." {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "/")
}

// ToJSON 序列化为清单 JSON 文本（工具侧生成用）。
func (m *Manifest) ToJSON() ([]byte, error) {
	root := map[string]any{
		"base": map[string]any{
			"file":    m.BaseFile,
			"version": m.BaseVersion,
		},
	}
	if m.PatchFile != "" {
		root["patch"] = map[string]any{
			"file":    m.PatchFile,
			"version": m.PatchVersion,
		}
	}
	root["local"] = m.LocalDir
	if len(m.Deleted) > 0 {
		deleted := make([]string, 0, len(m.Deleted))
		for d := range m.Deleted {
			deleted = append(deleted, d)
		}
		sort.Strings(deleted)
		root["deleted"] = deleted
	}
	return json.MarshalIndent(root, "", "  ")
}

// manifestPath 返回资源根下的清单文件路径。
func manifestPath(dir string) string {
	return filepath.Join(dir, manifestFileName)
}
