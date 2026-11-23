package fssoloworld

// npcshop_test.go 商店/仓库 View 单测：帧族形态（View61 四属性+逐物品
// 0x18 行）、模板业务列贯通（npcgen 折叠后的 ShopID/DepotID）、菜单
// mark 路由（缺目录安全降级）。

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/npcshop"
)

func TestNPCShopViewFrames(t *testing.T) {
	section := &npcshop.Section{ID: "Shop_yaopin_00101", NameID: "药堂", Type: 0, Pages: 2, Items: []npcshop.Item{
		{ConfigID: "drug_hp_small", Amount: 99, CapitalType: 0, Price: 500, Page: 1, Position: 0},
		{ConfigID: "exchange_token", Amount: 1, CapitalType: 3, Page: 2, Position: 0, ExchangeID: 13094},
	}}
	frames, err := npcShopViewFrames("Shop_yaopin_00101", section)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 3 { // create + 2 行
		t.Fatalf("帧数 = %d, want 3", len(frames))
	}
	create := frames[0]
	if create[0] != 0x15 {
		t.Fatalf("建视图 op = %#x", create[0])
	}
	if vid := binary.LittleEndian.Uint16(create[1:]); vid != 61 {
		t.Fatalf("视口 = %d, want 61", vid)
	}
	if !containsASCII(create, "Shop_yaopin_00101") {
		t.Fatalf("建视图缺 ShopID 串: % x", create)
	}
	// 行帧：op 0x18、视口 61、objectIndex=(page-1)*10+position+1。
	row0, row1 := frames[1], frames[2]
	for i, row := range [][]byte{row0, row1} {
		if row[0] != 0x18 {
			t.Fatalf("行 %d op = %#x", i, row[0])
		}
		if vid := binary.LittleEndian.Uint16(row[1:]); vid != 61 {
			t.Fatalf("行 %d 视口 = %d", i, vid)
		}
	}
	if idx := binary.LittleEndian.Uint16(row0[3:]); idx != 1 {
		t.Fatalf("行 0 objectIndex = %d, want 1", idx)
	}
	if idx := binary.LittleEndian.Uint16(row1[3:]); idx != 11 {
		t.Fatalf("行 1 objectIndex = %d, want 11（page2 pos0）", idx)
	}
	if !containsASCII(row0, "drug_hp_small") {
		t.Fatalf("行 0 缺 ConfigID: % x", row0)
	}
}

func containsASCII(frame []byte, s string) bool {
	for i := 0; i+len(s) <= len(frame); i++ {
		if string(frame[i:i+len(s)]) == s {
			return true
		}
	}
	return false
}

func TestNPCServicesFromFoldedColumns(t *testing.T) {
	// npcgen 2026-11-23 折叠后的真实档：胡大勇（Shop_weapon_00401）/
	// 金四爷（DepotID=1）业务列直通 npcServicesOf。
	services := npcServicesOf(map[string]string{"ShopID": "Shop_weapon_00401", "NpcType": "62"})
	if len(services) != 1 || services[0].mark != npcMarkShop || services[0].value != "Shop_weapon_00401" {
		t.Fatalf("商店列贯通异常: %+v", services)
	}
	services = npcServicesOf(map[string]string{"DepotID": "1", "NpcType": "67"})
	if len(services) != 1 || services[0].mark != npcMarkDepot || services[0].value != "1" {
		t.Fatalf("仓库列贯通异常: %+v", services)
	}
	// mark 匹配（npcMenuServiceByFuncID）。
	if _, ok := npcMenuServiceByFuncID(services, 807000000); !ok {
		t.Fatal("仓库 mark 807000000 未匹配")
	}
	if _, ok := npcMenuServiceByFuncID(nil, 805000000); ok {
		t.Fatal("空服务列不应匹配")
	}
	// born03 目录贯通：再生后模板行携带 ShopID（npcfeed 联查 → 服务列）。
	s := newNpcTestServer(t)
	found := 0
	for _, sp := range s.sceneNPCCatalog("born03_yanyuzhuang").spawns {
		if sp.ConfigID == "FuncNpc00051" {
			found++
			svc := npcServicesOf(sp.Template)
			if len(svc) != 1 || svc[0].mark != npcMarkShop || svc[0].value != "Shop_weapon_00401" {
				t.Fatalf("胡大勇目录服务异常: %+v", svc)
			}
		}
	}
	if found == 0 {
		t.Fatal("born03 目录未找到 FuncNpc00051（再生档异常）")
	}
}

func TestNPCShopMarkRouting(t *testing.T) {
	// mark 805000000：无目录/缺节安全降级（trace 留痕不 panic），目录
	// 命中走帧族（sendToClient 空 inner 扇出，零连接安全）。
	s := newNpcTestServer(t)
	pl := &Player{Conn1: 7, State: PlOnline}
	if s.openNPCShopView(pl, "Shop_nope") {
		t.Fatal("缺目录不应开窗")
	}
	// NINEYIN_SHOP_INI 指向夹具档 → 命中开窗。
	dir := t.TempDir()
	fixture := filepath.Join(dir, "shop.ini")
	if err := os.WriteFile(fixture, []byte("[Shop_weapon_00401]\nNameID=铁匠铺\nType=0\n0=sword_iron_001,1,0,2500,0,1,0,,\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NINEYIN_SHOP_INI", fixture)
	// 进程级缓存按 root 锚定：测试 Server 与前次调用同 root，需重置。
	npcShopMu.Lock()
	npcShopRootTried = ""
	npcShopMu.Unlock()
	if !s.openNPCShopView(pl, "Shop_weapon_00401") {
		t.Fatal("目录命中应开窗")
	}
	// 仓库开窗（无数据依赖）。
	s.openNPCDepotView(pl) // 无 panic 即过（帧走空 inner）
	// 菜单路由：服务 mark → 关对话+开窗（end-to-end 分支进入）。
	menu := &npcActiveMenu{
		objectID: 5, ownerID: 6,
		services: npcServicesOf(map[string]string{"ShopID": "Shop_weapon_00401", "NpcType": "62"}),
	}
	npcMenuMu.Lock()
	npcMenuState[pl.Conn1] = menu
	npcMenuMu.Unlock()
	if !s.handleNPCMenuSelect(pl, npcObjectRequest{ObjectID: 5, OwnerID: 6, FuncID: 805000000}) {
		t.Fatal("商店 mark 应已受理")
	}
	npcMenuMu.Lock()
	_, still := npcMenuState[pl.Conn1]
	npcMenuMu.Unlock()
	if still {
		t.Fatal("商店开窗后菜单应清除")
	}
}
