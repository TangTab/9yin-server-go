package charge

import (
	"testing"
)

// 官方 text_encode_msg 方言 golden：元素形态（裸命令词/`$`窄串/数字/
// `%f` 浮点/转义）。
func TestEncodeLineGolden(t *testing.T) {
	cases := []struct {
		name string
		vl   VarList
		want string
	}{
		{
			name: "register 请求（客户端形态）",
			vl: func() VarList {
				v := VarList{}
				v.AddString("register")
				v.AddInt(1)
				v.AddInt(101)
				v.AddString("本地测试")
				v.AddString("5D41402ABC4B2A76B9719D911017C592")
				v.AddInt(9600)
				return v
			}(),
			want: "$register 1 101 $\\xE6\\x9C\\xAC\\xE5\\x9C\\xB0\\xE6\\xB5\\x8B\\xE8\\xAF\\x95 $5D41402ABC4B2A76B9719D911017C592 9600",
		},
		{
			name: "register 应答（服务端形态）",
			vl: func() VarList {
				v := VarList{}
				v.AddString("register")
				v.AddString("register")
				v.AddInt(1)
				return v
			}(),
			want: "$register $register 1",
		},
		{
			name: "login 应答含浮点到期时间",
			vl: func() VarList {
				v := VarList{}
				v.AddString("gw1")
				v.AddString("login")
				v.AddInt(1)
				v.AddString("TEST001")
				v.AddInt(1)
				v.AddInt(42)
				v.AddString("TEST001")
				v.AddString("")
				v.AddInt(100)
				v.AddString("")
				v.AddInt(2)
				v.AddDouble(46023.5)
				v.AddInt(0)
				v.AddString("")
				return v
			}(),
			want: "$gw1 $login 1 $TEST001 1 42 $TEST001 $ 100 $ 2 46023.500000 0 $",
		},
	}
	for _, c := range cases {
		if got := c.vl.EncodeLine(); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

// 官方 text_decode_msg 方言：裸词=窄串、`$`/`#` 前缀、数字 int/int64/
// double 分档、转义回解。
func TestDecodeLine(t *testing.T) {
	vl, err := DecodeLine("gw1 login 7 TEST001 46023.5 12345678901 $a\\x20b #\\u4E2D\\u6587")
	if err != nil {
		t.Fatal(err)
	}
	if got := vl.StringVal(0); got != "gw1" {
		t.Errorf("[0] 裸词窄串: %q", got)
	}
	if got := vl.StringVal(1); got != "login" {
		t.Errorf("[1]: %q", got)
	}
	if vl[2].Kind != KindInt || vl.Int(2) != 7 {
		t.Errorf("[2] int 分档: %+v", vl[2])
	}
	if vl[4].Kind != KindDouble || vl.DoubleVal(4) != 46023.5 {
		t.Errorf("[4] double: %+v", vl[4])
	}
	if vl[5].Kind != KindInt64 || vl[5].I != 12345678901 {
		t.Errorf("[5] int64 分档: %+v", vl[5])
	}
	if got := vl.StringVal(6); got != "a b" {
		t.Errorf("[6] \\x20: %q", got)
	}
	if got := vl.StringVal(7); got != "中文" {
		t.Errorf("[7] 宽串 \\u: %q", got)
	}
}

// 编解码往返（含空格/百分号/反斜杠转义域）。
func TestWireRoundTrip(t *testing.T) {
	src := VarList{}
	src.AddString("echo 100%")
	src.AddString("login")
	src.AddInt(-3)
	src.AddDouble(1.25)
	src.AddWideStr("名字\\x")
	enc := src.EncodeLine()
	dst, err := DecodeLine(enc)
	if err != nil {
		t.Fatalf("decode %q: %v", enc, err)
	}
	if dst.StringVal(0) != "echo 100%" || dst.StringVal(1) != "login" {
		t.Errorf("串往返: %q / %q", dst.StringVal(0), dst.StringVal(1))
	}
	if dst.Int(2) != -3 {
		t.Errorf("int 往返: %d", dst.Int(2))
	}
	if dst.DoubleVal(3) != 1.25 {
		t.Errorf("double 往返: %f", dst.DoubleVal(3))
	}
	if dst.StringVal(4) != `名字\x` {
		t.Errorf("宽串往返: %q", dst.StringVal(4))
	}
}

func TestParseGameChargeINI(t *testing.T) {
	cfg, err := ParseGameChargeINI([]byte(`
[System]
Sleep=1
[Module]
FsKit.dll
[Main]
GameID=1
PointMinutes=6
AwakeTime=60
FlushErrorMax=0
UserShakeTime=3600
FlushLog=1
[Outer]
Addr=192.168.33.70
Port=6000
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GameID != 1 || cfg.PointMinutes != 6 || cfg.AwakeSec != 60 ||
		cfg.FlushErrorMax != 0 || !cfg.FlushLog {
		t.Errorf("Main 投影: %+v", cfg)
	}
	if cfg.Addr != "192.168.33.70" || cfg.Port != 6000 {
		t.Errorf("Outer: %+v", cfg)
	}
}
