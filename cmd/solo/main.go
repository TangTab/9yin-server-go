// solo 是单进程整合服入口，对应原版 FsSoloLoader + FsSolo* 模块集：
// 把 entry/lister/member/world/public 全部角色装进一个进程跑。
//
// 这是第一里程碑的载体：登录→选角→进场景→AOI→属性同步→落库，
// 跑通即拿到整套架构的 80%；之后再按角色拆出独立 cmd。
package main

import (
	"fmt"
	"os"
)

const version = "0.0.1-solo-skeleton"

// milestones 是第一里程碑的串联清单，随实现逐步勾掉：
//   - [ ] ecs 内核（World/Entity/Storage/Query）
//   - [ ] structgen 产出 component + 元数据
//   - [ ] config 加载 scenes.xml / logic_class.xml
//   - [ ] engine tick 循环 + proc/inproc Bus
//   - [ ] net/proto 最小集：握手 + 心跳（probe 对拍）
//   - [ ] login 主干：进场景 + AOI + 属性全量集下发
//   - [ ] persist：DDL 建库 + 角色存取
func main() {
	fmt.Printf("9yin-server-go solo %s — skeleton, engine not wired yet\n", version)
	fmt.Println("milestones: see cmd/solo/main.go 注释清单")
	os.Exit(0)
}
