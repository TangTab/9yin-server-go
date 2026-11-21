// decode_capture 解码 solounite -capture-dir 落盘的 wire 抓取
// （EE+XOR 原样字节）为帧序列摘要：opcode/长度/头部十六进制。
// 用法：go run ./tools/decode_capture <file.bin> [key-hex]
// key 缺省 FA6D3BCC（InitialKey；MessageEncodeKey 注释态时生效）。
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: decode_capture <file.bin> [key-hex]")
		os.Exit(2)
	}
	key := fsnet.NewKey(fsnet.InitialKeyValue)
	if len(os.Args) > 2 {
		v, err := strconv.ParseUint(os.Args[2], 16, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bad key:", err)
			os.Exit(2)
		}
		key = fsnet.NewKey(uint32(v))
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := fsnet.NewEscapeFramer(newByteReader(data), fsnet.DefaultMaxFrameSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, "framer:", err)
		os.Exit(1)
	}
	cipher := fsnet.NewXorCipher(key)
	// 可选提取：decode_capture <file> <key-hex> <frameNo> <out.bin>
	extract, outPath := -1, ""
	if len(os.Args) > 4 {
		extract, _ = strconv.Atoi(os.Args[3])
		outPath = os.Args[4]
	}
	for i := 0; ; i++ {
		wire, err := f.ReadFrame()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("#%d FRAMER-ERR %v (at offset %d)\n", i, err, len(data))
			break
		}
		plain := cipher.Decode(wire)
		if i == extract && outPath != "" {
			if err := os.WriteFile(outPath, plain, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "write:", err)
				os.Exit(1)
			}
		}
		head := 16
		if len(plain) < head {
			head = len(plain)
		}
		fmt.Printf("#%-3d op=0x%02X len=%-6d head=% x\n", i, plain[0], len(plain), plain[:head])
	}
}

type byteReader struct{ b []byte; off int }

func newByteReader(b []byte) *byteReader           { return &byteReader{b: b} }
func (r *byteReader) Write(p []byte) (int, error) { return len(p), nil }
func (r *byteReader) Read(p []byte) (int, error) {
	if r.off >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.off:])
	r.off += n
	return n, nil
}
