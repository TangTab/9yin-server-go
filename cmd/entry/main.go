// entry 是登录网关进程（对应原版 game_entry.exe + FsEntry.dll）：
// 接受客户端连接、账号认证、向 lister 查区服列表、把玩家转给 member。
//
// 第二里程碑启用（solo 形态下其职责在进程内实现）。
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("entry — 第二里程碑启用（solo 先行）")
	os.Exit(0)
}
