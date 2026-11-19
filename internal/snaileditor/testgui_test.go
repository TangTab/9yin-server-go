package snaileditor

import "github.com/lizhanyin/9yin-server-go/internal/snaileditornet"

// startTestGUI 起一个回环"编辑器 GUI"服务器（复用 snaileditornet.Server）。
func startTestGUI() (*snaileditornet.Server, error) {
	return snaileditornet.NewServer("127.0.0.1:0")
}
