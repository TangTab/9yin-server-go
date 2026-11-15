package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// CLIENT_CUSTOMMSG_ACTIVE_PARRY 是现代客户端用于主动招架控制的共享 C->S 入口。
// 它被刻意与 211 分开：
// 各个套路用于提交攻击数据，而招架则是玩家的战斗状态请求，
// 该请求必须等到后续处理来袭攻击的命中判定时
// 再进行求值。
const clientCustomActiveParry int32 = 218

// activeParryRequest 刻意对 custom_sender 已知的两个字段做无损保留：
// 子类型（subtype）及其随附参数（argument）。我们保留原始的
// TVarList 值，而不是把第二个值强转成猜测的
// 枚举；其确切的游戏内含义将由实机抓包来判定。
type activeParryRequest struct {
	Subtype  int32
	Argument clientCustomValue
	At       time.Time
}

// handleActiveParryCustom 确立权威的入口边界。
// 它接受现代 TVarList 写入方使用的所有整数编码（门派
// UI 已经证明某些表单会把枚举值以 double 类型发出）。
// 它不虚构视觉动作，也不虚构固定的招架窗口：
// 这些取决于原生的命中结果路径，而该路径尚未得到验证。
func handleActiveParryCustom(player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomActiveParry {
		return false, nil
	}
	if player == nil {
		return true, fmt.Errorf("active parry before scene player exists")
	}
	if len(custom.Values) < 3 {
		return true, fmt.Errorf("active parry requires subtype and argument")
	}
	subtype, ok := custom.Values[1].exactInt32()
	if !ok {
		return true, fmt.Errorf("active parry subtype must be an integral TVarList value, got %s", custom.Values[1])
	}
	request := activeParryRequest{Subtype: subtype, Argument: custom.Values[2], At: time.Now().UTC()}
	player.setActiveParry(request)
	log.Printf("%s: accepted active parry subtype=%d argument=%s", remote, subtype, request.Argument)
	return true, nil
}

// CLIENT_CUSTOMMSG_BLOCK_STATE（fork handleBlockStateCustom：值 0=按下
// 格挡、非 0=松开；Java handleManualParry engage==0 同义）。
const clientCustomBlockState int32 = 54

// propLogicState 是预警切换回帧的单属性索引（官方表 463 号，byte 型）。
const propLogicState uint16 = 463

// handleBlockStateCustom 见下方；handleChangeWarningCustom 还原 fork
// active_parry.go:186-250 的预警切换：零参消息，仅在普通(0)↔预警(2)间
// 翻转；出招中/格挡中/异常态拒绝。回单帧 0x10 LogicState byte。
func handleChangeWarningCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomChangeWarning {
		return false, nil
	}
	if player == nil {
		return true, fmt.Errorf("change warning before scene player exists")
	}
	if len(custom.Values) != 1 {
		log.Printf("%s: reject change warning with extra payload nvalues=%d", remote, len(custom.Values)-1)
		return true, fmt.Errorf("change warning expects no payload, got %d extra values", len(custom.Values)-1)
	}
	state := player.logicStateSnapshot()
	if state != 0 && state != 2 {
		log.Printf("%s: reject change warning from logic state %d", remote, state)
		return true, nil
	}
	if casting := player.currentSkillSnapshot(); casting != "" {
		log.Printf("%s: reject change warning while casting %s", remote, casting)
		return true, nil
	}
	if player.parryingSnapshot() {
		log.Printf("%s: reject change warning while parrying", remote)
		return true, nil
	}
	next := byte(2)
	if state == 2 {
		next = 0
	}
	player.setLogicState(next)
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propLogicState, Name: "LogicState", Value: clientdata.ByteValue(next)},
	})
	if err != nil {
		return true, fmt.Errorf("encode logic state: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return true, err
	}
	log.Printf("%s: change warning logic_state %d -> %d", remote, state, next)
	return true, nil
}

// handleBlockStateCustom 还原手动格挡（Java handleManualParry 同源，
// clientCustomBlockState=54 即 C2S m54）。应答=外服私服六件套序
// （writeOfficialParryTokenAck，a.pcapng 5.2/5.7s 定版：按下
// MoveSpeed+RunSpeedAdd+InParry+ParryUpSpeedAdd+BufferInfo4 无时长槽串
// +m169 添加+InManualParry；松开反向+m169 移除）——此前 fork 四属性合帧
// 应答缺速度族，客户端 m54 连发 5971 条等不到外服私服方言（2026-11-12 P2）。
// 按下方言双形：fork 协议模式 0/1 按值（0=按下）；官服族客户端发令牌版
// （i64 成对随机令牌，2026-11-15 真机 58929 会话实证），按 parryTokenTransition
// 状态机判定。重复按下/松开均回帧（客户端状态重同步，78 发按值去重
// =50 沿=50 回帧的 F 盘实证）。非战斗也可按格挡（不切 LogicState）。
func handleBlockStateCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) error {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomBlockState {
		return nil
	}
	if player == nil {
		return fmt.Errorf("block state before scene player exists")
	}
	if len(custom.Values) < 2 {
		return fmt.Errorf("block state requires a value, got %d values", len(custom.Values))
	}
	state := custom.Values[1]
	// 虚招破防后的不可格挡窗：m54 全部静默吞并记住散逸令牌（格挡.pcapng
	// 定版：破防后客户端发新令牌×4 重试约 6s，官方零回应——窗后同值到达
	// 继续静默防误判成新按下）；真实按下（异值）走正常受理并清散逸记忆。
	if player.cantParrySnapshot() {
		player.noteParryStrayToken(state)
		return nil
	}
	if player.parryStraySnapshot() != 0 {
		if player.parryStrayMatches(state) {
			return nil
		}
		player.clearParryStrayToken()
	}
	// 双方言判定：fork 协议模式客户端发 0/1（0=按下、非 0=松开；Java
	// handleManualParry engage==0 同义）；官服族客户端（B 线官方方言登录
	// 后）的 m54 是**每帧状态上报**——值=基调（首条上报，未格挡）、
	// 值≠基调=格挡中（parryStateReport；2026-11-15 真机 57719 会话
	// 6984 条同值零间隙定案）。0/1 按值直判。
	engage := false
	tokenDialect := false
	stateReport := func(value int64) bool {
		tokenDialect = true
		return player.parryStateReport(value)
	}
	switch state.Type {
	case 2:
		if state.Int32 == 0 || state.Int32 == 1 {
			engage = state.Int32 == 0
		} else {
			engage = stateReport(int64(state.Int32))
		}
	case 3:
		if state.Int64 == 0 || state.Int64 == 1 {
			engage = state.Int64 == 0
		} else {
			engage = stateReport(state.Int64)
		}
	case 4, 5:
		number, ok := state.exactInt32()
		if !ok {
			return fmt.Errorf("block state value is not integral: %s", state)
		}
		if number == 0 || number == 1 {
			engage = number == 0
		} else {
			engage = stateReport(int64(number))
		}
	default:
		return fmt.Errorf("block state value must be numeric, got %s", state)
	}

	alreadyParrying := player.parryingSnapshot()
	if alreadyParrying == engage {
		if tokenDialect {
			// 状态上报无变迁：零帧（每帧上报，回任何帧都会自激成洪流）。
			return nil
		}
		if !engage {
			// 游离的重复松开：客户端可能错过上一次回帧，重推空态。
			// 必须发**完整两帧对**（169 摘除事件 + 0x10 属性帧）——客户端的
			// 格挡认知锚在 msg169 BuffInParry 事件上，仅属性帧摘不掉：
			// 2026-11-12 真机死锁（旧纪元 bug 残留）客户端带"已格挡"认知
			// 只发 1 等确认、永不再发按下(0)，单属性帧 ack 无法解锁，
			// 两帧对一发即恢复。外服私服登录段本就有 remove→clear→fill→add
			// 重登记舞步（k.pcapng），多发一次摘除事件方言安全。
			if err := writeOfficialParryTokenAck(link, player, false); err != nil {
				return err
			}
			log.Printf("%s: ack stray manual parry release (client state resync)", remote)
		}
		return nil
	}

	player.releaseParryBuff()
	if engage {
		player.engageParryBuff()
		// 格挡不清怒(2026-11-07 撤销):Java handleManualParry 不动 SP,
		// 55866 复核四次 SP→0 单帧全部嵌在出招序列(294/337/20/24/CanUse)
		// 里、无一先于 msg169 格挡通告(按下帧甚至更靠后),且战斗态/
		// 非战斗态各占一半——"格挡开启清怒"系误读。格挡态本身按
		// "格挡被揍 +2"攒怒。
	}
	if err := writeOfficialParryTokenAck(link, player, engage); err != nil {
		return err
	}
	// 格挡期间临时战斗姿态（2026-11-16 定版策略）：官服按住右键=拔剑
	// 格挡的备战造型；真机实证只有 LogicState=1 立得住架势（战斗段 6s
	// 保持 vs 预警(2)段 50ms 即弃）。Z 持剑切换（m292）所有权不侵占——
	// 按下时记下 0/2 原值临时升 1，松开/破防原值回写。官方按下簇本身
	// 不带 463（a.pcapng），此帧是我方让客户端架势立住的撑持方言。
	if engage {
		if state := player.logicStateSnapshot(); state == 0 || state == 2 {
			player.setParryPreBlockState(state)
			player.setLogicState(1)
			player.writeLogicStateFrame(link, 1)
		}
	} else if pre := player.takeParryPreBlockState(); pre != 0xFF {
		player.setLogicState(pre)
		player.writeLogicStateFrame(link, pre)
	}
	log.Printf("%s: manual parry engage=%t", remote, engage)
	return nil
}

// ---- m54 外服私服令牌格挡问答（a.pcapng 5.144/5.695 定版，2026-11-12 P2） ----

var (
	propParryTokenMoveSpeed   = clientdata.MustPlayerProperty("MoveSpeed").Index
	propParryTokenRunSpeedAdd = clientdata.MustPlayerProperty("RunSpeedAdd").Index
	propParryTokenUpSpeedAdd  = clientdata.MustPlayerProperty("ParryUpSpeedAdd").Index
	propParryTokenBufferInfo4 = clientdata.MustPlayerProperty("BufferInfo4").Index
)

// handleParryTokenCustom 处理 C2S m54（格挡令牌问答，外服私服令牌/fork 0-1 双形态）。
// 外服私服客户端连发令牌版 m54 等六件套应答（a.pcapng：按下=MoveSpeed 1.9f+
// RunSpeedAdd -1.2f+InParry 1+ParryUpSpeedAdd -93+BufferInfo4 无时长槽串+
// m169 添加+InManualParry 1；松开=m169 移除+BufferInfo4 清空+InParry 0+
// MoveSpeed 4.8f+RunSpeedAdd 0.6f+ParryUpSpeedAdd 10+InManualParry 0，
// 「连发直到应答到达」）。
// 令牌值每会话成对随机，按下/松开判定=状态机（非格挡态收到→按下，
// a 会话三对完美交替实证）；fork 客户端 0/1 形态按值（1=按下持续上报、
// 0=松开）。此前全天 5971 条 m54 零应答——客户端 m54 连发不停的同源根因，
// 亦是「外服私服应答方言正确→客户端切外服私服模式」假说（P2）的探针。
// writeOfficialParryTokenAck 外服私服 m54 应答（单属性帧序照 a.pcapng，
// m169 与 78 通道同构九值帧；速度对照抄外服私服样本值 1.9/-1.2 → 4.8/0.6）。
// BufferInfo4 槽串=curated.md a.pcapng 5.2/5.7s 逐字节定版：按下
// '11CE,701E5A4,3AD5FB,1,0,0'、松开空串。seg5 到期/seg6 剩余跳恒 0——
// 格挡 buff 无时长、事件驱动（按下登记，松开/被打破摘除）。曾误抄
// 字面量 "26"/"1"：客户端解析不出 0x11CE 登记、非战斗姿态按下即塌
// （战斗态由 LogicState 撑住掩盖，与旧纪元 bug 同症不同因）。
func writeOfficialParryTokenAck(link sceneMessageConnection, player *playerActor, engage bool) error {
	single := func(index uint16, name string, value clientdata.Value) error {
		frame, err := objectSingleProperty(playerObjectID, playerOwnerID, index, name, value)
		if err != nil {
			return fmt.Errorf("encode parry token %s: %w", name, err)
		}
		return link.WriteFrame(frame)
	}
	name := player.snapshotName()
	notice, err := serverCustomIntMessage(169,
		customInt(boolInt32(!engage)),
		customString("BuffInParry"),
		customObject(playerObjectID, playerOwnerID),
		customObject(playerObjectID, playerOwnerID),
		customWideString(name),
		customWideString(name),
		customInt64(0),
		customInt(0),
	)
	if err != nil {
		return fmt.Errorf("encode parry token notice: %w", err)
	}
	if engage {
		// 速度族=a.pcapng 5.144s 按下簇逐字节（f32 位形直接复刻：
		// MoveSpeed 0x3FE28F5C≈1.764、RunSpeedAdd 0xC00851EC≈-2.04——
		// 旧注释的 1.9/-1.2 系误读；按下即塌疑与速度值不符客户端自检有关）。
		if err := single(propParryTokenMoveSpeed, "MoveSpeed", clientdata.Float32Value(math.Float32frombits(0x3FE28F5C))); err != nil {
			return err
		}
		if err := single(propParryTokenRunSpeedAdd, "RunSpeedAdd", clientdata.Float32Value(math.Float32frombits(0xC00851EC))); err != nil {
			return err
		}
		if err := single(propInParry, "InParry", clientdata.Int32Value(1)); err != nil {
			return err
		}
		if err := single(propParryTokenUpSpeedAdd, "ParryUpSpeedAdd", clientdata.Int32Value(-93)); err != nil {
			return err
		}
		// BufferInfo4 外服私服=六段 CSV 槽串（curated.md a.pcapng 5.2s
		// '11CE,701E5A4,3AD5FB,1,0,0'），seg5 到期/seg6 剩余跳恒 0：
		// 格挡 buff 无时长，靠 m169 事件对增删。曾误抄字面量 "26"，
		// 客户端解析不出 0x11CE 登记→非战斗姿态按下即被撤销（战斗态
		// 由 LogicState 撑住掩盖）。表线型 7(串)：按线型发串，发 int
		// 会被当串长度域解出 Out range（trace.log 23:53 实证）。
		if err := single(propParryTokenBufferInfo4, "BufferInfo4", clientdata.StringValue(
			fmt.Sprintf("%X,%X,%X,1,0,0", parryBuffStaticData, playerObjectID, playerOwnerID))); err != nil {
			return err
		}
		if err := link.WriteFrame(notice); err != nil {
			return err
		}
		if err := single(propParryValue, "ParryValue", clientdata.Int32Value(player.parryValueTotalSnapshot())); err != nil {
			return err
		}
		return single(propInManualParry, "InManualParry", clientdata.Int32Value(1))
	}
	if err := link.WriteFrame(notice); err != nil {
		return err
	}
	// 外服私服松开=BufferInfo4 清空串（curated.md a.pcapng 5.7s 'BufferInfo4='）。
	if err := single(propParryTokenBufferInfo4, "BufferInfo4", clientdata.StringValue("")); err != nil {
		return err
	}
	if err := single(propInParry, "InParry", clientdata.Int32Value(0)); err != nil {
		return err
	}
	// 松开恢复族=a.pcapng 5.695s 逐字节：MoveSpeed 0x40900000=4.5、
	// RunSpeedAdd 0x3F19999A=0.6。
	if err := single(propParryTokenMoveSpeed, "MoveSpeed", clientdata.Float32Value(math.Float32frombits(0x40900000))); err != nil {
		return err
	}
	if err := single(propParryTokenRunSpeedAdd, "RunSpeedAdd", clientdata.Float32Value(math.Float32frombits(0x3F19999A))); err != nil {
		return err
	}
	if err := single(propParryTokenUpSpeedAdd, "ParryUpSpeedAdd", clientdata.Int32Value(10)); err != nil {
		return err
	}
	return single(propInManualParry, "InManualParry", clientdata.Int32Value(0))
}
