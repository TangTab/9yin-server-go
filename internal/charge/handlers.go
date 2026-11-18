// handlers.go 业务命令族：login/logout/renew/gift/card/item/points/validate
// 按官方 Receive*（请求字段表，tmp_research/fscharge/report_receive.md）
// 与 Publish*（应答元素表，report_publish.md）逐元素复刻。
package charge

// upper 官方 Port_StringUpper 大写化（ASCII 域，账号/密码/卡号归一）。
func upper(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 'a' - 'A'
		}
	}
	return string(b)
}

// requireOnline 官方通用前置：GetServer + state==2。
func (s *Server) requireOnline(sess *session, cmd string) *gameServer {
	sv := s.getServerLocked(sess.connid, 0)
	if sv == nil {
		s.logf("(GameCharge::Receive%s)Server not found", cmd)
		return nil
	}
	if sv.state != serverStateOnline {
		s.logf("(GameCharge::Receive%s)Server state not online", cmd)
		return nil
	}
	return sv
}

// cmdLogin ReceiveLogin@0x71560：
// [echo,"login",?(2), account(3), passwd(4), ip(5), ?(6)]。
// 应答（PublishLogin@0x6a170，14 元素）：
// [echo,"login",1,account,result,accID,logonID,"",points,"",chargeType,
//  deadline(double),gmLevel,ext]
func (s *Server) cmdLogin(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "Login")
	if sv == nil {
		return
	}
	account := upper(vl.StringVal(3))
	passwd := upper(vl.StringVal(4))
	address := vl.StringVal(5)
	u := s.addUserLocked(sv.serverID, sess.connid, 0, account, passwd, address)

	res := s.Backend.Login(account, passwd, sv.serverID, s.Config.GameID, address)
	if res.Result == 1 {
		u.accID = res.AccID
		u.points = res.Points
		u.chargeTyp = res.ChargeType
		u.unitType = res.ChargeType
		u.deadline = res.Deadline
		u.logonID = res.LogonID
		u.lastFlush = daysSinceEpoch(s.now())
		if res.AccID != 0 {
			s.byAccID[res.AccID] = u
		}
		s.logf("Account: %s ID: %d Address: %s Logon ID: %s Server: %d login succeed",
			account, res.AccID, address, res.LogonID, sv.serverID)
	} else {
		s.logf("Account: %s Address: %s Server: %d login failed result %d",
			account, address, sv.serverID, res.Result)
		s.releaseUserLocked(u)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("login")
	out.AddInt(1)
	out.AddString(account)
	out.AddInt(res.Result)
	out.AddInt(res.AccID)
	out.AddString(res.LogonID)
	out.AddString("")
	out.AddInt(res.Points)
	out.AddString("")
	out.AddInt(res.ChargeType)
	out.AddDouble(res.Deadline)
	out.AddInt(res.GMLevel)
	out.AddString(res.ChargeInfo)
	s.respond(sess, out)
}

// cmdLogout ReceiveLogout@0x71c80：
// [echo,"logout",useridx(2), sess(3), reason(4), account(5), passwd(6)]。
// 应答（PublishLogout@0x6a660）：[echo,"logout",useridx,result]。
func (s *Server) cmdLogout(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "Logout")
	if sv == nil {
		return
	}
	useridx := vl.Int(2)
	var u *userInfo
	if useridx != 0 {
		u = s.getUserActiveLocked(sv.serverID, useridx, sess.connid, 0)
	} else {
		u = s.getUserByAccountLocked(upper(vl.StringVal(5)), sess.connid, 0)
	}
	if u == nil {
		s.logf("(GameCharge::ReceiveLogout)user not found")
		return
	}
	result := s.Backend.Logout(u.account, u.accID, u.points)
	if result == 1 {
		s.logf("Account: %s ID: %d Logon ID: %s logout succeed", u.account, u.accID, u.logonID)
	} else {
		s.logf("Account: %s ID: %d Logon ID: %s logout failed result %d",
			u.account, u.accID, u.logonID, result)
	}
	idx := u.index
	s.releaseUserLocked(u)
	out := VarList{}
	out.AddString(echo)
	out.AddString("logout")
	out.AddInt(idx)
	out.AddInt(result)
	s.respond(sess, out)
}

// cmdRenew ReceiveRenew@0x72220：
// [echo,"renew",useridx(2), sess(3), reason(4), order(5)]。
// 应答（PublishRenew@0x6b010，仅 chargeType==2 回发；踢人分支第 4 元素=2）：
// [echo,"renew",accID,chargeType,unitType,deadline(double)]。
func (s *Server) cmdRenew(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "Renew")
	if sv == nil {
		return
	}
	u := s.getUserActiveLocked(sv.serverID, vl.Int(2), sess.connid, 0)
	if u == nil {
		s.logf("(GameCharge::ReceiveRenew)user not found")
		return
	}
	s.renewUser(sess, echo, u)
}

// renewUser PublishRenew 主链（也用于 CheckUsers 主动推与顶号踢人）。
func (s *Server) renewUser(sess *session, echo string, u *userInfo) {
	result, chargeType, unitType, deadline := s.Backend.Renew(u.account, u.accID)
	if result == 1 {
		u.chargeTyp = chargeType
		u.unitType = unitType
		u.deadline = deadline
		u.failCnt = 0
		u.lastFlush = daysSinceEpoch(s.now())
		if s.Config.FlushLog {
			s.logf("Account: %s ID: %d Logon ID: %s flush succeed", u.account, u.accID, u.logonID)
		}
	} else {
		s.logf("Account: %s ID: %d flush failed result %d", u.account, u.accID, result)
		if u.lastFlush <= foreverFlush {
			// 从未落账：直接删号，不回发（官方 return 0 路径）。
			s.logf("Account: %s ID: %d Logon ID: %s deleted", u.account, u.accID, u.logonID)
			s.releaseUserLocked(u)
			return
		}
		u.failCnt++
		if s.Config.FlushErrorMax > 0 && u.failCnt > s.Config.FlushErrorMax &&
			!u.kicked && deadline < daysSinceEpoch(s.now()) {
			u.kicked = true
			s.logf("Account: %s ID: %d Logon ID: %s kick out", u.account, u.accID, u.logonID)
			out := VarList{}
			out.AddString(echo)
			out.AddString("renew")
			out.AddInt(u.accID)
			out.AddInt(2)
			out.AddInt(u.unitType)
			out.AddDouble(u.deadline)
			s.respond(sess, out)
			s.releaseUserLocked(u)
			return
		}
	}
	if chargeType != 2 {
		return // 官方：仅计费类型 2 回传续费结果
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("renew")
	out.AddInt(u.accID)
	out.AddInt(chargeType)
	out.AddInt(u.unitType)
	out.AddDouble(u.deadline)
	s.respond(sess, out)
}

// cmdQueryGift ReceiveQueryGift@0x72590：[echo,"query_gift",useridx(2)]。
// 应答（PublishQueryGift@0x6b610）：
// [echo,"query_gift",accID,N,(id,name,amount)×N]。
func (s *Server) cmdQueryGift(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "QueryGift")
	if sv == nil {
		return
	}
	u := s.getUserActiveLocked(sv.serverID, vl.Int(2), sess.connid, 0)
	if u == nil {
		s.logf("(GameCharge::ReceiveQueryGift)user not found")
		return
	}
	result, gifts := s.Backend.QueryGift(u.account, u.accID)
	s.logf("Account: %s ID: %d query gift %d returned", u.account, u.accID, len(gifts))
	out := VarList{}
	out.AddString(echo)
	out.AddString("query_gift")
	out.AddInt(u.accID)
	out.AddInt(len(gifts))
	for _, g := range gifts {
		out.AddInt(g.ID)
		out.AddString(g.Name)
		out.AddInt(g.Amount)
	}
	_ = result
	s.respond(sess, out)
}

// cmdGetGift ReceiveGetGift@0x727c0：
// [echo,"get_gift",useridx(2),giftID(3),serial(4),amount(5)]。
// 应答（PublishGetGift@0x6be00）：
// [echo,"get_gift",accID,result,giftID,serial,amount]。
func (s *Server) cmdGetGift(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "GetGift")
	if sv == nil {
		return
	}
	u := s.getUserActiveLocked(sv.serverID, vl.Int(2), sess.connid, 0)
	if u == nil {
		s.logf("(GameCharge::ReceiveGetGift)user not found")
		return
	}
	giftID := vl.Int(3)
	serial := vl.StringVal(4)
	amount := vl.Int(5)
	result, name := s.Backend.GetGift(u.account, u.accID, giftID, serial, amount)
	if result == 1 {
		s.logf("Account: %s ID: %d get gift ID: %s Amount: %d succeed",
			u.account, u.accID, name, amount)
	} else {
		s.logf("Account: %s ID: %d get gift failed result %d", u.account, u.accID, result)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("get_gift")
	out.AddInt(u.accID)
	out.AddInt(result)
	out.AddInt(giftID)
	out.AddString(name)
	out.AddInt(amount)
	s.respond(sess, out)
}

// cmdSellCard ReceiveSellCard@0x72ae0：
// [echo,"sell_card",useridx(2),cardType(3),cardNo(4,大写),price(5)]。
// 应答（PublishSellCard@0x6c0c0）：
// [echo,"sell_card",accID,result,cardNo,price,note]。
func (s *Server) cmdSellCard(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "SellCard")
	if sv == nil {
		return
	}
	u := s.getUserActiveLocked(sv.serverID, vl.Int(2), sess.connid, 0)
	if u == nil {
		s.logf("(GameCharge::ReceiveSellCard)user not found")
		return
	}
	t := CardTrade{
		UserIdx:  vl.Int(2),
		CardType: vl.StringVal(3),
		CardNo:   upper(vl.StringVal(4)),
		Price:    vl.Int(5),
	}
	result, cardNo, price, note := s.Backend.SellCard(u.account, u.accID, t)
	if result == 1 {
		s.logf("Account: %s ID: %d sell card NO: %s price: %d succeed",
			u.account, u.accID, cardNo, price)
	} else {
		s.logf("Account: %s ID: %d sell card NO: %s failed result %d",
			u.account, u.accID, cardNo, result)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("sell_card")
	out.AddInt(u.accID)
	out.AddInt(result)
	out.AddString(cardNo)
	out.AddInt(price)
	out.AddString(note)
	s.respond(sess, out)
}

// cmdUnsellCard ReceiveUnsellCard@0x72ee0：
// [echo,"unsell_card",useridx(2 可 0=系统),cardNo(3 原文)]。
// 应答（PublishUnsellCard@0x6c380）：[echo,"unsell_card",useridx,result,cardNo]。
func (s *Server) cmdUnsellCard(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "UnsellCard")
	if sv == nil {
		return
	}
	useridx := vl.Int(2)
	cardNo := vl.StringVal(3)
	var u *userInfo
	if useridx != 0 {
		u = s.getUserActiveLocked(sv.serverID, useridx, sess.connid, 0)
	}
	var result int
	if u != nil {
		result = s.Backend.UnsellCard(u.account, u.accID, cardNo)
		if result == 1 {
			s.logf("Account: %s ID: %d cancel sell card NO: %s succeed", u.account, u.accID, cardNo)
		} else {
			s.logf("Account: %s ID: %d cancel sell card NO: %s failed result %d",
				u.account, u.accID, cardNo, result)
		}
	} else {
		result = s.Backend.UnsellCard("", 0, cardNo)
		if result == 1 {
			s.logf("System cancel sell card NO: %s succeed", cardNo)
		} else {
			s.logf("System cancel sell card NO: %s failed result %d", cardNo, result)
		}
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("unsell_card")
	out.AddInt(useridx)
	out.AddInt(result)
	out.AddString(cardNo)
	s.respond(sess, out)
}

// cmdBuyCard ReceiveBuyCard@0x73180：
// [echo,"buy_card",useridx(2),cardType(3),cardNo(4,大写),password(5)]。
// 应答（PublishBuyCard@0x6c650）：[echo,"buy_card",accID,result,cardNo,fillAccount]。
func (s *Server) cmdBuyCard(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "BuyCard")
	if sv == nil {
		return
	}
	u := s.getUserActiveLocked(sv.serverID, vl.Int(2), sess.connid, 0)
	if u == nil {
		s.logf("(GameCharge::ReceiveBuyCard)user not found")
		return
	}
	t := CardTrade{
		UserIdx:  vl.Int(2),
		CardType: vl.StringVal(3),
		CardNo:   upper(vl.StringVal(4)),
		Password: vl.StringVal(5),
	}
	result, cardNo, fillAcc := s.Backend.BuyCard(u.account, u.accID, t)
	if result == 1 {
		s.logf("Account: %s ID: %d buy card NO: %s fill account: %s succeed",
			u.account, u.accID, cardNo, fillAcc)
	} else {
		s.logf("Account: %s ID: %d buy card NO: %s fill account: %s failed result %d",
			u.account, u.accID, cardNo, fillAcc, result)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("buy_card")
	out.AddInt(u.accID)
	out.AddInt(result)
	out.AddString(cardNo)
	out.AddString(fillAcc)
	s.respond(sess, out)
}

// cmdBuyItem ReceiveBuyItem@0x73590（无 state 校验）：
// [echo,"buy_item",useridx(2),account(3,大写),itemID(4),amount(5),serverID(6)]。
// 应答（PublishBuyItem@0x6c910）：[echo,"buy_item",useridx,result,item,amount,points]。
func (s *Server) cmdBuyItem(sess *session, echo string, vl VarList) {
	sv := s.getServerLocked(sess.connid, 0)
	if sv == nil {
		s.logf("(GameCharge::ReceiveBuyItem)Server not found")
		return
	}
	serverID := sv.serverID
	useridx := vl.Int(2)
	var u *userInfo
	if serverID != 0 {
		u = s.getUserActiveLocked(serverID, useridx, sess.connid, 0)
	}
	if serverID == 0 {
		serverID = vl.Int(6)
	}
	t := ItemTrade{
		UserIdx: useridx,
		Account: upper(vl.StringVal(3)),
		ItemID:  vl.Int(4),
		Amount:  vl.Int(5),
	}
	var account string
	var accID int
	if u != nil {
		account, accID = u.account, u.accID
	} else {
		account = t.Account
	}
	result, item, amount, points := s.Backend.BuyItem(account, accID, t)
	if result == 1 {
		if u != nil {
			s.logf("Account: %s ID: %d buy item: %s amount: %d points: %d succeed",
				account, accID, item, amount, points)
		} else {
			s.logf("Account ID: %d buy item: %s amount: %d points: %d succeed",
				useridx, item, amount, points)
		}
	} else if u != nil {
		s.logf("Account: %s ID: %d buy item: %s amount: %d points: %d failed result %d",
			account, accID, item, amount, points, result)
	} else {
		s.logf("Account ID: %d buy item: %s amount: %d points: %d failed result %d",
			useridx, item, amount, points, result)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("buy_item")
	out.AddInt(useridx)
	out.AddInt(result)
	out.AddString(item)
	out.AddInt(amount)
	out.AddInt(points)
	s.respond(sess, out)
}

// cmdItemInfo ReceiveItemInfo@0x739e0（不查 user）：
// [echo,"item_info",query(2,大写)]。
// 应答（PublishItemInfo@0x6cc50）：
// [echo,"item_info",result,total,start,N,echo2,(name,points)×N]。
func (s *Server) cmdItemInfo(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "ItemInfo")
	if sv == nil {
		return
	}
	query := upper(vl.StringVal(2))
	result, total, start, items := s.Backend.ItemInfo(query)
	if result == 1 {
		s.logf("Get Item info total %d start %d amount %d succeed", total, start, len(items))
	} else {
		s.logf("Get Item info failed result: %d", result)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("item_info")
	out.AddInt(result)
	out.AddInt(total)
	out.AddInt(start)
	out.AddInt(len(items))
	out.AddString(query)
	for _, it := range items {
		out.AddString(it.Name)
		out.AddInt(it.Points)
	}
	s.respond(sess, out)
}

// cmdBuyItemGive ReceiveBuyItemGive@0x73c60（无 state 校验）：
// [echo,"buy_item_give",useridx(2),account(3),itemID(4),amount(5),
//  serverID(6),giveTo(7,大写)]。
// 应答（PublishBuyItemGive@0x6cf50）：
// [echo,"buy_item_give",useridx,result,item,amount,points,orderID,givee]。
func (s *Server) cmdBuyItemGive(sess *session, echo string, vl VarList) {
	sv := s.getServerLocked(sess.connid, 0)
	if sv == nil {
		s.logf("(GameCharge::ReceiveBuyItemGive)Server not found")
		return
	}
	serverID := sv.serverID
	if serverID == 0 {
		serverID = vl.Int(6)
	}
	if serverID == 0 {
		s.logf("(GameCharge::ReceiveBuyItemGive)Need server id")
		return
	}
	useridx := vl.Int(2)
	var u *userInfo
	if sv.serverID != 0 {
		u = s.getUserActiveLocked(sv.serverID, useridx, sess.connid, 0)
	}
	t := ItemTrade{
		UserIdx: useridx,
		Account: upper(vl.StringVal(3)),
		ItemID:  vl.Int(4),
		Amount:  vl.Int(5),
		GiveTo:  upper(vl.StringVal(7)),
	}
	var account string
	var accID int
	if u != nil {
		account, accID = u.account, u.accID
	} else {
		account = t.Account
	}
	result, item, amount, points, orderID, givee := s.Backend.BuyItemGive(account, accID, t)
	out := VarList{}
	out.AddString(echo)
	out.AddString("buy_item_give")
	out.AddInt(useridx)
	out.AddInt(result)
	out.AddString(item)
	out.AddInt(amount)
	out.AddInt(points)
	out.AddInt(orderID)
	out.AddString(givee)
	s.respond(sess, out)
}

// cmdGetPoints ReceiveRetPoints@0x741c0：[echo,"get_points",useridx(2)]。
// 应答（PublishRetPoints@0x6d2d0）：[echo,"get_points",useridx,result,points]。
func (s *Server) cmdGetPoints(sess *session, echo string, vl VarList) {
	sv := s.requireOnline(sess, "RetPoints")
	if sv == nil {
		return
	}
	u := s.getUserActiveLocked(sv.serverID, vl.Int(2), sess.connid, 0)
	if u == nil {
		s.logf("(GameCharge::ReceiveRetPoints)user not found")
		return
	}
	result, points := s.Backend.RetPoints(u.account, u.accID)
	if result == 1 {
		s.logf("Account: %s ID: %d get points %d succeed", u.account, u.accID, points)
	} else {
		s.logf("Account: %s ID: %d get points %d failed result %d",
			u.account, u.accID, points, result)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("get_points")
	out.AddInt(u.index)
	out.AddInt(result)
	out.AddInt(points)
	s.respond(sess, out)
}

// cmdValidate ReceiveValidate@0x743e0（无连接前置）：
// [echo,"validate",account(2,大写),passwd(3,大写),ip(4),?(5),mode(6)]。
// 应答（PublishValidate@0x6d550）：
// [echo,"validate",1,account,result,accID,"",points,"",chargeType,
//  deadline(double),ext]
func (s *Server) cmdValidate(sess *session, echo string, vl VarList) {
	account := upper(vl.StringVal(2))
	passwd := upper(vl.StringVal(3))
	ip := vl.StringVal(4)
	mode := vl.Int(6)
	res := s.Backend.Validate(account, passwd, ip, mode)
	if res.Result == 1 {
		s.logf("Account: %s ID: %d validate succeed", account, res.AccID)
	} else {
		s.logf("Account: %s validate failed result %d", account, res.Result)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("validate")
	out.AddInt(1)
	out.AddString(account)
	out.AddInt(res.Result)
	out.AddInt(res.AccID)
	out.AddString("")
	out.AddInt(res.Points)
	out.AddString("")
	out.AddInt(res.ChargeType)
	out.AddDouble(res.Deadline)
	out.AddString(res.ChargeInfo)
	s.respond(sess, out)
}
