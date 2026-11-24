// attrdump-tmp：0x10 属性帧全量走查（一次性诊断）——按 obj 段分类，
// 逐帧解析 [flags][obj][own][count]{idx,值}（值按 0x09 表线型）。
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
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

func main() {
	key := fsnet.NewKey(fsnet.InitialKeyValue)
	if len(os.Args) > 2 {
		v, _ := strconv.ParseUint(os.Args[2], 16, 64)
		key = fsnet.NewKey(uint32(v))
	}
	data, err := res.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	f, err := fsnet.NewEscapeFramer(newReader(data), fsnet.DefaultMaxFrameSize)
	if err != nil {
		panic(err)
	}
	cipher := fsnet.NewXorCipher(key)

	type frameInfo struct {
		obj, own uint32
		props    string
	}
	bySegment := map[byte][]frameInfo{}
	for i := 0; ; i++ {
		wire, err := f.ReadFrame()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		plain := cipher.Decode(wire)
		if len(plain) < 12 || plain[0] != 0x10 {
			continue
		}
		obj := binary.LittleEndian.Uint32(plain[2:6])
		own := binary.LittleEndian.Uint32(plain[6:10])
		count := int(binary.LittleEndian.Uint16(plain[10:12]))
		var props []string
		off := 12
		for j := 0; j < count && off < len(plain); j++ {
			idx := binary.LittleEndian.Uint16(plain[off:])
			off += 2
			entry, ok := clientdata.PropertyByIndex(int(idx))
			if !ok {
				props = append(props, fmt.Sprintf("%d:?", idx))
				off += 4
				continue
			}
			val := ""
			switch entry.Type {
			case clientdata.WireByte:
				val = fmt.Sprintf("b%d", plain[off])
				off++
			case clientdata.WireWord:
				val = fmt.Sprintf("w%d", binary.LittleEndian.Uint16(plain[off:]))
				off += 2
			case clientdata.WireInt32:
				val = fmt.Sprintf("i%d", int32(binary.LittleEndian.Uint32(plain[off:])))
				off += 4
			case clientdata.WireFloat32:
				val = fmt.Sprintf("f%g", math.Float32frombits(binary.LittleEndian.Uint32(plain[off:])))
				off += 4
			case clientdata.WireInt64, clientdata.WireObject:
				val = fmt.Sprintf("q%x", binary.LittleEndian.Uint64(plain[off:]))
				off += 8
			case clientdata.WireString, clientdata.WireWideString:
				l := int(binary.LittleEndian.Uint32(plain[off:]))
				off += 4
				raw := plain[off : off+l]
				if entry.Type == clientdata.WireWideString {
					units := make([]uint16, 0, l/2)
					for k := 0; k+1 < l; k += 2 {
						units = append(units, binary.LittleEndian.Uint16(raw[k:]))
					}
					val = "w:" + decodeUnits(units)
				} else {
					val = "s:" + strings.TrimRight(string(raw), "\x00")
				}
				off += l
			default:
				val = "raw"
				off += 4
			}
			props = append(props, fmt.Sprintf("%s=%s", entry.Name, val))
		}
		seg := byte(obj >> 24)
		bySegment[seg] = append(bySegment[seg], frameInfo{obj, own, strings.Join(props, " ")})
	}
	segs := make([]byte, 0)
	for s := range bySegment {
		segs = append(segs, s)
	}
	sort.Slice(segs, func(i, j int) bool { return len(bySegment[segs[i]]) > len(bySegment[segs[j]]) })
	for _, s := range segs {
		frames := bySegment[s]
		fmt.Printf("== obj 段 %#x：%d 帧\n", s, len(frames))
		// 属性频次
		freq := map[string]int{}
		for _, fr := range frames {
			for _, p := range strings.Split(fr.props, " ") {
				if p != "" {
					freq[strings.Split(p, "=")[0]]++
				}
			}
		}
		names := make([]string, 0)
		for n := range freq {
			names = append(names, n)
		}
		sort.Slice(names, func(i, j int) bool { return freq[names[i]] > freq[names[j]] })
		line := "属性频次: "
		for k, n := range names {
			if k >= 18 {
				break
			}
			line += fmt.Sprintf("%s×%d ", n, freq[n])
		}
		fmt.Println(line)
		// 唯一对象数与样例帧
		objs := map[uint32]bool{}
		for _, fr := range frames {
			objs[fr.obj] = true
		}
		fmt.Printf("唯一对象 %d 个；样例 3 帧：\n", len(objs))
		for k, fr := range frames {
			if k >= 3 {
				break
			}
			fmt.Printf("  obj=%#x own=%#x %s\n", fr.obj, fr.own, fr.props)
		}
	}
}

func decodeUnits(units []uint16) string {
	rs := make([]rune, 0, len(units))
	for _, u := range units {
		if u == 0 {
			break
		}
		rs = append(rs, rune(u))
	}
	return string(rs)
}

type byteReader struct {
	b   []byte
	off int
}

func newReader(b []byte) *byteReader              { return &byteReader{b: b} }
func (r *byteReader) Write(p []byte) (int, error) { return len(p), nil }
func (r *byteReader) Read(p []byte) (int, error) {
	if r.off >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.off:])
	r.off += n
	return n, nil
}
