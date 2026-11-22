// pcapsplit 从 pcapng 抓包中按服务器端口提取 TCP 会话的 c2s/s2c 载荷流
//（EE+XOR wire 原样字节——可直接进 tools/decode_capture / propanalyze）。
//
// 用法：go run ./tools/pcapsplit <file.pcapng> <server-port> <out-prefix>
// 产物：<out-prefix>_c2s.bin / <out-prefix>_s2c.bin（按 seq 去重排序重组）。
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

type seg struct {
	seq  uint32
	data []byte
}

var serverPort uint16

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: pcapsplit <file.pcapng> <server-port> <out-prefix>")
		os.Exit(2)
	}
	serverPort = uint16(mustAtoi(os.Args[2]))
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	linkType := -1
	// 每 stream：dir(0=c2s,1=s2c) → segs；key = client port（非服务端口侧）。
	type streamKey struct {
		clientPort uint16
		dir        int
	}
	streams := map[streamKey][]seg{}
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
		case 0x00000006: // EPB
			if len(body) < 20 {
				break
			}
			capLen := int(binary.LittleEndian.Uint32(body[12:]))
			pkt := body[20 : 20+capLen]
			dir, cport, seq, payload := parsePacket(pkt, linkType)
			if dir < 0 {
				continue
			}
			cp := make([]byte, len(payload))
			copy(cp, payload)
			k := streamKey{cport, dir}
			streams[k] = append(streams[k], seg{seq, cp})
		}
		off += blkLen
	}
	var c2sOut, s2cOut []byte
	keys := make([]streamKey, 0, len(streams))
	for k := range streams {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].dir != keys[j].dir {
			return keys[i].dir < keys[j].dir
		}
		return keys[i].clientPort < keys[j].clientPort
	})
	for _, k := range keys {
		segs := streams[k]
		sort.Slice(segs, func(i, j int) bool { return lessSeq(segs[i].seq, segs[j].seq) })
		var reassembled []byte
		var expect uint32
		first := true
		for _, s := range segs {
			if first {
				expect, first = s.seq, false
			} else if s.seq != expect {
				if seqDelta(s.seq, expect) < 0 { // 旧重传覆盖
					skip := int(seqDelta(expect, s.seq))
					if skip >= len(s.data) {
						continue
					}
					s.data = s.data[skip:]
				} else {
					fmt.Fprintf(os.Stderr, "warn: stream cport=%d dir=%d gap at seq %d (expect %d)\n",
						k.clientPort, k.dir, s.seq, expect)
				}
			}
			reassembled = append(reassembled, s.data...)
			expect += uint32(len(s.data))
		}
		label := "s2c"
		if k.dir == 0 {
			label, c2sOut = "c2s", append(c2sOut, reassembled...)
		} else {
			s2cOut = append(s2cOut, reassembled...)
		}
		fmt.Fprintf(os.Stderr, "stream cport=%d dir=%s segs=%d bytes=%d\n", k.clientPort, label, len(segs), len(reassembled))
	}
	fmt.Fprintf(os.Stderr, "filter server-port=%d; total c2s=%d s2c=%d\n", serverPort, len(c2sOut), len(s2cOut))
	mustWrite(os.Args[3]+"_c2s.bin", c2sOut)
	mustWrite(os.Args[3]+"_s2c.bin", s2cOut)
}

// parsePacket 以太网(1)/RawIP(101) 下解 IPv4/TCP；dir: 0=c2s 1=s2c -1=无关。
func parsePacket(pkt []byte, linkType int) (dir int, clientPort uint16, seq uint32, payload []byte) {
	var ip []byte
	switch linkType {
	case 1: // Ethernet
		if len(pkt) < 14 || binary.BigEndian.Uint16(pkt[12:]) != 0x0800 {
			return -1, 0, 0, nil
		}
		ip = pkt[14:]
	case 101: // Raw IP
		ip = pkt
	default:
		return -1, 0, 0, nil
	}
	if len(ip) < 20 || ip[0]>>4 != 4 || ip[9] != 6 {
		return -1, 0, 0, nil
	}
	ihl := int(ip[0]&0xf) * 4
	if len(ip) < ihl+20 {
		return -1, 0, 0, nil
	}
	tcp := ip[ihl:]
	srcPort := binary.BigEndian.Uint16(tcp[0:])
	dstPort := binary.BigEndian.Uint16(tcp[2:])
	switch {
	case dstPort == serverPort:
		dir, clientPort = 0, srcPort
	case srcPort == serverPort:
		dir, clientPort = 1, dstPort
	default:
		return -1, 0, 0, nil
	}
	seq = binary.BigEndian.Uint32(tcp[4:])
	doff := int(tcp[12]>>4) * 4
	if doff < 20 || len(tcp) < doff {
		return -1, 0, 0, nil
	}
	return dir, clientPort, seq, tcp[doff:]
}

func lessSeq(a, b uint32) bool { return seqDelta(a, b) < 0 }

// seqDelta 有符号距离 a-b（环形）。
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

func mustAtoi(s string) int {
	v := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			fmt.Fprintln(os.Stderr, "bad port:", s)
			os.Exit(2)
		}
		v = v*10 + int(c-'0')
	}
	return v
}

func mustWrite(path string, b []byte) {
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
