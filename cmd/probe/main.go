// probe 是协议探测与对拍客户端：连接原版服务端（或本实现），
// 按用例回放消息流、抓取字节、与 testdata/captures/ 的 golden 对拍。
//
// 职责（对齐旧分支 protocol-probe 的经验）：
//   - 模拟客户端走登录→进场景→移动→技能 的消息流
//   - 原始字节落盘 testdata/captures/（golden 用例来源）
//   - 断言本实现 net/proto 的输出与原服字节一致
//
// 本包是全项目的 oracle 基础设施，比业务代码更早稳定。
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("probe — 未实现：待 net/proto 最小集就绪后接入")
	os.Exit(0)
}
