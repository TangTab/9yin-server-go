// tmp/login-client 是一次性验证工具：用 fork 登录加密构造 op02 账号登录帧，
// 连接运行中的游戏服并验证 0x04 角色列表应答（模拟客户端可行性取证）。
// 不属于构建产物，验证后可整体删除。
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/local/9yin-go-server/internal/auth"
	"github.com/local/9yin-go-server/internal/transport"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:19061", "server address")
	account := flag.String("account", "test001", "login account")
	password := flag.String("password", "test001", "login password")
	flag.Parse()

	frame, err := buildLoginFrame(*account, *password)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("login frame built: account=%s op02 len=%d\n", *account, len(frame))

	conn, err := net.DialTimeout("tcp", *addr, 3*time.Second)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(8 * time.Second)); err != nil {
		fatal(err)
	}
	writer, err := transport.NewFrameWriter(conn, 1<<20)
	if err != nil {
		fatal(err)
	}
	reader, err := transport.NewFrameReader(conn, 1<<20)
	if err != nil {
		fatal(err)
	}
	if err := writer.WriteFrame(frame, transport.InitialKey); err != nil {
		fatal(fmt.Errorf("write login frame: %w", err))
	}
	fmt.Println("op02 sent, waiting for role list...")

	// 登录成功：0x04 角色列表 → 0x03 世界信息 → 0x04 选角 → 快照簇 → 0x09 就绪。
	response, err := reader.ReadFrame(transport.InitialKey)
	if err != nil || len(response) == 0 || response[0] != 0x04 {
		fatal(fmt.Errorf("expect 0x04 role list, got err=%v len=%d", err, len(response)))
	}
	fmt.Printf("step 1: opcode=0x%02X len=%d (role list)\n", response[0], len(response))
	if err := writer.WriteFrame([]byte{0x03}, transport.InitialKey); err != nil {
		fatal(fmt.Errorf("write world-info request: %w", err))
	}
	response, err = reader.ReadFrame(transport.InitialKey)
	if err != nil || len(response) == 0 || response[0] != 0x05 {
		fatal(fmt.Errorf("expect 0x05 world info, got err=%v len=%d", err, len(response)))
	}
	fmt.Printf("step 2: opcode=0x%02X len=%d (world info)\n", response[0], len(response))

	// 选角：handler 不解析帧体，仅需满足状态机最小 0x54 字节。
	choose := make([]byte, 0x54)
	choose[0] = 0x04
	if err := writer.WriteFrame(choose, transport.InitialKey); err != nil {
		fatal(fmt.Errorf("write choose-role: %w", err))
	}
	fmt.Println("op04 choose-role sent, reading snapshot stream...")

	histogram := map[byte]int{}
	sawTable, sawReady := false, false
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		frame, err := reader.ReadFrame(transport.InitialKey)
		if err != nil {
			break // 快照流读完（超时/对端静默）
		}
		if len(frame) == 0 {
			continue
		}
		histogram[frame[0]]++
		switch frame[0] {
		case 0x09:
			if !sawTable {
				fmt.Printf("snapshot: 0x09 property table len=%d\n", len(frame))
			}
			sawTable = true
			// 属性表/记录表已到，发就绪通知（幂等，可在流中提前发）。
			if !sawReady {
				if err := writer.WriteFrame([]byte{0x09}, transport.InitialKey); err != nil {
					fatal(fmt.Errorf("write client-ready: %w", err))
				}
				sawReady = true
				fmt.Println("op09 client-ready sent")
			}
		case 0x0A:
			fmt.Printf("snapshot: 0x0A record schema len=%d\n", len(frame))
		}
	}
	if !sawTable {
		fatal(fmt.Errorf("snapshot stream had no 0x09 property table (frames=%v)", histogram))
	}
	// 就绪后应继续收到场景/属性推送；再收 2s 佐证会话存活。
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	afterReady := 0
	for {
		frame, err := reader.ReadFrame(transport.InitialKey)
		if err != nil {
			break
		}
		if len(frame) != 0 {
			afterReady++
			histogram[frame[0]]++
		}
	}
	total := 0
	for _, n := range histogram {
		total += n
	}
	fmt.Printf("snapshot+post-ready frames: total=%d after_ready=%d histogram=%v\n", total, afterReady, histogram)
	fmt.Println("LOGIN FLOW OK: 0x02 login -> 0x04 roles -> 0x03 -> 0x05 -> 0x04 choose -> snapshot -> 0x09 ready")
}

// buildLoginFrame 按 ParseLoginAccountIdentity/ParseLoginCredentials 的帧布局
// 构造完整 op02：25B 头（[0x15:0x19] 账号槽长）+ 账号槽 + u32 密码槽长 +
// 密码槽 + u32 clientType + u32 loginString 长 + loginString + u32 loginType。
func buildLoginFrame(account, password string) ([]byte, error) {
	accountCT, passwordCT := auth.EncryptFields(account, password)
	// 槽长须为 16 的倍数（validateSlotLength 的 %16 约束），占位 32B。
	loginString := make([]byte, 32)
	copy(loginString, "probe-loginclient")

	frame := make([]byte, 0, 25+len(accountCT)+4+len(passwordCT)+4+4+len(loginString)+4)
	header := make([]byte, 25)
	header[0] = 0x02
	binary.LittleEndian.PutUint32(header[0x15:0x19], uint32(len(accountCT)))
	frame = append(frame, header...)
	frame = append(frame, accountCT...)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(passwordCT)))
	frame = append(frame, passwordCT...)
	frame = binary.LittleEndian.AppendUint32(frame, 2) // clientType
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(loginString)))
	frame = append(frame, loginString...)
	frame = binary.LittleEndian.AppendUint32(frame, 0) // loginType
	return frame, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
