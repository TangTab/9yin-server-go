package fspublic

import (
	"fmt"
)

// LogicModuleVersion 官方版本门 0x3001（GamePublic::Start 逐字比对；
// 快照内 FsPubLogic 0x300E/0x300F 均被拒——见全解 §10）。
const LogicModuleVersion uint32 = 0x3001

// Callee 是 FsPubLogic pub_callee 的 Go 等价（IPubManager 面 +
// OnFrameExecute/OnServerState/OnRoleOffline）。Assembly 装配时注入
// Public.Callee。
type Callee interface {
	// OnPublicStart 全空间装载完、开放服务时（官方 vt+4）。
	OnPublicStart(k *Kernel)
	// OnServerClose world 关闭令到达（进 Closing 前，官方 vt+8）。
	OnServerClose(k *Kernel)
	// OnPubSpaceLoaded 单空间装载完（官方 vt+C）。
	OnPubSpaceLoaded(k *Kernel, space string)
	// OnPubDataLoaded 单数据对象装载完（官方 vt+10）。
	OnPubDataLoaded(k *Kernel, space, data string)
	// OnPublicMessage member→public 透传消息（官方 vt+14；msgid 在
	// args[2]——{space, dataName, msgID, operands...}）。
	OnPublicMessage(k *Kernel, ident, msgID uint32, args []Value) bool
	// OnFrameExecute Opened 态每秒帧回调（官方 vt 转发）。
	OnFrameExecute(k *Kernel, elapsed int)
	// OnServerState world 服务器状态（public_res_server_state_t）。
	OnServerState(k *Kernel, state ServerState)
	// OnRoleOffline store 离线角色应答行。
	OnRoleOffline(k *Kernel, rows [][]Value)
}

// 公共消息 id（FsPubLogic ProcessCommonMessage 定案；对应 member
// 推送 op 0x3..0xC 的修改原语）。
const (
	MsgSetPubDataProp          uint32 = 1000 // [3]=attrName [4]=value
	MsgIncPubDataProp          uint32 = 1001 // [3]=attrName [4]=delta(int)
	MsgClearPubDataRec         uint32 = 1100 // [3]=recName
	MsgRemovePubDataRecRow     uint32 = 1101 // [3]=recName [4]=row
	MsgAddPubDataRecRowValue   uint32 = 1102 // [3]=recName [4..]=列值
	MsgSetPubDataRecValue      uint32 = 1103 // [3]=recName [4]=row [5]=col [6]=value
	MsgIncPubDataRecValue      uint32 = 1104 // [3]=recName [4]=row [5]=col [6]=delta
	MsgRemovePubDataRecRowByKey uint32 = 1105
	MsgSetPubDataRecValueByKey  uint32 = 1106
	MsgIncPubDataRecValueByKey  uint32 = 1107
)

// Domain 是 CDomainBase 的 Go 等价：挂载在空间内的玩法域。
type Domain interface {
	// DomainName 官方 GetDomainName(kernel)（"%d" 后缀=ServerID）。
	DomainName(k *Kernel) string
	// GetSave 该域数据是否持久化到 store。
	GetSave() bool
	// OnCreate 新建域数据对象时建表。
	OnCreate(k *Kernel, space *PubSpace, dataName string)
	// OnLoad 从 store 载入后初始化。
	OnLoad(k *Kernel, space *PubSpace)
	// OnMessage 域消息。
	OnMessage(k *Kernel, space *PubSpace, msgID uint32, args []Value)
}

// DefaultCallee 是 FsPubLogic（pub_callee + DomainManager + GuildManager）
// 的 Go 骨架：内建公共修改原语（1000..1107）+ 域注册表分发。
//
// 官方 22 域（Team/EnergyWell/Arena/Guild...）的玩法语义不在架构复刻
// 范围；空域注册表 + OnCreate/OnLoad/OnMessage 挂点已对齐两级协议。
type DefaultCallee struct {
	// Spaces 名→域分发（DomainManager→"domain"/"guild" 双名兼容——
	// 2013 DLL 自报 "domain"、2014 lua 建 "pubdata" 的版本差定案）。
	domains []domainEntry
	// OnLog 日志面。
	OnLog func(format string, args ...any)
}

type domainEntry struct {
	spaceNames []string // 兼容别名（如 {"domain", "pubdata"}）
	domain     Domain
}

// NewDefaultCallee 建空域 Callee。
func NewDefaultCallee() *DefaultCallee {
	return &DefaultCallee{}
}

// RegisterDomain 注册域（spaceNames 为空间别名集）。
func (c *DefaultCallee) RegisterDomain(spaceNames []string, d Domain) {
	c.domains = append(c.domains, domainEntry{spaceNames: spaceNames, domain: d})
}

func (c *DefaultCallee) log(format string, args ...any) {
	if c.OnLog != nil {
		c.OnLog(format, args...)
	}
}

// --- Callee 面 ---

// OnPublicStart 官方：manager 全量 OnPublicStart（域在此 new，Go 版为
// 预注册）。
func (c *DefaultCallee) OnPublicStart(k *Kernel) {
	c.log("pub logic started (version %#x)", LogicModuleVersion)
}

// OnServerClose 官方全量转发。
func (c *DefaultCallee) OnServerClose(k *Kernel) {}

// OnPubSpaceLoaded 官方：manager 空间匹配后对每域 OnLoad。
func (c *DefaultCallee) OnPubSpaceLoaded(k *Kernel, space string) {
	for _, e := range c.domains {
		if matchSpace(e.spaceNames, space) {
			if s, ok := k.p.Spaces.Find(space); ok {
				e.domain.OnLoad(k, s)
			}
		}
	}
}

// OnPubDataLoaded 官方：空间匹配后按数据名（=域名）OnCreate。
func (c *DefaultCallee) OnPubDataLoaded(k *Kernel, space, data string) {
	for _, e := range c.domains {
		if matchSpace(e.spaceNames, space) {
			if s, ok := k.p.Spaces.Find(space); ok {
				e.domain.OnCreate(k, s, data)
			}
		}
	}
}

// OnFrameExecute 官方全量转发（域级定时在此驱动由 Timer 承担）。
func (c *DefaultCallee) OnFrameExecute(k *Kernel, elapsed int) {}

// OnServerState 官方全量转发。
func (c *DefaultCallee) OnServerState(k *Kernel, state ServerState) {
	c.log("server state %q %v", state.Name, state.Props)
}

// OnRoleOffline 官方全量转发。
func (c *DefaultCallee) OnRoleOffline(k *Kernel, rows [][]Value) {
	c.log("role offline rows %d", len(rows))
}

// OnPublicMessage 官方分派：1000..1999 走 ProcessCommonMessage，其余按
// args[1]（数据名=域名）查域。
func (c *DefaultCallee) OnPublicMessage(k *Kernel, ident, msgID uint32, args []Value) bool {
	if msgID >= 1000 && msgID <= 1999 {
		return c.processCommonMessage(k, ident, msgID, args)
	}
	if len(args) < 2 {
		return false
	}
	spaceName := valueString(args[0])
	dataName := valueText(args[1])
	for _, e := range c.domains {
		if !matchSpace(e.spaceNames, spaceName) {
			continue
		}
		if s, ok := k.p.Spaces.Find(spaceName); ok {
			if dataName == e.domain.DomainName(k) || dataName == "" {
				e.domain.OnMessage(k, s, msgID, args)
				return true
			}
		}
	}
	c.log("(pub_callee::OnPublicMessage)no domain for %s/%s msg %d", spaceName, dataName, msgID)
	return false
}

// processCommonMessage 官方 ProcessCommonMessage（十原语定案）。
func (c *DefaultCallee) processCommonMessage(k *Kernel, ident, msgID uint32, args []Value) bool {
	if len(args) < 3 {
		c.log("(pub_callee::ProcessCommonMessage)parameter error")
		return false
	}
	p := k.p
	spaceName := valueString(args[0])
	dataName := valueText(args[1])
	space, ok := p.Spaces.Find(spaceName)
	if !ok {
		c.log("(pub_callee)%s space not exists: %s", msgName(msgID), spaceName)
		return false
	}
	d, ok := space.Get(dataName)
	if !ok {
		c.log("(pub_callee)%s data not exists: %s", msgName(msgID), dataName)
		return false
	}
	key := d.Key
	switch msgID {
	case MsgSetPubDataProp:
		if len(args) < 5 {
			return false
		}
		name := valueText(args[3])
		v := args[4]
		save := false
		if at, ok := d.GetAttr(name); ok {
			save = at.Save
		}
		if err := d.SetAttr(name, v.Type, v, save); err != nil {
			c.log("(pub_callee::SetPubDataProp)%v", err)
			return false
		}
		d.Modify = true
		p.PushAddAttr(spaceName, dataName, key, v)
		return true
	case MsgIncPubDataProp:
		if len(args) < 5 {
			return false
		}
		name := valueText(args[3])
		delta := args[4].I64
		at, ok := d.GetAttr(name)
		if !ok {
			c.log("(pub_callee::IncPubDataProp)attr not exists: %s", name)
			return false
		}
		v := at.Val
		v.I64 += delta
		if err := d.SetAttr(name, at.Type, v, at.Save); err != nil {
			c.log("(pub_callee::IncPubDataProp)%v", err)
			return false
		}
		d.Modify = true
		p.PushAddAttr(spaceName, dataName, key, v)
		return true
	case MsgClearPubDataRec:
		if len(args) < 4 {
			return false
		}
		ri, ok := d.GetRec(valueText(args[3]))
		if !ok {
			return false
		}
		ri.Rec.Rows = nil
		d.Modify = true
		p.PushClearRecRow(spaceName, dataName, key)
		return true
	case MsgRemovePubDataRecRow:
		if len(args) < 5 {
			return false
		}
		ri, ok := d.GetRec(valueText(args[3]))
		if !ok {
			return false
		}
		row := int(args[4].I64)
		if row < 0 || row >= len(ri.Rec.Rows) {
			c.log("(pub_callee::RemovePubDataRecRow)row error")
			return false
		}
		ri.Rec.Rows = append(ri.Rec.Rows[:row], ri.Rec.Rows[row+1:]...)
		d.Modify = true
		p.PushRemoveRecRow(spaceName, dataName, key, row)
		return true
	case MsgAddPubDataRecRowValue:
		if len(args) < 4 {
			return false
		}
		ri, ok := d.GetRec(valueText(args[3]))
		if !ok {
			return false
		}
		vals := args[4:]
		if err := ri.Rec.AddRow(vals); err != nil {
			c.log("(pub_callee::AddPubDataRecRowValue)%v", err)
			return false
		}
		d.Modify = true
		p.PushAddRecRow(spaceName, dataName, key, len(ri.Rec.Rows)-1, vals)
		return true
	case MsgSetPubDataRecValue:
		if len(args) < 7 {
			return false
		}
		ri, ok := d.GetRec(valueText(args[3]))
		if !ok {
			return false
		}
		row, col := int(args[4].I64), int(args[5].I64)
		if err := ri.Rec.SetGrid(row, col, args[6]); err != nil {
			c.log("(pub_callee::SetPubDataRecValue)%v", err)
			return false
		}
		d.Modify = true
		p.PushSetRecGrid(spaceName, dataName, key, row, col, args[6])
		return true
	case MsgIncPubDataRecValue:
		if len(args) < 7 {
			return false
		}
		ri, ok := d.GetRec(valueText(args[3]))
		if !ok {
			return false
		}
		row, col := int(args[4].I64), int(args[5].I64)
		if row < 0 || row >= len(ri.Rec.Rows) || col < 0 || col >= ri.Rec.Cols {
			c.log("(pub_callee::IncPubDataRecValue)row/col error")
			return false
		}
		v := ri.Rec.Rows[row][col]
		v.I64 += args[6].I64
		if err := ri.Rec.SetGrid(row, col, v); err != nil {
			c.log("(pub_callee::IncPubDataRecValue)%v", err)
			return false
		}
		d.Modify = true
		p.PushSetRecGrid(spaceName, dataName, key, row, col, v)
		return true
	case MsgRemovePubDataRecRowByKey, MsgSetPubDataRecValueByKey, MsgIncPubDataRecValueByKey:
		// 官方 *ByKey 族：按键列值定位行后复用上三原语（键列索引
		// CPubData::SetRecKey 建）；Go 侧按键列扫描定位。
		return c.processByKey(k, space, d, msgID, args)
	}
	return false
}

// processByKey 官方 \*ByKey 三原语：args[3]=recName [4]=key 列值
//（string/widestr）[5..]按对应原语。
func (c *DefaultCallee) processByKey(k *Kernel, space *PubSpace, d *PubData, msgID uint32, args []Value) bool {
	if len(args) < 5 {
		return false
	}
	ri, ok := d.GetRec(valueText(args[3]))
	if !ok || len(ri.Rec.Keys) == 0 {
		c.log("(pub_callee::*ByKey)rec no keys")
		return false
	}
	keyCol := ri.Rec.Keys[0].Col
	want := valueText(args[4])
	row := -1
	for r, rv := range ri.Rec.Rows {
		if valueText(rv[keyCol]) == want {
			row = r
			break
		}
	}
	if row < 0 {
		c.log("(pub_callee::*ByKey)key not found: %s", want)
		return false
	}
	// 重写成绝对坐标原语
	switch msgID {
	case MsgRemovePubDataRecRowByKey:
		return c.processCommonMessage(k, 0, MsgRemovePubDataRecRow, append(args[:4:4], IntValue(int32(row))))
	case MsgSetPubDataRecValueByKey:
		if len(args) < 7 {
			return false
		}
		fixed := append(append(args[:4:4], IntValue(int32(row))), IntValue(int32(keyCol)))
		return c.processCommonMessage(k, 0, MsgSetPubDataRecValue, append(fixed, args[6]))
	case MsgIncPubDataRecValueByKey:
		if len(args) < 7 {
			return false
		}
		fixed := append(append(args[:4:4], IntValue(int32(row))), IntValue(int32(keyCol)))
		return c.processCommonMessage(k, 0, MsgIncPubDataRecValue, append(fixed, args[6]))
	}
	return false
}

func matchSpace(names []string, space string) bool {
	for _, n := range names {
		if n == space {
			return true
		}
	}
	return false
}

func valueString(v Value) string {
	if v.Type == VTypeWideStr {
		return v.Text
	}
	return v.Text
}

func valueText(v Value) string { return v.Text }

func msgName(id uint32) string {
	switch id {
	case MsgSetPubDataProp:
		return "SetPubDataProp"
	case MsgIncPubDataProp:
		return "IncPubDataProp"
	case MsgClearPubDataRec:
		return "ClearPubDataRec"
	case MsgRemovePubDataRecRow:
		return "RemovePubDataRecRow"
	case MsgAddPubDataRecRowValue:
		return "AddPubDataRecRowValue"
	case MsgSetPubDataRecValue:
		return "SetPubDataRecValue"
	case MsgIncPubDataRecValue:
		return "IncPubDataRecValue"
	}
	return fmt.Sprintf("Msg(%d)", id)
}
