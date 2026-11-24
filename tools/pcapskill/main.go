// pcapskill 抓包逐帧 hex 转储（技能链字节级复刻用）：
// pcapng → <port> TCP 双向重组 → EE+XOR 帧解码 → 按时间窗全帧 hex。
//
// 用法: pcapskill <file.pcapng> <port> <t0> <t1> [filter]
//
//	filter（可组合，逗号分隔）: c2s / s2c / opXX（如 op11,op13）/ mNNN（自定义消息号）
//
// 注：S2C 流 t=522.8 处有一处 7B 带外标记（见 docs/official/初始女号抓包
// m54手动格挡全解.md §六），标准帧器到该点硬停——技能施放全部在此之前。
package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	net "github.com/lizhanyin/9yin-server-go/internal/net"
)

type segRun struct {
	seq uint32
	ts  float64
	b   []byte
}

type event struct {
	ts    float64
	dir   int // 0=c2s 1=s2c
	idx   int
	op    byte
	frame []byte
}

func main() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "usage: pcapskill <file.pcapng> <port> <t0> <t1> [filter]")
		os.Exit(2)
	}
	data, err := res.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	port := uint16(mustInt(os.Args[2]))
	t0, t1 := mustFloat(os.Args[3]), mustFloat(os.Args[4])
	filter := map[string]bool{}
	for _, f := range strings.Split(strings.ToLower(arg(5)), ",") {
		if f != "" {
			filter[f] = true
		}
	}

	linkType, tsres, base := -1, 6.0, -1.0
	type streamKey struct {
		cport uint16
		dir   int
	}
	streams := map[streamKey][]segRun{}
	off := 0
	for off+12 <= len(data) {
		blkType := binary.LittleEndian.Uint32(data[off:])
		blkLen := int(binary.LittleEndian.Uint32(data[off+4:]))
		if blkLen < 12 || off+blkLen > len(data) {
			break
		}
		body := data[off+8 : off+blkLen-4]
		switch blkType {
		case 0x00000001: // IDB：链路类型 + if_tsresol 选项
			linkType = int(binary.LittleEndian.Uint16(body[0:]))
			po := 8
			for po+4 <= len(body) {
				code, ln := int(binary.LittleEndian.Uint16(body[po:])), int(binary.LittleEndian.Uint16(body[po+2:]))
				if code == 0 {
					break
				}
				if code == 9 && ln >= 1 {
					tsres = math.Pow(10, float64(body[po+4]&0x7f))
				}
				po += 4 + ((ln + 3) & ^3)
			}
		case 0x00000006: // EPB
			if len(body) < 20 {
				break
			}
			ts := (float64(binary.LittleEndian.Uint32(body[4:]))*(1<<32) +
				float64(binary.LittleEndian.Uint32(body[8:]))) / tsres
			capLen := int(binary.LittleEndian.Uint32(body[12:]))
			pkt := body[20 : 20+capLen]
			var ip []byte
			switch linkType {
			case 1:
				if len(pkt) >= 14 && binary.BigEndian.Uint16(pkt[12:]) == 0x0800 {
					ip = pkt[14:]
				}
			case 101:
				ip = pkt
			}
			if len(ip) >= 20 && ip[0]>>4 == 4 && ip[9] == 6 {
				ihl := int(ip[0]&0xf) * 4
				tcp := ip[ihl:]
				if len(tcp) >= 20 {
					sp, dp := binary.BigEndian.Uint16(tcp[0:]), binary.BigEndian.Uint16(tcp[2:])
					doff := int(tcp[12]>>4) * 4
					dir, cport := -1, uint16(0)
					if dp == port && sp != port {
						dir, cport = 0, sp
					} else if sp == port && dp != port {
						dir, cport = 1, dp
					}
					if dir >= 0 && len(tcp) > doff && len(tcp[doff:]) > 0 {
						if base < 0 {
							base = ts
						}
						streams[streamKey{cport, dir}] = append(streams[streamKey{cport, dir}],
							segRun{binary.BigEndian.Uint32(tcp[4:]), ts - base, append([]byte(nil), tcp[doff:]...)})
					}
				}
			}
		}
		off += blkLen
	}
	if base < 0 {
		fmt.Fprintln(os.Stderr, "no matching packets")
		os.Exit(1)
	}

	var events []event
	for sk, segs := range streams {
		sort.Slice(segs, func(i, j int) bool { return seqDelta(segs[i].seq, segs[j].seq) < 0 })
		var stream []byte
		var tsAt []float64 // 每字节到达时刻（帧时间=帧尾字节时刻）
		var expect uint32
		first := true
		for _, s := range segs {
			d := s.b
			if first {
				expect, first = s.seq, false
			} else if s.seq != expect {
				if dd := seqDelta(s.seq, expect); dd < 0 {
					if int(-dd) >= len(d) {
						continue
					}
					d = d[-dd:]
				} else {
					stream = append(stream, make([]byte, dd)...)
					tsAt = append(tsAt, make([]float64, dd)...)
				}
			}
			stream = append(stream, d...)
			for range d {
				tsAt = append(tsAt, s.ts)
			}
			expect += uint32(len(d))
		}
		dir := sk.dir
		f, err := net.NewEscapeFramer(&byteReader{b: stream}, net.DefaultMaxFrameSize)
		if err != nil {
			fmt.Fprintln(os.Stderr, "framer:", err)
			os.Exit(1)
		}
		cipher := net.NewXorCipher(net.NewKey(net.InitialKeyValue))
		pos := 0
		for i := 0; ; i++ {
			wire, err := f.ReadFrame()
			if err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
					fmt.Fprintf(os.Stderr, "dir=%d #%d framer stop at stream offset %d (t≈%.3f): %v\n",
						dir, i, pos, frameTs(tsAt, pos), err)
				}
				break
			}
			plain := cipher.Decode(wire)
			// wire 长 = 帧体 + 帧内 EE 转义数 + 终止符 2
			wireLen := len(wire) + bytesCount(wire, 0xEE) + 2
			events = append(events, event{frameTs(tsAt, pos+wireLen-1), dir, i, plain[0], plain})
			pos += wireLen
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].ts != events[j].ts {
			return events[i].ts < events[j].ts
		}
		return events[i].dir < events[j].dir
	})

	for _, ev := range events {
		if ev.ts < t0 || ev.ts > t1 || !match(ev, filter) {
			continue
		}
		d := "C2S"
		if ev.dir == 1 {
			d = "S2C"
		}
		fmt.Printf("t=%9.3f %s #%-4d op=0x%02X len=%-4d % x\n", ev.ts, d, ev.idx, ev.op, len(ev.frame), ev.frame)
	}
}

// match filter 命中判定（空 filter=全放行）。
func match(ev event, filter map[string]bool) bool {
	if len(filter) == 0 {
		return true
	}
	if ev.dir == 0 && filter["c2s"] || ev.dir == 1 && filter["s2c"] {
		return true
	}
	if filter[fmt.Sprintf("op%x", ev.op)] {
		return true
	}
	if m, err := entergen.ParseCustomMessage(ev.frame); err == nil && len(m.Values) > 0 && m.Values[0].Type == 2 {
		return filter[fmt.Sprintf("m%d", m.Values[0].Int32)]
	}
	return false
}

// frameTs 流偏移处字节的到达时刻（越界取末字节）。
func frameTs(tsAt []float64, off int) float64 {
	if off < 0 {
		off = 0
	}
	if off >= len(tsAt) {
		off = len(tsAt) - 1
	}
	if len(tsAt) == 0 {
		return 0
	}
	return tsAt[off]
}

func bytesCount(b []byte, v byte) int {
	n := 0
	for _, x := range b {
		if x == v {
			n++
		}
	}
	return n
}

type byteReader struct {
	b   []byte
	off int
}

func (r *byteReader) Write(p []byte) (int, error) { return len(p), nil }
func (r *byteReader) Read(p []byte) (int, error) {
	if r.off >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.off:])
	r.off += n
	return n, nil
}

func seqDelta(a, b uint32) int32 {
	d := int64(a) - int64(b)
	if d > 1<<31 {
		d -= 1 << 32
	}
	if d < -(1 << 31) {
		d += 1 << 32
	}
	return int32(d)
}

func arg(i int) string {
	if len(os.Args) > i {
		return os.Args[i]
	}
	return ""
}

func mustInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bad int:", s)
		os.Exit(2)
	}
	return v
}

func mustFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bad float:", s)
		os.Exit(2)
	}
	return v
}
