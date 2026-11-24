//go:build !windows

package genutil

import (
	"errors"
	"syscall"
)

// isTransientWriteError 报告写文件错误是否属于短暂占用、值得重试。
// 类 Unix 平台上对应的情形是 ETXTBSY（文件正被执行映像占用）。
func isTransientWriteError(err error) bool {
	return errors.Is(err, syscall.ETXTBSY)
}
