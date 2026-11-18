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
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	gonet "net"
	"os"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/net"
	"github.com/lizhanyin/9yin-server-go/internal/proto"
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
		doLogin  = flag.Bool("login", false, "登录模式：重放 golden 首飞并断言 0x04 应答（与 -addr 组合）")
		flight   = flag.String("flight", "testdata/captures/login-connect-001.bin", "登录模式：golden 首飞线路帧路径")
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

	if *doLogin {
		os.Exit(probeLogin(*addr, *flight, opt, *plain))
	}
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

// probeLogin 登录模式：解码 golden 首飞线路帧得 121B 明文 op02，经
// 同装配线路重放，断言 41B 无角色 0x04 应答（含日期字段合理性）、
// op03 → 0x05 五书 worldInfo、op05 建角 → 有角色 0x04。
func probeLogin(addr, flightPath string, opt net.Options, plain bool) int {
	raw, err := os.ReadFile(flightPath)
	if err != nil {
		fatal(err)
	}
	framer, err := net.NewEscapeFramer(bytes.NewBuffer(raw), 4096)
	if err != nil {
		fatal(err)
	}
	wire, err := framer.ReadFrame()
	if err != nil {
		fatal(fmt.Errorf("首飞解帧: %w", err))
	}
	// golden 线路帧为 XOR 编码态；明文基线下（plain）golden 不适用，
	// 该组合直接报错以免误判。
	if plain {
		fatal(fmt.Errorf("登录模式 golden 为 XOR 编码态，不支持 -plain"))
	}
	flight := net.NewXorCipher(net.InitialKey).Decode(wire)
	fmt.Printf("probe: 首飞解码 %d 字节（op=%02X）\n", len(flight), flight[0])

	conn, err := net.Dial(addr, opt)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()
	fmt.Printf("probe: 已连接 %s（%s）\n", addr, lineDesc(plain))

	if err := conn.WriteMsg(flight); err != nil {
		fatal(err)
	}
	reply, err := conn.ReadMsg()
	if err != nil {
		fatal(err)
	}
	fail := 0
	// 0x04 两种合法形态：41B 无角色 / 追加段有角色（服务端角色仓持久，
	// 重复运行 probe 时首登即有角色）
	hasRole := len(reply) > 0x29
	if (len(reply) != 0x29 && !hasRole) || reply[0] != 0x04 {
		fmt.Printf("probe: FAIL — 0x04 应答形态不符（op=%02X len=%d）\n", reply[0], len(reply))
		fail++
	} else if hasRole {
		fmt.Printf("probe: PASS — 0x04 有角色应答（len=%d，账号已有角色）\n", len(reply))
	} else {
		u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(reply[off:]) }
		now := time.Now()
		fmt.Printf("probe: PASS — 0x04 无角色应答 status=%d 时刻=%04d-%02d-%02d %02d:%02d:%02d\n",
			u32(0x01), u32(0x09), u32(0x0D), u32(0x11), u32(0x15), u32(0x19), u32(0x1D))
		if y := u32(0x09); y != uint32(now.Year()) {
			fmt.Printf("probe: FAIL — 应答年份 %d 非 %d\n", y, now.Year())
			fail++
		}
	}

	if err := conn.WriteMsg([]byte{0x03}); err != nil {
		fatal(err)
	}
	info, err := conn.ReadMsg()
	if err != nil {
		fatal(err)
	}
	if len(info) < 3 || info[0] != 0x05 {
		fmt.Printf("probe: FAIL — worldInfo 应答形态不符（op=%02X len=%d）\n", info[0], len(info))
		fail++
	} else {
		units := make([]uint16, 0, (len(info)-3)/2)
		for i := 3; i+1 < len(info); i += 2 {
			u := binary.LittleEndian.Uint16(info[i:])
			if u == 0 {
				break
			}
			units = append(units, u)
		}
		fmt.Printf("probe: PASS — worldInfo type=%d text=%q\n",
			binary.LittleEndian.Uint16(info[1:]), string(utf16.Decode(units)))
	}

	// 建角 op05 → 有角色 0x04（追加段 TVarList 回显名字与 rolePara）
	if err := conn.WriteMsg(buildProbeCreateRole("小虾米")); err != nil {
		fatal(err)
	}
	created, err := conn.ReadMsg()
	if err != nil {
		fatal(err)
	}
	if len(created) <= 0x29 || created[0] != 0x04 {
		fmt.Printf("probe: FAIL — 建角应答形态不符（op=%02X len=%d）\n", created[0], len(created))
		fail++
	} else {
		body, err := proto.DecodeVarList(created[0x29:])
		if err != nil || len(body.Values) != 6 || body.Values[2].Text != "小虾米" {
			fmt.Printf("probe: FAIL — 建角追加段不符（err=%v 值数=%d）\n", err, len(body.Values))
			fail++
		} else {
			fmt.Printf("probe: PASS — 建角 0x04 名字=%q 剧本=%q\n",
				body.Values[2].Text, strings.Split(body.Values[3].Text, ",")[32])
		}
	}

	// 选角 op04 → 0x0B EntryScene（断言场景串回显）
	if err := conn.WriteMsg([]byte{0x04}); err != nil {
		fatal(err)
	}
	entry, err := conn.ReadMsg()
	if err != nil {
		fatal(err)
	}
	if !validateEntryScene(entry, "school09") {
		fail++
	}

	if fail > 0 {
		return 1
	}
	return 0
}

// validateEntryScene 断言 0x0B 进场帧：头部 + 四属性段走位，
// idx7/idx8 场景串须为预期出生场景。
func validateEntryScene(entry []byte, wantResource string) bool {
	if len(entry) < 15 || entry[0] != 0x0B {
		fmt.Printf("probe: FAIL — 0x0B 形态不符（op=%02X len=%d）\n", entry[0], len(entry))
		return false
	}
	u16 := func(off int) uint16 { return binary.LittleEndian.Uint16(entry[off:]) }
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(entry[off:]) }
	if n := u16(13); n != 4 {
		fmt.Printf("probe: FAIL — 0x0B 属性数 %d want 4\n", n)
		return false
	}
	off := 15
	strs := map[uint16]string{}
	for i := 0; i < 4 && off+2 <= len(entry); i++ {
		idx := u16(off)
		off += 2
		switch idx {
		case 178: // [u32 8][8B 遮蔽常量]
			off += 4 + 8
		case 1468: // u16
			off += 2
		default: // 窄串 [u32 len 含 NUL][字节][NUL]
			n := int(u32(off))
			off += 4
			if n < 1 || off+n > len(entry) {
				fmt.Printf("probe: FAIL — 0x0B 属性 %d 越界\n", idx)
				return false
			}
			strs[idx] = string(entry[off : off+n-1])
			off += n
		}
	}
	cfg, hasCfg := strs[7]
	res, hasRes := strs[8]
	if !hasCfg || !hasRes {
		fmt.Printf("probe: FAIL — 0x0B 缺 idx7/idx8（%v）\n", strs)
		return false
	}
	if res != wantResource {
		fmt.Printf("probe: FAIL — 0x0B 场景 %q want %q\n", res, wantResource)
		return false
	}
	fmt.Printf("probe: PASS — 0x0B 进场 config=%q resource=%q\n", cfg, res)
	return true
}

// buildProbeCreateRole 构造 op05 建角请求（女号外观样本）。
func buildProbeCreateRole(name string) []byte {
	appearance := []string{
		"book1",
		`icon\players\girl01_create.png`,
		"", "",
		`obj\char\g_hair\g_hair1`,
		"cloth_g0001", "pants_g0001", "shoes_g0001",
		"1",
	}
	units := utf16.Encode([]rune(name))
	frame := make([]byte, 5, 5+len(units)*2+16*len(appearance))
	frame[0] = 0x05
	for _, u := range units {
		frame = binary.LittleEndian.AppendUint16(frame, u)
	}
	frame = append(frame, 0, 0)
	for len(frame) < 0x49 {
		frame = append(frame, 0)
	}
	for _, s := range appearance {
		frame = append(frame, 6)
		frame = binary.LittleEndian.AppendUint32(frame, uint32(len(s)+1))
		frame = append(frame, s...)
		frame = append(frame, 0)
	}
	return frame
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
