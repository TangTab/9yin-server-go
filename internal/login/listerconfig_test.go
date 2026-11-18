package login

import (
	"os"
	"strings"
	"testing"
)

// goldenIni 是官方 GameLister.ini 的逐字转录（FsLister 变体，注释保留；
// 解析器跳过注释行，中文注释编码与解析无关）。
const goldenIni = `[System]
// 循环休眠毫秒数（小于0则不休眠）
Sleep=1
// 主实体名称
MainEntity=MainConsole
// 启动脚本名称
MainScript=main

// 指定加载的组件模块（为空则全部加载）
[Module]
FsKit.dll
FsComm.dll
FsLister.dll

// 脚本文件
[Script]
// 所在路径
Path=..\Lua\GameLister
// 是否即时重新加载
Reload=1

[Main]
// 网络接入商编号
IDC=0
// 游戏分区编号
DistrictID=1
// 列表服务器编号
ListerID=0
// 连接数量上限
MaxPlayer=1024
// IO线程数
SockThreads=1

// 外网IP地址
[Outer]
Addr=127.0.0.1
Port=4000

// 内网地址
[Inner]
//Addr=127.0.0.1

// 内网广播地址
[Broadcast]
//Addr=127.255.255.255
//Port=4001
`

func TestParseGameListerINIGolden(t *testing.T) {
	cfg, err := ParseGameListerINI([]byte(goldenIni))
	if err != nil {
		t.Fatal(err)
	}
	want := GameListerConfig{
		IDC: 0, DistrictID: 1, ListerID: 0, MaxPlayer: 1024, SockThreads: 1,
		OuterAddr: "127.0.0.1", OuterPort: 4000,
	}
	if cfg != want {
		t.Fatalf("解析 %+v want %+v", cfg, want)
	}
}

func TestParseGameListerINISoloVariant(t *testing.T) {
	// game_lister.ini（FsSoloLister 变体）：MaxConnects 键 + ClientVersion
	// 版本门 + Inner/Broadcast 值实际生效形态。
	ini := `[Main]
IDC=2
DistrictID=3
ListerID=1
MaxConnects=2048
ClientVersion=105

[Outer]
Addr=10.1.2.3
Port=4001

[Inner]
Addr=10.1.2.3

[Broadcast]
Addr=10.1.2.255
Port=15000
`
	cfg, err := ParseGameListerINI([]byte(ini))
	if err != nil {
		t.Fatal(err)
	}
	want := GameListerConfig{
		IDC: 2, DistrictID: 3, ListerID: 1, MaxPlayer: 2048,
		ClientVersion: 105,
		OuterAddr:     "10.1.2.3", OuterPort: 4001,
		InnerAddr:     "10.1.2.3",
		BroadcastAddr: "10.1.2.255", BroadcastPort: 15000,
	}
	if cfg != want {
		t.Fatalf("解析 %+v want %+v", cfg, want)
	}
}

func TestLoadOrCreateGameListerINI(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/GameLister.ini"
	def := []byte("[Outer]\nAddr=127.0.0.1\nPort=4000\n")

	// 首次：不存在 → 释放默认并返回
	data, created, err := LoadOrCreateGameListerINI(path, def)
	if err != nil || !created || string(data) != string(def) {
		t.Fatalf("首次: data=%q created=%v err=%v", data, created, err)
	}
	if onDisk, _ := os.ReadFile(path); string(onDisk) != string(def) {
		t.Fatalf("落盘内容 %q want %q", onDisk, def)
	}

	// 二次：已存在 → 读现有，绝不覆盖（用户改动保留）
	user := []byte("[Outer]\nAddr=10.0.0.9\nPort=4100\n")
	if err := os.WriteFile(path, user, 0o644); err != nil {
		t.Fatal(err)
	}
	data, created, err = LoadOrCreateGameListerINI(path, def)
	if err != nil || created || string(data) != string(user) {
		t.Fatalf("二次: data=%q created=%v err=%v（应保留用户改动）", data, created, err)
	}
}

func TestParseGameListerINIErrors(t *testing.T) {
	cases := []struct{ name, ini, wantErr string }{
		{"缺 Outer.Addr", "[Outer]\nPort=4000\n", "缺 [Outer] Addr"},
		{"缺 Outer.Port", "[Outer]\nAddr=1.2.3.4\n", "[Outer] Port 非法"},
		{"端口越界", "[Outer]\nAddr=1.2.3.4\nPort=70000\n", "[Outer] Port 非法"},
		{"整数键非整数", "[Main]\nIDC=abc\n[Outer]\nAddr=1.2.3.4\nPort=4000\n", "须为整数"},
		{"无等号行", "[Outer]\n乱行\n", "无 '='"},
	}
	for _, c := range cases {
		_, err := ParseGameListerINI([]byte(c.ini))
		if err == nil || !strings.Contains(err.Error(), c.wantErr) {
			t.Fatalf("%s: err=%v want 含 %q", c.name, err, c.wantErr)
		}
	}
}
