package snaileditornet

import (
	"fmt"
	"os/exec"
	"strings"
)

// SVN 配置面（官方 ClientSession/FileCon 的 TortoiseProc 集成）。

// DefaultTortoiseProcProcPath 官方注册表值 HKLM\SOFTWARE\TortoiseSVN\ProcPath
// 的典型落点；StartSnailEditor 读取失败时回落此路径。
const DefaultTortoiseProcProcPath = `TortoiseProc.exe`

// TortoiseProcPath 读 HKLM\SOFTWARE\TortoiseSVN\ProcPath（官方 ReadRegValue 路径）。
// 读不到返回默认值。
func TortoiseProcPath() string {
	if v, ok := readRegistryString(`SOFTWARE\TortoiseSVN`, "ProcPath"); ok {
		return v
	}
	return DefaultTortoiseProcProcPath
}

// BuildSvnUpdateArgs 构造官方 TortoiseProc 命令行参数（定案串拼接）：
//
//	/command:update /path:"<path>" /notempfile /closeonend:1
func BuildSvnUpdateArgs(path string) []string {
	return []string{
		"/command:update",
		fmt.Sprintf(`/path:"%s"`, path),
		"/notempfile",
		"/closeonend:1",
	}
}

// RunSvnUpdate 起一次 TortoiseProc update（官方 StartAppAndWait 等价：
// 起进程并等待退出）。返回错误仅覆盖"无法启动"场景。
func RunSvnUpdate(procPath, localRoot string) error {
	cmd := exec.Command(procPath, BuildSvnUpdateArgs(localRoot)...)
	return cmd.Run()
}

// NormalizeSvnPath 官方 FMSG_SERVER_SVN_UPDATE 处理：'/' → '\\'。
func NormalizeSvnPath(p string) string { return strings.ReplaceAll(p, "/", `\`) }

// SplitPaths 官方路径列表分隔（观察为空白/换行分隔 + 逐条规范化）。
func SplitPaths(payload string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(payload, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ' ' || r == '\t'
	}) {
		out = append(out, NormalizeSvnPath(f))
	}
	return out
}
