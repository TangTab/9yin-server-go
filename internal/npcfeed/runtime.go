package npcfeed

// runtime.go 运行时生成的 NPC：不在场景摆放表里，由服务端按模板现合成
//（GM 训练木桩）。物化帧、战斗体、交互查找与静态 NPC 走同一条链，区别
// 只在对象号现分、血量可自定义、死亡后不重生。

import (
	"fmt"
	"os"
	"strings"
)

// extrasEnabled NINEYIN_RUNTIME_EXTRAS=off 关停运行时对象 0x28 帧的追加段
// （MaxHP/HP）——真机 A/B 开关：生成的木桩不显示时对照，分清是追加段不被
// 客户端接受，还是对象号 / 建档环节的问题。关停后血量仍在服务端生效，
// 只是客户端血条按本地表。
func extrasEnabled() bool { return os.Getenv("NINEYIN_RUNTIME_EXTRAS") != "off" }

// SpawnFromTemplate 按模板 ID 合成一条待物化的摆放。
//
// overrides 叠在模板行上、combatOverrides 叠在战斗行上（列名 → 值）：官方
// 模板自带的值在生成场景未必适用。maxHP > 0 时血量取它（Spawn.MaxHP），
// 否则取战斗行 MaxLife。对象号由调用方分配。
func SpawnFromTemplate(templates map[string]TemplateRow, combat map[string]CombatRow, configID string,
	overrides, combatOverrides map[string]string, maxHP int32, x, y, z, orient float32) (Spawn, error) {
	tpl, ok := templates[configID]
	if !ok {
		return Spawn{}, fmt.Errorf("npcfeed: 模板 %s 缺席", configID)
	}
	attrs := mergeRows(tpl.Attrs, overrides)
	script := attrs["script"]
	if script == "" {
		script = tpl.Script
	}
	spawn := Spawn{ConfigID: configID, File: "runtime", Script: script,
		X: x, Y: y, Z: z, Orient: orient,
		BornAction: bornActionOf(nil, attrs), Template: attrs,
		MaxHP: maxHP, Once: true}
	if row, hit := combat[strings.TrimSpace(attrs["FightID"])]; hit {
		spawn.Combat = &CombatRow{ID: row.ID, Attrs: mergeRows(row.Attrs, combatOverrides)}
	}
	return spawn, nil
}

// mergeRows 克隆 base 后叠上 overlay（base 为 nil 也给非 nil 表——Template
// 约定非 nil）。
func mergeRows(base, overlay map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overlay))
	for name, value := range base {
		out[name] = value
	}
	for name, value := range overlay {
		out[name] = value
	}
	return out
}
