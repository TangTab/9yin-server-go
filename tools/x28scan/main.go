// x28scan 一次性诊断：以 0x38 心跳(30s)为锚测持续帧速率与构成。
package main

import (
	"fmt"
	"io"
	"os"

	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

func main() {
	data, _ := os.ReadFile(os.Args[1])
	f, _ := fsnet.NewEscapeFramer(&byteReader{b: data}, fsnet.DefaultMaxFrameSize)
	cipher := fsnet.NewXorCipher(fsnet.NewKey(fsnet.InitialKeyValue))
	var seg map[byte]int
	segs := []map[byte]int{}
	var total int
	for {
		wire, err := f.ReadFrame()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		p := cipher.Decode(wire)
		total++
		if p[0] == 0x38 { // 心跳锚：切窗
			if seg != nil {
				segs = append(segs, seg)
			}
			seg = map[byte]int{}
			continue
		}
		if seg == nil {
			seg = map[byte]int{}
		}
		seg[p[0]]++
	}
	if seg != nil && len(seg) > 0 {
		segs = append(segs, seg)
	}
	fmt.Printf("total=%d 30s-segments=%d\n", total, len(segs))
	for i, s := range segs {
		fmt.Printf("seg%02d (~30s):", i)
		sum := 0
		for op := byte(0); ; op++ {
			if n, ok := s[op]; ok {
				fmt.Printf(" %02x:%d", op, n)
				sum += n
			}
			if op == 0xff {
				break
			}
		}
		fmt.Printf("  SUM=%d (%.1f/s)\n", sum, float64(sum)/30)
	}
}

type byteReader struct{ b []byte; off int }

func (r *byteReader) Write(p []byte) (int, error) { return len(p), nil }
func (r *byteReader) Read(p []byte) (int, error) {
	if r.off >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.off:])
	r.off += n
	return n, nil
}
