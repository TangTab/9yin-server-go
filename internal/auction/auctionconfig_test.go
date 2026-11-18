package auction

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEmbeddedDefaultINI(t *testing.T) {
	cfg, err := ParseGameAuctionINI(defaultGameAuctionINI)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DistrictID != 1 || cfg.ServerID != 11 {
		t.Fatalf("ids = %d/%d", cfg.DistrictID, cfg.ServerID)
	}
	if cfg.InnerPort != 16003 {
		t.Fatalf("inner port = %d", cfg.InnerPort)
	}
	if cfg.SaveMinutes != 20 || cfg.SaveBatchSize != 0 {
		t.Fatalf("save = %d/%d", cfg.SaveMinutes, cfg.SaveBatchSize)
	}
	if cfg.MainEntity != "SoloAuction" || cfg.AuctionLogicDll != "fsauctionlogic.dll" {
		t.Fatalf("system keys = %q/%q", cfg.MainEntity, cfg.AuctionLogicDll)
	}
	// 注释掉的键 = 缺省
	if cfg.InnerAddr != "" || cfg.BroadcastAddr != "" || cfg.BroadcastPort != 0 {
		t.Fatalf("commented keys must default: %+v", cfg)
	}
	// Go 扩展：资源根
	if got := cfg.ItemTypeXML(); got != "Res"+string(os.PathSeparator)+"ini"+string(os.PathSeparator)+
		"Trade"+string(os.PathSeparator)+"AuctionHouse"+string(os.PathSeparator)+"ItemType.xml" {
		t.Fatalf("ItemTypeXML = %q", got)
	}
}

func TestLoadGameAuctionINICreateAndReuse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game_auction.ini")

	// 首次：释放内置默认
	cfg, _, created, err := LoadGameAuctionINI(path)
	if err != nil || !created {
		t.Fatalf("first load: created=%v err=%v", created, err)
	}
	if cfg.InnerPort != 16003 {
		t.Fatalf("released default port = %d", cfg.InnerPort)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not released: %v", err)
	}

	// 用户改写后：绝不覆盖，采用用户值
	body := "[Main]\nDistrictID=2\nServerID=33\n[Inner]\nPort=16013\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, created, err = LoadGameAuctionINI(path)
	if err != nil || created {
		t.Fatalf("second load: created=%v err=%v", created, err)
	}
	if cfg.DistrictID != 2 || cfg.ServerID != 33 || cfg.InnerPort != 16013 {
		t.Fatalf("user values = %+v", cfg)
	}
}

func TestParseOfficialGBKINI(t *testing.T) {
	const official = `F:\Game\Online\9yin\BGserverCJ\Debug\game_auction.ini`
	raw, err := os.ReadFile(official)
	if err != nil {
		t.Skipf("官方快照不可读：%v", err)
	}
	cfg, err := ParseGameAuctionINI(raw)
	if err != nil {
		t.Fatalf("官方 GBK 文件解析失败：%v", err)
	}
	if cfg.DistrictID != 1 || cfg.ServerID != 11 {
		t.Fatalf("official ids = %d/%d", cfg.DistrictID, cfg.ServerID)
	}
	if cfg.AuctionLogicDll != "fsauctionlogic.dll" {
		t.Fatalf("logic dll = %q", cfg.AuctionLogicDll)
	}
	// 官方 [Inner] 只注释 Addr → 全地址侦听语义
	if cfg.InnerAddr != "" {
		t.Fatalf("official InnerAddr = %q", cfg.InnerAddr)
	}
}

func TestParseINIValidation(t *testing.T) {
	if _, err := ParseGameAuctionINI([]byte("[Main]\nServerID=11\n[Inner]\nPort=16003\n")); err == nil {
		t.Fatal("缺 DistrictID 须报错")
	}
	if _, err := ParseGameAuctionINI([]byte("[Main]\nDistrictID=1\nServerID=11\n[Inner]\nPort=70000\n")); err == nil {
		t.Fatal("非法端口须报错")
	}
	// [Inner] Port 缺省 → main.lua 缺省 16003（官方文件即不写该键）
	cfg2, err := ParseGameAuctionINI([]byte("[Main]\nDistrictID=1\nServerID=11\n[Inner]\n//Addr=127.0.0.1\n"))
	if err != nil || cfg2.InnerPort != 16003 {
		t.Fatalf("缺省端口回落: port=%d err=%v", cfg2.InnerPort, err)
	}
	if _, err := ParseGameAuctionINI([]byte("[Main]\nDistrictID=x\nServerID=11\n[Inner]\nPort=16003\n")); err == nil {
		t.Fatal("非整数须报错")
	}
	// 未知节/键 + Module 裸行 + DB 连接串：前向兼容不报错
	ok := "[Module]\nfskit.dll\n[Account]\nConnectString=Provider=SQLOLEDB;x\n[Main]\nDistrictID=1\nServerID=11\n[Inner]\nPort=16003\n"
	if _, err := ParseGameAuctionINI([]byte(ok)); err != nil {
		t.Fatalf("兼容键不应报错：%v", err)
	}
}
