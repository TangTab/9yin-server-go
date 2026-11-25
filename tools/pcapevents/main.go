// pcapevents（原 m54scan）带时间轴的抓包事件分析器：
// pcapng → 8032 端口 TCP 双向重组（带每字节到达时间戳）→ EE+XOR 帧解码 →
// C2S 0x0A 自定义消息（m54 格挡为主）/ S2C 0x09 表 + 0x10 属性帧事件流。
//
// 用法: go run ./tools/pcapevents <file.pcapng> [server-port] [dumpall|dumprec]
//
//	dumpall = 全事件一行一帧（属性名解码/自定义消息值表），供同目录
//	rushband.py / staged.py / m337check.py 文本分析
//	dumprec = dumpall 同款全事件，另把记录族帧（0x11/0x12/0x13/0x14）按
//	官方 0x0A schema 解成「记录名 行 列=值」（战斗簇 fight_relation_rec /
//	attack_back_rec / cooldown_rec 逐格取证用；dumpall 行格式不变）
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
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
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pcapevents <file.pcapng> [server-port]")
		os.Exit(2)
	}
	port := uint16(8032)
	if len(os.Args) > 2 {
		v, _ := strconv.Atoi(os.Args[2])
		port = uint16(v)
	}
	data, err := res.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	linkType := -1
	tsres := 6.0
	t0 := -1.0
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
		case 0x00000001: // IDB
			linkType = int(binary.LittleEndian.Uint16(body[0:]))
			po := 8
			for po+4 <= len(body) {
				code, ln := int(binary.LittleEndian.Uint16(body[po:])), int(binary.LittleEndian.Uint16(body[po+2:]))
				if code == 0 {
					break
				}
				if code == 9 && ln >= 1 {
					b := body[po+4]
					if b&0x80 != 0 {
						tsres = math.Pow(2, float64(b&0x7f))
					} else {
						tsres = math.Pow(10, float64(b&0x7f))
					}
				}
				po += 4 + ((ln + 3) & ^3)
			}
		case 0x00000006: // EPB
			if len(body) < 20 {
				break
			}
			tsHi := binary.LittleEndian.Uint32(body[4:])
			tsLo := binary.LittleEndian.Uint32(body[8:])
			// pcapng: raw64 以 10^-tsres 秒为单位 → 秒 = raw64 / tsres
			ts := (float64(tsHi)*(1<<32) + float64(tsLo)) / tsres
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
					sp := binary.BigEndian.Uint16(tcp[0:])
					dp := binary.BigEndian.Uint16(tcp[2:])
					doff := int(tcp[12]>>4) * 4
					dir := -1
					cport := uint16(0)
					if dp == port && sp != port {
						dir, cport = 0, sp
					} else if sp == port && dp != port {
						dir, cport = 1, dp
					}
					if dir >= 0 && len(tcp) > doff {
						payload := tcp[doff:]
						if len(payload) > 0 {
							if t0 < 0 {
								t0 = ts
							}
							cp := make([]byte, len(payload))
							copy(cp, payload)
							k := streamKey{cport, dir}
							streams[k] = append(streams[k], segRun{binary.BigEndian.Uint32(tcp[4:]), ts - t0, cp})
						}
					}
				}
			}
		}
		off += blkLen
	}
	if t0 < 0 {
		fmt.Fprintln(os.Stderr, "no matching packets")
		os.Exit(1)
	}

	var events []event
	for sk, segs := range streams {
		fmt.Fprintf(os.Stderr, "stream cport=%d dir=%d segs=%d bytes=%d\n", sk.cport, sk.dir, len(segs), func() int {
			n := 0
			for _, s := range segs {
				n += len(s.b)
			}
			return n
		}())
		dir := sk.dir
		sort.Slice(segs, func(i, j int) bool { return seqDelta(segs[i].seq, segs[j].seq) < 0 })
		var stream []byte
		var tsArr []float64
		var expect uint32
		first := true
		for _, s := range segs {
			d := s.b
			if first {
				expect, first = s.seq, false
			} else if s.seq != expect {
				if dd := seqDelta(s.seq, expect); dd < 0 {
					skip := int(-dd)
					if skip >= len(d) {
						continue
					}
					d = d[skip:]
				} else {
					fmt.Fprintf(os.Stderr, "warn dir=%d gap seq %d expect %d (%d bytes)\n", dir, s.seq, expect, dd)
					// 前跳：空洞字节填 ts=前段末时间
					for i := uint32(0); i < uint32(dd); i++ {
						stream = append(stream, 0)
						tsArr = append(tsArr, s.ts)
					}
					expect = s.seq
				}
			}
			stream = append(stream, d...)
			for range d {
				tsArr = append(tsArr, s.ts)
			}
			expect += uint32(len(d))
		}
		rd := &byteReader{b: stream}
		f, err := net.NewEscapeFramer(rd, net.DefaultMaxFrameSize)
		if err != nil {
			fmt.Fprintln(os.Stderr, "framer:", err)
			os.Exit(1)
		}
		cipher := net.NewXorCipher(net.NewKey(net.InitialKeyValue))
		pos := 0
		for i := 0; ; i++ {
			wire, err := f.ReadFrame()
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "dir=%d #%d framer-err %v\n", dir, i, err)
				}
				break
			}
			// wire 长度 = 帧体 + 帧内 EE 转义 + 终止符 2
			ee := 0
			for _, b := range wire {
				if b == 0xEE {
					ee++
				}
			}
			wireLen := len(wire) + ee + 2
			end := pos + wireLen - 1
			ts := 0.0
			if end >= 0 && end < len(tsArr) {
				ts = tsArr[end]
			}
			plain := cipher.Decode(wire)
			events = append(events, event{ts, dir, i, plain[0], plain})
			pos += wireLen
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].ts != events[j].ts {
			return events[i].ts < events[j].ts
		}
		return events[i].dir < events[j].dir
	})

	// S2C 0x09 属性表（取最后一次完整表）。
	var entries []propEntry
	for _, ev := range events {
		if ev.dir == 1 && len(ev.frame) >= 3 && ev.frame[0] == 0x09 {
			n := int(binary.LittleEndian.Uint16(ev.frame[1:]))
			o, es := 3, []propEntry{}
			for j := 0; j < n && o < len(ev.frame); j++ {
				z := o
				for z < len(ev.frame) && ev.frame[z] != 0 {
					z++
				}
				if z+1 >= len(ev.frame) {
					break
				}
				es = append(es, propEntry{string(ev.frame[o:z]), int(ev.frame[z+1])})
				o = z + 2
			}
			if len(es) == n {
				entries = es
			}
		}
	}
	fmt.Fprintf(os.Stderr, "prop table: %d entries\n", len(entries))

	// dumpall 模式：全部事件打一行（S2C 0x10 解属性、0x1E 打原始 hex 头 48B）。
	// dumprec 同款，记录族帧另按 schema 解码（见 decodeRecordFrame）。
	mode := ""
	if len(os.Args) > 3 {
		mode = os.Args[3]
	}
	if mode == "dumpall" || mode == "dumprec" {
		for _, ev := range events {
			d := "C2S"
			if ev.dir == 1 {
				d = "S2C"
			}
			switch {
			case ev.op == 0x10 && len(ev.frame) >= 12:
				obj := binary.LittleEndian.Uint32(ev.frame[2:])
				n := int(binary.LittleEndian.Uint16(ev.frame[10:]))
				o := 12
				ps := ""
				for j := 0; j < n && o < len(ev.frame); j++ {
					idx := int(binary.LittleEndian.Uint16(ev.frame[o:]))
					o += 2
					name := fmt.Sprintf("idx%d", idx)
					wt := 0
					if idx < len(entries) {
						name, wt = entries[idx].name, entries[idx].code
					}
					val, width := decodeValue(ev.frame, o, wt)
					o += width
					ps += fmt.Sprintf(" %s=%s", name, val)
					if width == 0 {
						break
					}
				}
				fmt.Printf("A t=%9.3f %s #%d 0x10 obj=%08x:%s\n", ev.ts, d, ev.idx, obj, ps)
			case ev.op == 0x1E || ev.op == 0x27 || ev.op == 0x0A:
				hl := 96
				if len(ev.frame) < hl {
					hl = len(ev.frame)
				}
				if m, err := entergen.ParseCustomMessage(ev.frame); err == nil && len(m.Values) >= 1 && m.Values[0].Type == 2 {
					vs := ""
					for _, vv := range m.Values[1:] {
						vs += fmt.Sprintf(" %s", valStr(vv))
					}
					fmt.Printf("A t=%9.3f %s #%d custom m%d%s\n", ev.ts, d, ev.idx, m.Values[0].Int32, vs)
				} else if m, ok, err := entergen.ParseActivityCustomMessage(ev.frame); ok && err == nil && len(m.Values) >= 1 && m.Values[0].Type == 2 {
					vs := ""
					for _, vv := range m.Values[1:] {
						vs += fmt.Sprintf(" %s", valStr(vv))
					}
					fmt.Printf("A t=%9.3f %s #%d actenv m%d%s\n", ev.ts, d, ev.idx, m.Values[0].Int32, vs)
				} else {
					fmt.Printf("A t=%9.3f %s #%d op=0x%02X len=%d hex=% x\n", ev.ts, d, ev.idx, ev.op, len(ev.frame), ev.frame[:hl])
				}
			default:
				if mode == "dumprec" && ev.dir == 1 {
					if desc, ok := decodeRecordFrame(ev.frame); ok {
						fmt.Printf("A t=%9.3f %s #%d rec op=0x%02X len=%d %s\n", ev.ts, d, ev.idx, ev.op, len(ev.frame), desc)
						continue
					}
				}
				fmt.Printf("A t=%9.3f %s #%d op=0x%02X len=%d\n", ev.ts, d, ev.idx, ev.op, len(ev.frame))
			}
		}
		return
	}

	// m54 事件 + 上下文窗口。
	var m54s []int
	for k, ev := range events {
		if ev.dir == 0 && ev.op == 0x0A {
			msg, ok, err := entergen.ParseActivityCustomMessage(ev.frame)
			if ok && err == nil && len(msg.Values) >= 1 && msg.Values[0].Type == 2 && msg.Values[0].Int32 == 54 {
				m54s = append(m54s, k)
			}
		}
	}
	fmt.Fprintf(os.Stderr, "events=%d m54=%d\n", len(events), len(m54s))
	for _, k := range m54s {
		ev := events[k]
		seq := uint32(0)
		if len(ev.frame) >= 21 {
			seq = binary.LittleEndian.Uint32(ev.frame[17:21])
		}
		msg, _, _ := entergen.ParseActivityCustomMessage(ev.frame)
		vs := ""
		for j, v := range msg.Values {
			if j == 0 {
				continue
			}
			vs += fmt.Sprintf(" [%d] tag=%d %s", j, v.Type, valStr(v))
		}
		fmt.Printf("M54 t=%8.3f #%d seq=%d n=%d%s\n", ev.ts, ev.idx, seq, len(msg.Values), vs)
	}

	// 值变迁点上下文窗口：±1.5s 双向帧时间线（S2C 0x10 解属性名）。
	if len(m54s) > 0 {
		prevVal := int64(0)
		for wi, k := range m54s {
			ev := events[k]
			msg, _, _ := entergen.ParseActivityCustomMessage(ev.frame)
			v := int64(0)
			if len(msg.Values) > 1 {
				v = msg.Values[1].Int64
			}
			if wi > 0 && v == prevVal {
				continue
			}
			prevVal = v
			fmt.Printf("=== WINDOW t=%.3f value=%d ===\n", ev.ts, v)
			for _, e2 := range events {
				if e2.ts < ev.ts-1.5 || e2.ts > ev.ts+1.5 {
					continue
				}
				switch {
				case e2.dir == 0 && e2.op == 0x0A:
					if m, ok, err := entergen.ParseActivityCustomMessage(e2.frame); ok && err == nil && len(m.Values) >= 1 && m.Values[0].Type == 2 {
						vs := ""
						for _, vv := range m.Values[1:] {
							vs += fmt.Sprintf(" %s", valStr(vv))
						}
						fmt.Printf("  t=%8.3f C2S m%d %s\n", e2.ts, m.Values[0].Int32, vs)
					} else {
						fmt.Printf("  t=%8.3f C2S 0x0A len=%d\n", e2.ts, len(e2.frame))
					}
				case e2.dir == 1 && e2.op == 0x10 && len(e2.frame) >= 12:
					obj := binary.LittleEndian.Uint32(e2.frame[2:])
					n := int(binary.LittleEndian.Uint16(e2.frame[10:]))
					o := 12
					ps := ""
					for j := 0; j < n && o < len(e2.frame); j++ {
						idx := int(binary.LittleEndian.Uint16(e2.frame[o:]))
						o += 2
						name := fmt.Sprintf("idx%d", idx)
						wt := 0
						if idx < len(entries) {
							name, wt = entries[idx].name, entries[idx].code
						}
						val, width := decodeValue(e2.frame, o, wt)
						o += width
						ps += fmt.Sprintf(" %s=%s", name, val)
						if width == 0 {
							break
						}
					}
					fmt.Printf("  t=%8.3f S2C 0x10 obj=%08x:%s\n", e2.ts, obj, ps)
				case e2.dir == 1:
					fmt.Printf("  t=%8.3f S2C op=0x%02X len=%d\n", e2.ts, e2.op, len(e2.frame))
				default:
					fmt.Printf("  t=%8.3f C2S op=0x%02X len=%d\n", e2.ts, e2.op, len(e2.frame))
				}
			}
		}
	}
}

func decodeValue(b []byte, off int, wt int) (string, int) {
	if off >= len(b) {
		return "<short>", 0
	}
	switch wt {
	case 1:
		return fmt.Sprintf("byte(%d)", b[off]), 1
	case 2:
		if off+2 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("word(%d)", binary.LittleEndian.Uint16(b[off:])), 2
	case 3:
		if off+4 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("int(%d)", int32(binary.LittleEndian.Uint32(b[off:]))), 4
	case 4:
		if off+8 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("i64(%d)", int64(binary.LittleEndian.Uint64(b[off:]))), 8
	case 5:
		if off+4 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("f32(%v)", math.Float32frombits(binary.LittleEndian.Uint32(b[off:]))), 4
	case 6:
		if off+8 > len(b) {
			return "<short>", 0
		}
		return fmt.Sprintf("f64(%v)", math.Float64frombits(binary.LittleEndian.Uint64(b[off:]))), 8
	case 7:
		if off+4 > len(b) {
			return "<short>", 0
		}
		ln := int(binary.LittleEndian.Uint32(b[off:]))
		if ln < 1 || off+4+ln > len(b) {
			return fmt.Sprintf("<bad str len %d>", ln), 0
		}
		return fmt.Sprintf("str(%q)", string(b[off+4:off+4+ln-1])), 4 + ln
	case 8:
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
	case 9:
		if off+8 > len(b) {
			return "<short>", 0
		}
		v := binary.LittleEndian.Uint64(b[off:])
		return fmt.Sprintf("obj(%08x/%08x)", uint32(v), uint32(v>>32)), 8
	}
	return fmt.Sprintf("<code%d raw % x>", wt, b[off:min(off+8, len(b))]), min(8, len(b)-off)
}

// decodeRecordFrame 记录族帧按官方 0x0A schema 解码（entergen frames.go
// 行加/删行/逐格/挂载四种布局的逆）：
//
//	0x11 [obj][own][rec u16][行 u16][行数 u16]{行内各列按列型}
//	0x12 [obj][own][rec u16][行 u16]
//	0x13 [obj][own][rec u16][格数 u16][行 u16]{[列 u8][按列型]}
//	0x14 [sub][obj][own][rec u16]
//
// 列值借用 decodeValue（线型码同源）。非记录族/畸形帧返回 false。
func decodeRecordFrame(b []byte) (string, bool) {
	if len(b) < 14 {
		return "", false
	}
	obj, own := binary.LittleEndian.Uint32(b[2:]), binary.LittleEndian.Uint32(b[6:])
	rec := int(binary.LittleEndian.Uint16(b[10:]))
	schema, known := clientdata.RecordSchemaByIndex(rec)
	name := fmt.Sprintf("rec%d", rec)
	if known {
		name = schema.Name
	}
	head := fmt.Sprintf("obj=%08x/%04x %s", obj, own, name)
	cell := func(o int, colType int) (string, int) {
		if colType == 0 || o >= len(b) {
			return "<short>", 0
		}
		return decodeValue(b, o, colType)
	}
	switch b[0] {
	case 0x11:
		if len(b) < 16 || !known {
			return "", false
		}
		row, rows := int(binary.LittleEndian.Uint16(b[12:])), int(binary.LittleEndian.Uint16(b[14:]))
		out := ""
		o := 16
		for r := 0; r < rows; r++ {
			out += fmt.Sprintf(" row=%d{", row+r)
			for col, ct := range schema.ColumnTypes {
				val, width := cell(o, int(ct))
				if width == 0 {
					return head + out + " <short>}", true
				}
				o += width
				out += fmt.Sprintf(" c%d=%s", col, val)
			}
			out += "}"
		}
		return head + fmt.Sprintf(" add%s", out), true
	case 0x12:
		return fmt.Sprintf("%s del row=%d", head, binary.LittleEndian.Uint16(b[12:])), true
	case 0x13:
		if len(b) < 16 || !known {
			return "", false
		}
		count, row := int(binary.LittleEndian.Uint16(b[12:])), int(binary.LittleEndian.Uint16(b[14:]))
		out := ""
		o := 16
		for i := 0; i < count; i++ {
			if o >= len(b) {
				break
			}
			col := int(b[o])
			o++
			ct := 0
			if col < len(schema.ColumnTypes) {
				ct = int(schema.ColumnTypes[col])
			}
			val, width := cell(o, ct)
			if width == 0 {
				out += fmt.Sprintf(" c%d=<short>", col)
				break
			}
			o += width
			out += fmt.Sprintf(" c%d=%s", col, val)
		}
		return fmt.Sprintf("%s grid row=%d%s", head, row, out), true
	case 0x14:
		if len(b) < 12 {
			return "", false
		}
		return fmt.Sprintf("sub=%d obj=%08x/%04x %s mount", b[1], obj, own, name), true
	}
	return "", false
}

func valStr(v entergen.CustomInboundValue) string {
	switch v.Type {
	case 2:
		return fmt.Sprintf("i32(%d)", v.Int32)
	case 3:
		return fmt.Sprintf("i64(%d)", v.Int64)
	case 4:
		return fmt.Sprintf("f32(%v)", v.Float32)
	case 5:
		return fmt.Sprintf("f64(%v)", v.Float64)
	case 6, 7:
		return fmt.Sprintf("str(%q)", v.Text)
	default:
		return fmt.Sprintf("raw(% x)", v.Raw)
	}
}

type propEntry struct {
	name string
	code int
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
