package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

// 九阴志（jyz）域：章节/事件状态是服务端驱动的「功能开放」数据源。
// 客户端 form_9yinzhi 界面与原生记录消费者（按官方下标 120 绑定
// data_record）据此解锁章节门槛的系统；lua 侧查不到 414/770/
// data_record 引用，全部走 C++ 原生。
//
// 官方方言（2026-11-08 官服 c.pcapng 实证，替代此前 native-menu
// 逆向的 0x1E 419 单 int 方言——419 在官服记录表另有其主：plug_record
// 奇珍系统，与九阴志无关）：
//
//   - S2C 0x1E custom 414 ['gamestep', step, flag]（抓包 #111：
//     `1e 04 00 02 9e010000 06 09...gamestep\0 02 06000000 02 01000000`），
//     登录爆发①紧跟账户帧 655 之后、0x0B 场景头之前；
//   - data_record（官方下标 120，1 列 u16）58 行**打包一帧**（#256：
//     `11 00 <obj> <owner> 78 00 0000 3a00 <u16×58>`），行值=事件
//     真实进度（26146/36588/…），非 native-menu 的逐行全 2；
//   - C2S custom 770 ('', step) 登录早期上报本地步骤（#12）；
//   - S2C custom 770 事件明细应答（#2026，由 mind 排行查询链触发）：
//     [770, 0, (id, 0, 1, state)×8]，id=75601..75606/75641/75642，
//     state∈{1,2}。
//
// init 默认 jyzStep=6、jyzEventState=2，环境变量
// NINEYIN_JYZ_STEP（仅 >0 生效）与 NINEYIN_JYZ_EVENT_STATE
// （>=0 生效，低 8 位截断）可覆盖。
const (
	// serverJyzGameStepMessage 0x1E 首值=414，次值窄字符串 'gamestep'
	// 命令名寻址（官服 #111 逐字节）。
	serverJyzGameStepMessage int32 = 414
	// serverJyzLegacyStepMessage 0x1E 419 单 int 步骤帧（native-menu
	// #1683 `1e 02 00 02 a3010000 02 06000000`；419 记录表在官服另有
	// 其主=plug_record 奇珍，此帧仅 native 方言兼容）。
	serverJyzLegacyStepMessage int32 = 419
	// serverJyzDetailMessage 0x1E 事件明细应答帧 ID（官服 #2026）。
	serverJyzDetailMessage int32 = 770
	// clientCustomJyzQuery 客户端步骤上报/查询（官服 C2S #12 ('', 6)）。
	clientCustomJyzQuery int32 = 770
	// jyzEventCount data_record 官方打包行数（#256 rowCount=0x3a）。
	jyzEventCount = 58
)

// jyzEventEntry 770 明细应答里的单事件条目：官服四元组 (id, 0, 1, state)。
type jyzEventEntry struct {
	id    int32
	state int32
}

// jyzDetailEntries 官服 #2026 抓包逐值照录的默认事件集。
// 状态 1=进行中前段、2=后段（75605/75641/75606 为 2）。
var jyzDetailEntries = []jyzEventEntry{
	{75601, 1}, {75602, 1}, {75603, 1}, {75604, 1}, {75642, 1},
	{75605, 2}, {75641, 2}, {75606, 2},
}

// jyzConfig 九阴志会话配置。
type jyzConfig struct {
	step       int32
	eventState uint16
}

// defaultJyzConfig 步骤缺省与官服 #111 gamestep=6 一致。
func defaultJyzConfig() jyzConfig {
	return jyzConfig{step: 6, eventState: 2}
}

// resolveJyzConfig 复刻 native-menu init 的环境变量语义：STEP 仅接受
// 正整数，EVENT_STATE 接受非负整数且按低 8 位截断（asm MOVZX DL）。
func resolveJyzConfig(getenv func(string) string) jyzConfig {
	cfg := defaultJyzConfig()
	if v, err := strconv.Atoi(getenv("NINEYIN_JYZ_STEP")); err == nil && v > 0 {
		cfg.step = int32(v)
	}
	if v, err := strconv.Atoi(getenv("NINEYIN_JYZ_EVENT_STATE")); err == nil && v >= 0 {
		cfg.eventState = uint16(uint8(v))
	}
	return cfg
}

// jyzRecordsConfig 进程级配置（启动环境决定，运行期不变）。
var jyzRecordsConfig = resolveJyzConfig(os.Getenv)

// jyzStepFrame 0x1E 414 ['gamestep', step, 1]（官服 #111 同形）。
// 注意：此帧必须在 switch sync 之后收尾下发（pushJyzGameStepFinal），
// sync 会重置 414 域状态，先发的 gamestep 会被抹掉（native #1743→#1744
// 帧序铁律；此前 gamestep 在 sync 前发导致功能入口不出）。
func jyzStepFrame(cfg jyzConfig) ([]byte, error) {
	return serverCustomIntMessage(serverJyzGameStepMessage, customString("gamestep"), customInt(cfg.step), customInt(1))
}

// jyzLegacyStepFrame 0x1E 419 单 int 步骤帧（native-menu #1683 同形，
// 官服不发）。quest49 簇之后、data_record 之前下发。
func jyzLegacyStepFrame(cfg jyzConfig) ([]byte, error) {
	return serverCustomIntMessage(serverJyzLegacyStepMessage, customInt(cfg.step))
}

// jyzDetailFrame 0x1E 770 [0, (id, 0, 1, state)×n]（官服 #2026 同形）。
func jyzDetailFrame() ([]byte, error) {
	values := make([]serverCustomValue, 0, 2+4*len(jyzDetailEntries))
	values = append(values, customInt(0))
	for _, entry := range jyzDetailEntries {
		values = append(values, customInt(entry.id), customInt(0), customInt(1), customInt(entry.state))
	}
	return serverCustomIntMessage(serverJyzDetailMessage, values...)
}

// pushJyzRecords ClientReady 一次性下发九阴志数据部（native 62311
// #1680-#1742 帧序）：quest49 簇 → 419 步骤 → data_record 58 行单帧
// 打包（官服 #256 方言）。gamestep 步骤帧不在此处——见
// pushJyzGameStepFinal（必须在 switch sync 之后收尾）。
func pushJyzRecords(link sceneMessageConnection, cfg jyzConfig, ownedMountPacks map[uint32]bool) error {
	if err := pushJyzQuestCluster(link, ownedMountPacks); err != nil {
		return fmt.Errorf("push quest cluster: %w", err)
	}
	if err := pushJyzDataRecords(link, cfg); err != nil {
		return err
	}
	log.Printf("pushed jyz data: quest49 cluster + 419 step=%d + data_record rows=%d", cfg.step, jyzEventCount)
	return nil
}

// pushJyzDataRecords 仅 419 步骤 + data_record 打包行（770 查询应答
// 复用；quest 簇只进 ClientReady 一次性链，重发会重复行）。
func pushJyzDataRecords(link sceneMessageConnection, cfg jyzConfig) error {
	legacy, err := jyzLegacyStepFrame(cfg)
	if err != nil {
		return fmt.Errorf("encode jyz legacy step: %w", err)
	}
	if err := link.WriteFrame(legacy); err != nil {
		return fmt.Errorf("write jyz legacy step: %w", err)
	}
	rows := make([]uint16, jyzEventCount)
	for i := range rows {
		rows[i] = cfg.eventState
	}
	if err := link.WriteFrame(serverRecordAddWordRows(playerObjectID, playerOwnerID, recordDataRecord, rows)); err != nil {
		return fmt.Errorf("add data_record rows: %w", err)
	}
	return nil
}

// pushJyzGameStepFinal 开关同步之后收尾下发 414 'gamestep' 步骤帧
// （native #1744 在 switch sync #1743 之后；sync 重置 414 域状态，
// 先发的 gamestep 会被抹掉——功能入口不出面的根因）。
func pushJyzGameStepFinal(link sceneMessageConnection, cfg jyzConfig) error {
	frame, err := jyzStepFrame(cfg)
	if err != nil {
		return fmt.Errorf("encode jyz gamestep: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return fmt.Errorf("write jyz gamestep: %w", err)
	}
	return nil
}

// handleJyzQueryCustom 应答 C2S 770 (player_name, msgID)：回 419+data
// 数据部 + 414 gamestep + 770 事件明细帧。官服对该查询无即时应答
// （步骤帧已在登录推过、明细帧由 UI 查询链触发），这里补齐保证
// 客户端状态完整；quest 簇不重发（会重复行）。
func handleJyzQueryCustom(link sceneMessageConnection, custom clientCustomMessage, remote string) error {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomJyzQuery {
		return fmt.Errorf("jyz query custom malformed: %v", custom.Values)
	}
	if err := pushJyzDataRecords(link, jyzRecordsConfig); err != nil {
		return err
	}
	if err := pushJyzGameStepFinal(link, jyzRecordsConfig); err != nil {
		return err
	}
	detail, err := jyzDetailFrame()
	if err != nil {
		return fmt.Errorf("encode jyz detail: %w", err)
	}
	if err := link.WriteFrame(detail); err != nil {
		return fmt.Errorf("write jyz detail: %w", err)
	}
	log.Printf("%s: answered jyz query via data + gamestep step=%d + detail entries=%d", remote, jyzRecordsConfig.step, len(jyzDetailEntries))
	return nil
}
