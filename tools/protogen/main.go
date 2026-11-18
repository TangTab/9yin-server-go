// protogen：消息规格 → Go 代码生成器（对应原版 FsVerEnc 的 gen_encode 机制）。
//
// v1 输入为原版 lua 消息目录（单一事实源，无需手写规格）：
//
//	testdata/lua/share/client_custom_define.lua   C2S 自定义消息（数值 ID + 格式注释）
//	testdata/lua/share/server_custom_define.lua   S2C 自定义消息（字符串名 + 格式注释）
//
// 输出 internal/proto/custom_gen.go：
//   - CustomMsgDef 目录（ID/名字/格式原文/尽力解析的字段类型序列/可变参标记）
//   - C2SCustomByID / C2SCustomByName / S2CCustomByKey / S2CCustomByName 索引
//
// 消息体统一为 TVarList（varlist.go），字段类型序列即 TVarList 类型标签。
// 引擎级消息（0x05/0x18/...）在 message.go 手写——它们定义在 C++ 里，
// 线索源是 PDB 符号与抓包，不来自 lua。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	in := flag.String("in", "", "lua define 目录（默认 testdata/lua/share；设置 NINEYIN_DATA 指向原版根目录时直读 <NINEYIN_DATA>/Res/lua/share）")
	out := flag.String("out", "internal/proto", "生成产物目录")
	flag.Parse()

	dir := resolveIn(*in, filepath.Join("Res", "lua", "share"), "testdata/lua/share")
	if err := run(dir, *out); err != nil {
		fmt.Fprintln(os.Stderr, "protogen:", err)
		os.Exit(1)
	}
}

// resolveIn 输入目录解析优先级：-in 显式参数 > NINEYIN_DATA 环境变量
// （原版根目录，直读不拷贝）> 仓库内 testdata 快照（保证无外部依赖可构建）。
func resolveIn(flagVal, envSub, fallback string) string {
	if flagVal != "" {
		return flagVal
	}
	if root := os.Getenv("NINEYIN_DATA"); root != "" {
		return filepath.Join(root, filepath.FromSlash(envSub))
	}
	return fallback
}

func run(in, out string) error {
	c2s, err := parseDefineFile(filepath.Join(in, "client_custom_define.lua"))
	if err != nil {
		return err
	}
	s2c, err := parseDefineFile(filepath.Join(in, "server_custom_define.lua"))
	if err != nil {
		return err
	}
	if len(c2s) < 100 || len(s2c) < 50 {
		return fmt.Errorf("解析结果异常偏少：C2S=%d S2C=%d", len(c2s), len(s2c))
	}
	src, err := genCatalog(c2s, s2c)
	if err != nil {
		return err
	}
	dst := filepath.Join(out, "custom_gen.go")
	if err := os.WriteFile(dst, src, 0o644); err != nil {
		return err
	}
	fmt.Printf("protogen: C2S %d 条、S2C %d 条 → %s\n", len(c2s), len(s2c), dst)
	return nil
}
