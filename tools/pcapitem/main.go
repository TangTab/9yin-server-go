// pcapitem 抓包物品使用链（msg31）逐帧取证：pcapng → <port> TCP 双向重组
// （带每字节到达时刻）→ EE+XOR 帧解码 → 以每条 C2S msg31 为锚的时间窗，
// 窗内帧全解码：
//
//	0x10   属性帧（按 S2C 0x09 属性表解名；视图对象位一并标出）
//	0x18   视图行新增（视图号/行号/行属性——背包行即在此看 ConfigID）
//	0x19   视图行删除（视图号/行号）
//	0x11 / 0x13  记录行新增/改格（记录号按官方 schema 解名，格值给 hex）
//	0x1E / 0x0A  自定义消息值表（0x0A 活动信封）
//	其余   hex（截 64B）
//
// 用法: pcapitem <file.pcapng> [port=8032] [before=1.5] [after=3.0] [msgid=31]
//
//	-time0 <t> -time1 <t>   不用消息锚，直接输出 [t0,t1] 全窗（可省 before/after）。
//
// 心跳噪声（C2S 0x18 应答、周期 m924/m294 对）不计入窗——它们不是使用链的一环。
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	net "github.com/lizhanyin/9yin-server-go/internal/net"
)

// msgHeartbeat 上行心跳（与配对的下行 m294 一并滤除）。
const msgHeartbeat = 924

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

type propEntry struct {
	name string
	code int
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pcapitem <file.pcapng> [port] [before] [after] [msgid]")
		fmt.Fprintln(os.Stderr, "       pcapitem <file.pcapng> -time0 <t0> -time1 <t1> [port]")
		os.Exit(2)
	}
	data, err := res.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	port, before, after, msgID := 8032, 1.5, 3.0, 31
	time0, time1, windowed := -1.0, -1.0, false
	positional := []float64{}
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-time0":
			i++
			time0, windowed = mustFloat(arg(i)), true
		case "-time1":
			i++
			time1, windowed = mustFloat(arg(i)), true
		case "-port":
			i++
			port = mustInt(arg(i))
		default:
			positional = append(positional, mustFloat(os.Args[i]))
		}
	}
	// 位置参数按「给出的个数」解释：3 个=port/before/after（其前可带 msgid
	// 的第 4 个），2 个=before/after，1 个=after。
	switch len(positional) {
	case 4:
		msgID = int(positional[3])
		fallthrough
	case 3:
		port, before, after = int(positional[0]), positional[1], positional[2]
	case 2:
		before, after = positional[0], positional[1]
	case 1:
		after = positional[0]
	case 0:
	default:
		fmt.Fprintln(os.Stderr, "too many positional args")
		os.Exit(2)
	}

	events := readEvents(data, uint16(port))
	entries := propTable(events)
	fmt.Fprintf(os.Stderr, "events=%d prop-entries=%d\n", len(events), len(entries))

	if windowed {
		dumpWindow(events, entries, time0, time1, nil)
		return
	}
	anchors := 0
	for i := range events {
		if events[i].dir != 0 || messageID(events[i].frame) != msgID {
			continue
		}
		anchors++
		anchor := events[i]
		fmt.Printf("===== #%d m%d @ t=%.3f =====\n", anchors, msgID, anchor.ts)
		dumpWindow(events, entries, anchor.ts-before, anchor.ts+after, &anchor)
	}
	fmt.Fprintf(os.Stderr, "anchors m%d=%d\n", msgID, anchors)
}

// messageID 帧首值自定义消息号（两种载体 0x1E/0x0A；非自定义消息 -1）。
func messageID(frame []byte) int {
	if m, err := entergen.ParseCustomMessage(frame); err == nil {
		return customHead(m)
	}
	if m, ok, err := entergen.ParseActivityCustomMessage(frame); ok && err == nil {
		return customHead(m)
	}
	return -1
}

func customHead(m entergen.CustomInbound) int {
	if len(m.Values) == 0 || m.Values[0].Type != 2 {
		return -1
	}
	return int(m.Values[0].Int32)
}

// dumpWindow 时间窗内全部帧逐行解码；anchor 非空时相对时刻按锚计。
func dumpWindow(events []event, entries []propEntry, t0, t1 float64, anchor *event) {
	base := t0
	if anchor != nil {
		base = anchor.ts
	}
	for _, ev := range events {
		if ev.ts < t0 || ev.ts > t1 || noise(ev) {
			continue
		}
		d := "C2S"
		if ev.dir == 1 {
			d = "S2C"
		}
		fmt.Printf("%+8.3f %s #%-5d %s\n", ev.ts-base, d, ev.idx, describe(ev, entries))
	}
}

// noise 心跳噪声：C2S 0x18 应答、m924 上行与配对的 m294 下行。
func noise(ev event) bool {
	if ev.dir == 0 && ev.op == 0x18 {
		return true
	}
	return ev.op == 0x0A && messageID(ev.frame) == msgHeartbeat ||
		ev.op == 0x0A && messageID(ev.frame) == 294
}

// describe 单帧可读形。
func describe(ev event, entries []propEntry) string {
	switch ev.op {
	case 0x10:
		return describeProps(ev.frame, entries)
	case 0x18:
		return describeViewAdd(ev.frame, entries)
	case 0x19:
		if len(ev.frame) >= 5 {
			return fmt.Sprintf("0x19 ViewRemove view=%d idx=%d",
				binary.LittleEndian.Uint16(ev.frame[1:]), binary.LittleEndian.Uint16(ev.frame[3:]))
		}
	case 0x11, 0x13:
		if len(ev.frame) >= 16 {
			return describeRecord(ev.frame)
		}
	case 0x1E, 0x27:
		if m, err := entergen.ParseCustomMessage(ev.frame); err == nil {
			return customLine("custom", m)
		}
	case 0x0A:
		if m, ok, err := entergen.ParseActivityCustomMessage(ev.frame); ok && err == nil {
			return customLine("actenv", m)
		}
	}
	return hexLine(ev.frame)
}

func describeProps(frame []byte, entries []propEntry) string {
	view := frame[1] != 0
	obj := binary.LittleEndian.Uint32(frame[2:])
	own := binary.LittleEndian.Uint32(frame[6:])
	head := fmt.Sprintf("0x10 obj=%08x own=%08x", obj, own)
	if view {
		head = fmt.Sprintf("0x10 VIEWOBJ view=%d idx=%d", obj, own)
	}
	return head + ":" + propList(frame, 12, int(binary.LittleEndian.Uint16(frame[10:])), entries)
}

func describeViewAdd(frame []byte, entries []propEntry) string {
	view := binary.LittleEndian.Uint16(frame[1:])
	row := binary.LittleEndian.Uint16(frame[3:])
	n := int(binary.LittleEndian.Uint16(frame[5:]))
	return fmt.Sprintf("0x18 ViewAdd view=%d idx=%d n=%d:%s", view, row, n, propList(frame, 7, n, entries))
}

// propList 「(idx u16, 值)」序列解名（值线型取属性表；表外给 idx 名）。
func propList(frame []byte, off, n int, entries []propEntry) string {
	var sb strings.Builder
	for j := 0; j < n && off < len(frame); j++ {
		if off+2 > len(frame) {
			break
		}
		idx := int(binary.LittleEndian.Uint16(frame[off:]))
		off += 2
		name, wt := fmt.Sprintf("idx%d", idx), 0
		if idx < len(entries) {
			name, wt = entries[idx].name, entries[idx].code
		}
		val, width := decodeValue(frame, off, wt)
		if width == 0 {
			fmt.Fprintf(&sb, " %s=%s", name, val)
			break
		}
		off += width
		fmt.Fprintf(&sb, " %s=%s", name, val)
	}
	return sb.String()
}

// describeRecord 记录行帧（0x11 整行 / 0x13 改格）：记录号按官方 schema
// 解名，格值按列线型解（schema 缺席给 hex）。
func describeRecord(frame []byte) string {
	rec := binary.LittleEndian.Uint16(frame[10:])
	row := binary.LittleEndian.Uint16(frame[12:])
	head := fmt.Sprintf("0x%02X rec=%d", frame[0], rec)
	name, ok := clientdata.RecordSchemaByIndex(int(rec))
	if !ok {
		return head + " (schema 缺席) " + hexTail(frame, 16)
	}
	obj := binary.LittleEndian.Uint32(frame[2:])
	own := binary.LittleEndian.Uint32(frame[6:])
	if frame[0] == 0x13 {
		n := int(binary.LittleEndian.Uint16(frame[12:])) // 0x13：u16 格数；行号在 +14
		row = binary.LittleEndian.Uint16(frame[14:])
		var sb strings.Builder
		off := 16
		for j := 0; j < n && off < len(frame); j++ {
			col := int(frame[off])
			off++
			if col >= len(name.ColumnTypes) {
				break
			}
			val, width := decodeValue(frame, off, int(name.ColumnTypes[col]))
			if width == 0 {
				break
			}
			off += width
			fmt.Fprintf(&sb, " [r%d c%d]=%s", row, col, val)
		}
		return fmt.Sprintf("0x13 GRID obj=%08x own=%08x rec=%d(%s)%s", obj, own, rec, name.Name, sb.String())
	}
	// 0x11：[起始行 u16][行数 u16=1][行内各列]
	count := int(binary.LittleEndian.Uint16(frame[14:]))
	var sb strings.Builder
	off := 16
	for col := 0; col < len(name.ColumnTypes) && off < len(frame); col++ {
		val, width := decodeValue(frame, off, int(name.ColumnTypes[col]))
		if width == 0 {
			break
		}
		off += width
		fmt.Fprintf(&sb, " [c%d]=%s", col, val)
	}
	return fmt.Sprintf("0x11 ROW obj=%08x own=%08x rec=%d(%s) row=%d rows=%d:%s",
		obj, own, rec, name.Name, row, count, sb.String())
}

func hexLine(frame []byte) string {
	if len(frame) == 0 {
		return "<empty>"
	}
	return fmt.Sprintf("0x%02X hex=%s", frame[0], hexTail(frame, 0))
}

// hexTail hex 自 off 起（截 64B，超出标注总长）。
func hexTail(frame []byte, off int) string {
	limit := len(frame)
	if limit > off+64 {
		limit = off + 64
	}
	suffix := ""
	if len(frame) > limit {
		suffix = fmt.Sprintf(" …(%dB)", len(frame))
	}
	return fmt.Sprintf("% x%s", frame[off:limit], suffix)
}

func customLine(kind string, m entergen.CustomInbound) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s m%d", kind, customHead(m))
	for _, v := range m.Values[1:] {
		sb.WriteString(" " + valStr(v))
	}
	return sb.String()
}

// propTable S2C 0x09 属性表（取最后一次完整表）。
func propTable(events []event) []propEntry {
	var entries []propEntry
	for _, ev := range events {
		if ev.dir != 1 || len(ev.frame) < 3 || ev.frame[0] != 0x09 {
			continue
		}
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
	return entries
}

// readEvents pcapng → 帧事件流（按时间排序；dir 0=C2S 1=S2C）。序号空洞
// 填零字节续解（带外标记不致整流硬停，见 pcapskill 头注）。
func readEvents(data []byte, port uint16) []event {
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
			if len(body) >= 2 {
				linkType = int(binary.LittleEndian.Uint16(body))
			}
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
		case 0x00000006: // EPB
			if len(body) < 20 {
				break
			}
			ts := (float64(binary.LittleEndian.Uint32(body[4:]))*(1<<32) +
				float64(binary.LittleEndian.Uint32(body[8:]))) / tsres
			capLen := int(binary.LittleEndian.Uint32(body[12:]))
			if 20+capLen > len(body) {
				break
			}
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
					dir, cport := -1, uint16(0)
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
							streams[streamKey{cport, dir}] = append(streams[streamKey{cport, dir}],
								segRun{binary.BigEndian.Uint32(tcp[4:]), ts - t0, cp})
						}
					}
				}
			}
		}
		off += blkLen
	}

	var events []event
	for sk, segs := range streams {
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
					if skip := int(-dd); skip >= len(d) {
						continue
					} else {
						d = d[skip:]
					}
				} else {
					for i := int32(0); i < dd; i++ {
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
			end := pos + wireLen - 1
			ts := 0.0
			if end >= 0 && end < len(tsArr) {
				ts = tsArr[end]
			} else if len(tsArr) > 0 {
				ts = tsArr[len(tsArr)-1]
			}
			plain := cipher.Decode(wire)
			if len(plain) == 0 {
				pos += wireLen
				continue
			}
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
	return events
}

// decodeValue 按属性表线型解一个值（返回可读形与字节数；0=截断/非法）。
func decodeValue(b []byte, off, wt int) (string, int) {
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
			if u := binary.LittleEndian.Uint16(b[p:]); u == 0 {
				break
			} else {
				runes = append(runes, rune(u))
			}
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
	if i < len(os.Args) {
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
