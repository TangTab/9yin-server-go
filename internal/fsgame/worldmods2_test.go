package fsgame

import (
	"testing"
)

// assemble14 装配十四批一百一十五模块（player.xml includes 链自带
// honor/title/punisher/exile/in_water/force_repute/force_state/unenthrall）。
func assemble14(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	k, mods := assemble13(t)
	var r Registry
	if err := RegisterFourteenthBatch(&r); err != nil {
		t.Fatal(err)
	}
	more, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return k, append(mods, more...)
}

// TestFourteenthBatchAssemble 十四期装配序 + schema 抽查。
func TestFourteenthBatchAssemble(t *testing.T) {
	_, mods := assemble14(t)
	want := []string{"HonorValueModule", "AirportModule", "PunisherModule",
		"UnderWaterModule", "CampModule", "ReputeModule", "TitleModule",
		"WorldRankModule", "ExileModule", "NewsModule", "ForceStateModule",
		"ForceModule", "MovieModule", "UnenthrallModule"}
	var got []string
	for _, m := range mods {
		for _, w := range want {
			if m.Name() == w {
				got = append(got, w)
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("装配缺模块: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
}

// TestHonorRankAndPunisher 荣誉升阶 + 排行镜像 + 怒火增减衰减。
func TestHonorRankAndPunisher(t *testing.T) {
	k, _ := assemble14(t)
	hm := module2[*HonorValueModule](t, k, "HonorValueModule")
	wr := module2[*WorldRankModule](t, k, "WorldRankModule")

	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	_ = k.SetInt(p, "PlayerHonor", 900)
	_ = k.SetInt(p, "UpgradeHonor", 1000)
	_ = k.SetInt(p, "RankLevel", 0)
	if !hm.AddHonor(k, p, 200) {
		t.Fatal("荣誉入账失败")
	}
	if o.GetInt("RankLevel") != 1 || o.GetInt("PlayerHonor") != 1100 {
		t.Fatalf("升阶异常: rank=%d honor=%d", o.GetInt("RankLevel"), o.GetInt("PlayerHonor"))
	}
	// 上限钳。
	_ = k.SetInt(p, "CanAddHonorPoint", 50)
	hm.AddHonor(k, p, 100)
	if o.GetInt("PlayerHonor") != 1150 {
		t.Fatalf("上限钳异常: %d", o.GetInt("PlayerHonor"))
	}
	// 排行镜像（荣誉临界同步域 0）。
	if wr.ranks[0][p] != 1150 {
		t.Fatalf("排行镜像异常: %d", wr.ranks[0][p])
	}
	// 怒火：cmd 601 +30 → 30；衰减 1 拍 → 29；死亡清零。
	k.DispatchIntCommand("player", 601, p, NullPersistID, VarList{int32(30)})
	if o.GetInt("RFC") != 30 {
		t.Fatalf("怒火异常: %d", o.GetInt("RFC"))
	}
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("RFC") != 29 {
		t.Fatalf("怒火衰减异常: %d", o.GetInt("RFC"))
	}
	_ = k.SetInt(p, "Dead", 1)
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("RFC") != 0 {
		t.Fatal("死亡应清怒火")
	}
}

// TestTitleLifecycle 称号闭环：解锁（title_rec）→ RecHook 联动 → 佩戴 →
// 限时到期移除。
func TestTitleLifecycle(t *testing.T) {
	k, _ := assemble14(t)
	tm := module2[*TitleModule](t, k, "TitleModule")
	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)

	if !tm.NxAddPlayerTitle(k, p, 7, 0) {
		t.Fatal("解锁失败")
	}
	r := o.Record("title_rec")
	if r.Rows() != 1 || r.Row(0).Int(0) != 7 {
		t.Fatalf("解锁行异常: %v", r.Row(0))
	}
	// RecHook 联动（记录通知链——append 触发 rec:1）。
	if !hasStr(tm.Events(), "rec:1") {
		t.Fatalf("RecHook 联动缺失: %v", tm.Events())
	}
	if !tm.NxSetPlayerTitle(k, p, 7) || o.GetInt("TitleID") != 7 {
		t.Fatal("佩戴失败")
	}
	if tm.NxSetPlayerTitle(k, p, 9) {
		t.Fatal("未持有不可佩戴")
	}
	// 限时称号：2 拍到期移除 + 佩戴中同卸。
	tm.NxAddPlayerTitle(k, p, 8, 2)
	tm.NxSetPlayerTitle(k, p, 8)
	k.Bus().TickHeartBeats(k, 1)
	if r.Rows() != 2 {
		t.Fatalf("未到期不应移除: %d", r.Rows())
	}
	k.Bus().TickHeartBeats(k, 1)
	if r.Rows() != 1 || o.GetInt("TitleID") != 0 || !hasStr(tm.Events(), "overtime") {
		t.Fatalf("到期移除异常: rows=%d title=%d %v", r.Rows(), o.GetInt("TitleID"), tm.Events())
	}
}

// TestWaterExileAndForce 水下呼吸/溺伤 + 流放刑期/释放 + 势力声望链。
func TestWaterExileAndForce(t *testing.T) {
	k, _ := assemble14(t)
	uw := module2[*UnderWaterModule](t, k, "UnderWaterModule")
	em := module2[*ExileModule](t, k, "ExileModule")
	rm := module2[*ReputeModule](t, k, "ReputeModule")
	fm := module2[*ForceModule](t, k, "ForceModule")
	cm := module2[*CampModule](t, k, "CampModule")

	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	// 水下：呼吸 100，速度 10/拍 → 10 拍耗尽（耗尽当拍即起溺伤——两泵
	// 同拍序：先耗后判）。
	_ = o.Set("CurUnderWaterValue", float32(100))
	_ = o.Set("UnderWaterSpeed", float32(10))
	_ = k.SetInt(p, "HP", 100)
	k.DispatchCommand("player", 478, p, NullPersistID, nil)
	for i := 0; i < 9; i++ {
		k.Bus().TickHeartBeats(k, 1)
	}
	if o.GetFloat("CurUnderWaterValue") != 10 || len(uw.Events()) != 0 {
		t.Fatalf("呼吸消耗异常: %v %v", o.GetFloat("CurUnderWaterValue"), uw.Events())
	}
	k.Bus().TickHeartBeats(k, 1) // 第 10 拍：归零 + 当拍溺伤
	if o.GetFloat("CurUnderWaterValue") != 0 || o.GetInt("HP") != 95 || !hasStr(uw.Events(), "drown") {
		t.Fatalf("耗尽溺伤异常: breath=%v hp=%d %v", o.GetFloat("CurUnderWaterValue"), o.GetInt("HP"), uw.Events())
	}
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("HP") != 90 {
		t.Fatalf("持续溺伤异常: %d", o.GetInt("HP"))
	}
	// 出水回复。
	k.DispatchCommand("player", 479, p, NullPersistID, nil)
	k.Bus().TickHeartBeats(k, 1)
	if o.GetFloat("CurUnderWaterValue") != 5 {
		t.Fatalf("出水回复异常: %v", o.GetFloat("CurUnderWaterValue"))
	}
	// 流放：通缉值 120 → 重犯档；监禁 1 天；刑期到满释放。
	_ = k.SetInt(p, "ArrestValue", 120)
	if o.GetInt("ExileStatus") != 2 || !hasStr(em.Events(), "arrest:120→2") {
		t.Fatalf("通缉档异常: %d %v", o.GetInt("ExileStatus"), em.Events())
	}
	if !em.NxSetStatus(k, p, 1) {
		t.Fatal("监禁失败")
	}
	em.hbCheckLockTime(k, p, 0) // 未到期不释放
	if o.GetInt("ExileStatus") != 2 {
		t.Fatal("未到期不应释放")
	}
	_ = o.Set("ExileLockDate", float64(1)) // 锚到远古 → 已到期
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("ExileStatus") != 0 || o.GetInt("ArrestValue") != 0 || !hasStr(em.Events(), "release") {
		t.Fatalf("释放异常: %d %v", o.GetInt("ExileStatus"), em.Events())
	}
	// 势力链：JoinForce → Force 写 + 版本递增 + 阵营查询 + 声望入账。
	_ = k.SetInt(p, "Force", 0)
	_ = k.SetInt(p, "ForceVersion", 0)
	if !fm.JoinForce(k, p, 5) {
		t.Fatal("入势力失败")
	}
	if o.GetInt("Force") != 5 || o.GetInt("ForceVersion") != 1 || cm.Camp(k, p) != 5 {
		t.Fatalf("势力链异常: force=%d ver=%d", o.GetInt("Force"), o.GetInt("ForceVersion"))
	}
	rm.NxAddRepute(k, p, 5, 30)
	if rm.ReputeOf(p, 5) != 30 || o.GetInt("ForceVersion") != 2 {
		t.Fatalf("声望异常: %d ver=%d", rm.ReputeOf(p, 5), o.GetInt("ForceVersion"))
	}
}

// TestAirportMovieNewsUnenthrall 航线传送 + 影片免打 + 公告 + 防沉迷累计。
func TestAirportMovieNewsUnenthrall(t *testing.T) {
	k, _ := assemble14(t)
	am := module2[*AirportModule](t, k, "AirportModule")
	mm := module2[*MovieModule](t, k, "MovieModule")
	nm := module2[*NewsModule](t, k, "NewsModule")
	um := module2[*UnenthrallModule](t, k, "UnenthrallModule")

	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	o.SetPos(0, 0, 0)
	// 航线：激活 → cust 553 直达 (30,40) → 解除。
	if !am.NxActiveAirport(k, p, 30, 40) {
		t.Fatal("航线激活失败")
	}
	k.DispatchCommand("player", 553, p, NullPersistID, nil)
	if x, _, z := o.Pos(); x != 30 || z != 40 {
		t.Fatalf("航线传送异常: %v,%v", x, z)
	}
	if !hasStr(am.Events(), "fly:30,40") {
		t.Fatalf("航线流水缺失: %v", am.Events())
	}
	// 影片：播放 → CantBeAttack 置位 → 结束解除。
	_ = k.SetInt(p, "CantBeAttack", 0)
	if !mm.NxBeiginMovie(k, p, 12) {
		t.Fatal("播放失败")
	}
	if o.GetInt("CantBeAttack") != 1 {
		t.Fatal("播放期免打未置")
	}
	mm.EndMovie(k, p)
	if o.GetInt("CantBeAttack") != 0 || !hasStr(mm.Events(), "movie-end") {
		t.Fatal("结束未解除")
	}
	// 公告。
	nm.NxPublishNews(3, "全服公告：例行维护")
	if !hasStr(nm.Events(), "news:3:全服公告：例行维护") {
		t.Fatalf("公告流水缺失: %v", nm.Events())
	}
	// 防沉迷：累计 +1 秒/拍。
	_ = o.Set("UnenthrallTime", 3*3600-1)
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("UnenthrallTime") != 3*3600 || !hasStr(um.Events(), "fatigue:3h") {
		t.Fatalf("防沉迷累计异常: %d %v", o.GetInt("UnenthrallTime"), um.Events())
	}
}
