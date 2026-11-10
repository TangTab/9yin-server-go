package main

import (
	"encoding/binary"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

type entryFrameRecorder struct{ frames [][]byte }

func (r *entryFrameRecorder) WriteFrame(frame []byte) error {
	r.frames = append(r.frames, append([]byte(nil), frame...))
	return nil
}

// TestSendEntrySceneTailWidthWalk 0x0B 属性段按表列型走查闸：
// attrCount 条 [idx][值] 必须恰好走到帧尾——此前尾属性 idx99 写 u32
// 而表线型为 int64，客户端读 8B 越出帧尾（真机 trace.log
// RecvProperty "Out range" 定案）。双方言表都过。
func TestSendEntrySceneTailWidthWalk(t *testing.T) {
	for _, mode := range []string{"fork", "official2433"} {
		t.Run(mode, func(t *testing.T) {
			clientdata.SwapPropertyTableForTest(t, mode)
			rec := &entryFrameRecorder{}
			scene := role.Scene{Config: `ini\scene\city05_ChengDu`, Resource: "city05"}
			if err := sendEntryScene(rec, scene); err != nil {
				t.Fatal(err)
			}
			frame := rec.frames[0]
			if frame[0] != 0x0B {
				t.Fatalf("op=%#x", frame[0])
			}
			count := binary.LittleEndian.Uint16(frame[13:])
			off := 15
			for k := 0; k < int(count); k++ {
				idx := binary.LittleEndian.Uint16(frame[off:])
				entry, ok := clientdata.PropertyByIndex(int(idx))
				if !ok || entry.Index != idx {
					t.Fatalf("field#%d idx=%d 不在表", k, idx)
				}
				off += 2
				switch entry.Type {
				case clientdata.WireByte:
					off++
				case clientdata.WireWord:
					off += 2
				case clientdata.WireInt32:
					off += 4
				case clientdata.WireInt64:
					if off+8 > len(frame) {
						t.Fatalf("field#%d %s int64 越界（尾宽不足 8B）", k, entry.Name)
					}
					off += 8
				case clientdata.WireFloat32:
					off += 4
				case clientdata.WireString:
					slen := binary.LittleEndian.Uint32(frame[off:])
					off += 4 + int(slen)
				case clientdata.WireWideString:
					blen := binary.LittleEndian.Uint32(frame[off:])
					off += 4 + int(blen)
				default:
					t.Fatalf("field#%d %s 线型 %d 未覆盖", k, entry.Name, entry.Type)
				}
			}
			if off != len(frame) {
				t.Fatalf("%s: 属性段终点 %d ≠ 帧长 %d", mode, off, len(frame))
			}
		})
	}
}
