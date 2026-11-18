// public 是公共数据进程（对应原版 game_public.exe + FsPublic.dll）：
// 跨场景共享数据（拍卖、排行、全局状态）。
// 第二里程碑启用。
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("public — 第二里程碑启用")
	os.Exit(0)
}
