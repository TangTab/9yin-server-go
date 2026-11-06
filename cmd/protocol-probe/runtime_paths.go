package main

import (
	"os"
	"path/filepath"
	"strings"
)

const serverRootEnvironment = "NINEYIN_SERVER_ROOT"

var (
	runtimeProjectRoot       = locateRuntimeProjectRoot()
	defaultModernShareRoot   = runtimeProjectPath("resources", "modern", "share")
	defaultModernActionRoot  = runtimeProjectPath("resources", "modern", "ini", "action")
	defaultModernTextRoot    = runtimeProjectPath("resources", "modern", "text")
	defaultLegacyNPCFuncPath = runtimeProjectPath("resources", "legacy", "Npc_Func.xml")
)

func runtimeProjectPath(parts ...string) string {
	all := append([]string{runtimeProjectRoot}, parts...)
	return filepath.Join(all...)
}

func locateRuntimeProjectRoot() string {
	if configured := strings.TrimSpace(os.Getenv(serverRootEnvironment)); configured != "" {
		return filepath.Clean(configured)
	}

	var starts []string
	if executable, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(executable))
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		starts = append(starts, workingDirectory)
	}
	for _, start := range starts {
		if root, ok := findRuntimeProjectRoot(start); ok {
			return root
		}
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		return filepath.Clean(workingDirectory)
	}
	return "."
}

func findRuntimeProjectRoot(start string) (string, bool) {
	for current := filepath.Clean(start); ; current = filepath.Dir(current) {
		if isRuntimeProjectRoot(current) {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
	}
}

func isRuntimeProjectRoot(path string) bool {
	if info, err := os.Stat(filepath.Join(path, "resources", "modern", "share")); err == nil && info.IsDir() {
		return true
	}
	info, err := os.Stat(filepath.Join(path, "go.mod"))
	return err == nil && !info.IsDir()
}
