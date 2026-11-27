package fssoloworld

// gmitemsearch_test.go 物品检索单测：纯函数匹配/分页/限行、gm items
// echo 行协议（夹具 share/item 目录）。

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func gmItemSearchRows() []gmItemRow {
	return []gmItemRow{
		{ConfigID: "additem_0011", Name: "初级止血药", ViewID: 1, MaxAmount: 99},
		{ConfigID: "material_ore_01", Name: "铁矿石", ViewID: 3, MaxAmount: 999},
		{ConfigID: "wuqi_dao_01", Name: "精钢刀", ViewID: 2, EquipType: "Blade"},
		{ConfigID: "task_letter_01", Name: "家书", ViewID: 4},
	}
}

// TestGMItemSearchPure 纯函数检索：名称/ID 子串（不分大小写）、页过滤、
// limit 钳位、空 query 全量。
func TestGMItemSearchPure(t *testing.T) {
	rows := gmItemSearchRows()
	if got := gmItemSearch(rows, "止血", 0, 50); len(got) != 1 || got[0].ConfigID != "additem_0011" {
		t.Fatalf("名称子串: %+v", got)
	}
	if got := gmItemSearch(rows, "wuqi", 0, 50); len(got) != 1 || got[0].ViewID != 2 {
		t.Fatalf("ID 子串: %+v", got)
	}
	if got := gmItemSearch(rows, "", 3, 50); len(got) != 1 || got[0].ConfigID != "material_ore_01" {
		t.Fatalf("页过滤: %+v", got)
	}
	if got := gmItemSearch(rows, "", 0, 2); len(got) != 2 {
		t.Fatalf("limit 截断: %d", len(got))
	}
	if got := gmItemSearch(rows, "", 0, 0); len(got) != 4 {
		t.Fatalf("limit<1 回退 50: %d", len(got))
	}
	if got := gmItemSearch(rows, "不存在", 0, 50); len(got) != 0 {
		t.Fatalf("无命中: %+v", got)
	}
}

// TestGMItemsCommand 行协议：gm items <query>——结果行
// `>configID|name|view|equipType|maxAmount` + 回执行 + >end。
func TestGMItemsCommand(t *testing.T) {
	s, _ := newCustomTestServer(t)
	root := t.TempDir()
	itemDir := filepath.Join(root, "share", "item")
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ini := "[additem_0011]\nName=初级止血药\nItemType=8\nMaxAmount=99\n" +
		"[wuqi_dao_01]\nscript=Equipment\nEquipType=Blade\nName=精钢刀\n"
	if err := os.WriteFile(filepath.Join(itemDir, "item_static.ini"), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NINEYIN_DATA_ROOT", root)
	resetGMItemSearch()
	t.Cleanup(resetGMItemSearch)

	got := echoGMItems(t, s, "刀")
	if !strings.Contains(got, ">wuqi_dao_01|精钢刀|2|Blade|0\r") {
		t.Fatalf("装备行协议异常: %q", got)
	}
	if !strings.Contains(got, "gm items: 1 行") || !strings.HasSuffix(got, ">end\r") {
		t.Fatalf("回执/收尾异常: %q", got)
	}
	// 页过滤 + 空命中。
	if got := echoGMItems(t, s, "刀 1"); strings.Contains(got, "wuqi_dao_01|") {
		t.Fatalf("页 1 过滤应无装备行: %q", got)
	}
	if got := echoGMItems(t, s, "铁矿石"); !strings.Contains(got, "gm items: 0 行") {
		t.Fatalf("空命中回执: %q", got)
	}
}

// echoGMItems 直调 echo 分发器跑 gm items，回收全部应答行。
func echoGMItems(t *testing.T, s *Server, args string) string {
	t.Helper()
	client, server := net.Pipe()
	defer client.Close()
	done := make(chan struct{})
	go func() {
		(&EchoConsole{srv: s}).command(server, "gm items "+args)
		server.Close()
		close(done)
	}()
	out := make([]byte, 0, 4096)
	buf := make([]byte, 512)
	for {
		n, err := client.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			break
		}
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("echo command timeout")
	}
	return string(out)
}
