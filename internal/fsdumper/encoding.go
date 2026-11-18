package fsdumper

import (
	"bytes"
	"errors"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// gbkDecode 官方配置/XML/头文件均为 GBK（gb2312 声明）；
// 用 GB18030 超集解码（对 GBK 内容无损）。
func gbkDecode(b []byte) (string, error) {
	out, err := simplifiedchinese.GB18030.NewDecoder().Bytes(b)
	if err != nil {
		return "", err
	}
	if bytes.ContainsRune(out, 0xFFFD) {
		return "", errors.New("fsdumper: GBK 解码出现替换符")
	}
	return string(out), nil
}
