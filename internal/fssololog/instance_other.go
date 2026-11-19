//go:build !windows

package fssololog

import (
	"os"
	"path/filepath"
)

// acquireInstanceMutex 非 Windows 回退：临时目录排它锁文件
// （O_CREATE|O_EXCL）。跨进程互斥成立；进程崩溃残留的锁文件会
// 拒绝后续实例（Windows 命名互斥体无此问题）——官方部署面为
// Windows，此路径仅开发兜底。
func acquireInstanceMutex(name string) (func(), bool) {
	path := filepath.Join(os.TempDir(), name+".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, false
	}
	f.Close()
	return func() { os.Remove(path) }, true
}
