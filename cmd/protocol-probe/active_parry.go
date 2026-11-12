package main

import (
	"fmt"
	"log"
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
// clientCustomBlockState=54 即 C2S m54）。应答=官服六件套序
// （writeOfficialParryTokenAck，a.pcapng 5.144/5.695 定版：按下
// MoveSpeed1.9f+RunSpeedAdd-1.2f+InParry+ParryUpSpeedAdd-93+BufferInfo4=26
// +m169 添加+InManualParry；松开反向+m169 移除）——此前 fork 四属性合帧
// 应答缺速度族，客户端 m54 连发 5971 条等不到官服方言（2026-11-12 P2）。
// 重复按下静默；重复松开仍回帧（客户端状态重同步，78 发按值去重
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
	// fork 的 tag2/tag3 双编码：i32 与 i64 都按数值比较 0。
	state := custom.Values[1]
	engage := false
	switch state.Type {
	case 2:
		engage = state.Int32 == 0
	case 3:
		engage = state.Int64 == 0
	case 4, 5:
		number, ok := state.exactInt32()
		if !ok {
			return fmt.Errorf("block state value is not integral: %s", state)
		}
		engage = number == 0
	default:
		return fmt.Errorf("block state value must be numeric, got %s", state)
	}

	alreadyParrying := player.parryingSnapshot()
	if alreadyParrying == engage {
		if !engage {
			// 游离的重复松开：客户端可能错过上一次回帧，重推空态。
			// 必须发**完整两帧对**（169 摘除事件 + 0x10 属性帧）——客户端的
			// 格挡认知锚在 msg169 BuffInParry 事件上，仅属性帧摘不掉：
			// 2026-11-12 真机死锁（旧纪元 bug 残留）客户端带"已格挡"认知
			// 只发 1 等确认、永不再发按下(0)，单属性帧 ack 无法解锁，
			// 两帧对一发即恢复。官服登录段本就有 remove→clear→fill→add
			// 重登记舞步（k.pcapng），多发一次摘除事件方言安全。
			if err := writeOfficialParryTokenAck(link, player, false); err != nil {
				return err
			}
			log.Printf("%s: ack stray manual parry release (client state resync)", remote)
		}
		return nil
	}

	now := time.Now().UTC()
	player.releaseParryBuff()
	if engage {
		player.engageParryBuff(now)
		// 格挡不清怒(2026-11-07 撤销):Java handleManualParry 不动 SP,
		// 55866 复核四次 SP→0 单帧全部嵌在出招序列(294/337/20/24/CanUse)
		// 里、无一先于 msg169 格挡通告(按下帧甚至更靠后),且战斗态/
		// 非战斗态各占一半——"格挡开启清怒"系误读。格挡态本身按
		// "格挡被揍 +2"攒怒。
	}
	if err := writeOfficialParryTokenAck(link, player, engage); err != nil {
		return err
	}
	log.Printf("%s: manual parry engage=%t buff_slot=%d", remote, engage, parryBuffSlot)
	return nil
}

// ---- m54 官服令牌格挡问答（a.pcapng 5.144/5.695 定版，2026-11-12 P2） ----

var (
	propParryTokenMoveSpeed   = clientdata.MustPlayerProperty("MoveSpeed").Index
	propParryTokenRunSpeedAdd = clientdata.MustPlayerProperty("RunSpeedAdd").Index
	propParryTokenUpSpeedAdd  = clientdata.MustPlayerProperty("ParryUpSpeedAdd").Index
	propParryTokenBufferInfo4 = clientdata.MustPlayerProperty("BufferInfo4").Index
)

// handleParryTokenCustom 处理 C2S m54（格挡令牌问答，官服令牌/fork 0-1 双形态）。
// 官服客户端连发令牌版 m54 等六件套应答（a.pcapng：按下=MoveSpeed 1.9f+
// RunSpeedAdd -1.2f+InParry 1+ParryUpSpeedAdd -93+BufferInfo4 26+m169 添加+
// InManualParry 1；松开=m169 移除+BufferInfo4 1+InParry 0+MoveSpeed 4.8f+
// RunSpeedAdd 0.6f+ParryUpSpeedAdd 10+InManualParry 0，「连发直到应答到达」）。
// 令牌值每会话成对随机，按下/松开判定=状态机（非格挡态收到→按下，
// a 会话三对完美交替实证）；fork 客户端 0/1 形态按值（1=按下持续上报、
// 0=松开）。此前全天 5971 条 m54 零应答——客户端 m54 连发不停的同源根因，
// 亦是「官服应答方言正确→客户端切官服模式」假说（P2）的探针。
// writeOfficialParryTokenAck 官服 m54 应答（单属性帧序照 a.pcapng，
// m169 与 78 通道同构九值帧；速度对照抄官服样本值 1.9/-1.2 → 4.8/0.6）。
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
		if err := single(propParryTokenMoveSpeed, "MoveSpeed", clientdata.Float32Value(1.9)); err != nil {
			return err
		}
		if err := single(propParryTokenRunSpeedAdd, "RunSpeedAdd", clientdata.Float32Value(-1.2)); err != nil {
			return err
		}
		if err := single(propInParry, "InParry", clientdata.Int32Value(1)); err != nil {
			return err
		}
		if err := single(propParryTokenUpSpeedAdd, "ParryUpSpeedAdd", clientdata.Int32Value(-93)); err != nil {
			return err
		}
		// BufferInfo4 表线型=7(串)：无 type 字节方言下必须发串（官服帧带
		// 显式 type=0x01 override 故可发 int；我们无 type 字节→按表线型，
		// 发 int 会被当串长度域解出 Out range，trace.log 23:53 实证）。
		if err := single(propParryTokenBufferInfo4, "BufferInfo4", clientdata.StringValue("26")); err != nil {
			return err
		}
		if err := link.WriteFrame(notice); err != nil {
			return err
		}
		return single(propInManualParry, "InManualParry", clientdata.Int32Value(1))
	}
	if err := link.WriteFrame(notice); err != nil {
		return err
	}
	if err := single(propParryTokenBufferInfo4, "BufferInfo4", clientdata.StringValue("1")); err != nil {
		return err
	}
	if err := single(propInParry, "InParry", clientdata.Int32Value(0)); err != nil {
		return err
	}
	if err := single(propParryTokenMoveSpeed, "MoveSpeed", clientdata.Float32Value(4.8)); err != nil {
		return err
	}
	if err := single(propParryTokenRunSpeedAdd, "RunSpeedAdd", clientdata.Float32Value(0.6)); err != nil {
		return err
	}
	if err := single(propParryTokenUpSpeedAdd, "ParryUpSpeedAdd", clientdata.Int32Value(10)); err != nil {
		return err
	}
	return single(propInManualParry, "InManualParry", clientdata.Int32Value(0))
}
