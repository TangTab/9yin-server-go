// npc_staticprop.go：NPC 实力包表（share/npc/staticprop.ini）装载与守方
// 战斗数值推导（2026-11-08 NPC 属性块迁移批次）。
//
// 背景：官方 base block 0..98 是场景面板块，不含防御/封劲/闪避——官服
// NPC 战斗数值按"实力包"（模板列 PowerNo@60）在服务端推导后写入视图块
// （PhyDef@584/StiDef@586/JujDef@587/NegDef@588/MasDef@589/Dodge@562/
// VaDef@573）。staticprop.ini 是该推导的成长基数表（745 节，Go/Java/
// 客户端 Lua 此前均未消费）：每节 17 键——五维 Base（Str/Dex/Ing/Spi/
// Sta）、HPBase（浮点）、MPBase、五系 DefBase、PI/AI 防御 Base、
// RandomPropType、VarLevel；[-1] 为兜底节。
//
// 组合规则无官方参照，按「属性 = Base × Level」推断（锚：§七 白狼封劲
// 20–40 ≈ StaBase1×L30）。推导集中在 deriveNPCCombatStats 一处，可调。
package main

import (
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

// staticPropRow 是单个实力包的成长基数（每级增量）。
type staticPropRow struct {
	strBase, dexBase, ingBase, spiBase, staBase float64
	hpBase, mpBase                              float64
	phyDefBase, stiDefBase, jujDefBase          float64
	negDefBase, masDefBase                      float64
	piPhyDefBase, piMagicDefBase                float64
	aiPhyDefBase, aiMagicDefBase                float64
	randomPropType                              float64
	varLevel                                    float64
}

// staticPropFallbackID 是 [-1] 兜底节（模板 PowerNo 空/缺时使用）。
const staticPropFallbackID = int32(-1)

// installedStaticPropTable 是启动期装载的实力包表（nil＝表缺失，推导按
// 零值——战斗链不受阻）。
var installedStaticPropTable map[int32]staticPropRow

// staticPropPath 是实力包表路径（与 skill 资源同根 resources/modern/share）。
func staticPropPath() string {
	return filepath.Join(defaultModernShareRoot, "npc", "staticprop.ini")
}

// loadStaticPropTable 解析 staticprop.ini：段号→成长基数。优先资源仓
// （res-base.rep），仓缺失该文件时回退 os.Open 散文件（部署端
// resources/modern 为解包树）；两者都失败返回 nil（守方按零值推导，
// 战斗链不受阻），坏键跳过。
func loadStaticPropTable() map[int32]staticPropRow {
	table, err := loadINISections(staticPropPath())
	if err != nil {
		file, openErr := os.Open(staticPropPath())
		if openErr != nil {
			log.Printf("NPC 实力包表不可用（守方数值按零值推导）：store=%v os=%v", err, openErr)
			return nil
		}
		defer file.Close()
		table, err = scanINISections(file)
		if err != nil {
			log.Printf("NPC 实力包表解析失败（守方数值按零值推导）：%v", err)
			return nil
		}
	}
	return parseStaticPropTable(table)
}

// parseStaticPropTable 是段表→实力包基数的纯解析（测试直读仓内文件用）。
func parseStaticPropTable(table iniTable) map[int32]staticPropRow {
	result := make(map[int32]staticPropRow, len(table))
	for section, fields := range table {
		id, err := strconv.ParseInt(section, 10, 32)
		if err != nil {
			continue
		}
		result[int32(id)] = staticPropRow{
			strBase:       iniFieldFloat(fields, "StrBase"),
			dexBase:       iniFieldFloat(fields, "DexBase"),
			ingBase:       iniFieldFloat(fields, "IngBase"),
			spiBase:       iniFieldFloat(fields, "SpiBase"),
			staBase:       iniFieldFloat(fields, "StaBase"),
			hpBase:        iniFieldFloat(fields, "HPBase"),
			mpBase:        iniFieldFloat(fields, "MPBase"),
			phyDefBase:    iniFieldFloat(fields, "PhyDefBase"),
			stiDefBase:    iniFieldFloat(fields, "StiDefBase"),
			jujDefBase:    iniFieldFloat(fields, "JujDefBase"),
			negDefBase:    iniFieldFloat(fields, "NegDefBase"),
			masDefBase:    iniFieldFloat(fields, "MasDefBase"),
			piPhyDefBase:  iniFieldFloat(fields, "PIPhyDefBase"),
			piMagicDefBase: iniFieldFloat(fields, "PIMagicDefBase"),
			aiPhyDefBase:  iniFieldFloat(fields, "AIPhyDefBase"),
			aiMagicDefBase: iniFieldFloat(fields, "AIMagicDefBase"),
			randomPropType: iniFieldFloat(fields, "RandomPropType"),
			varLevel:       iniFieldFloat(fields, "VarLevel"),
		}
	}
	return result
}

func iniFieldFloat(fields []iniField, key string) float64 {
	value, err := strconv.ParseFloat(iniValue(fields, key), 64)
	if err != nil {
		return 0
	}
	return value
}

// npcCombatStats 是 NPC 的守方战斗数值快照（注册期推导一次，静态）。
type npcCombatStats struct {
	phyDef, stiDef, jujDef, negDef, masDef int32 // 分系防御
	evasion                                int32 // 闪避值
	// resilience（封劲）暂不推导：玩家侧暴击值未成长（默认 16），接入
	// NPC 封劲（StaBase×Level≈30）会把暴击率清零——待装备/宝物批把玩家
	// 暴击值抬到官服量级（数百+）后一并接通。
}

// deriveNPCCombatStats 按「Base × Level」推导守方数值（规则集中此处，
// 可调）。powerNo 取自模板列 PowerNo（缺省/查不到→[-1] 兜底节）；level
// 取自模板列 Level（缺省 1）。表缺失时全零（保持旧行为）。
func deriveNPCCombatStats(powerNo, level int32) npcCombatStats {
	if level <= 0 {
		level = 1
	}
	table := installedStaticPropTable
	if table == nil {
		return npcCombatStats{}
	}
	row, ok := table[powerNo]
	if !ok {
		row, ok = table[staticPropFallbackID]
		if !ok {
			return npcCombatStats{}
		}
	}
	grow := func(base float64) int32 {
		return int32(math.Round(base * float64(level)))
	}
	return npcCombatStats{
		phyDef:  grow(row.phyDefBase),
		stiDef:  grow(row.stiDefBase),
		jujDef:  grow(row.jujDefBase),
		negDef:  grow(row.negDefBase),
		masDef:  grow(row.masDefBase),
		evasion: grow(row.dexBase),
	}
}
