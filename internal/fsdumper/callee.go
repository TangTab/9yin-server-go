package fsdumper

import "context"

// DefaultCallee 对应 FsDumpLogic 的 dump_callee（vtable 0x1016FC88）：
// 装配 DumpRole + DumpRes 两单例并按官方语义转发。
type DefaultCallee struct {
	Role *DumpRole
	Res  *DumpRes
}

// NewDefaultCallee 组装默认回调面（需先各自 New 成）。
func NewDefaultCallee(role *DumpRole, res *DumpRes) *DefaultCallee {
	return &DefaultCallee{Role: role, Res: res}
}

// OnDumperStart 官方：DumpRole::OnDumperStart + DumpRes::OnDumperStart。
func (c *DefaultCallee) OnDumperStart(k *Kernel) {
	if c.Role != nil {
		// 官方 DumpRole 启动即 GetTableSchem("output", 表) → 建表 →
		// 版本就绪；Go 在此同步完成（无列名异步回环）。
		if db, err := k.Access("output"); err == nil {
			if err := c.Role.EnsureTable(context.Background(), db); err != nil {
				k.NotifyEcho(err.Error())
			}
		}
	}
	if c.Res != nil {
		c.Res.OnDumperStart(k)
	}
}

func (c *DefaultCallee) OnFrameExecute(k *Kernel) {} // 官方空实现

func (c *DefaultCallee) OnServerClose(k *Kernel) {} // 官方空实现

// OnServerConnected 官方只转 DumpRes::OnServerConnected；Go 同位补
// DumpRole 模板下发（官方在版本不符时回发，连接即发等价且更简）。
func (c *DefaultCallee) OnServerConnected(k *Kernel, clientID int) {
	if c.Role != nil {
		k.SendServerMsg(clientID, c.Role.BuildVersionMsg())
	}
}

// OnRecvGameServerMsg 官方：首字段=="dump_role_msg" 才收；转
// DumpRes 再转 DumpRole。
func (c *DefaultCallee) OnRecvGameServerMsg(k *Kernel, clientID int, msg *textVarList) {
	if msg.Len() == 0 || msg.String(0) != "dump_role_msg" {
		return
	}
	if c.Role != nil {
		c.Role.HandleGameServerMsg(k, clientID, msg)
	}
}
