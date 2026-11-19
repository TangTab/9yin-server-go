//go:build !windows

package snaileditornet

// readRegistryString 非 Windows 无注册表。
func readRegistryString(path, name string) (string, bool) { return "", false }
