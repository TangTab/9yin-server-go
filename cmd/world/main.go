// world 是场景服进程（对应原版 game_world.exe + FsWorld.dll）：
// 场景生命周期、AOI、战斗 tick。
// 第二里程碑启用。
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("world — 第二里程碑启用")
	os.Exit(0)
}
