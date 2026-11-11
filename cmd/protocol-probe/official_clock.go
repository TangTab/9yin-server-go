package main

// 官服 1Hz 时钟对（技能.pcapng 全程实证：msg294×1289 + op0x38×1187，
// 覆盖 1301s 整场，~1.00s cadence）——客户端的服务端时间基准通道：
//
//	msg294 = [294, 进程ms(i64, 活值), 开机戳µs(i64, 常量), 开机unix秒
//	         (i64, 常量), 运行秒(f64, 活值)]（0x1E 信封，44B 定长）
//	op0x38 = [38, 进程ms(u32), 0, 序号(u32, 每帧+1), 04 00 00,
//	         clock(u32), 0, 01, clock(u32), 0]（30B 定长；
//	         clock=(运行秒<<16)|(ms&0xffff)）
//
// 走踢帧（rushWalkKickFrame）后客户端要按服务端时间插值 0x21 位移——
// 无钟基准则模型冻结不走（2026-11-12 三轮真机：踢帧字节逐槽镜像后仍
// 原地不动，官服踢后 0.485s 即起走且全程带钟）。官服钟对为全程常开；
// 本服先按最小变量 scoped：冲锋 pending 在途期间 1Hz 维持 + 受理时
// 先发一对（赶在 +150ms 踢帧之前），验证后再评估全局常开。

import (
	"encoding/binary"
	"log"
	"sync/atomic"
	"time"
)

var officialClockProcStart = time.Now()

// officialClock294Frame msg294 对钟帧（44B 定长同官服）。
func officialClock294Frame(now time.Time) ([]byte, error) {
	elapsed := now.Sub(officialClockProcStart)
	return serverCustomIntMessage(294,
		customInt64(int64(elapsed.Milliseconds())),
		customInt64(officialClockProcStart.UnixMicro()),
		customInt64(officialClockProcStart.Unix()),
		customFloat64(elapsed.Seconds()),
	)
}

// officialClock38Frame op0x38 时钟帧（30B 定长同官服）。
func officialClock38Frame(now time.Time) []byte {
	elapsed := now.Sub(officialClockProcStart)
	procMs := uint32(elapsed.Milliseconds())
	clock := uint32(elapsed.Seconds())<<16 | (procMs & 0xFFFF)
	frame := make([]byte, 0, 30)
	frame = append(frame, 0x38)
	frame = binary.LittleEndian.AppendUint32(frame, procMs)
	frame = append(frame, 0)
	frame = binary.LittleEndian.AppendUint32(frame, officialClock38Seq.Add(1))
	frame = append(frame, 4, 0, 0)
	frame = binary.LittleEndian.AppendUint32(frame, clock)
	frame = binary.LittleEndian.AppendUint32(frame, 0)
	frame = append(frame, 1)
	frame = binary.LittleEndian.AppendUint32(frame, clock)
	frame = binary.LittleEndian.AppendUint32(frame, 0)
	return frame
}

// officialClock38Seq op0x38 每帧序号（官服逐帧 +1）。
var officialClock38Seq atomic.Uint32

// writeOfficialClockPair 立即发一对时钟帧（受理簇后、踢帧前引导用）。
func writeOfficialClockPair(link sceneMessageConnection, remote string) {
	now := time.Now()
	if frame, err := officialClock294Frame(now); err == nil {
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("%s: clock 294: %v", remote, err)
		}
	} else {
		log.Printf("%s: clock 294 encode: %v", remote, err)
	}
	if err := link.WriteFrame(officialClock38Frame(now)); err != nil {
		log.Printf("%s: clock 38: %v", remote, err)
	}
}

// scheduleOfficialClockTicker 冲锋 pending 在途期间的 1Hz 钟对维持
// （自重排 AfterFunc 链；pending 消失/换技能即停）。
func scheduleOfficialClockTicker(link sceneMessageConnection, player *playerActor, skillID, remote string) {
	var tick func()
	tick = func() {
		if pending := player.peekPendingCast(time.Now().UTC()); pending == nil || pending.skillID != skillID {
			return
		}
		writeOfficialClockPair(link, remote)
		time.AfterFunc(time.Second, tick)
	}
	time.AfterFunc(time.Second, tick)
}
