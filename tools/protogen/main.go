// protogen：消息规格 → Go 序列化代码生成器。
//
// 输入：internal/proto/proto.yaml（消息号 ↔ handler 映射、字段顺序与宽度）。
// 规格的两个线索源：
//   - 原版 FsGame.pdb 的 OnXxxMsg 符号（枚举全部 handler 与消息号）
//   - cmd/probe 对原服抓包的字节级验证（字段序/宽度/符号性）
//
// 输出（internal/proto/，生成部分）：
//   - 消息结构体 + Marshal/Unmarshal
//   - 消息号注册表与分发辅助
//
// 验收：生成的 codec 对 testdata/captures/ 抓包切片逐字节 round-trip。
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	spec := flag.String("spec", "internal/proto/proto.yaml", "消息规格文件")
	out := flag.String("out", "internal/proto", "生成产物目录")
	flag.Parse()

	fmt.Printf("protogen — 未实现：spec=%s out=%s\n", *spec, *out)
	fmt.Println("实现要点：规格解析 → 类型映射 → 序列化模板 → gofmt → golden 测试脚手架")
	os.Exit(1)
}
