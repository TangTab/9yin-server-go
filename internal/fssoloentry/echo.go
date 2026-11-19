package fssoloentry

import (
	"fmt"
	"os"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/entry"
)

// EchoWelcome 官方欢迎行（soloentry echo_open："welcome server <sid>
// entry <eid>\r"）。
func EchoWelcome(serverID, entryID uint32) string {
	return fmt.Sprintf("welcome server %d entry %d\r", serverID, entryID)
}

// EchoResponse 官方应答前缀（">" + 信息 + "\r"）。
func EchoResponse(info string) string { return ">" + info + "\r" }

// EchoHandler 装配 solo 维护面命令集（官方 echo_input：quit/loadip/end
// 具名 + 其余走 Lua 反射面 undefined command 兜底；复刻以状态查询命令
// 等价反射面）。
//
//	blockIP 为网关黑名单（loadip 目标）；state 返回状态行；shutdown
// 由宿主注入（官方 close()：WriteLine + Stop + nx_quit）。
func EchoHandler(serverID, entryID uint32, blockIP *entry.BlockIPList,
	state func() string, shutdown func()) (welcome string, onInput func(string) []string) {
	welcome = EchoWelcome(serverID, entryID)
	onInput = func(line string) []string {
		cmd := strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		switch cmd {
		case "quit":
			// 官方："server will shutdown..." → close()
			out := []string{EchoResponse("server will shutdown...")}
			if shutdown != nil {
				go shutdown()
			}
			return out
		case "loadip":
			out := []string{EchoResponse("reload block ip list...")}
			out = append(out, LoadIPFile("ip.txt", blockIP)...)
			out = append(out, EchoResponse("reload block ip list ended"))
			return out
		case "end":
			// 官方 CloseEcho：仅断开本维护连接（宿主以停止 echo 口等价）
			return nil
		case "state", "stat":
			if state != nil {
				return []string{EchoResponse(state())}
			}
			return []string{EchoResponse("state unavailable")}
		case "help":
			return []string{EchoResponse("commands: state loadip quit end")}
		default:
			return []string{EchoResponse("undefined command: " + cmd)}
		}
	}
	return welcome, onInput
}

// LoadIPFile 复刻 Lua load_ip：ip.txt 行 "beg,end"（分隔符 ,/TAB/空格
// 依次回退）→ AddBlockIP；官方 nx_log 逐行留痕，此处返回装载日志。
func LoadIPFile(path string, blockIP *entry.BlockIPList) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // nx_file_exists 门：缺文件静默
	}
	var logs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		sep := strings.IndexAny(line, ",\t ")
		if sep < 0 {
			continue
		}
		beg := line[:sep]
		end := line[sep+1:]
		if beg == "" || end == "" {
			continue
		}
		if blockIP.AddBlockIP(beg, end) {
			logs = append(logs, "add block ip: "+beg+" to "+end)
		} else {
			logs = append(logs, "add block ip failed: "+line)
		}
	}
	return logs
}
