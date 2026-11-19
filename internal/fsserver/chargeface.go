package fsserver

import (
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/charge"
)

// ChargeClient 复刻 CChargeSockHook（§5.6）：UserSock 异步 TCP 客户端 +
// 文本 VarList 行协议（`$s_%d_%d_%d_%d ` 前缀 + text_encode_msg + CRLF）。
type ChargeClient struct {
	Serial   int
	District uint32
	Server   uint32
	ident    int

	// SendLine 原始行发送（含 CRLF；由装配层注入 TCP 连接器）。
	SendLine func(line string) error
	// OnVarList 收侧 decoded VarList 回调（队列化后 ProcessChargeMsg 消费）。
	OnVarList func(v charge.VarList)
}

// NewChargeClient …
func NewChargeClient(district, server uint32) *ChargeClient {
	return &ChargeClient{District: district, Server: server}
}

// prefix `$s_%d_%d_%d_%d `（GetSerial + ident + district + server）。
func (c *ChargeClient) prefix() string {
	return fmt.Sprintf("$s_%d_%d_%d_%d ", c.Serial, c.ident, c.District, c.Server)
}

// SendIdentMsg …（MakeSendData 手补 0D 0A 语义）。
func (c *ChargeClient) SendIdentMsg(v charge.VarList) error {
	if c.SendLine == nil {
		return fmt.Errorf("(CChargeSockHook::SendIdentMsg)error connect")
	}
	return c.SendLine(c.prefix() + v.EncodeLine() + "\r\n")
}

// SendRegister 连接建立后的 register 命令。
func (c *ChargeClient) SendRegister(name string) error {
	var v charge.VarList
	v.AddString("register")
	v.AddString(name)
	return c.SendIdentMsg(v)
}

// Keepalive …（DoWait 1s "keep"）。
func (c *ChargeClient) Keepalive() error {
	var v charge.VarList
	v.AddString("keep")
	return c.SendIdentMsg(v)
}

// ReceiveLine 收侧行解码（OnReceive → text_decode_msg → OnVarList）。
func (c *ChargeClient) ReceiveLine(line string) {
	line = trimEOL(line)
	if line == "" {
		return
	}
	v, err := charge.DecodeLine(line)
	if err != nil {
		return // "(GameWorld::ProcessChargeMsg)decode failed"
	}
	if len(v) < 2 {
		return // "(GameWorld::ProcessChargeMsg)message too short"
	}
	if c.OnVarList != nil {
		c.OnVarList(v)
	}
}

func trimEOL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// chargeCmdNames ProcessChargeMsg strcmp 链 17 项（§5.6 官方序；
// buy_item_give 与 buy_item 同 handler）。
var chargeCmdNames = []string{
	"register", "unregister", "reset", "login", "logout", "renew",
	"query_gift", "get_gift", "sell_card", "unsell_card", "buy_card",
	"buy_item", "buy_item_give", "item_info", "get_points", "validate", "custom",
}

// ChargeCmdName …（跳表导出测试用）。
func ChargeCmdName(i int) string {
	if i < 0 || i >= len(chargeCmdNames) {
		return ""
	}
	return chargeCmdNames[i]
}

// PutChargeVarList 文本命令入队（OnVarList → 队列 → Pump 消费）。
func (w *World) PutChargeVarList(v charge.VarList) {
	w.mu.Lock()
	w.qCharge = append(w.qCharge, msg{typ: 0xFFFFFFFF, payload: any(v)})
	w.mu.Unlock()
}

// processChargeMsg 文本命令分派：v[0]=回显标记、v[1]=命令名、v[2..]=参数。
func (w *World) processChargeMsg(m msg) {
	v, ok := m.payload.(charge.VarList)
	if !ok || len(v) < 2 {
		w.logf("(GameWorld::ProcessChargeMsg)message too short")
		return
	}
	cmd := v.StringVal(1)
	switch cmd {
	case "login":
		w.chargeLogin(v)
	case "logout":
		w.chargeLogout(v)
	case "validate":
		w.chargeValidate(v)
	case "renew":
		w.chargeRenew(v)
	case "query_gift", "get_gift", "sell_card", "unsell_card", "buy_card",
		"buy_item", "buy_item_give", "item_info", "get_points":
		// 礼品/点卡族：找到在线玩家后透传场景（官方 "no charge" 门在
		// member 侧已过；world 侧仅路由）
		w.chargeToScene(v)
	case "register", "unregister", "reset", "custom":
		w.logf("charge %s", cmd)
	default:
		w.logf("(GameWorld::ProcessChargeMsg)error message type")
	}
}

// chargeAcct 参数取账号（v[2]）。
func chargeAcct(v charge.VarList) string {
	if len(v) > 2 {
		return v.StringVal(2)
	}
	return ""
}

// chargeLogin 登录应答（md5 比对结果）。
func (w *World) chargeLogin(v charge.VarList) {
	acct := chargeAcct(v)
	ok := len(v) > 3 && v.Int(3) != 0
	p := w.Players.FindByAccount(acct)
	if p == nil {
		return
	}
	if !ok {
		w.logf("(GameWorld::PlayerStateUnknown)get account password error")
		w.sendErrClient(p, ErrLoginParse)
		w.breakClient(p)
		w.Players.RemovePlayer(p.Ident, p.Serial)
		return
	}
	w.PlayerStateMachine(p, EventTick, nil, 0)
}

// chargeLogout …
func (w *World) chargeLogout(v charge.VarList) {
	p := w.Players.FindByAccount(chargeAcct(v))
	if p == nil {
		w.logf("(GameWorld::StoreMsgAccountLogoff)logout failed")
		return
	}
	w.PlayerStateMachine(p, EventClosed, nil, 0)
}

// chargeValidate 矩阵卡挑战应答。
func (w *World) chargeValidate(v charge.VarList) {
	p := w.Players.FindByAccount(chargeAcct(v))
	if p == nil {
		return
	}
	w.PlayerStateMachine(p, EventGetVerify, []byte{1}, 1)
}

// chargeRenew 续费应答（"renew warning"/"renew break"）。
func (w *World) chargeRenew(v charge.VarList) {
	p := w.Players.FindByAccount(chargeAcct(v))
	if p == nil {
		return
	}
	if len(v) > 3 && v.Int(3) != 0 {
		p.ActiveTime = timeNow()
	} else if p.State == PStateOnline {
		w.logf("(GameWorld::PlayerStateOnline)renew break")
		w.setPlayerState(p, PStateOffline)
		w.PlayerStateMachine(p, EventTick, nil, 0)
	}
}

// chargeToScene 礼品/点卡族透传场景（官方 SendToScene 路由）。
func (w *World) chargeToScene(v charge.VarList) {
	p := w.Players.FindByAccount(chargeAcct(v))
	if p == nil || p.State != PStateOnline {
		return
	}
	var out charge.VarList
	out.AddString(v.StringVal(1))
	for i := 2; i < len(v); i++ {
		out = append(out, v[i])
	}
	_ = out
	w.sendToScene(p.SceneID, 0x2A, []byte(v.StringVal(1)))
}
