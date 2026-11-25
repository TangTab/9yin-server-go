// pcapevents（原 m54scan）带时间轴的抓包事件分析器：
// pcapng → 8032 端口 TCP 双向重组（带每字节到达时间戳）→ EE+XOR 帧解码 →
// C2S 0x0A 自定义消息（m54 格挡为主）/ S2C 0x09 表 + 0x10 属性帧事件流。
// 属性名表取自包内登录阶段的 S2C 0x09；中途开抓的包没有这一帧时回退仓库
// 内嵌的官服 2433 属性表（internal/clientdata）。
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
	"github.com/lizhanyin/9yin-server-go/internal/fssolorobot"
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
		// 带外标记：干净 EE EE 终止符后插入 7 字节 `00×6 EE`（官服女号抓包
		// t=522.8 实证一处，详见 docs/official/初始女号抓包m54手动格挡全解.md
		// §六）。标准帧器读到标记尾部的非法转义对会硬停，其后整段流丢失
		//（女号抓包 600~624s 的战斗段此前因此未解出）。分析工具侧剥离标记，
		// 仓库 net 包的 EscapeFramer 保持协议忠实不改。
		if removed := stripOutOfBandMarkers(stream, tsArr); removed > 0 {
			fmt.Fprintf(os.Stderr, "dir=%d stripped %d out-of-band marker(s)\n", dir, removed)
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
	// 中途开抓的包没有登录阶段的 0x09（属性帧只能解到下标）：回退仓库内嵌的
	// 官服 2433 属性表（clientdata，与服务端下发 0x09 同一张）。
	source := "capture 0x09"
	if len(entries) == 0 {
		official := clientdata.OfficialPropertyEntries()
		max := 0
		for _, pe := range official {
			if int(pe.Index) > max {
				max = int(pe.Index)
			}
		}
		entries = make([]propEntry, max+1)
		for i := range entries {
			entries[i] = propEntry{fmt.Sprintf("idx%d", i), 0}
		}
		for _, pe := range official {
			entries[pe.Index] = propEntry{pe.Name, pe.TypeCode}
		}
		source = "embedded official 2433"
	}
	fmt.Fprintf(os.Stderr, "prop table: %d entries (%s)\n", len(entries), source)

	// dumpall 模式：全部事件打一行（S2C 0x10 解属性、0x1E 打原始 hex 头 48B）。
	// dumprec 同款，记录族帧另按 schema 解码（见 decodeRecordFrame）。
	mode := ""
	if len(os.Args) > 3 {
		mode = os.Args[3]
	}
	hexMax := 0
	if v, e := strconv.Atoi(os.Getenv("NINEYIN_PCAP_HEX")); e == nil {
		hexMax = v
	}
	if mode == "dumpall" || mode == "dumprec" {
		for _, ev := range events {
			d := "C2S"
			if ev.dir == 1 {
				d = "S2C"
			}
			switch {
			case ev.op == 0x10 && len(ev.frame) >= 12:
				ps, _ := propList(ev.frame, 12, int(binary.LittleEndian.Uint16(ev.frame[10:])), entries)
				fmt.Printf("A t=%9.3f %s #%d 0x10 obj=%08x:%s\n", ev.ts, d, ev.idx, binary.LittleEndian.Uint32(ev.frame[2:]), ps)
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
			case ev.op == 0x2C:
				// 压缩批量位置帧（mode-0x47 QuickLZ）：明文体 [u16 条数][条数×44B：
				// obj/own/x/y/z/朝向/…]。旁观者视角下他人位置（含受控位移）的来源。
				if body := unwrap47(ev.frame); len(body) >= 2 {
					cnt := int(binary.LittleEndian.Uint16(body))
					es := ""
					for j := 0; j < cnt && 2+(j+1)*44 <= len(body); j++ {
						b := body[2+j*44:]
						es += fmt.Sprintf(" {%08x %.2f,%.2f,%.2f o=%.2f}", binary.LittleEndian.Uint32(b),
							math.Float32frombits(binary.LittleEndian.Uint32(b[8:])), math.Float32frombits(binary.LittleEndian.Uint32(b[12:])),
							math.Float32frombits(binary.LittleEndian.Uint32(b[16:])), math.Float32frombits(binary.LittleEndian.Uint32(b[20:])))
					}
					line := fmt.Sprintf("A t=%9.3f %s #%d op=0x2C len=%d n=%d%s", ev.ts, d, ev.idx, len(ev.frame), cnt, es)
					if hexMax > 0 { // 明文体头（条目尾部字段取证用）
						line += fmt.Sprintf(" phex=% x", body[:min(hexMax, len(body))])
					}
					fmt.Println(line)
					continue
				}
				printOther(ev, d, mode, hexMax)
			case ev.op == 0x2D && len(ev.frame) >= 3:
				// 多对象属性帧：u16 对象数 + 每对象 {obj, own, u16 属性数, {下标 u16, 值}…}
				//（值编码同 0x10）。旁观者视角下他人血量/状态等属性的来源之一。
				k := int(binary.LittleEndian.Uint16(ev.frame[1:]))
				o, ps := 3, ""
				for j := 0; j < k && o+10 <= len(ev.frame); j++ {
					obj := binary.LittleEndian.Uint32(ev.frame[o:])
					o += 8
					n := int(binary.LittleEndian.Uint16(ev.frame[o:]))
					o += 2
					one, w := propList(ev.frame, o, n, entries)
					if w == 0 && n > 0 {
						break
					}
					o += w
					ps += fmt.Sprintf(" {%08x%s}", obj, one)
				}
				fmt.Printf("A t=%9.3f %s #%d 0x2D k=%d%s\n", ev.ts, d, ev.idx, k, ps)
			case ev.op == 0x21 && len(ev.frame) >= 3:
				// 批量移动广播（entergen.ServerAllDestFrame 布局：u16 条数 +
				// 条数×44B：obj/own/目的地 xyz/朝向/速度/回显槽×3/移动标志）。
				n := int(binary.LittleEndian.Uint16(ev.frame[1:]))
				es := ""
				for j := 0; j < n && 3+(j+1)*44 <= len(ev.frame); j++ {
					b := ev.frame[3+j*44:]
					f := func(o int) float64 {
						return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[o:])))
					}
					es += fmt.Sprintf(" {%08x dst=%.2f,%.2f,%.2f o=%.2f spd=%.2f rot=%.2f m2=%.2f m3=%.2f f=%d}",
						binary.LittleEndian.Uint32(b), f(8), f(12), f(16), f(20), f(24), f(28), f(32), f(36),
						binary.LittleEndian.Uint32(b[40:]))
				}
				line := fmt.Sprintf("A t=%9.3f %s #%d op=0x21 len=%d n=%d%s", ev.ts, d, ev.idx, len(ev.frame), n, es)
				if hexMax > 0 { // 原始字节头（与服务端编码器逐字节对拍用）
					line += fmt.Sprintf(" hex=% x", ev.frame[:min(hexMax, len(ev.frame))])
				}
				fmt.Println(line)
			case ev.op == 0x1F && len(ev.frame) >= 25:
				// 位置帧（entergen.LocationFrame 布局；受控位移/腾空取证用，
				// 旧 token `op=0x1F len=` 保留，字段追加在行尾）。
				fmt.Printf("A t=%9.3f %s #%d op=0x1F len=%d obj=%08x own=%08x x=%v y=%v z=%v orient=%v\n", ev.ts, d, ev.idx, len(ev.frame),
					binary.LittleEndian.Uint32(ev.frame[1:]), binary.LittleEndian.Uint32(ev.frame[5:]),
					math.Float32frombits(binary.LittleEndian.Uint32(ev.frame[9:])), math.Float32frombits(binary.LittleEndian.Uint32(ev.frame[13:])),
					math.Float32frombits(binary.LittleEndian.Uint32(ev.frame[17:])), math.Float32frombits(binary.LittleEndian.Uint32(ev.frame[21:])))
			default:
				printOther(ev, d, mode, hexMax)
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

// stripOutOfBandMarkers 原地剥离流中的带外标记并保持逐字节时间戳对齐，
// 返回剥离数。标记形：`EE EE | 00 00 00 00 00 00 EE | <下一帧首字节≠00/EE>`
// ——只删中间 7 字节（终止符与下一帧首字节保留）。合法帧不会命中：帧内
// EE 必以 `EE 00`（字面量）或 `EE EE`（终止符）续接，标记尾部是单个 EE
// 后跟普通字节。
func stripOutOfBandMarkers(stream []byte, ts []float64) int {
	const markerLen = 7
	removed := 0
	for i := 0; i+markerLen+2 < len(stream); i++ {
		if stream[i] != 0xEE || stream[i+1] != 0xEE {
			continue
		}
		zeros := true
		for k := 2; k < markerLen+1; k++ {
			if stream[i+k] != 0 {
				zeros = false
				break
			}
		}
		if !zeros || stream[i+markerLen+1] != 0xEE {
			continue
		}
		next := stream[i+markerLen+2]
		if next == 0x00 || next == 0xEE {
			continue
		}
		at := i + 2
		stream = append(stream[:at], stream[at+markerLen:]...)
		if ts != nil {
			ts = append(ts[:at], ts[at+markerLen:]...)
		}
		removed++
		i = at // 从下一帧首字节继续扫
	}
	return removed
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

// printOther 是未专门解码的帧的一行输出：dumprec 模式下记录族帧先按 schema
// 解（解得出即替代普通行）；其余打 `op len`，按开关追加原始 hex 头与
// mode-0x47 包装帧的明文体长度。
func printOther(ev event, d, mode string, hexMax int) {
	if mode == "dumprec" && ev.dir == 1 {
		if desc, ok := decodeRecordFrame(ev.frame); ok {
			fmt.Printf("A t=%9.3f %s #%d rec op=0x%02X len=%d %s\n", ev.ts, d, ev.idx, ev.op, len(ev.frame), desc)
			return
		}
	}
	line := fmt.Sprintf("A t=%9.3f %s #%d op=0x%02X len=%d", ev.ts, d, ev.idx, ev.op, len(ev.frame))
	// NINEYIN_PCAP_HEX=<字节数>：未解码帧追加原始 hex 头（缺省不开，行格式不变）。
	if hexMax > 0 && len(ev.frame) > 0 {
		line += fmt.Sprintf(" hex=% x", ev.frame[:min(hexMax, len(ev.frame))])
	}
	if body := unwrap47(ev.frame); body != nil {
		line += fmt.Sprintf(" plain=%d", len(body))
		if hexMax > 0 && len(body) > 0 {
			line += fmt.Sprintf(" phex=% x", body[:min(hexMax, len(body))])
		}
	}
	fmt.Println(line)
}

// unwrap47 解 mode-0x47 QuickLZ 包装帧（0x28/0x29/0x2C/0x2E 族），返回明文体；
// 非该包装或解压失败返回 nil（解压器是 fssolorobot 里官方 qlz_decompress 的复刻）。
func unwrap47(frame []byte) []byte {
	if len(frame) < 10 || frame[1] != 0x47 {
		return nil
	}
	d, err := fssolorobot.QLZSizeDecompressed(frame[1:])
	if err != nil || d < 0 || d > 1<<20 {
		return nil
	}
	body := make([]byte, d)
	n, err := fssolorobot.QLZDecompress(frame[1:], body)
	if err != nil || n != d {
		return nil
	}
	return body
}

// propList 解属性帧体 {下标 u16, 值}×n（o 为体的起始偏移；0x10 头 12B、
// 0x2D 每对象头 10B），返回 ` 名=值` 串与消耗的字节数，线型按属性表。
func propList(frame []byte, o, n int, entries []propEntry) (string, int) {
	ps, start := "", o
	for j := 0; j < n && o < len(frame); j++ {
		idx := int(binary.LittleEndian.Uint16(frame[o:]))
		o += 2
		name := fmt.Sprintf("idx%d", idx)
		wt := 0
		if idx < len(entries) {
			name, wt = entries[idx].name, entries[idx].code
		}
		val, width := decodeValue(frame, o, wt)
		o += width
		ps += fmt.Sprintf(" %s=%s", name, val)
		if width == 0 {
			break
		}
	}
	return ps, o - start
}
