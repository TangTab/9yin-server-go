package npcfeed

// runtime_test.go 运行时生成对象（runtime.go）：模板 / 战斗行列覆写、自定义
// 血量进 0x28 帧、静态摆放的帧不受影响。

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestSpawnFromTemplateMergesOverrides(t *testing.T) {
	templates := map[string]TemplateRow{
		"mon_test": {ID: "mon_test", Script: "AttackNpc", Attrs: map[string]string{
			"ID": "mon_test", "script": "AttackNpc", "Resource": `npc\x`, "FightID": "763",
			"CantBeKilled": "1", "CantAttack": "0", "BornActionRec": "stand",
		}},
	}
	combat := map[string]CombatRow{"763": {ID: "763", Attrs: map[string]string{"ID": "763", "MaxLife": "2530", "Dodge": "126"}}}

	spawn, err := SpawnFromTemplate(templates, combat, "mon_test",
		map[string]string{"CantAttack": "1", "Name": "训练木桩"}, map[string]string{"Dodge": "0"}, 500000, 1, 2, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"CantAttack": "1", "Name": "训练木桩", // 覆写生效
		"CantBeKilled": "1", "Resource": `npc\x`, "FightID": "763", // 模板其余列保留
	} {
		if got := spawn.Template[name]; got != want {
			t.Errorf("模板列 %s = %q, want %q", name, got, want)
		}
	}
	if spawn.Script != "AttackNpc" {
		t.Errorf("Script = %q, want AttackNpc", spawn.Script)
	}
	if spawn.Combat == nil || spawn.Combat.Attrs["Dodge"] != "0" || spawn.Combat.Attrs["MaxLife"] != "2530" {
		t.Errorf("战斗行覆写不符: %+v", spawn.Combat)
	}
	if spawn.MaxHP != 500000 || !spawn.Once || spawn.BornAction != "stand" {
		t.Errorf("运行时字段不符: MaxHP=%d Once=%t BornAction=%q", spawn.MaxHP, spawn.Once, spawn.BornAction)
	}
	if spawn.X != 1 || spawn.Y != 2 || spawn.Z != 3 || spawn.Orient != 4 || spawn.ConfigID != "mon_test" {
		t.Errorf("摆放字段不符: %+v", spawn)
	}
	// 源表不被改写。
	if templates["mon_test"].Attrs["CantAttack"] != "0" || combat["763"].Attrs["Dodge"] != "126" {
		t.Error("SpawnFromTemplate 改写了源表")
	}
	if _, err := SpawnFromTemplate(templates, combat, "mon_missing", nil, nil, 0, 0, 0, 0, 0); err == nil {
		t.Error("模板缺席应报错")
	}
	// 没有战斗行的模板：Combat 为空，自定义血量仍由 MaxHP 承载。
	bare, err := SpawnFromTemplate(map[string]TemplateRow{
		"mon_bare": {ID: "mon_bare", Script: "AttackNpc", Attrs: map[string]string{"ID": "mon_bare"}},
	}, combat, "mon_bare", nil, nil, 7, 0, 0, 0, 0)
	if err != nil || bare.Combat != nil || bare.MaxHP != 7 || bare.Template["ID"] != "mon_bare" {
		t.Errorf("无战斗行模板: err=%v combat=%v MaxHP=%d", err, bare.Combat, bare.MaxHP)
	}
}

func TestFrame0x28RuntimeExtras(t *testing.T) {
	base := Spawn{
		ObjectID: ObjectIDAt(0), OwnerID: OwnerIDOf(ObjectIDAt(0)),
		ConfigID: "mon_test",
		Template: map[string]string{"ID": "mon_test", "Resource": `npc\x`},
	}
	plain := decode0x28Fields(t, base.Frame0x28())
	for _, name := range []string{"MaxHP", "HP"} {
		if _, ok := plain[name]; ok {
			t.Fatalf("静态摆放的帧不应带 %s", name)
		}
	}

	base.MaxHP, base.Once = 500000, true
	fields := decode0x28Fields(t, base.Frame0x28())
	for _, name := range []string{"MaxHP", "HP"} {
		if got := int32(binary.LittleEndian.Uint32(fields[name])); got != 500000 {
			t.Errorf("%s = %d, want 500000", name, got)
		}
	}
	// 追加段只增不改：其余字段逐字节同静态帧。
	for name, value := range plain {
		if !bytes.Equal(fields[name], value) {
			t.Errorf("追加段改动了既有字段 %s", name)
		}
	}
	// 真机 A/B 开关：关停后帧回到不带追加段的形态。
	t.Setenv("NINEYIN_RUNTIME_EXTRAS", "off")
	if off := decode0x28Fields(t, base.Frame0x28()); len(off) != len(plain) {
		t.Errorf("关停追加段后字段数 = %d, want %d（同静态帧）", len(off), len(plain))
	}
}
