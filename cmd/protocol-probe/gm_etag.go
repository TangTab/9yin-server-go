// Package main · gm_etag.go —— 游戏服 GM 目录/会话端点的 304 写出器
// （设计文档 §5.1 S3/S4）。口径：
//   - 标签 = 端点名 [+内容哈希|+名称代数] + 进程 boot 戳。进程重启必变
//     （客户端缓存全量失效），进程内内容不变则标签不变；
//   - 依赖 hub.itemNames 的目录（时装/卡牌/发型）额外带名称缓存代数
//     gmItemNames.generation()：名称表后台重载后标签随之变化，客户端
//     自动重拉（即使 boot 戳没变）；
//   - writeGMTaggedJSON 先比对 If-None-Match 再序列化，304 路径零成本。
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/local/9yin-go-server/internal/gmapi"
)

// gmBootStamp 进程启动戳（hex 纳秒）。所有标签都拼它：重启即全量失效。
var gmBootStamp = fmt.Sprintf("%x", time.Now().UnixNano())

// gmWriteTaggedJSON 命中 If-None-Match 回 304 空体；否则 200 + ETag + JSON。
// tag 必须在调用前算好（不触发 value 序列化）。
func gmWriteTaggedJSON(writer http.ResponseWriter, request *http.Request, tag string, value any) {
	etag := gmapi.ETagValue(tag)
	if gmapi.ETagMatches(request.Header.Get("If-None-Match"), etag) {
		gmapi.WriteNotModified(writer)
		return
	}
	gmapi.SetETag(writer, tag)
	writeGMJSON(writer, http.StatusOK, value)
}

// gmWriteContentTagged 内容即标签（小体积运行态端点用：声望/身份状态等）。
// 每次请求都序列化一次算哈希——这些载荷只有几十行，成本可忽略。
func gmWriteContentTagged(writer http.ResponseWriter, request *http.Request, name string, value any) {
	tag := gmapi.HashTag(name, gmBootStamp, marshalForTag(value))
	gmWriteTaggedJSON(writer, request, tag, value)
}

// gmNameGen 名称缓存代数（未启用/未装载为 0）。时装/卡牌目录标签拼它。
func gmNameGen() uint64 { return gmItemNames.generation() }

// gmCatalogTag 静态目录装载完成时算一次的内容标签（name + 行哈希 + boot）。
func gmCatalogTag(name string, rows any) string {
	return gmapi.HashTag(name, marshalForTag(rows), gmBootStamp)
}

func marshalForTag(value any) string {
	return gmapi.HashJSON(value)
}
