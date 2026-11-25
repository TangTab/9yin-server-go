package fssoloworld

// combat_test.go 战斗接入测试：命中节拍、伤害结算与帧序按官服抓包定版
// （docs/official/战斗接入与技能Lua宿主.md）。夹具 = 技能夹具（可拨时钟
// + 下行帧捕获）+ 一只合成怪物（属性已知）+ 可控随机源；战斗事件由
// CombatPump 手动推进。
//
// 断言面是「可读化的帧序」（combatText）：本人属性帧 me:名=值、怪物属性
// 帧 npc:名=值、技能行视图 view<视图>.<行>:名=值、关系表行
// me:rec114…、自定义消息 m<号>(对象,值…)，时间相关的取值归一为 T。

import (
	"encoding/binary"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

// combatFixture 技能夹具 + 合成怪物。
type combatFixture struct {
	*skillFixture
	monster *npcFighter
	scene   string
}

// combatSceneSeq 合成场景序号：每个用例一个独立场景键（数据根里不存在
// 该摆放目录 → 空目录，只放测试怪物；不污染真档场景缓存，用例间也不
// 共享同一只怪物的死亡态）。
var combatSceneSeq int64

// newCombatFixture 玩家站 (100,5,200) 朝 +X（朝向 1.571），怪物在
// (100+dx, 5, 200)；combat 为战斗行属性（MaxLife/Dodge/…）。收招定时器
// 换到战斗时钟上：与命中同钟（CombatPump 驱动），不随真实时间漂移。
func newCombatFixture(t *testing.T, dx float64, combat map[string]string) *combatFixture {
	t.Helper()
	f := newSkillFixture(t)
	if combat == nil {
		combat = map[string]string{}
	}
	if _, ok := combat["MaxLife"]; !ok {
		combat["MaxLife"] = "100"
	}
	for name, value := range map[string]string{"Level": "5", "MeleeWeaponDamageBase": "10",
		"MeleeWeaponDamageRand": "10", "Dodge": "3", "PhysicsAttackHit": "50"} {
		if _, ok := combat[name]; !ok {
			combat[name] = value
		}
	}
	prevRoll := combatRoll
	combatRoll = func() float64 { return 0.99 } // 不闪避、不暴击
	combatAfterFunc = func(d time.Duration, run func()) *time.Timer { f.s.after(d, run); return nil }
	t.Cleanup(func() {
		combatRoll, combatAfterFunc = prevRoll, time.AfterFunc
	})
	t.Setenv("NINEYIN_NPC_AI", "off")
	if lua := f.s.combatLua(); !lua.engine.Enabled() || lua.actions == nil {
		t.Skip("公式引擎或招式节拍产物缺席（go run ./tools/datagen skill action）")
	}
	catalog := f.s.sceneNPCCatalog(fmt.Sprintf("combat_test_%d", atomic.AddInt64(&combatSceneSeq, 1)))
	spawn := npcfeed.Spawn{ObjectID: 0x0E000100, OwnerID: 0x1000, X: float32(100 + dx), Y: 5, Z: 200,
		ConfigID: "test_monster", Script: "AttackNpc", Template: map[string]string{},
		Combat: &npcfeed.CombatRow{ID: "test", Attrs: combat}}
	catalog.combatMu.Lock()
	catalog.spawns = append(catalog.spawns, spawn)
	catalog.combatMu.Unlock()
	f.pl.npcMu.Lock()
	f.pl.npcScene = catalog.dir
	f.pl.npcMu.Unlock()
	f.pl.roleState.HP = 97
	f.pl.roleState.Position = [4]float64{100, 5, 200, math.Pi / 2}
	return &combatFixture{skillFixture: f, monster: f.s.fighterFor(catalog.dir, spawn), scene: catalog.dir}
}

// target 怪物对象键。
func (f *combatFixture) target() *uint64 {
	key := uint64(f.monster.spawn.ObjectID) | uint64(f.monster.spawn.OwnerID)<<32
	return &key
}

func (f *combatFixture) pump(d time.Duration) []string {
	f.advance(d)
	f.s.CombatPump()
	return combatText(f.s, f.pl, f.monster, f.take())
}

// combatText 下行帧 → 可读串。
func combatText(s *Server, pl *Player, monster *npcFighter, frames [][]byte) []string {
	me := pl.wireIdent().orFixed()
	npcID := monster.spawn.ObjectID
	le16 := func(b []byte) int { return int(binary.LittleEndian.Uint16(b)) }
	le32 := func(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
	label := func(obj, own uint32) string {
		switch {
		case obj == me.objectID && own == me.ownerID:
			return "me"
		case obj == npcID:
			return "npc"
		}
		return fmt.Sprintf("o:%d/%d", obj, own)
	}
	out := make([]string, 0, len(frames))
	for _, frame := range frames {
		if len(frame) == 0 {
			continue
		}
		switch frame[0] {
		case 0x10:
			who := label(le32(frame[2:]), le32(frame[6:]))
			if frame[1] == 1 {
				who = fmt.Sprintf("view%d.%d", le32(frame[2:]), le32(frame[6:]))
			}
			for i, offset := 0, 12; i < le16(frame[10:]) && offset+2 <= len(frame); i++ {
				entry, ok := clientdata.PropertyByIndex(le16(frame[offset:]))
				offset += 2
				if !ok {
					out = append(out, who+":<未知属性>")
					break
				}
				var value string
				switch entry.Type {
				case clientdata.WireByte:
					value, offset = fmt.Sprint(frame[offset]), offset+1
				case clientdata.WireWord:
					value, offset = fmt.Sprint(binary.LittleEndian.Uint16(frame[offset:])), offset+2
				case clientdata.WireInt32:
					value, offset = fmt.Sprint(int32(le32(frame[offset:]))), offset+4
				case clientdata.WireInt64:
					value, offset = "T", offset+8
				case clientdata.WireFloat32:
					value, offset = fmt.Sprintf("%.3f", math.Float32frombits(le32(frame[offset:]))), offset+4
				case clientdata.WireString:
					length := int(le32(frame[offset:]))
					value, offset = string(frame[offset+4:offset+4+length-1]), offset+4+length
				case clientdata.WireObject:
					value, offset = label(le32(frame[offset:]), le32(frame[offset+4:])), offset+8
				default:
					value, offset = "?", len(frame)
				}
				out = append(out, fmt.Sprintf("%s:%s=%s", who, entry.Name, value))
			}
		case 0x11, 0x12, 0x13:
			record, _ := clientdata.RecordSchemaByIndex(le16(frame[10:]))
			who := label(le32(frame[2:]), le32(frame[6:]))
			switch frame[0] {
			case 0x11:
				out = append(out, fmt.Sprintf("%s:rec%d+row%d{%s}", who, record.Index, le16(frame[12:]),
					recordCells(record, frame[16:], le16(frame[14:]))))
			case 0x12:
				out = append(out, fmt.Sprintf("%s:rec%d-row%d", who, record.Index, le16(frame[12:])))
			case 0x13:
				row := le16(frame[14:])
				for i, offset := 0, 16; i < le16(frame[12:]) && offset < len(frame); i++ {
					column := int(frame[offset])
					value, width := cellValue(record, column, frame[offset+1:])
					out = append(out, fmt.Sprintf("%s:rec%d[%d.%d]=%s", who, record.Index, row, column, value))
					offset += 1 + width
				}
			}
		case 0x1E:
			message, err := entergen.ParseCustomMessage(frame)
			if err != nil || len(message.Values) == 0 {
				out = append(out, "<自定义帧解码失败>")
				continue
			}
			parts := make([]string, 0, len(message.Values))
			for _, value := range message.Values[1:] {
				switch value.Type {
				case 2:
					parts = append(parts, fmt.Sprintf("i:%d", value.Int32))
				case 3:
					parts = append(parts, "i:T64")
				case 4:
					parts = append(parts, fmt.Sprintf("f:%.3f", value.Float32))
				case 6:
					parts = append(parts, "s:"+value.Text)
				case 8:
					parts = append(parts, label(binary.LittleEndian.Uint32(value.Raw[:]), binary.LittleEndian.Uint32(value.Raw[4:])))
				default:
					parts = append(parts, fmt.Sprintf("tag%d", value.Type))
				}
			}
			out = append(out, fmt.Sprintf("m%d(%s)", message.Values[0].Int32, strings.Join(parts, ",")))
		case 0x1F:
			out = append(out, fmt.Sprintf("loc(%s,%.3f,%.3f,%.3f,%.3f)", label(le32(frame[1:]), le32(frame[5:])),
				math.Float32frombits(le32(frame[9:])), math.Float32frombits(le32(frame[13:])),
				math.Float32frombits(le32(frame[17:])), math.Float32frombits(le32(frame[21:]))))
		case 0x21:
			// AllDest 记录（entergen.ServerAllDestFrame 布局）：位置 / 速度 / 回显槽 / 标志。
			f32 := func(o int) float64 { return float64(math.Float32frombits(le32(frame[o:]))) }
			for j := 0; j < le16(frame[1:]) && 3+(j+1)*44 <= len(frame); j++ {
				at := 3 + j*44
				out = append(out, fmt.Sprintf("alldest(%s,%.3f,%.3f,%.3f,o=%.3f,spd=%.1f,m2=%.1f,m3=%.1f,f=%#x)",
					label(le32(frame[at:]), le32(frame[at+4:])), f32(at+8), f32(at+12), f32(at+16), f32(at+20),
					f32(at+24), f32(at+32), f32(at+36), le32(frame[at+40:])))
			}
		default:
			out = append(out, fmt.Sprintf("op%02X", frame[0]))
		}
	}
	// fight_relation_rec 第 2 列是对钟取值（随夹具时钟变），归一为 T。
	relation := regexp.MustCompile(`(rec114(?:\+row\d+\{c0=[^,]*,c1=|\[\d+\.1\]=))\d+`)
	for i, text := range out {
		out[i] = relation.ReplaceAllString(text, "${1}T")
	}
	return out
}

// recordCells 整行加帧的各列。
func recordCells(schema clientdata.RecordSchema, frame []byte, rows int) string {
	var parts []string
	offset := 0
	for r := 0; r < rows && offset < len(frame); r++ {
		for column := range schema.ColumnTypes {
			if offset >= len(frame) {
				break
			}
			value, width := cellValue(schema, column, frame[offset:])
			parts = append(parts, fmt.Sprintf("c%d=%s", column, value))
			offset += width
		}
	}
	return strings.Join(parts, ",")
}

// cellValue 按 schema 列型读一格（时间类与对象列归一）。
func cellValue(schema clientdata.RecordSchema, column int, frame []byte) (string, int) {
	if column >= len(schema.ColumnTypes) || len(frame) < 2 {
		return "?", 0
	}
	switch schema.ColumnTypes[column] {
	case 1:
		if len(frame) < 1 {
			break
		}
		return fmt.Sprint(frame[0]), 1
	case 2:
		return fmt.Sprint(binary.LittleEndian.Uint16(frame)), 2
	case 3:
		if len(frame) < 4 {
			break
		}
		return fmt.Sprint(int32(binary.LittleEndian.Uint32(frame))), 4
	case 4:
		if len(frame) < 8 {
			break
		}
		return "T", 8
	case 9:
		if len(frame) < 8 {
			break
		}
		return fmt.Sprintf("obj(%d/%d)", binary.LittleEndian.Uint32(frame), binary.LittleEndian.Uint32(frame[4:])), 8
	}
	return "?", 0
}

func (f *combatFixture) hp() int {
	f.monster.mu.Lock()
	defer f.monster.mu.Unlock()
	return f.monster.hp
}
