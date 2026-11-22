package fssoloworld

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// gmPropWhiteList gm set 白名单：字段名 → roleState 落点 + 热发属性
//（与 entergen.basePropertyFrames 同名同型；money 对拍 2433 线
// CapitalType1）。HP/MP/SP 同时刷新 Max 语义（0x10 两属性一帧）。
type gmPropSpec struct {
	apply func(pl *Player, v int64)
	frame func(pl *Player) ([]byte, error)
}

var gmPropWhiteList = map[string]gmPropSpec{
	"str":    {apply: func(pl *Player, v int64) { pl.roleState.Str = int(v) }, frame: propFrameInt32("Str", func(pl *Player) int32 { return int32(pl.roleState.Str) })},
	"dex":    {apply: func(pl *Player, v int64) { pl.roleState.Dex = int(v) }, frame: propFrameInt32("Dex", func(pl *Player) int32 { return int32(pl.roleState.Dex) })},
	"ing":    {apply: func(pl *Player, v int64) { pl.roleState.Ing = int(v) }, frame: propFrameInt32("Ing", func(pl *Player) int32 { return int32(pl.roleState.Ing) })},
	"spi":    {apply: func(pl *Player, v int64) { pl.roleState.Spi = int(v) }, frame: propFrameInt32("Spi", func(pl *Player) int32 { return int32(pl.roleState.Spi) })},
	"sta":    {apply: func(pl *Player, v int64) { pl.roleState.Sta = int(v) }, frame: propFrameInt32("Sta", func(pl *Player) int32 { return int32(pl.roleState.Sta) })},
	"parry":  {apply: func(pl *Player, v int64) { pl.roleState.Parry = int(v) }, frame: propFrameInt32("MaxParry", func(pl *Player) int32 { return int32(pl.roleState.Parry) })},
	"money":  {apply: func(pl *Player, v int64) { pl.roleState.Money = v }, frame: moneyFrame},
}

// hp/mp/sp：值与 Max 同步（0x10 一帧两属性）。
func init() {
	for name, both := range map[string][2]string{
		"hp": {"HP", "MaxHP"}, "mp": {"MP", "MaxMP"}, "sp": {"SP", "MaxSP"},
	} {
		cur, max := both[0], both[1]
		field := map[string]func(pl *Player) *int{
			"hp": func(pl *Player) *int { return &pl.roleState.HP },
			"mp": func(pl *Player) *int { return &pl.roleState.MP },
			"sp": func(pl *Player) *int { return &pl.roleState.SP },
		}[name]
		gmPropWhiteList[name] = gmPropSpec{
			apply: func(pl *Player, v int64) { *field(pl) = int(v) },
			frame: func(pl *Player) ([]byte, error) {
				return entergen.ObjectPropertyFrame(1, 1, 0, []entergen.IndexedProperty{
					entergen.NamedProperty(cur, entergen.Int32Value(int32(*field(pl)))),
					entergen.NamedProperty(max, entergen.Int32Value(int32(*field(pl)))),
				})
			},
		}
	}
}

func propFrameInt32(name string, get func(*Player) int32) func(*Player) ([]byte, error) {
	return func(pl *Player) ([]byte, error) {
		return entergen.ObjectPropertyFrame(1, 1, 0, []entergen.IndexedProperty{
			entergen.NamedProperty(name, entergen.Int32Value(get(pl))),
		})
	}
}

func moneyFrame(pl *Player) ([]byte, error) {
	return entergen.ObjectPropertyFrame(1, 1, 0, []entergen.IndexedProperty{
		entergen.NamedProperty("CapitalType1", entergen.Int64Value(pl.roleState.Money)),
	})
}

// gmCommand 面板 GM 测试命令族（echo `gm ` 前缀；who 解析与 kick 同权：
// FindByName 先、FindByAccount 兜底）。无 roleState（未进场/旧档）回错。
// 返回行沿用 plist 方言：逐行 `>…` + `>end` 收尾。
func (e *EchoConsole) gmCommand(nc net.Conn, fields []string) {
	s := e.srv
	if len(fields) < 2 {
		fmt.Fprint(nc, ">usage: gm <state|set|money|tp|msg|view|flush|save> <who> [args]\r>end\r")
		return
	}
	sub, who := fields[0], fields[1]
	pl := s.players.FindByName(who)
	if pl == nil {
		pl = s.players.FindByAccount(who)
	}
	if pl == nil {
		fmt.Fprintf(nc, ">gm %s: player %q not found\r>end\r", sub, who)
		return
	}
	if !pl.haveRoleState && sub != "state" {
		fmt.Fprintf(nc, ">gm %s: player %q has no role state (not entered)\r>end\r", sub, who)
		return
	}
	send := func(frame []byte, err error) bool {
		if err != nil {
			fmt.Fprintf(nc, ">frame error: %v\r", err)
			return false
		}
		s.sendToClient(pl, frame)
		return true
	}
	defer fmt.Fprint(nc, ">end\r")
	switch sub {
	case "state":
		st := pl.roleState
		fmt.Fprintf(nc, ">account=%s name=%s state=%s\r", pl.Account, pl.Name, pl.State)
		fmt.Fprintf(nc, ">str=%d dex=%d ing=%d spi=%d sta=%d\r", st.Str, st.Dex, st.Ing, st.Spi, st.Sta)
		fmt.Fprintf(nc, ">hp=%d mp=%d sp=%d parry=%d money=%d\r", st.HP, st.MP, st.SP, st.Parry, st.Money)
		fmt.Fprintf(nc, ">scene=%s pos=%.3f,%.3f,%.3f,%.3f\r", st.Scene,
			st.Position[0], st.Position[1], st.Position[2], st.Position[3])
		fmt.Fprintf(nc, ">skills=%d neigong=%d cur=%s facultySeed=%d\r",
			len(st.Skills), len(st.NeiGong), st.CurNeiGong, st.FacultySeed)
	case "set":
		if len(fields) < 4 {
			fmt.Fprint(nc, ">usage: gm set <who> <prop> <value>\r")
			return
		}
		prop := strings.ToLower(fields[2])
		spec, ok := gmPropWhiteList[prop]
		if !ok {
			fmt.Fprintf(nc, ">gm set: unknown prop %q (whitelist: str dex ing spi sta hp mp sp parry money)\r", fields[2])
			return
		}
		v, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil || v < 0 {
			fmt.Fprintf(nc, ">gm set: bad value %q\r", fields[3])
			return
		}
		spec.apply(pl, v)
		if frame, ferr := spec.frame(pl); ferr != nil {
			fmt.Fprintf(nc, ">frame error: %v\r", ferr)
		} else {
			s.sendToClient(pl, frame)
		}
		fmt.Fprintf(nc, ">gm set %s %s=%d sent (0x10)\r", who, prop, v)
	case "money":
		if len(fields) < 3 {
			fmt.Fprint(nc, ">usage: gm money <who> <value>\r")
			return
		}
		v, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || v < 0 {
			fmt.Fprintf(nc, ">gm money: bad value %q\r", fields[2])
			return
		}
		pl.roleState.Money = v
		send(moneyFrame(pl))
		fmt.Fprintf(nc, ">gm money %s=%d sent (CapitalType1)\r", who, v)
	case "tp":
		if len(fields) < 5 {
			fmt.Fprint(nc, ">usage: gm tp <who> <x> <y> <z> [orient]\r")
			return
		}
		var v [4]float64
		for i := 0; i < 3; i++ {
			f, err := strconv.ParseFloat(fields[2+i], 64)
			if err != nil {
				fmt.Fprintf(nc, ">gm tp: bad coord %q\r", fields[2+i])
				return
			}
			v[i] = f
		}
		v[3] = 1
		if len(fields) >= 6 {
			if f, err := strconv.ParseFloat(fields[5], 64); err == nil {
				v[3] = f
			}
		}
		pl.roleState.Position = v
		s.sendToClient(pl, entergen.LocationFrame(1, 1, float32(v[0]), float32(v[1]), float32(v[2]), float32(v[3])))
		fmt.Fprintf(nc, ">gm tp %s → %.3f,%.3f,%.3f,%.3f sent (0x1F)\r", who, v[0], v[1], v[2], v[3])
	case "msg":
		if len(fields) < 4 {
			fmt.Fprint(nc, ">usage: gm msg <who> <msgId> [intArgs...]\r")
			return
		}
		id, err := strconv.ParseInt(fields[2], 10, 32)
		if err != nil || id < 256 || id > 100000 {
			fmt.Fprintf(nc, ">gm msg: bad msgId %q\r", fields[2])
			return
		}
		var vals []entergen.CustomValue
		for _, a := range fields[3:] {
			iv, err := strconv.ParseInt(a, 10, 32)
			if err != nil {
				fmt.Fprintf(nc, ">gm msg: bad int arg %q\r", a)
				return
			}
			vals = append(vals, entergen.CustomInt(int32(iv)))
		}
		if ok := send(entergen.CustomMessageFrame(int32(id), vals...)); ok {
			fmt.Fprintf(nc, ">gm msg %d(%d args) sent\r", id, len(vals))
		}
	case "view":
		if len(fields) < 3 || (fields[2] != "40" && fields[2] != "43" && fields[2] != "46") {
			fmt.Fprint(nc, ">usage: gm view <who> <40|43|46> (40=武学 43=内功 46=轻功)\r")
			return
		}
		st := pl.roleState
		var frames [][]byte
		var err error
		switch fields[2] {
		case "40":
			frames, err = entergen.SkillViewFrames(st)
		case "43":
			frames, err = entergen.NeiGongViewFrames(st)
		default:
			frames, err = entergen.QingGongViewFrames(st)
		}
		if err != nil {
			fmt.Fprintf(nc, ">view frames error: %v\r", err)
			return
		}
		for _, f := range frames {
			s.sendToClient(pl, f)
		}
		fmt.Fprintf(nc, ">gm view %s: view %s rebuilt (%d frames)\r", who, fields[2], len(frames))
	case "flush":
		// 重放 ReadyFrames（登录信息簇+位置，0x0B 进场帧不含——对拍/
		// 客户端卡死恢复用）。Sequence 自增防与原进场序撞号。
		frames, err := entergen.ReadyFrames(s.enterContext(pl), time.Now())
		if err != nil {
			fmt.Fprintf(nc, ">flush error: %v\r", err)
			return
		}
		for _, f := range frames {
			s.sendToClient(pl, f)
		}
		fmt.Fprintf(nc, ">gm flush %s: ready frames replayed (%d)\r", who, len(frames))
	case "save":
		if !s.sendRoleSave(pl) {
			fmt.Fprint(nc, ">gm save: store not connected, save not sent\r")
			return
		}
		fmt.Fprintf(nc, ">gm save %s: 0x04 sent to store\r", who)
	default:
		fmt.Fprintf(nc, ">gm: unknown subcommand %q\r", sub)
	}
}
