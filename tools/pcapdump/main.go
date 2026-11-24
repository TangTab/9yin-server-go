// pcapdump 把 pcapng 里某服务端口的 TCP 会话解成帧后转储为二进制记录，
// 供离线分析脚本逐帧取证（pcapitem/pcapevents 是「按消息锚开窗」的即视
// 工具，本工具只做无损转储、不解业务字段）。
//
// 链路：pcapng → <port> TCP 双向重组（按连接分开；同客户端端口再见 SYN
// 即另起一代）→ EE 转义成帧 → XOR 解密 → 逐帧落盘。
//
// 用法：pcapdump <file.pcapng> <out.bin> [port]
//
//	port 缺省自动识别：取「服务端侧端口 <49152 的会话」里双向负载总和
//	最大者（游戏线口负载远大于 6001 列表口与 HTTP 口）。
//
// 记录格式（小端）：
//
//	头  "PCFR" u16 版本=1 u16 保留 u64 帧数 u16 服务端口 u16 保留
//	帧  f64 相对秒 u64 绝对微秒 u8 方向(0=C2S 1=S2C) u16 客户端端口
//	    u32 流内序号 u32 字节数 字节…
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"

	net "github.com/lizhanyin/9yin-server-go/internal/net"
)

const ephemeralFloor = 49152 // Windows 10+ 动态端口下限：其上视为客户端临时口

type segRun struct {
	seq uint32
	ts  float64 // 相对首包秒
	abs uint64  // 绝对微秒
	syn bool
	b   []byte
}

type stream struct {
	sport uint16
	cport uint16
	dir   int
	segs  []segRun
}

type frame struct {
	ts    float64
	abs   uint64
	dir   int
	cport uint16
	idx   int
	b     []byte
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: pcapdump <file.pcapng> <out.bin> [port]")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	convs, streams := parse(data)
	port := uint16(0)
	if len(os.Args) > 3 {
		v, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintln(os.Stderr, "bad port:", os.Args[3])
			os.Exit(2)
		}
		port = uint16(v)
	} else {
		for p, n := range convs {
			if port == 0 || n > convs[port] {
				port = p
			}
		}
	}
	if port == 0 {
		fmt.Fprintln(os.Stderr, "no server-side conversation found")
		os.Exit(1)
	}
	var chosen []stream
	for _, st := range streams {
		if st.sport == port && len(st.segs) > 0 {
			chosen = append(chosen, st)
		}
	}
	sort.Slice(chosen, func(i, j int) bool { return chosen[i].segs[0].abs < chosen[j].segs[0].abs })
	for _, st := range chosen {
		n := 0
		for _, s := range st.segs {
			n += len(s.b)
		}
		fmt.Fprintf(os.Stderr, "stream cport=%d dir=%d segs=%d bytes=%d\n", st.cport, st.dir, len(st.segs), n)
	}
	frames := decode(chosen)
	if err := write(os.Args[2], port, frames); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	hist := map[[2]int]int{}
	for _, f := range frames {
		if len(f.b) > 0 {
			hist[[2]int{f.dir, int(f.b[0])}]++
		}
	}
	keys := make([]int, 0, len(hist))
	for k := range hist {
		keys = append(keys, k[0]<<8|k[1])
	}
	sort.Ints(keys)
	fmt.Fprintf(os.Stderr, "port=%d frames=%d -> %s\n", port, len(frames), os.Args[2])
	for _, k := range keys {
		d := "C2S"
		if k>>8 == 1 {
			d = "S2C"
		}
		fmt.Fprintf(os.Stderr, "  %s op=0x%02X ×%d\n", d, k&0xff, hist[[2]int{k >> 8, k & 0xff}])
	}
}

// parse pcapng → 会话负载量表（服务端口→字节数）+ 全部流。
func parse(data []byte) (map[uint16]int, []stream) {
	linkTypes := map[uint32]int{}
	tsres := 6.0
	type flowKey struct {
		src, dst string
		sport    uint16
		dport    uint16
	}
	type dirKey struct {
		cport uint16
		dir   int
	}
	type genKey struct {
		dirKey
		gen int
	}
	type flow struct {
		sport, cport uint16
		dir          int
		segs         []segRun
	}
	flows := map[flowKey]*flow{}
	convs := map[uint16]int{}
	t0 := -1.0
	for off := 0; off+12 <= len(data); {
		blkType := binary.LittleEndian.Uint32(data[off:])
		blkLen := int(binary.LittleEndian.Uint32(data[off+4:]))
		if blkLen < 12 || off+blkLen > len(data) {
			break
		}
		body := data[off+8 : off+blkLen-4]
		switch blkType {
		case 0x00000001: // IDB（接口号 = 出现序，body 无自述编号）
			if len(body) >= 8 {
				linkTypes[uint32(len(linkTypes))] = int(binary.LittleEndian.Uint16(body))
				po := 8
				for po+4 <= len(body) {
					code, ln := int(binary.LittleEndian.Uint16(body[po:])), int(binary.LittleEndian.Uint16(body[po+2:]))
					if code == 0 {
						break
					}
					if code == 9 && ln >= 1 {
						if b := body[po+4]; b&0x80 != 0 {
							tsres = math.Pow(2, float64(b&0x7f))
						} else {
							tsres = math.Pow(10, float64(b&0x7f))
						}
					}
					po += 4 + ((ln + 3) & ^3)
				}
			}
		case 0x00000006: // EPB
			if len(body) < 20 {
				break
			}
			iface := binary.LittleEndian.Uint32(body)
			raw := uint64(binary.LittleEndian.Uint32(body[4:]))<<32 | uint64(binary.LittleEndian.Uint32(body[8:]))
			ts := float64(raw) / tsres
			capLen := int(binary.LittleEndian.Uint32(body[12:]))
			if 20+capLen > len(body) {
				break
			}
			pkt := body[20 : 20+capLen]
			var ip []byte
			switch linkTypes[iface] {
			case 1:
				if len(pkt) >= 14 && binary.BigEndian.Uint16(pkt[12:]) == 0x0800 {
					ip = pkt[14:]
				}
			case 101:
				ip = pkt
			}
			if len(ip) < 20 || ip[0]>>4 != 4 || ip[9] != 6 {
				break
			}
			tcp := ip[int(ip[0]&0xf)*4:]
			if len(tcp) < 20 {
				break
			}
			sp, dp := binary.BigEndian.Uint16(tcp), binary.BigEndian.Uint16(tcp[2:])
			doff := int(tcp[12]>>4) * 4
			if len(tcp) < doff || doff < 20 {
				break
			}
			// 服务端侧 = 非临时口一侧；两侧都是/都不是临时口则不成会话。
			dir, sport, cport := -1, uint16(0), uint16(0)
			switch {
			case dp < ephemeralFloor && sp >= ephemeralFloor:
				dir, sport, cport = 0, dp, sp
			case sp < ephemeralFloor && dp >= ephemeralFloor:
				dir, sport, cport = 1, sp, dp
			default:
				break
			}
			if t0 < 0 {
				t0 = ts
			}
			payload := tcp[doff:]
			convs[sport] += len(payload)
			key := flowKey{dot(ip[12:16]), dot(ip[16:20]), sp, dp}
			fl := flows[key]
			if fl == nil {
				fl = &flow{sport: sport, cport: cport, dir: dir}
				flows[key] = fl
			}
			run := segRun{
				seq: binary.BigEndian.Uint32(tcp[4:]), ts: ts - t0, abs: uint64(math.Round(ts * 1e6)),
				syn: tcp[13]&0x02 != 0 && tcp[13]&0x10 == 0,
			}
			if len(payload) > 0 {
				run.b = append([]byte(nil), payload...)
			}
			fl.segs = append(fl.segs, run)
		}
		off += blkLen
	}
	// 同一方向（cport, dir）上再见 SYN 即另起一代：捕获跨多次连接复用
	// 同一客户端端口时，帧流不互相污染。
	groups := map[genKey]*stream{}
	byKey := map[dirKey][]*flow{}
	for _, fl := range flows {
		byKey[dirKey{fl.cport, fl.dir}] = append(byKey[dirKey{fl.cport, fl.dir}], fl)
	}
	for k, fls := range byKey {
		for _, fl := range fls {
			sort.Slice(fl.segs, func(i, j int) bool { return fl.segs[i].abs < fl.segs[j].abs })
		}
		sort.Slice(fls, func(i, j int) bool { return fls[i].segs[0].abs < fls[j].segs[0].abs })
		gen := 0
		for _, fl := range fls {
			sawSyn := false
			for _, s := range fl.segs {
				if s.syn && !sawSyn {
					gen++
					sawSyn = true
				}
				sk := genKey{k, gen}
				g := groups[sk]
				if g == nil {
					g = &stream{sport: fl.sport, cport: k.cport, dir: k.dir}
					groups[sk] = g
				}
				if len(s.b) > 0 {
					g.segs = append(g.segs, s)
				}
			}
		}
	}
	var out []stream
	for _, g := range groups {
		if len(g.segs) > 0 {
			out = append(out, *g)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].cport != out[j].cport {
			return out[i].cport < out[j].cport
		}
		return out[i].dir < out[j].dir
	})
	return convs, out
}

func dot(b []byte) string { return string([]byte{b[0], '.', b[1], '.', b[2], '.', b[3]}) }

// decode 各流按 seq 重组、成帧、解密。
func decode(streams []stream) []frame {
	var frames []frame
	for _, st := range streams {
		sort.Slice(st.segs, func(i, j int) bool { return seqDelta(st.segs[i].seq, st.segs[j].seq) < 0 })
		var stream []byte
		var tsArr []float64
		var absArr []uint64
		var expect uint32
		first := true
		for _, s := range st.segs {
			d := s.b
			if first {
				expect, first = s.seq, false
			} else if s.seq != expect {
				if dd := seqDelta(s.seq, expect); dd < 0 {
					if skip := int(-dd); skip >= len(d) {
						continue
					} else {
						d = d[skip:]
					}
				} else {
					for i := int32(0); i < dd; i++ { // 序号空洞填零续解
						stream = append(stream, 0)
						tsArr = append(tsArr, s.ts)
						absArr = append(absArr, s.abs)
					}
					expect = s.seq
				}
			}
			stream = append(stream, d...)
			for range d {
				tsArr = append(tsArr, s.ts)
				absArr = append(absArr, s.abs)
			}
			expect += uint32(len(d))
		}
		if len(stream) == 0 {
			continue
		}
		f, err := net.NewEscapeFramer(&byteReader{b: stream}, net.DefaultMaxFrameSize)
		if err != nil {
			continue
		}
		cipher := net.NewXorCipher(net.NewKey(net.InitialKeyValue))
		pos := 0
		for i := 0; ; i++ {
			wire, err := f.ReadFrame()
			if err != nil {
				break
			}
			wireLen := len(wire) + bytesCount(wire, 0xEE) + 2
			end := min(max(pos+wireLen-1, 0), len(tsArr)-1)
			plain := cipher.Decode(wire)
			if len(plain) == 0 {
				pos += wireLen
				continue
			}
			frames = append(frames, frame{
				ts: tsArr[end], abs: absArr[end], dir: st.dir, cport: st.cport, idx: i, b: plain,
			})
			pos += wireLen
		}
	}
	sort.SliceStable(frames, func(i, j int) bool {
		if frames[i].ts != frames[j].ts {
			return frames[i].ts < frames[j].ts
		}
		if frames[i].cport != frames[j].cport {
			return frames[i].cport < frames[j].cport
		}
		return frames[i].dir < frames[j].dir
	})
	return frames
}

func write(path string, port uint16, frames []frame) error {
	out := make([]byte, 0, len(frames)*64)
	out = append(out, 'P', 'C', 'F', 'R')
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint16(out, 0)
	out = binary.LittleEndian.AppendUint64(out, uint64(len(frames)))
	out = binary.LittleEndian.AppendUint16(out, port)
	out = binary.LittleEndian.AppendUint16(out, 0)
	for _, f := range frames {
		out = binary.LittleEndian.AppendUint64(out, math.Float64bits(f.ts))
		out = binary.LittleEndian.AppendUint64(out, f.abs)
		out = append(out, byte(f.dir))
		out = binary.LittleEndian.AppendUint16(out, f.cport)
		out = binary.LittleEndian.AppendUint32(out, uint32(f.idx))
		out = binary.LittleEndian.AppendUint32(out, uint32(len(f.b)))
		out = append(out, f.b...)
	}
	return os.WriteFile(path, out, 0o644)
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

func bytesCount(b []byte, v byte) int {
	n := 0
	for _, x := range b {
		if x == v {
			n++
		}
	}
	return n
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
