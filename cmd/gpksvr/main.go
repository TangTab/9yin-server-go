// gpksvr 是官方 GPKitSvr.dll 反外挂动态码组件的 Go 复刻自检工具:
// 按官方 Init 配对规则装载 DynCodeBin(Server\*.bin 代码 +
// Server\Client\*.bin 数据),跑一轮 RandomPick→GetData→Call1/Call2
// 挑战-应答闭环,并可对 128 字节回执包做 CCSAuth::CheckAuthReply 校验。
//
// 用法:
//
//	gpksvr -bin DynCodeBin/Server -data DynCodeBin/Client -conf gpkconf.ini
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/lizhanyin/9yin-server-go/internal/gpkit"
)

func main() {
	codeDir := flag.String("bin", `DynCodeBin\Server`, "代码目录(官方 Init 第 1 参)")
	dataDir := flag.String("data", `DynCodeBin\Client`, "数据目录名(官方 Init 第 2 参)")
	confPath := flag.String("conf", "gpkconf.ini", "gpkconf.ini 路径(可缺省)")
	authPath := flag.String("auth", "AuthData.dat", "AuthData.dat 路径(可缺省)")
	flag.Parse()

	svr := gpkit.Create()
	if b, err := os.ReadFile(*confPath); err == nil {
		svr.LoadConf(b)
		fmt.Printf("conf: %+v\n", svr.Conf())
	}
	if b, err := os.ReadFile(*authPath); err == nil {
		svr.LoadAuthData(b)
		fmt.Printf("auth: %d bytes\n", len(svr.AuthData()))
	}

	n := svr.Init(*codeDir, *dataDir)
	if n < 0 {
		fmt.Fprintln(os.Stderr, "GPK Code Load Failed")
		os.Exit(1)
	}
	fmt.Printf("loaded %d dyn code(s)\n", n)
	if n == 0 {
		return
	}

	idx := svr.RandomPick()
	fmt.Printf("RandomPick -> %d (hack %d)\n", idx, idx+10)
	if w, err := gpkit.HackWarning(idx + 10); err == nil {
		fmt.Printf("  warning text: %q\n", w)
	}
	data, err := svr.GetData(uint32(idx) << 16)
	if err != nil {
		fmt.Fprintln(os.Stderr, "GPK GetCltDynCode Failed!!:", err)
		os.Exit(1)
	}
	preview := 16
	if len(data) < preview {
		preview = len(data)
	}
	fmt.Printf("challenge data[%d]: %s\n", len(data), hex.EncodeToString(data[:preview]))

	v1, err := svr.Call1(0xDEAD0001, 0xC4, uint32(idx)<<16)
	if err != nil {
		fmt.Fprintln(os.Stderr, "GPK Encode Failed!!:", err)
		os.Exit(1)
	}
	v2, err := svr.Call2(v1, 0xC4, uint32(idx)<<16)
	if err != nil {
		fmt.Fprintln(os.Stderr, "GPK check failed:", err)
		os.Exit(1)
	}
	fmt.Printf("fn1 -> %08x  fn2 -> %08x\n", v1, v2)

	// 演示 x64 CCSAuth::CheckAuthReply:构造合法 128B 回执(掩码+序号)。
	pkt := make([]byte, gpkit.AuthReplySize)
	binary.LittleEndian.PutUint16(pkt[0:], uint16(idx))
	binary.LittleEndian.PutUint32(pkt[4:8], ^binary.LittleEndian.Uint32(pkt[0:4]))
	if _, err := gpkit.CheckAuthReply(pkt, uint16(idx), nil); err != nil {
		fmt.Println("CheckAuthReply:", err)
		os.Exit(1)
	}
	if _, err := gpkit.CheckAuthReply(bytes.Repeat([]byte{0}, gpkit.AuthReplySize), uint16(idx), nil); err != nil {
		fmt.Printf("CheckAuthReply(bad pkt) rejected: %v\n", err)
	}
	fmt.Println("ok")
}
