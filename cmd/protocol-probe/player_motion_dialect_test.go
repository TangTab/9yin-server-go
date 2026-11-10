package main

import (
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// TestPlayerMotionPropsMatchActiveTable 运动属性族双方言闸（§14.3）：
// baseMotion/qinggongMotion 全部条目的索引与值线型必须与当前方言表的
// 玩家域行一致。2026-11-10 真机：qinggongMotionProperties 26 处
// f(655,"DriftSpeed") 式 fork 字面量在官服方言下整体错位——玩家 0x0D
// 166 属性行在 CurNeiGong(655) 处按 string 读 f32 流即失步，客户端
// 丢弃/截断玩家档案（武学面板与 NPC 可见性连坐）。
func TestPlayerMotionPropsMatchActiveTable(t *testing.T) {
	for _, mode := range []string{"fork", "official2433"} {
		t.Run(mode, func(t *testing.T) {
			clientdata.SwapPropertyTableForTest(t, mode)
			props := append(baseMotionProperties(playerMotion{}), qinggongMotionProperties()...)
			if len(props) < 20 {
				t.Fatalf("motion props=%d, want >=20", len(props))
			}
			for _, p := range props {
				entry, ok := clientdata.PlayerProperty(p.Name)
				if !ok {
					t.Fatalf("%s: %q absent from active table", mode, p.Name)
				}
				if entry.Index != p.Index {
					t.Fatalf("%s: %q index=%d, table=%d", mode, p.Name, p.Index, entry.Index)
				}
				if entry.Type != p.Value.Type {
					t.Fatalf("%s: %q wire=%v, table=%v", mode, p.Name, p.Value.Type, entry.Type)
				}
			}
		})
	}
}
