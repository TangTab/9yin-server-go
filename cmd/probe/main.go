// probe 是协议探测与对拍客户端（全项目的 oracle 基础设施）：
//
// 阶段一（本实现）：
//   - 默认模式：连本服网关跑「握手+心跳往返」——TCP 建立（线路握手）
//     + golden ping（fork 49840 抓包 326 对钉死的字节形态）逐字节断言 pong
//   - -svrlist 模式：连区服列表分发服务，发 golden 请求 "svrlist\n"，
//     断言应答符合方言（$svrlist #名字 $地址 端口 尾标 CRLF）
//
// 后续阶段：连原版服务端回放登录流首段、原始字节落盘 testdata/captures/
// 供 golden 用例；对拍基线可切明文（-plain，对应 CJ 分支）。
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	gonet "net"
	"os"
	"strings"
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
		addr     = flag.String("addr", "127.0.0.1:9600", "目标服务地址（心跳模式）")
		svrlist  = flag.String("svrlist", "", "svrlist 模式：区服列表服务地址（空=心跳模式）")
		count    = flag.Int("count", 3, "心跳轮数")
		interval = flag.Duration("interval", time.Second, "轮间隔")
		plain    = flag.Bool("plain", false, "明文线路（CJ 对拍基线，不用 XOR）")
	)
	flag.Parse()

	if *svrlist != "" {
		os.Exit(probeSvrlist(*svrlist))
	}

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

// probeSvrlist svrlist 模式：发 golden 请求、读应答至断连、按方言断言。
func probeSvrlist(addr string) int {
	conn, err := gonet.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()
	fmt.Printf("probe: 已连接 %s（svrlist 明文行协议）\n", addr)

	if _, err := conn.Write([]byte("svrlist\n")); err != nil {
		fatal(err)
	}
	var sb strings.Builder
	buf := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		n, err := conn.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break // 应答后服务端断连，EOF 即完整
		}
	}
	resp := sb.String()
	fmt.Printf("应答: %q\n", resp)

	// 方言断言：$svrlist #<\u名> $<host> <port> <flag> CRLF
	if !strings.HasPrefix(resp, "$svrlist #") || !strings.HasSuffix(resp, "\r\n") {
		fmt.Println("probe: FAIL — 应答不符合 svrlist 方言（头/尾）")
		return 1
	}
	fields := strings.Fields(strings.TrimSuffix(resp, "\r\n"))
	if len(fields) != 5 || !strings.HasPrefix(fields[2], "$") {
		fmt.Printf("probe: FAIL — 字段数/形态错误: %v\n", fields)
		return 1
	}
	fmt.Printf("probe: PASS — 名=%s 通告=%s:%s 尾标=%s\n",
		unescapeUnicode(fields[1][1:]), fields[2][1:], fields[3], fields[4])
	return 0
}

// unescapeUnicode 还原 \uXXXX 转义（仅打印用；非法序列原样保留）。
func unescapeUnicode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if len(s)-i >= 6 && s[i] == '\\' && s[i+1] == 'u' {
			var v rune
			if _, err := fmt.Sscanf(s[i+2:i+6], "%04X", &v); err == nil {
				b.WriteRune(v)
				i += 5
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
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
