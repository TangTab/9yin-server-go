package fssoloworld

// itemuse_test.go msg31 使用道具受理链测试：官服抓包逐字节对拍（武学书
// 读条流/行上锁/扣件帧）、钱袋入账与冷却、瞬时使用、堆叠扣件、拒绝面、
// 分发接线；以及入包堆叠上限。

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

func useMsg(view, slot, count int32) entergen.CustomInbound {
	return entergen.CustomInbound{Values: []entergen.CustomInboundValue{
		{Type: 2, Int32: 31}, {Type: 2, Int32: view}, {Type: 2, Int32: slot}, {Type: 2, Int32: count},
	}}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// 官服原始帧（资料\初始账号登陆-完整引导-女.pcapng，m31 #1，tools/pcapitem
// 解码）。玩家对象号官服为 350000f3/2a100010、本仓为 1/1——State 帧按本仓
// 口径现拼，其余（m25/m26/行锁/扣件）逐字节对拍。
const (
	officialUseFlow = "1e0400021900000002dc050000060d000000666c6f775f7072657061726500060d0000007573656974656d5f666c6f7700"
	officialLock    = "100102000000050000000100720701000000"
	officialUnlock  = "100102000000050000000100720700000000"
	officialFlowEnd = "1e0100021a000000"
	officialConsume = "1902000500"
)

func playerPropFrame(t *testing.T, name string, value entergen.PropertyValue) []byte {
	t.Helper()
	frame, err := entergen.ObjectPropertyFrame(forkPlayerObjectID, forkPlayerOwnerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty(name, value)})
	if err != nil {
		t.Fatal(err)
	}
	return frame
}

type itemUseFixture struct {
	s   *Server
	pl  *Player
	out *[][]byte
}

func newItemUseFixture(t *testing.T) *itemUseFixture {
	t.Helper()
	s := newNpcTestServer(t)
	pl := s.players.AddPlayer(7, 0, "")
	pl.State, pl.haveRoleState = PlOnline, true
	var frames [][]byte
	prev := clientSendObserver
	clientSendObserver = func(got *Player, frame []byte) {
		if got == pl {
			frames = append(frames, frame)
		}
	}
	t.Cleanup(func() { clientSendObserver = prev })
	return &itemUseFixture{s: s, pl: pl, out: &frames}
}

func (f *itemUseFixture) take() [][]byte {
	frames := *f.out
	*f.out = nil
	return frames
}

// finish 手动驱动读条结束（定时器走真实时钟，测试直接触发；晚到的真实
// 定时器发现流已清除即空转）。
func (f *itemUseFixture) finish(t *testing.T) {
	t.Helper()
	f.pl.itemUseMu.Lock()
	flow := f.pl.itemUse
	f.pl.itemUseMu.Unlock()
	if flow == nil {
		t.Fatal("无进行中的读条流")
	}
	f.s.finishItemUse(f.pl, flow)
}

func (f *itemUseFixture) busy() bool {
	f.pl.itemUseMu.Lock()
	defer f.pl.itemUseMu.Unlock()
	return f.pl.itemUse != nil
}

// 武学书（官服 #1 对拍）：受理 = m25 读条流 + 行上锁 + stand/动作；读条中
// 再按静默；结束 = m26 → 清态 → 解锁 → View40 追加行 → MaxLevel/CanUse →
// m141/m1 → 扣件。已学的书再用：不动。
func TestUseItemSkillBookOfficialDialect(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "testdata", "ini", "Item", "path_config.xml"),
		filepath.Join("..", "..", "testdata", "ini", "Skill", "skill_static.xml"),
	} {
		if _, err := res.Stat(path); err != nil {
			t.Skipf("构建产物缺席（go run ./tools/datagen item / ./tools/datagen skill）: %v", err)
		}
	}
	f := newItemUseFixture(t)
	st := &f.pl.roleState
	st.BagItems = []rolestate.BagItem{
		{Slot: 5, ConfigID: "book_CS_base_pgd01", ItemType: 23, ViewID: 1, Amount: 1, MaxAmount: 1, Seq: 1},
	}
	f.s.dispatchCustom(f.pl, useMsg(2, 5, 1))
	accept := f.take()
	if len(accept) != 4 {
		t.Fatalf("受理帧数 %d want 4: % x", len(accept), accept)
	}
	if !bytes.Equal(accept[0], mustHex(t, officialUseFlow)) {
		t.Errorf("m25 读条流不符官服: % x", accept[0])
	}
	if !bytes.Equal(accept[1], mustHex(t, officialLock)) {
		t.Errorf("行上锁帧不符官服: % x", accept[1])
	}
	if !bytes.Equal(accept[2], playerPropFrame(t, "State", entergen.StringValue("stand"))) ||
		!bytes.Equal(accept[3], playerPropFrame(t, "State", entergen.StringValue("life_ss_01"))) {
		t.Errorf("State 帧应 stand → life_ss_01（功能包 118）: % x | % x", accept[2], accept[3])
	}
	// 状态在受理时落定：学会招式、扣件、读条进行中。
	if len(st.Skills) != 1 || st.Skills[0].ConfigID != "CS_base_pgd01" || st.Skills[0].StaticData != 5459 ||
		st.Skills[0].ItemType != 1000 || st.Skills[0].Level != 1 {
		t.Fatalf("武学书应学会 CS_base_pgd01: %+v", st.Skills)
	}
	if len(st.BagItems) != 0 || !f.busy() {
		t.Fatalf("应扣件并置读条态: bag=%+v busy=%t", st.BagItems, f.busy())
	}
	// 读条中再按：静默。
	f.s.dispatchCustom(f.pl, useMsg(2, 6, 1))
	if again := f.take(); len(again) != 0 {
		t.Fatalf("读条进行中应零应答: % x", again)
	}
	f.finish(t)
	tail := f.take()
	if len(tail) != 9 {
		t.Fatalf("尾段帧数 %d want 9: % x", len(tail), tail)
	}
	if !bytes.Equal(tail[0], mustHex(t, officialFlowEnd)) {
		t.Errorf("m26 不符官服: % x", tail[0])
	}
	if !bytes.Equal(tail[1], playerPropFrame(t, "State", entergen.StringValue(""))) {
		t.Errorf("清态帧异常: % x", tail[1])
	}
	if !bytes.Equal(tail[2], mustHex(t, officialUnlock)) {
		t.Errorf("行解锁帧不符官服: % x", tail[2])
	}
	// View40 追加行 idx=1 携 ConfigID；随后 MaxLevel（逐级行数）/ CanUse
	// 视图对象帧。
	row := tail[3]
	if row[0] != 0x18 || uint16(row[1])|uint16(row[2])<<8 != entergen.ViewportSkill ||
		uint16(row[3])|uint16(row[4])<<8 != 1 || !frameContains([][]byte{row}, []byte("CS_base_pgd01")) {
		t.Errorf("View40 追加行异常: % x", row)
	}
	learned, err := f.s.skillCatalog().Definition("CS_base_pgd01", 1)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []entergen.PropertyValue{
		entergen.Int32Value(int32(max(learned.Levels, 1))), entergen.ByteValue(1),
	} {
		frame, err := entergen.ObjectPropertyFrame(uint32(entergen.ViewportSkill), 1, 1,
			[]entergen.IndexedProperty{entergen.NamedProperty([]string{"MaxLevel", "CanUse"}[i], want)})
		if err != nil || !bytes.Equal(tail[4+i], frame) {
			t.Errorf("视图对象帧 %d 异常: % x (err=%v)", i, tail[4+i], err)
		}
	}
	if !frameContains([][]byte{tail[6]}, []byte("CS_base_pgd01")) || tail[6][4] != 141 {
		t.Errorf("m141 学会通知异常: % x", tail[6])
	}
	if !frameContains([][]byte{tail[7]}, []byte("9491")) || !frameContains([][]byte{tail[7]}, []byte("CS_base_pgd01")) {
		t.Errorf("m1 系统提示异常: % x", tail[7])
	}
	if !bytes.Equal(tail[8], mustHex(t, officialConsume)) {
		t.Errorf("扣件帧不符官服: % x", tail[8])
	}
	if f.busy() {
		t.Error("读条结束后应清读条态")
	}
	// 晚到的真实定时器：流已清，空转。
	f.s.finishItemUse(f.pl, &itemUseFlow{})
	if extra := f.take(); len(extra) != 0 {
		t.Fatalf("已结束的流不应再发帧: % x", extra)
	}

	// 已学的书再用：无可生效目标 → 零应答、不扣件。
	st.BagItems = []rolestate.BagItem{{Slot: 5, ConfigID: "book_CS_base_pgd01", ItemType: 23, ViewID: 1, Amount: 1, Seq: 2}}
	f.s.dispatchCustom(f.pl, useMsg(2, 5, 1))
	if extra := f.take(); len(extra) != 0 || len(st.BagItems) != 1 || len(st.Skills) != 1 || f.busy() {
		t.Fatalf("已学书应拒且不扣件: frames=%d bag=%d skills=%d busy=%t", len(extra), len(st.BagItems), len(st.Skills), f.busy())
	}
}

// 微型构建产物夹具（NINEYIN_ITEM_XML 显式覆盖目录）。
func useItemProduct(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("path_config.xml", `<Object><Property ID="1" Type="ItemUse" Path="ItemUse.xml"/></Object>`)
	write("ItemUse.xml", `<Object>
	<Property ID="money_bag" script="CapitalItem" ItemType="37" ViewID="1" Name="钱袋" FuncPack="1" CapitalType="1" CapitalValue="500" CoolDownCategory="24001" CoolDownTime="1000" MaxAmount="1"/>
	<Property ID="inst_money" script="CapitalItem" ItemType="37" ViewID="1" Name="散碎银两" FuncPack="2" CapitalType="1" CapitalValue="50"/>
	<Property ID="gold_bag" script="CapitalItem" ItemType="37" ViewID="1" Name="金袋" FuncPack="1" CapitalType="2" CapitalValue="10"/>
	<Property ID="box_x" script="BoxItem" ItemType="2102" ViewID="1" Name="礼盒" FuncPack="1" DropID="drop_x"/>
	<Property ID="task_mark" script="ToolItem" ItemType="1200" ViewID="4" Name="星图"/>
</Object>`)
	write("item_func_static.xml", `<Object><Property ID="1" PrepareTime="1000"/><Property ID="2"/></Object>`)
	t.Setenv("NINEYIN_ITEM_XML", dir)
	resetItemCatalog()
	return dir
}

// 钱袋：读条受理 → 入账 + 冷却行 + 扣件；冷却中再按静默；金袋（未建模
// 币种）/礼盒/不可用物零应答不动状态。
func TestUseItemMoneyBagAndCooldown(t *testing.T) {
	useItemProduct(t)
	f := newItemUseFixture(t)
	st := &f.pl.roleState
	bag := func() []rolestate.BagItem {
		return []rolestate.BagItem{
			{Slot: 3, ConfigID: "money_bag", ItemType: 37, ViewID: 1, Amount: 1, MaxAmount: 1, Seq: 1},
			{Slot: 4, ConfigID: "money_bag", ItemType: 37, ViewID: 1, Amount: 1, MaxAmount: 1, Seq: 2},
			{Slot: 5, ConfigID: "gold_bag", ItemType: 37, ViewID: 1, Amount: 1, Seq: 3},
			{Slot: 6, ConfigID: "box_x", ItemType: 2102, ViewID: 1, Amount: 1, Seq: 4},
		}
	}
	st.BagItems = bag()

	// 受理：m25(1000ms) + 行上锁 + stand（功能包无动作）。
	f.s.dispatchCustom(f.pl, useMsg(2, 3, 1))
	accept := f.take()
	flow := mustHex(t, "1e0400021900000002e8030000060d000000666c6f775f7072657061726500060d0000007573656974656d5f666c6f7700")
	lock3 := mustHex(t, "100102000000030000000100720701000000")
	if len(accept) != 3 || !bytes.Equal(accept[0], flow) || !bytes.Equal(accept[1], lock3) ||
		!bytes.Equal(accept[2], playerPropFrame(t, "State", entergen.StringValue("stand"))) {
		t.Fatalf("钱袋受理帧异常（%d 帧）: % x", len(accept), accept)
	}
	if st.Money != 500 || len(st.BagItems) != 3 || !f.busy() {
		t.Fatalf("受理后状态异常: money=%d bag=%+v busy=%t", st.Money, st.BagItems, f.busy())
	}
	f.pl.combat.mu.Lock()
	rows := len(f.pl.combat.cooldowns)
	f.pl.combat.mu.Unlock()
	if rows != 1 {
		t.Fatalf("应写 1 行冷却，得 %d", rows)
	}
	f.finish(t)
	tail := f.take()
	// 尾段：m26 → 清态 → 解锁 → 碎银帧 → 冷却行（0x11 空行 + 4×0x13 逐格）
	// → 扣件。
	if len(tail) != 10 || !bytes.Equal(tail[0], mustHex(t, officialFlowEnd)) ||
		!bytes.Equal(tail[2], mustHex(t, "100102000000030000000100720700000000")) ||
		tail[3][0] != 0x10 || !bytes.Equal(tail[3], playerPropFrame(t, "CapitalType1", entergen.Int64Value(500))) ||
		tail[4][0] != 0x11 || tail[5][0] != 0x13 || tail[8][0] != 0x13 ||
		!bytes.Equal(tail[9], mustHex(t, "1902000300")) {
		t.Fatalf("钱袋尾段帧异常（%d 帧）: % x", len(tail), tail)
	}
	// 冷却行四格：类别 24001（int 格）→ 起 → 止（i64）→ 队 0xFFFF。
	if !bytes.Contains(tail[5], mustHex(t, "c15d0000")) || !bytes.Contains(tail[8], mustHex(t, "ffff")) {
		t.Errorf("冷却行格值异常: % x | % x", tail[5], tail[8])
	}

	// 冷却中再按同类别：静默、不扣件。
	f.s.dispatchCustom(f.pl, useMsg(2, 4, 1))
	if extra := f.take(); len(extra) != 0 || st.Money != 500 || len(st.BagItems) != 3 || f.busy() {
		t.Fatalf("冷却中应零应答: frames=%d money=%d bag=%d busy=%t", len(extra), st.Money, len(st.BagItems), f.busy())
	}
	// 冷却过期后放行。
	f.pl.combat.mu.Lock()
	for i := range f.pl.combat.cooldowns {
		f.pl.combat.cooldowns[i].end = 0
	}
	f.pl.combat.mu.Unlock()
	f.s.dispatchCustom(f.pl, useMsg(2, 4, 1))
	if extra := f.take(); len(extra) != 3 || st.Money != 1000 || len(st.BagItems) != 2 {
		t.Fatalf("冷却过期应放行: frames=%d money=%d bag=%d", len(extra), st.Money, len(st.BagItems))
	}
	f.finish(t)
	f.take()

	// 拒绝面：金袋（币种未建模）/礼盒（效果面未接）/无行/畸形——零应答。
	for _, msg := range []entergen.CustomInbound{useMsg(2, 5, 1), useMsg(2, 6, 1), useMsg(2, 9, 1),
		useMsg(999, 3, 1), useMsg(2, 0, 1), useMsg(2, 3, 0)} {
		f.s.dispatchCustom(f.pl, msg)
	}
	if extra := f.take(); len(extra) != 0 || st.Money != 1000 || len(st.BagItems) != 2 || f.busy() {
		t.Fatalf("拒绝面应零应答不动状态: frames=%d money=%d bag=%d busy=%t", len(extra), st.Money, len(st.BagItems), f.busy())
	}

	// 瞬时使用（功能包无 PrepareTime）：无读条流，入账帧 + 扣件即发。
	st.BagItems = append(st.BagItems, rolestate.BagItem{Slot: 7, ConfigID: "inst_money", ItemType: 37, ViewID: 1, Amount: 1, Seq: 9})
	f.s.dispatchCustom(f.pl, useMsg(2, 7, 1))
	instant := f.take()
	if len(instant) != 2 || !bytes.Equal(instant[0], playerPropFrame(t, "CapitalType1", entergen.Int64Value(1050))) ||
		!bytes.Equal(instant[1], mustHex(t, "1902000700")) || st.Money != 1050 || f.busy() {
		t.Fatalf("瞬时使用异常（%d 帧）: % x money=%d busy=%t", len(instant), instant, st.Money, f.busy())
	}
}

// 堆叠余量：扣 1 件 = 0x19 + 0x18（减堆），行保留。
func TestUseItemStackConsume(t *testing.T) {
	useItemProduct(t)
	f := newItemUseFixture(t)
	st := &f.pl.roleState
	st.BagItems = []rolestate.BagItem{{Slot: 2, ConfigID: "inst_money", ItemType: 37, ViewID: 1, Amount: 3, MaxAmount: 99, Seq: 1}}
	f.s.dispatchCustom(f.pl, useMsg(2, 2, 1))
	frames := f.take()
	// 瞬时使用：碎银帧 → 扣件（0x19）→ 减堆行成对（0x19 + 0x18）。
	if len(frames) != 4 || frames[0][0] != 0x10 || frames[1][0] != 0x19 || frames[2][0] != 0x19 ||
		frames[3][0] != 0x18 {
		t.Fatalf("减堆帧序异常: % x", frames)
	}
	row := frames[3]
	if !frameContains([][]byte{row}, []byte("inst_money")) || uint16(row[3])|uint16(row[4])<<8 != 2 {
		t.Fatalf("减堆行帧异常: % x", row)
	}
	// Amount=2（属性 idx 1901=0x076D，int32）。
	if !bytes.Contains(row, mustHex(t, "6d0702000000")) {
		t.Errorf("减堆行应携 Amount=2: % x", row)
	}
	if len(st.BagItems) != 1 || st.BagItems[0].Amount != 2 || st.Money != 50 {
		t.Fatalf("减堆后状态异常: %+v money=%d", st.BagItems, st.Money)
	}
}

// 入包堆叠上限：按模板行 MaxAmount 叠堆/开新行；放不下整单一件不发。
func TestGrantItemUnitsStackLimit(t *testing.T) {
	useItemProduct(t)
	f := newItemUseFixture(t)
	st := &f.pl.roleState
	if ok := f.s.grantShopItemUnits(f.pl, "money_bag", 3); !ok {
		t.Fatal("发放 3 件不可堆叠钱袋应成功")
	}
	if len(st.BagItems) != 3 || st.BagItems[0].Amount != 1 || st.BagItems[0].MaxAmount != 1 ||
		st.BagItems[2].Slot != 3 {
		t.Fatalf("不可堆叠物应逐件成行: %+v", st.BagItems)
	}
	// 目录未标 MaxAmount（inst_money）：不设限，单行承载。
	f.take()
	if ok := f.s.grantShopItemUnits(f.pl, "inst_money", 5); !ok {
		t.Fatal("发放散碎银两应成功")
	}
	if len(st.BagItems) != 4 || st.BagItems[3].Amount != 5 || st.BagItems[3].MaxAmount != 0 {
		t.Fatalf("未标上限应单行承载: %+v", st.BagItems)
	}
	// 同 configID 再发：补进行内。
	f.s.grantShopItemUnits(f.pl, "inst_money", 2)
	if st.BagItems[3].Amount != 7 || len(st.BagItems) != 4 {
		t.Fatalf("应叠入既有行: %+v", st.BagItems[3])
	}
	// 页满：90 格占满后整单拒。
	f.take()
	for len(st.BagItems) < 90 {
		if !f.s.grantShopItemUnits(f.pl, "money_bag", 1) {
			t.Fatalf("第 %d 行发放失败", len(st.BagItems)+1)
		}
	}
	f.take()
	if f.s.grantShopItemUnits(f.pl, "money_bag", 1) || len(st.BagItems) != 90 {
		t.Fatalf("页满应拒且不动状态: rows=%d", len(st.BagItems))
	}
	if extra := f.take(); len(extra) != 0 {
		t.Fatalf("页满拒不应发帧: %d", len(extra))
	}
	if _, ok := f.s.itemCatalogOf().Lookup("absent"); ok {
		t.Error("未知物品不应有目录条目")
	}
	if f.s.grantShopItemUnits(f.pl, "absent", 1) {
		t.Error("未知物品不应发放")
	}
}

// gm item / 邮件领附件的单件入包：模板行数据（ItemType/MaxAmount）随
// 目录入档；目录不认识的物品保持裸行。
func TestGrantBagItemCarriesTemplate(t *testing.T) {
	useItemProduct(t)
	f := newItemUseFixture(t)
	st := &f.pl.roleState
	item, push, err := f.s.grantBagItem(f.pl, "money_bag", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if item.ItemType != 37 || item.MaxAmount != 1 || st.BagItems[0].ItemType != 37 || len(push) != 2 {
		t.Fatalf("入包行应携模板行数据: %+v push=%d", item, len(push))
	}
	bare, _, err := f.s.grantBagItem(f.pl, "unknown_item", 2, 4)
	if err != nil || bare.ItemType != 0 || bare.MaxAmount != 0 || bare.Amount != 2 || bare.ViewID != 4 {
		t.Fatalf("目录外物品应保持裸行: %+v err=%v", bare, err)
	}
}
