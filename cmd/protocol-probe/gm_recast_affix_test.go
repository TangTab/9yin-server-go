package main

import (
	"strings"
	"testing"
)

// GM 精确包号词条操作域（gm_recast_affix.go）：PropBuilt 反解、
// add/clear 端到端（recastConfirm 穿戴分支同链）与分发。

func TestPropBuiltToAffixEntries(t *testing.T) {
	if entries := propBuiltToAffixEntries(""); entries != nil {
		t.Fatalf("empty built entries=%v", entries)
	}
	entries := propBuiltToAffixEntries("1,13040;2,13234")
	if len(entries) != 2 || entries[0] != (gmAffixEntry{Pack: 13040, Kind: 1}) ||
		entries[1] != (gmAffixEntry{Pack: 13234, Kind: 2}) {
		t.Fatalf("entries=%v", entries)
	}
	// 畸形段：空段/非数字/负包号丢弃；kind 越界按 1 兜底。
	entries = propBuiltToAffixEntries("9,13040;;abc;3,-5;0,42")
	if len(entries) != 2 || entries[0] != (gmAffixEntry{Pack: 13040, Kind: 1}) ||
		entries[1] != (gmAffixEntry{Pack: 42, Kind: 1}) {
		t.Fatalf("malformed entries=%v", entries)
	}
}

func TestApplyRecastAddAffix(t *testing.T) {
	h, conn, player := newRecastTestSession(t) // 槽 22 已穿 blade_test，propBuilt="1,13040;1,13041"

	// skill 追加：13234 族双表同号包须落 kind=2（招式威力显示）。
	text, err := applyRecastAddAffix(h, 22, []int32{13234, 13236}, "skill")
	if err != nil {
		t.Fatalf("add skill: %v", err)
	}
	if !strings.Contains(text, "追加词条 2 条（共 4 条）") {
		t.Fatalf("text=%q", text)
	}
	row, _ := player.wornItemAt(22)
	if row.propBuilt != "1,13040;1,13041;2,13234;2,13236" {
		t.Fatalf("propBuilt=%q", row.propBuilt)
	}
	if len(conn.Frames()) == 0 {
		t.Fatalf("wear frames not pushed")
	}

	// prop 追加在最新行之上继续叠加。
	if _, err := applyRecastAddAffix(h, 22, []int32{13042}, "prop"); err != nil {
		t.Fatalf("add prop: %v", err)
	}
	row, _ = player.wornItemAt(22)
	if !strings.HasSuffix(row.propBuilt, ";1,13042") || len(strings.Split(row.propBuilt, ";")) != 5 {
		t.Fatalf("propBuilt=%q", row.propBuilt)
	}

	// 未穿槽拒绝。
	if _, err := applyRecastAddAffix(h, 3, []int32{13040}, "prop"); err == nil {
		t.Fatalf("empty slot should fail")
	}
}

func TestApplyRecastClearAffix(t *testing.T) {
	h, _, player := newRecastTestSession(t)

	text, err := applyRecastClearAffix(h, 22)
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if !strings.Contains(text, "已清空槽 22 词条（原 2 条）") {
		t.Fatalf("text=%q", text)
	}
	row, _ := player.wornItemAt(22)
	if row.propBuilt != "" {
		t.Fatalf("propBuilt=%q want empty", row.propBuilt)
	}

	// 清空后再清（原 0 条）与未穿槽拒绝。
	if text, err := applyRecastClearAffix(h, 22); err != nil || !strings.Contains(text, "原 0 条") {
		t.Fatalf("re-clear text=%q err=%v", text, err)
	}
	if _, err := applyRecastClearAffix(h, 44); err == nil {
		t.Fatalf("empty slot should fail")
	}
}

func TestApplyRecastAffixCommandDispatch(t *testing.T) {
	h, _, _ := newRecastTestSession(t)
	if _, err := applyRecastAffixCommand(h, "recast_add_affix", 22, []int32{13040}, "prop"); err != nil {
		t.Fatalf("dispatch add: %v", err)
	}
	if _, err := applyRecastAffixCommand(h, "recast_unknown", 22, nil, ""); err == nil {
		t.Fatalf("unknown action should fail")
	}
}
