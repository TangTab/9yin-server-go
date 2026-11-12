// Package gmapi · etag.go —— 目录/会话端点 304 协商的共享工具
// （docs/gm-console-axui-redesign-design.md §5.1 S3/S4）。游戏服 handler
// 与独立面板代理两端都要做 If-None-Match 匹配，口径必须完全一致，故放契约层。
package gmapi

import (
	"encoding/json"
	"hash/fnv"
	"net/http"
	"strings"
	"sync"
)

// ETagValue 组弱校验值：`W/"<tag>"`。
func ETagValue(tag string) string { return `W/"` + tag + `"` }

// ETagMatches 判断请求的 If-None-Match 是否命中 etag。按 RFC 9110 弱比较：
// 两侧都剥离 W/ 前缀与空白后比对（本系全部端点只用弱校验）；逗号列表
// 逐项匹配；`*` 匹配任意。header 为空返回 false。
func ETagMatches(headerValue, etag string) bool {
	if headerValue == "" {
		return false
	}
	target := strings.TrimPrefix(etag, "W/")
	for _, raw := range strings.Split(headerValue, ",") {
		item := strings.TrimSpace(raw)
		if item == "*" {
			return true
		}
		item = strings.TrimPrefix(item, "W/")
		if item == target {
			return true
		}
	}
	return false
}

// SetETag 在响应上写出弱校验 ETag 头（须在 WriteHeader 之前调用）。
func SetETag(writer http.ResponseWriter, tag string) {
	writer.Header().Set("ETag", ETagValue(tag))
}

// WriteNotModified 写 304 空体（命中 If-None-Match 时的唯一出口）。
func WriteNotModified(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusNotModified)
}

// HashTag 把若干字符串拼起来算 FNV-1a 64 位哈希（hex）。用于“内容确定 →
// 标签确定”的目录端点：静态目录装载后哈希一次，请求期零成本。
func HashTag(parts ...string) string {
	hasher := fnv.New64a()
	for _, part := range parts {
		_, _ = hasher.Write([]byte(part))
		_, _ = hasher.Write([]byte{0}) // 分隔符防拼接歧义
	}
	return b62Hex(hasher.Sum64())
}

// b62Hex 用固定宽度 hex（16 位）表示，避免正负号。
func b62Hex(value uint64) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		out[i] = digits[value&0xf]
		value >>= 4
	}
	return string(out)
}

// weatherTagOnce 天气目录是契约层静态闭名单，进程内不变 → 标签算一次。
var weatherTagOnce struct {
	sync.Once
	value string
}

// WeatherTag 天气目录的内容标签（WeatherCatalog 的哈希）。两端（内嵌面板
// 与独立面板）共用同一份目录数据，标签一致。
func WeatherTag() string {
	weatherTagOnce.Do(func() {
		weatherTagOnce.value = HashTag("weather", marshalHashInput(WeatherCatalog()))
	})
	return weatherTagOnce.value
}

// marshalHashInput 把任意值 JSON 化做哈希输入（编码失败回退空串，
// 静态目录类型均为可编码结构体，实际不会走到）。
func marshalHashInput(value any) string {
	return HashJSON(value)
}

// HashJSON 把任意值 JSON 化为字符串（供调用方拼进 HashTag）。
// 编码失败回退空串——静态目录类型均为可编码结构体，实际不会走到。
func HashJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}
