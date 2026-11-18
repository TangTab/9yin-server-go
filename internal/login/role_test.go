package login

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/net"
	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

func TestParseAndBuildCreateRole(t *testing.T) {
	// 女号外观：photo girl、发型 g_、衣裤鞋 _g 裸配置
	appearance := []string{
		"book1",
		`icon\players\girl01_create.png`,
		"face-xxx",
		"",
		`obj\char\g_hair\g_hair1`,
		"cloth_g0001", "pants_g0001", "shoes_g0001",
		"1",
	}
	frame := buildCreateRoleFrame("小虾米", appearance)
	name, got := parseCreateRole(frame)
	if name != "小虾米" {
		t.Fatalf("名字 %q", name)
	}
	if len(got) != len(appearance) {
		t.Fatalf("外观串 %d want %d: %v", len(got), len(appearance), got)
	}
	for i := range appearance {
		if got[i] != appearance[i] {
			t.Fatalf("外观[%d] %q want %q", i, got[i], appearance[i])
		}
	}
}

func TestInferSex(t *testing.T) {
	cases := []struct {
		appearance []string
		want       byte
	}{
		{[]string{"", `icon\players\girl01_create.png`}, 1},
		{[]string{"", `icon\players\boy01_create.png`}, 0},
		{[]string{"", "", "", "", `obj\char\g_hair\g_hair1`}, 1},
		{[]string{"", "", "", "", "", "cloth_g0001"}, 1},
		{[]string{}, 0}, // 无证据=男
	}
	for _, c := range cases {
		if got := inferSex(c.appearance); got != c.want {
			t.Fatalf("inferSex(%v)=%d want %d", c.appearance, got, c.want)
		}
	}
}

func TestRoleParaFields(t *testing.T) {
	r := &Role{Appearance: []string{"book2", `icon\players\girl01_create.png`}}
	fields := roleParaFields(r)
	if len(fields) != 36 {
		t.Fatalf("字段数 %d want 36", len(fields))
	}
	if fields[0] != "1" { // 女号
		t.Fatalf("性别字段 %q", fields[0])
	}
	if fields[7] != `obj\char\g_hair\g_hair1` || fields[33] != fields[7] {
		t.Fatalf("发型字段 %q/%q", fields[7], fields[33])
	}
	if fields[9] != "cloth_g0001" || fields[10] != "pants_g0001" || fields[11] != "shoes_g0001" {
		t.Fatalf("衣裤鞋 %q %q %q", fields[9], fields[10], fields[11])
	}
	if fields[21] != "stand" || fields[22] != "127.0.0.1" {
		t.Fatalf("f21/f22 %q/%q", fields[21], fields[22])
	}
	if fields[23] != "title001" || fields[24] != "scene_name_017" {
		t.Fatalf("称号/场景 %q/%q", fields[23], fields[24])
	}
	if fields[25] != "book2" || fields[32] != "book2" {
		t.Fatalf("剧本 %q/%q", fields[25], fields[32])
	}
}

func TestOneRoleLoginSuccessLayout(t *testing.T) {
	now := time.Date(2026, 11, 18, 21, 0, 0, 0, time.Local)
	r := &Role{Name: "小虾米", Appearance: []string{"book1"}}
	msg := oneRoleLoginSuccessFrame(now, r)

	if msg[0] != OpServerPlayerRoles {
		t.Fatalf("opcode %02X", msg[0])
	}
	if got := binary.LittleEndian.Uint32(msg[0x25:0x29]); got != 1 {
		t.Fatalf("0x25=%d want 1", got)
	}

	// 追加段：6 值 TVarList（02,02,07,07,02,05）
	body, err := proto.DecodeVarList(msg[0x29:])
	if err != nil {
		t.Fatal(err)
	}
	if len(body.Values) != 6 {
		t.Fatalf("追加段值数 %d want 6", len(body.Values))
	}
	if body.Values[2].Text != "小虾米" {
		t.Fatalf("名字 %q", body.Values[2].Text)
	}
	if para := body.Values[3].Text; strings.Count(para, ",") != 35 {
		t.Fatalf("rolePara 应 36 字段: %q", para)
	}
}

// TestFlowCreateRoleOverGateway 端到端：首飞登录（无角色）→ 建角（有角色
// 0x04）→ 断线重连再登录（仓持久，仍有角色）。
func TestFlowCreateRoleOverGateway(t *testing.T) {
	flight := capturedFirstFlight(t)
	flow := &Flow{}
	g := &Gateway{Addr: "127.0.0.1:0", OnFrame: flow.Handle}
	addr, err := g.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })
	go g.Serve()

	conn, err := net.Dial(addr.String(), net.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMsg(flight); err != nil {
		t.Fatal(err)
	}
	reply, err := conn.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if len(reply) != 0x29 { // 首登无角色=41B
		t.Fatalf("首登应答长度 %d want 41", len(reply))
	}

	// 建角 → 有角色 0x04
	create := buildCreateRoleFrame("小虾米", []string{"book1", `icon\players\girl01_create.png`})
	if err := conn.WriteMsg(create); err != nil {
		t.Fatal(err)
	}
	reply, err = conn.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if len(reply) <= 0x29 || reply[0] != 0x04 {
		t.Fatalf("建角应答形态不符 len=%d", len(reply))
	}
	body, err := proto.DecodeVarList(reply[0x29:])
	if err != nil {
		t.Fatal(err)
	}
	if body.Values[2].Text != "小虾米" {
		t.Fatalf("建角应答名字 %q", body.Values[2].Text)
	}
	conn.Close()

	// 重连：账号仓持久，直接有角色
	conn2, err := net.Dial(addr.String(), net.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
	if err := conn2.WriteMsg(flight); err != nil {
		t.Fatal(err)
	}
	reply2, err := conn2.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if len(reply2) <= 0x29 {
		t.Fatalf("重登应有角色应答 len=%d", len(reply2))
	}
	body2, err := proto.DecodeVarList(reply2[0x29:])
	if err != nil {
		t.Fatal(err)
	}
	if body2.Values[2].Text != "小虾米" {
		t.Fatalf("重登角色名 %q", body2.Values[2].Text)
	}
}
