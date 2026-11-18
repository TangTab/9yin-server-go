// structgen：struct XML → Go 组件定义代码生成器。
//
// 输入：testdata/struct/ 下的对象属性定义 XML（源自原版 Res/struct/），
// 形如：
//
//	<property name="Sex" type="byte" public="true" private="true"
//	          save="true" desc="性别"/>
//
// 输出（internal/component/，只放生成物）：
//   - 组件结构体（按对象文件分组为少量大组件）
//   - 字段元数据表：属性名/类型/public/private/save/realtime
//   - 属性号映射表（供 proto 层使用）
//
// 产出被三处共用：sync（推送哪些字段）、persist（落库哪些字段）、
// proto（属性号）。规格变更一律改 XML 后重跑生成，不手改产物。
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	in := flag.String("in", "testdata/struct", "struct XML 输入目录")
	out := flag.String("out", "internal/component", "生成产物目录")
	flag.Parse()

	fmt.Printf("structgen — 未实现：in=%s out=%s\n", *in, *out)
	fmt.Println("实现要点：GBK→UTF-8 转码 → 属性解析 → 分组聚合 → 模板渲染 → gofmt")
	os.Exit(1)
}
