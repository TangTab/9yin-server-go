// propanalyze 对拍分析：解码 wire 抓取（solounite -capture-dir / mitm
// 落盘 EE+XOR 原样字节）后，把全部 0x10 属性帧按「该会话自己的 0x09 表帧」
// 解析为 idx/name/value 清单——2433 线人物数据推送形态对拍用。
//
// 关键口径：0x10 帧的 idx 只在会话内 0x09 表（[0x09][u16 N]{name\0+u8 code}×N，
// idx=条目位置）下才有意义；跨表对拍必须先解表帧。
//
// 用法：go run ./tools/propanalyze <file.bin> [only-obj-hex]
// only-obj-hex 形如 0300eb99（只列该 objectID 的帧）。
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	net "github.com/lizhanyin/9yin-server-go/internal/net"
)

type entry struct {
	name string
	code int
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: propanalyze <file.bin> [only-obj-hex]")
		os.Exit(2)
	}
	onlyObj := uint64(0)
	if len(os.Args) > 2 {
		v, err := strconv.ParseUint(os.Args[2], 16, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bad obj:", err)
			os.Exit(2)
		}
		onlyObj = v
	}
	data, err := res.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := net.NewEscapeFramer(newByteReader(data), net.DefaultMaxFrameSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, "framer:", err)
		os.Exit(1)
	}
	cipher := net.NewXorCipher(net.NewKey(net.InitialKeyValue))
	var frames [][]byte
	for {
		wire, err := f.ReadFrame()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "framer-err: %v\n", err)
			break
		}
		frames = append(frames, cipher.Decode(wire))
	}
	// 先解会话内 0x09 表帧（取最后一个——重发以末次为准）。
	var entries []entry
	for i, plain := range frames {
		if len(plain) < 3 || plain[0] != 0x09 {
			continue
		}
		n := int(binary.LittleEndian.Uint16(plain[1:]))
		off, es := 3, []entry{}
		for j := 0; j < n && off < len(plain); j++ {
			z := off
			for z < len(plain) && plain[z] != 0 {
				z++
			}
			if z+1 >= len(plain) {
				break
			}
			es = append(es, entry{string(plain[off:z]), int(plain[z+1])})
			off = z + 2
		}
		fmt.Fprintf(os.Stderr, "#%d 0x09 table: %d entries (parsed %d)\n", i, n, len(es))
		if len(es) == n {
			entries = es
		}
	}
	for i, plain := range frames {
		if len(plain) < 12 || plain[0] != 0x10 {
			continue
		}
		obj := binary.LittleEndian.Uint32(plain[2:])
		if onlyObj != 0 && uint64(obj) != onlyObj {
			continue
		}
		n := int(binary.LittleEndian.Uint16(plain[10:]))
		off := 12
		for j := 0; j < n && off < len(plain); j++ {
			idx := int(binary.LittleEndian.Uint16(plain[off:]))
			off += 2
			name := fmt.Sprintf("idx%d", idx)
			wt := 0
			if idx < len(entries) {
				name, wt = entries[idx].name, entries[idx].code
			}
			val, width := decodeValue(plain, off, wt)
			off += width
			fmt.Printf("#%-4d obj=%08x n=%d %s = %s\n", i, obj, n, name, val)
			if width == 0 {
				break
			}
		}
		if off != len(plain) {
			fmt.Printf("#%-4d obj=%08x TRAILER %d bytes (% x)\n", i, obj, len(plain)-off, plain[off:])
		}
	}
}

func decodeValue(b []byte, off int, wt int) (string, int) {
	if off >= len(b) {
		return "<short>", 0
	}
	switch wt {
	case 1: // byte
		return fmt.Sprintf("byte(%d)", b[off]), 1
	case 2: // word
		if off+2 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("word(%d)", binary.LittleEndian.Uint16(b[off:])), 2
	case 3: // int32
		if off+4 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("int(%d)", int32(binary.LittleEndian.Uint32(b[off:]))), 4
	case 4: // int64
		if off+8 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("i64(%d)", int64(binary.LittleEndian.Uint64(b[off:]))), 8
	case 5: // float32
		if off+4 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("f32(%v)", math.Float32frombits(binary.LittleEndian.Uint32(b[off:]))), 4
	case 6: // float64
		if off+8 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("f64(%v)", math.Float64frombits(binary.LittleEndian.Uint64(b[off:]))), 8
	case 7: // string
		if off+4 > len(b) {
			return "<short>", 0
		}
		ln := int(binary.LittleEndian.Uint32(b[off:]))
		if ln < 1 || off+4+ln > len(b) {
			return fmt.Sprintf("<bad str len %d>", ln), 0
		}
		return fmt.Sprintf("str(%q)", string(b[off+4:off+4+ln-1])), 4 + ln
	case 8: // wide：线长 = 码元字节数（含 NUL）
		if off+4 > len(b) {
			return "<short>", 0
		}
		ln := int(binary.LittleEndian.Uint32(b[off:]))
		if ln < 2 || ln%2 != 0 || off+4+ln > len(b) {
			return fmt.Sprintf("<bad wide len %d>", ln), 0
		}
		runes := []rune{}
		for p := off + 4; p+1 < off+4+ln; p += 2 {
			u := binary.LittleEndian.Uint16(b[p:])
			if u == 0 {
				break
			}
			runes = append(runes, rune(u))
		}
		return fmt.Sprintf("wide(%q)", string(runes)), 4 + ln
	case 9: // object
		if off+8 > len(b) {
			return "<short>", 0
		}
		v := binary.LittleEndian.Uint64(b[off:])
		return fmt.Sprintf("obj(%08x/%08x)", uint32(v), uint32(v>>32)), 8
	}
	return fmt.Sprintf("<code%d raw % x>", wt, b[off:min(off+8, len(b))]), min(8, len(b)-off)
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
