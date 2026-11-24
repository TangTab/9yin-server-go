// Package genutil 是 datagen 各构建器共用的落盘策略。
package genutil

import (
	"bytes"
	"os"
	"path/filepath"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// 覆盖写被别的进程短暂占住时的重试：次数与基础间隔（逐次线性加长）。
// 声明为变量仅为测试可缩短。
var (
	writeRetries    = 5
	writeRetryDelay = 150 * time.Millisecond
)

// writeFile 实际写盘的函数；声明为变量仅为测试可替换。
var writeFile = os.WriteFile

// WriteFile 写出生成物（建目录）。行尾跟随数据根里同名现有文件的样式：
// 现有文件是 CRLF，就把生成内容的 LF 行尾写成 CRLF；否则原样写出。
//
// 数据根里的旧产物行尾并不统一——经 git 检出的是 CRLF，本地生成后直接
// 并入层的是 LF。内容没变的重建结果要与旧版逐字节相同，并入新层时才会
// 被 store pack 去重，而不是仅仅因为行尾不同就整份重存。没有同名现有
// 文件时按生成器原样（LF）写出。
//
// 覆盖写的是刚生成不久的文件，可能正被杀毒 / 搜索索引 / 编辑器占着
// （Windows 报共享冲突或「user-mapped section open」）：这类占用稍等即
// 释放，自动重试；其它错误原样返回。
func WriteFile(path string, data []byte) error {
	if existingUsesCRLF(path) {
		data = toCRLF(data)
	}
	return writeRetrying(path, data)
}

// WriteFileVerbatim 原样写出（建目录，不改行尾）：用于字节透传的产物
// （巡逻路线等原样搬运的 ini）。与源逐字节相同，并入资源层时才能与既有
// 层去重。占用重试同 WriteFile。
func WriteFileVerbatim(path string, data []byte) error {
	return writeRetrying(path, data)
}

func writeRetrying(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var err error
	for attempt := 0; ; attempt++ {
		if err = writeFile(path, data, 0o644); err == nil || attempt >= writeRetries || !isTransientWriteError(err) {
			return err
		}
		time.Sleep(writeRetryDelay * time.Duration(attempt+1))
	}
}

// existingUsesCRLF 报告数据根里同名文件的首个换行是否为 CRLF；文件不存在
// 或没有换行即 false。经资源仓读取，层内文件与散文件一视同仁。
func existingUsesCRLF(path string) bool {
	data, err := res.ReadFileShared(path)
	if err != nil {
		return false
	}
	i := bytes.IndexByte(data, '\n')
	return i > 0 && data[i-1] == '\r'
}

// toCRLF 把不被 CR 前导的 LF 换成 CRLF；已是 CRLF 的不动，无需转换时
// 返回原切片。
func toCRLF(data []byte) []byte {
	if !bytes.Contains(data, []byte("\n")) {
		return data
	}
	out := make([]byte, 0, len(data)+bytes.Count(data, []byte("\n")))
	for i, b := range data {
		if b == '\n' && (i == 0 || data[i-1] != '\r') {
			out = append(out, '\r')
		}
		out = append(out, b)
	}
	return out
}
