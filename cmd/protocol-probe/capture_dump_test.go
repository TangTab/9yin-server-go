package main

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/transport"
)

// fakeCaptureAddr 提供确定性的远端地址用于命名断言。
type fakeCaptureAddr struct{ network, str string }

func (a fakeCaptureAddr) Network() string { return a.network }
func (a fakeCaptureAddr) String() string  { return a.str }

// fakeCaptureStream 是最小的 net.Conn 桩：写入的字节落入 out，
// 读取来自 in。captureTap 的 tee 行为不依赖真实网络。
type fakeCaptureStream struct {
	in  *bytes.Buffer
	out *bytes.Buffer
}

func (s *fakeCaptureStream) Read(p []byte) (int, error)  { return s.in.Read(p) }
func (s *fakeCaptureStream) Write(p []byte) (int, error) { return s.out.Write(p) }
func (s *fakeCaptureStream) Close() error                { return nil }
func (s *fakeCaptureStream) LocalAddr() net.Addr         { return fakeCaptureAddr{"tcp", "127.0.0.1:19061"} }
func (s *fakeCaptureStream) RemoteAddr() net.Addr        { return fakeCaptureAddr{"tcp", "127.0.0.1:55866"} }
func (s *fakeCaptureStream) SetDeadline(time.Time) error { return nil }

func (s *fakeCaptureStream) SetReadDeadline(time.Time) error  { return nil }
func (s *fakeCaptureStream) SetWriteDeadline(time.Time) error { return nil }

func TestCaptureSessionPrefixMatchesReferenceNaming(t *testing.T) {
	if got := captureSessionPrefix(fakeCaptureAddr{"tcp", "127.0.0.1:55866"}); got != "127-0-0-1-55866" {
		t.Fatalf("prefix=%q, want 127-0-0-1-55866", got)
	}
}

// TestCapturedConnectionDumpsReferenceCompatibleStream 验证三个契约：
// 1. 文件名与参考服抓包一致（<ip-pid>_c2s.bin / _s2c.bin）；
// 2. 记录的是 socket 上的最终线上字节（XOR+转义+EE EE 帧尾），而不是明文；
// 3. 生成的 .bin 能被参考解析器的规则（EE EE 切帧 → 去转义 → 帧内偏移
//    XOR KEY=CC 3B 6D FA）解回原始明文帧。
func TestCapturedConnectionDumpsReferenceCompatibleStream(t *testing.T) {
	dir := t.TempDir()
	stream := &fakeCaptureStream{in: bytes.NewBuffer(nil), out: bytes.NewBuffer(nil)}
	conn, tap := newCapturedConn(stream, dir)
	if tap == nil {
		t.Fatal("capture tap was not installed")
	}

	link, err := transport.NewConnection(conn, transport.DefaultConnectionConfig())
	if err != nil {
		t.Fatal(err)
	}
	outbound := [][]byte{
		{0x09, 0x01, 0x00, 'a', 0, 3},
		{0x10, 0, 1, 0, 0, 0, 2, 0},
	}
	for _, frame := range outbound {
		if err := link.WriteFrame(frame); err != nil {
			t.Fatal(err)
		}
	}
	inbound := [][]byte{
		{0x18, 0xEE, 0x00, 0x42},
		{0x11, 0x37},
	}
	for _, frame := range inbound {
		encoded, err := transport.EncodeFrame(frame, transport.InitialKey, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := stream.in.Write(encoded); err != nil {
			t.Fatal(err)
		}
		got, err := link.ReadFrame()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, frame) {
			t.Fatalf("roundtrip frame=%x want %x", got, frame)
		}
	}
	if err := link.Close(); err != nil {
		t.Fatal(err)
	}

	c2sPath := filepath.Join(dir, "127-0-0-1-55866_c2s.bin")
	s2cPath := filepath.Join(dir, "127-0-0-1-55866_s2c.bin")
	for _, path := range []string{c2sPath, s2cPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing capture dump: %v", err)
		}
	}

	// 参考解析器的等价实现：EE EE 切帧 → EE 00 去转义 → 帧内偏移 XOR。
	key := []byte{0xCC, 0x3B, 0x6D, 0xFA}
	decodeFrames := func(buf []byte) [][]byte {
		var frames [][]byte
		var body []byte
		for i := 0; i < len(buf); {
			if buf[i] == 0xEE && i+1 < len(buf) {
				switch buf[i+1] {
				case 0x00:
					body = append(body, 0xEE)
					i += 2
				case 0xEE:
					plain := make([]byte, len(body))
					for j, b := range body {
						plain[j] = b ^ key[j%4]
					}
					frames = append(frames, plain)
					body = nil
					i += 2
				default:
					t.Fatalf("invalid escape EE %02X", buf[i+1])
				}
			} else {
				body = append(body, buf[i])
				i++
			}
		}
		return frames
	}

	s2cRaw, err := os.ReadFile(s2cPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeFrames(s2cRaw); len(got) != len(outbound) {
		t.Fatalf("s2c frames=%d want %d", len(got), len(outbound))
	} else {
		for i, frame := range got {
			if !bytes.Equal(frame, outbound[i]) {
				t.Fatalf("s2c frame[%d]=%x want %x", i, frame, outbound[i])
			}
		}
	}

	c2sRaw, err := os.ReadFile(c2sPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeFrames(c2sRaw); len(got) != len(inbound) {
		t.Fatalf("c2s frames=%d want %d", len(got), len(inbound))
	} else {
		for i, frame := range got {
			if !bytes.Equal(frame, inbound[i]) {
				t.Fatalf("c2s frame[%d]=%x want %x", i, frame, inbound[i])
			}
		}
	}
}
