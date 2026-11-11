package main

// 骑乘技能受理簇（官服 official_players 抓包定版，2026-11-11）。
//
// 马背技能栏（客户端 form_main_shortcut_ride，引擎 239 槽 server_start_ride
// 打开）按键走标准 C2S 211 USE_SKILL 通路，技能 ID 即 RideSkillID（疾奔
// riding_jiben / 冲刺 riding_chongci / 解除坐骑 riding_dismount 等）。官服
// 观战抓包（official_players out2 timeline）三次 jiben + 一次 chongci 实证
// 服务端只回「受理簇」，无 msg20/284/CD 行/buff 帧：
//
//	property CurSkillTarget = 自己
//	[chongci] custom 24 [obj 自己, 54, "hit", 窄串 HitEffect, obj 自己]
//	custom 205 [obj 自己, 窄串 技能ID, obj 自己]（skillHitConfirmFrame 同构）
//	property CurSkillTarget = 0-0
//
// msg24 54 帧的 HitEffect 取自四表链 skill_new → skill_static →
// skill_normal_varprop(Flow) → flow_config[Flow].HitEffect（chongci flow
// 439 = att_bossclone002_shouji01，抓包逐位一致；jiben flow 1216 无
// HitEffect → 官服三验均无 54 帧）。速度提升与技能栏 CD 显示由客户端本地
// 结算（骑乘移动客户端权威；槽栏 grid:SetCoolType(CoolDownCategory) 本地
// 起表），服务端不补帧——effects_data 的 buf_riding_chongci 1000ms 是 INI
// 约定推导（生成文件），抓包无 m169/槽帧实证，照官服方言不发。

import (
	"log"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// rideSkillDialect 单骑乘技能的表链方言。
type rideSkillDialect struct {
	hitEffect string // flow_config[Flow].HitEffect（空=受理簇不带 msg24 54 帧）
	cooldown  int32  // varprop CoolDownTime（毫秒；日志/GM 回显用，不下发）
	flow      int32  // flow_config 行号（日志用）
}

// rideSkillTables 四表链进程级懒装载（npcSkillActions 同款；资源仓缺表
// 降级空方言=受理簇恒 205 形态，行为与旧版静默一致）。
var rideSkillTables = sync.OnceValue(func() map[string]rideSkillDialect {
	load := func(path string) (iniTable, bool) {
		table, err := loadINISections(path)
		if err != nil {
			log.Printf("ride skill dialect resource %s unavailable: %v", path, err)
			return nil, false
		}
		return table, true
	}
	newTable, ok := load(filepath.Join(modernSkillRoot, "skill_new.ini"))
	if !ok {
		return map[string]rideSkillDialect{}
	}
	staticTable, _ := load(filepath.Join(modernSkillRoot, "skill_static.ini"))
	varTable, _ := load(filepath.Join(modernSkillRoot, "skill_normal_varprop.ini"))
	flowTable, _ := load(filepath.Join(defaultModernShareRoot, "rule", "flow_config.ini"))
	dialects := resolveRideSkillDialects(newTable, staticTable, varTable, flowTable)
	log.Printf("ride skill dialects ready skills=%d", len(dialects))
	return dialects
})

// resolveRideSkillDialects 四表链解析：skill_new[技能].StaticData →
// skill_static[StaticData].MinVarPropNo → skill_normal_varprop[行].
// Flow/CoolDownTime → flow_config[Flow].HitEffect（生产懒装载与测试
// 夹具共用内核）。
func resolveRideSkillDialects(newTable, staticTable, varTable, flowTable iniTable) map[string]rideSkillDialect {
	dialects := make(map[string]rideSkillDialect)
	if len(newTable) == 0 {
		return dialects
	}
	key := func(v int32) string { return strconv.FormatInt(int64(v), 10) }
	for skillID, newFields := range newTable {
		staticFields := staticTable[key(iniInt(newFields, "StaticData"))]
		if len(staticFields) == 0 {
			continue
		}
		varFields := varTable[key(iniInt(staticFields, "MinVarPropNo"))]
		if len(varFields) == 0 {
			continue
		}
		flow := iniInt(varFields, "Flow")
		if flow <= 0 {
			continue
		}
		dialects[skillID] = rideSkillDialect{
			hitEffect: iniValue(flowTable[key(flow)], "HitEffect"),
			cooldown:  iniInt(varFields, "CoolDownTime"),
			flow:      flow,
		}
	}
	return dialects
}

// rideSkillDialectFor 取单技能方言（未索引/缺表 → 零值=纯 205 簇）。
func rideSkillDialectFor(skillID string) rideSkillDialect {
	return rideSkillTables()[skillID]
}

// rideSkillHitEffectFrame 即 msg24 骑乘命中动作帧（官服 chongci 实证）：
// [24, obj(自己), 54, 窄串"hit", 窄串 HitEffect, obj(自己)]。
func rideSkillHitEffectFrame(effect string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(playerObjectID, playerOwnerID),
		customInt(54),
		customString("hit"),
		customString(effect),
		customObject(playerObjectID, playerOwnerID),
	)
}

// writeRideSkillCastCluster 官方骑乘技能受理簇（四帧序：置目标 → 可选
// 54 命中帧 → 205 广播 → 清目标；簇内两属性帧仅过线，不动 playerActor
// 持久目标位——官服同拍置清，快照无痕）。
func writeRideSkillCastCluster(link sceneMessageConnection, skillID string, dialect rideSkillDialect) error {
	write := func(frame []byte, err error) error {
		if err != nil {
			return err
		}
		return link.WriteFrame(frame)
	}
	if err := write(rushPropertyFrame(propCurSkillTarget, "CurSkillTarget",
		clientdata.ObjectValue(playerObjectID, playerOwnerID))); err != nil {
		return err
	}
	if dialect.hitEffect != "" {
		if err := write(rideSkillHitEffectFrame(dialect.hitEffect)); err != nil {
			return err
		}
	}
	if err := write(skillHitConfirmFrame(playerObjectID, playerOwnerID, skillID,
		playerObjectID, playerOwnerID)); err != nil {
		return err
	}
	return write(rushPropertyFrame(propCurSkillTarget, "CurSkillTarget",
		clientdata.ObjectValue(0, 0)))
}

// emitRideSkillCast 骑乘技能受理（mountSkillGate 放行后调用）：解析表链
// 方言并写受理簇。dismount 族不进此路（闸内转下马收尾）。
func (m *mountSession) emitRideSkillCast(skillID string) error {
	dialect := rideSkillDialectFor(skillID)
	log.Printf("%s: ride skill cast id=%s hitEffect=%q flow=%d cd=%dms",
		m.remote, skillID, dialect.hitEffect, dialect.flow, dialect.cooldown)
	return writeRideSkillCastCluster(m.link, skillID, dialect)
}
