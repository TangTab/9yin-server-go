// member 是角色服务进程（对应原版 game_member.exe + FsMember.dll）：
// 角色列表/创建/选择、角色数据存取调度。
// 第二里程碑启用。
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("member — 第二里程碑启用")
	os.Exit(0)
}
