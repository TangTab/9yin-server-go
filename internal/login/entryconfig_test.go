package login

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// officialGameEntryINI 是官方 F:\Game\Online\9yin\BGserverCJ\Debug\
// game_entry.ini 原文（小写系）节选：含注释键（//Addr=127.0.0.1）、
// [Module] 裸行、[Script] 键——方言兼容性的对拍底本。
const officialGameEntryINI = `[System]
Sleep=-1
MainEntity=SoloEntry
MainScript=main

[Module]
fskit.dll
fscomm.dll
fssoloentry.dll

[Script]
Path=..\lua\soloentry
Reload=1

[Main]
//BindCore=0
IDC=1
DistrictID=1
ServerID=11
EntryID=0
MaxConnects=1024
//MessageEncodeKey=A5375CAE
//MessageVerifyDll=fsverify.dll

[Outer]
Addr=192.168.33.70
Port=2001

[Inner]
//Addr=127.0.0.1

[Broadcast]
//Addr=127.255.255.255
//Port=4001
`

func TestParseOfficialGameEntryINI(t *testing.T) {
	cfg, err := ParseGameEntryINI([]byte(officialGameEntryINI))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MainEntity != "SoloEntry" || cfg.MainScript != "main" {
		t.Fatalf("System 节：%q/%q", cfg.MainEntity, cfg.MainScript)
	}
	if cfg.IDC != 1 || cfg.DistrictID != 1 || cfg.ServerID != 11 || cfg.EntryID != 0 {
		t.Fatalf("Main 编号：%d/%d/%d/%d", cfg.IDC, cfg.DistrictID, cfg.ServerID, cfg.EntryID)
	}
	if cfg.MaxConnects != 1024 {
		t.Fatalf("MaxConnects=%d", cfg.MaxConnects)
	}
	// 注释掉的键=缺省（官方方言：//MessageEncodeKey 视为未设置）
	if cfg.MessageEncodeKey != "" || cfg.MessageVerifyDll != "" || cfg.ClientVersion != 0 {
		t.Fatalf("注释键应缺省：%q/%q/%d", cfg.MessageEncodeKey, cfg.MessageVerifyDll, cfg.ClientVersion)
	}
	if cfg.OuterAddr != "192.168.33.70" || cfg.OuterPort != 2001 {
		t.Fatalf("Outer：%s:%d", cfg.OuterAddr, cfg.OuterPort)
	}
	// [Inner]/[Broadcast] 全注释 → 零值（仅呈现字段）
	if cfg.InnerAddr != "" || cfg.BroadcastAddr != "" || cfg.BroadcastPort != 0 {
		t.Fatalf("注释段应零值：%q/%q/%d", cfg.InnerAddr, cfg.BroadcastAddr, cfg.BroadcastPort)
	}
}

func TestParseGameEntryINIErrors(t *testing.T) {
	if _, err := ParseGameEntryINI([]byte("[Outer]\nAddr=1.2.3.4\n")); err == nil {
		t.Fatal("缺 [Outer] Port 应报错")
	}
	if _, err := ParseGameEntryINI([]byte("[Main]\nIDC=x\n[Outer]\nAddr=1.2.3.4\nPort=2001\n")); err == nil {
		t.Fatal("非整数应报错")
	}
	if _, err := ParseGameEntryINI([]byte("[Main]\n裸行\n[Outer]\nAddr=1.2.3.4\nPort=2001\n")); err == nil {
		t.Fatal("非 [Module] 节裸行应报错")
	}
	// Go 扩展键：ClientVersion / MessageEncodeKey
	cfg, err := ParseGameEntryINI([]byte(
		"[Main]\nClientVersion=333\nMessageEncodeKey=A5375CAE\n" +
			"[Outer]\nAddr=127.0.0.1\nPort=9600\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientVersion != 333 || cfg.MessageEncodeKey != "A5375CAE" {
		t.Fatalf("扩展键：%d/%q", cfg.ClientVersion, cfg.MessageEncodeKey)
	}
}

// TestConvertEncodeKey 官方 SoloEntry::ConvertEncodeKey 语义：hex 串→u32
// （官方样本 A5375CAE→0xA5375CAE；Go 兼容小写；空=0=未设置）。
func TestConvertEncodeKey(t *testing.T) {
	cases := []struct {
		in   string
		want uint32
		ok   bool
	}{
		{"A5375CAE", 0xA5375CAE, true},
		{"a5375cae", 0xA5375CAE, true},
		{"FA6D3BCC", 0xFA6D3BCC, true},
		{"", 0, true},
		{"XYZ", 0, false},
		{"123456789", 0, false},
	}
	for _, c := range cases {
		got, err := ConvertEncodeKey(c.in)
		if c.ok && (err != nil || got != c.want) {
			t.Fatalf("ConvertEncodeKey(%q)=%#x,%v want %#x", c.in, got, err, c.want)
		}
		if !c.ok && err == nil {
			t.Fatalf("ConvertEncodeKey(%q) 应报错", c.in)
		}
	}
}

func TestLoadOrCreateGameEntryINI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game_entry.ini")

	data, created, err := LoadOrCreateGameEntryINI(path, []byte(officialGameEntryINI))
	if err != nil || !created || string(data) != officialGameEntryINI {
		t.Fatalf("首次释放：created=%t err=%v", created, err)
	}
	// 已存在的绝不覆盖：写脏后再载，内容保持用户改动。
	if err := os.WriteFile(path, []byte("// user edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, created, err = LoadOrCreateGameEntryINI(path, []byte(officialGameEntryINI))
	if err != nil || created || !strings.Contains(string(data), "user edit") {
		t.Fatalf("二次装载不得覆盖：created=%t data=%q err=%v", created, data, err)
	}
}
