// probe 是协议探测与对拍客户端（全项目的 oracle 基础设施）：
//
// 阶段一（本实现）：连本服网关跑「握手+心跳往返」——TCP 建立（线路握手）
// + golden ping（fork 49840 抓包 326 对钉死的字节形态）逐字节断言 pong。
//
// 后续阶段：连原版服务端回放登录流首段、原始字节落盘 testdata/captures/
// 供 golden 用例；对拍基线可切明文（-plain，对应 CJ 分支）。
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/net"
)

// goldenPings 旧分支心跳对拍的实测样本（ping→pong，首字节 18→38 其余逐字节回显）。
var goldenPings = [][2]string{
	{"183154470000140000000000", "383154470000140000000000"},
	{"18f15b470000150000000000", "38f15b470000150000000000"},
}

func main() {
	var (
		addr     = flag.String("addr", "127.0.0.1:9600", "目标服务地址")
		count    = flag.Int("count", 3, "心跳轮数")
		interval = flag.Duration("interval", time.Second, "轮间隔")
		plain    = flag.Bool("plain", false, "明文线路（CJ 对拍基线，不用 XOR）")
	)
	flag.Parse()

	opt := net.Options{}
	if *plain {
		opt.ReadCipher = net.IdentityCipher{}
		opt.WriteCipher = net.IdentityCipher{}
	}

	conn, err := net.Dial(*addr, opt)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()
	fmt.Printf("probe: 已连接 %s（%s）\n", *addr, lineDesc(*plain))

	pass, fail := 0, 0
	for i := 0; i < *count; i++ {
		c := goldenPings[i%len(goldenPings)]
		ping, _ := hex.DecodeString(c[0])
		want, _ := hex.DecodeString(c[1])

		start := time.Now()
		if err := conn.WriteMsg(ping); err != nil {
			fatal(fmt.Errorf("发送第 %d 轮: %w", i+1, err))
		}
		got, err := conn.ReadMsg()
		rtt := time.Since(start)
		if err != nil {
			fatal(fmt.Errorf("接收第 %d 轮: %w", i+1, err))
		}
		if string(got) != string(want) {
			fmt.Printf("第 %d/%d 轮 FAIL\n  sent %x\n  got  %x\n  want %x\n",
				i+1, *count, ping, got, want)
			fail++
		} else {
			fmt.Printf("第 %d/%d 轮 PASS  %x → %x  (%s)\n",
				i+1, *count, ping, got, rtt.Round(time.Microsecond))
			pass++
		}
		if i+1 < *count && *interval > 0 {
			time.Sleep(*interval)
		}
	}

	fmt.Printf("probe: %d 通过 / %d 失败\n", pass, fail)
	if fail > 0 {
		os.Exit(1)
	}
}

func lineDesc(plain bool) string {
	if plain {
		return "EE 定界 + 明文"
	}
	return "EE 定界 + XOR 初始键"
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "probe:", err)
	os.Exit(1)
}
