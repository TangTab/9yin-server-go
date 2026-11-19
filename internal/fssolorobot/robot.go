// Package fssolorobot 复刻官方 FsSoloRobot.dll + game_robot.exe
// （solo 集群压测机器人客户端，BGserverCJ\Debug，2014-03-22 快照，
// FsSoloRobot.pdb 3487 公共符号）。
//
// 权威事实源：FsSoloRobot.pdb 符号直读 + 反汇编终局
// （tmp_research/solorobot/）+ 官方 game_robot.ini +
// Lua\solorobot\main.lua 原文；全解沉淀
// docs/official/FsSoloRobot全解与Go复刻.md（本包伴生文档）。
//
// # 官方架构（反汇编定案）
//
//	game_robot.exe（通用 solo 壳，MainEntity=SoloRobot + MainScript=main）
//	 └ FsSoloRobot.dll ＝ SoloRobot 实体（IEntity，自带 CReactor/CEchoHook）
//	    ├ 外连受控方：向 [Outer]（缺省 127.0.0.1:2001，网关）发起大量
//	    │  TCP 连接，每连接模拟一个玩家：
//	    │  login(0x02) → login_succeed(0x04) → choose(0x04)/create(0x05)
//	    │  → entry_scene(0x0B) → 应答(0x09) → ingame 每 MoveSeconds 秒
//	    │  move(0x0B)（±5m 随机走，dest.y=4.0）
//	    ├ 线路帧法：0xEE 转义（EE 00）+ EE EE 终止，无长度前缀无加密
//	    │  （MessageEncodeKey 为注释掉的死配置）
//	    ├ 0x28 压缩包裹：QuickLZ level-2，解压体 = add_object 报文体
//	    ├ 模块状态机（solorobotstate.cpp）：Unknown --(Start)--> Opened
//	    │  --(Pause)--> Paused --(Resume)--> Opened；Closed 不可达
//	    ├ 双定时器：检查 1s（Opened 态驱动移动循环）、
//	    │  起连 ConnectConcurrent/ConnectSpeed 秒（每次 ≤ConnectConcurrent 个）
//	    └ echo 维护口（CEchoHook 服务端，[Echo] 缺省 127.0.0.1:15009）：
//	       单客户端、welcome game robot、quit/end（官方 Lua on_input 族）
//
// # 与官方的差异（原则：线上字节一致，进程内设施取 Go 等价）
//
//   - CReactor（IOCP）不复刻：每连接一读协程 + 自管写锁；连接失败
//     经 dial 错误回调（官方 OnConnect 第 2 参 == 0 路径）
//   - 总量语义保持：total_raised 单调不回落（无自动重连）、
//     ConnectCount=成功数、FailedCount=失败数
//   - 解压缓冲：官方全实体共享单缓冲（单线程反应器安全），Go 版
//     按消息分配
//   - echo 调试命令：官方 Lua 实体自省族（getp/method/...）不适用
//     Go 实体，内置 quit/end/status（欢迎语与 quit/end 语义一致）
package fssolorobot

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"time"
)

// ModuleState 是官方模块状态机枚举（solorobotstate.cpp，this+0x2c）：
// 0=Unknown / 1=Opened / 2=Paused / 3=Closed。
type ModuleState int

// 官方四态（GetState@0x10060b00 呈现名）。
const (
	ModuleUnknown ModuleState = iota
	ModuleOpened
	ModulePaused
	ModuleClosed
)

// String 官方 GetState 的呈现名。
func (s ModuleState) String() string {
	switch s {
	case ModuleOpened:
		return "Opened"
	case ModulePaused:
		return "Paused"
	case ModuleClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// 官方状态机事件（RobotStateMachine 第一参）：0=TimerCheck（每秒）、
// 1=Start/Resume、2=Pause/Close。
const (
	moduleEventTick   = 0
	moduleEventStart  = 1
	moduleEventPause  = 2
)

// Robot 复刻 SoloRobot 实体。零值不可用，经 NewRobot 构造。
type Robot struct {
	cfg Config

	mu             sync.Mutex
	module         ModuleState
	connects       map[int]*connect
	nextConnID     int
	totalRaised    int // 官方 +0xd8：累计发起（单调不回落 → 无重连）
	connectCount   int // 官方 +0xdc：TCP 建立成功数
	failedCount    int // 官方 +0xe0：连接失败数

	echo  *EchoConsole
	trace func(format string, args ...any) // 官方 WriteLine = printf+CORE_TRACE
	dial  func(addr string, port int) (net.Conn, error)
	onQuit func() // echo quit → 官方 close() → nx_quit()

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewRobot 建实体。cfg 为官方配置投影；trace 为 WriteLine 等价
// （nil = 丢弃）；dial 可注入假网关（nil = net.DialTimeout）；
// onQuit 在 echo 收到 quit 时回调（官方会退出进程）。
func NewRobot(cfg Config, trace func(string, ...any), dial func(string, int) (net.Conn, error), onQuit func()) *Robot {
	if trace == nil {
		trace = func(string, ...any) {}
	}
	if dial == nil {
		dial = func(addr string, port int) (net.Conn, error) {
			d := net.Dialer{Timeout: 15 * time.Second}
			return d.Dial("tcp", net.JoinHostPort(addr, fmt.Sprint(port)))
		}
	}
	cfg.applyDefaults()
	r := &Robot{
		cfg:      cfg,
		connects: map[int]*connect{},
		trace:    trace,
		dial:     dial,
		onQuit:   onQuit,
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.echo = NewEchoConsole(r.echoLine)
	return r
}

// applyDefaults 兜底（正常经 ParseGameRobotINI 已带官方 Lua 缺省）。
func (c *Config) applyDefaults() {
	if c.MaxConnects <= 0 {
		c.MaxConnects = 1024
	}
	if c.ConnectConcurrent <= 0 {
		c.ConnectConcurrent = 10
	}
	if c.ConnectSpeed <= 0 {
		c.ConnectSpeed = 50
	}
	if c.MoveSeconds <= 0 {
		c.MoveSeconds = 5
	}
	if c.AccountPrefix == "" {
		c.AccountPrefix = "robot"
	}
	if c.PasswordPrefix == "" {
		c.PasswordPrefix = "robot"
	}
	if c.OuterAddr == "" {
		c.OuterAddr = "127.0.0.1"
	}
	if c.OuterPort <= 0 {
		c.OuterPort = 2001
	}
	if c.EchoAddr == "" {
		c.EchoAddr = "127.0.0.1"
	}
	if c.EchoPort <= 0 {
		c.EchoPort = 15009
	}
}

// WriteLine 官方 SoloRobot::WriteLine（printf "%s\r\n" + trace）。
func (r *Robot) WriteLine(format string, args ...any) {
	r.trace(format, args...)
}

// Start 官方 SoloRobot::Start@0x1005c350：echo 受面（EchoAddr 非空才开）
// → 检查定时器（1s）→ 起连定时器（ConnectConcurrent/ConnectSpeed 秒）
// → 状态机事件 1（'connect server opened'）。
func (r *Robot) Start() error {
	if r.cfg.EchoAddr != "" && r.cfg.EchoAddr != "off" {
		if err := r.echo.Start(r.cfg.EchoAddr, r.cfg.EchoPort); err != nil {
			r.trace("start echo console failed: %v", err)
		} else {
			r.trace("echo socket %s:%d", r.cfg.EchoAddr, r.cfg.EchoPort)
		}
	}
	r.wg.Add(2)
	go r.loopTimer(1*time.Second, r.timerCheck)
	raise := time.Duration(float64(r.cfg.ConnectConcurrent)/float64(r.cfg.ConnectSpeed)*float64(time.Second)) / time.Nanosecond
	if raise < time.Millisecond {
		raise = time.Millisecond
	}
	go r.loopTimer(raise, r.timerRaise)
	r.trace("outer socket %s:%d", r.cfg.OuterAddr, r.cfg.OuterPort)
	r.trace("game robot started")
	r.stateMachine(moduleEventStart, 0, 0)
	return nil
}

// Stop 官方 SoloRobot::Stop@0x1005c590：停双定时器 → echo Stop →
// 反应器 Stop（Go：断开全部连接并等读协程）。模块状态不迁移（官方同）。
func (r *Robot) Stop() {
	r.cancel()
	r.echo.Stop()
	r.mu.Lock()
	conns := make([]*connect, 0, len(r.connects))
	for _, c := range r.connects {
		conns = append(conns, c)
	}
	r.mu.Unlock()
	for _, c := range conns {
		if c.nc != nil {
			c.nc.Close()
		}
	}
	r.wg.Wait()
}

func (r *Robot) loopTimer(interval time.Duration, fn func()) {
	defer r.wg.Done()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-t.C:
			fn()
		}
	}
}

// stateMachine 官方 RobotStateMachine@0x10060be0 + 四态处理器
// （solorobotstate.cpp）。返回是否接受。
func (r *Robot) stateMachine(event, arg1, arg2 int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.module {
	case ModuleUnknown: // RobotStateUnknown@0x10060d10
		switch event {
		case moduleEventStart:
			r.trace("connect server opened")
			r.module = ModuleOpened
			return true
		case moduleEventPause:
			r.trace("Connect server already closed")
			return true
		case moduleEventTick:
			return true
		}
	case ModuleOpened: // RobotStateOpened@0x10060ea0
		switch event {
		case moduleEventPause:
			r.trace("connect server paused")
			r.module = ModulePaused
			return true
		case moduleEventStart:
			r.trace("Connect server already opened")
			return true
		case moduleEventTick:
			r.moveLoopLocked()
			return true
		}
	case ModulePaused: // RobotStatePaused@0x100611b0
		switch event {
		case moduleEventStart:
			r.trace("Connect server resumed")
			r.module = ModuleOpened
			return true
		case moduleEventPause:
			r.trace("Connect server already paused")
			return true
		case moduleEventTick:
			return true
		}
	case ModuleClosed: // RobotStateClosed@0x10061270
		switch event {
		case moduleEventPause:
			r.trace("Connect server close all")
			return true
		case moduleEventTick:
			return true
		}
	}
	state := r.module.String()
	r.trace("(SoloRobot::RobotState%s)error input %d", state, event)
	return false
}

// moveLoopLocked 官方 RobotStateOpened 事件 0（TimerCheck@每秒）：
// 所有连接 wait_seconds++；ingame 且 wait > MoveSeconds → 清零并
// 发 move（dest = 当前 x/z 各加 ±5m 随机）。
// 调用方持有 r.mu。
func (r *Robot) moveLoopLocked() {
	for _, c := range r.connects {
		c.waitSeconds++
		if c.state == ConnInGame && c.waitSeconds > r.cfg.MoveSeconds {
			c.waitSeconds = 0
			destX := c.x + moveRandDelta()
			destZ := c.z + moveRandDelta()
			r.sendToServerLocked(c, buildMove(c.x, c.y, c.z, destX, destZ))
		}
	}
}

// moveRandDelta 官方 rand()*10.0/32767.0 - 5.0（C rand 上限 32767）。
func moveRandDelta() float32 {
	return float32((float64(rand.IntN(32768))/32767.0)*10.0 - 5.0)
}

// timerCheck 官方 TimerCheck@0x1005ef00：状态机事件 0。
func (r *Robot) timerCheck() { r.stateMachine(moduleEventTick, 0, 0) }

// timerRaise 官方 TimerRaise@0x1005ef60：总量未满则本拍最多
// ConnectConcurrent 个新连接（连接异步，槽位即刻占位）。
func (r *Robot) timerRaise() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := 0; i < r.cfg.ConnectConcurrent; i++ {
		if r.totalRaised >= r.cfg.MaxConnects {
			break
		}
		id := r.nextConnID
		r.nextConnID++
		c := &connect{index: r.totalRaised}
		r.connects[id] = c
		r.totalRaised++
		r.trace("raise connect address %s:%d, total %d", r.cfg.OuterAddr, r.cfg.OuterPort, r.totalRaised)
		r.wg.Add(1)
		go r.connectLoop(id, c)
	}
}

// connectLoop 官方 CreateConnector + OnConnect/OnReceive/OnClose 三回调
// 的 Go 等价：拨号（失败 = 官方 OnConnect succeeded==0 路径）→ 成功则
// send_login + 状态 logging → 读循环至断开（OnClose 路径）。
func (r *Robot) connectLoop(id int, c *connect) {
	defer r.wg.Done()
	nc, err := r.dial(r.cfg.OuterAddr, r.cfg.OuterPort)
	r.mu.Lock()
	if _, alive := r.connects[id]; !alive {
		r.mu.Unlock()
		if nc != nil {
			nc.Close()
		}
		return
	}
	if err != nil {
		// 官方 OnConnect 失败路径：销毁连接、failed_count++。
		delete(r.connects, id)
		r.failedCount++
		r.trace("connect address %s:%d failed, total %d", r.cfg.OuterAddr, r.cfg.OuterPort, r.failedCount)
		r.mu.Unlock()
		return
	}
	c.nc = nc
	c.peer = nc.RemoteAddr().String()
	c.state = ConnConnected
	r.connectCount++
	r.mu.Unlock()
	r.trace("connect address %s:%d succeed, total %d", r.cfg.OuterAddr, r.cfg.OuterPort, r.connectCount)

	r.onConnectSuccess(id, c)

	buf := make([]byte, 16<<10)
	for {
		n, rerr := nc.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			r.mu.Lock()
			_, alive := r.connects[id]
			if alive {
				c.feed(chunk, func(payload []byte) {
					r.receiveServerMsgLocked(id, c, payload)
				})
			}
			r.mu.Unlock()
			if !alive {
				nc.Close()
				return
			}
		}
		if rerr != nil {
			r.onClose(id, c, nc)
			return
		}
	}
}

// onConnectSuccess 官方 OnConnect 成功路径尾段：send_login → 状态 2
// logging（0x1005e917）。
func (r *Robot) onConnectSuccess(id int, c *connect) {
	account := fmt.Sprintf("%s%d", r.cfg.AccountPrefix, c.index)
	password := fmt.Sprintf("%s%d", r.cfg.PasswordPrefix, c.index)
	r.mu.Lock()
	r.sendToServerLocked(c, buildLogin(account, password, ""))
	c.state = ConnLogging
	r.mu.Unlock()
}

// onClose 官方 OnClose@0x1005eb50：销毁连接（计数不动）、WriteLine。
func (r *Robot) onClose(id int, c *connect, nc net.Conn) {
	nc.Close()
	r.mu.Lock()
	delete(r.connects, id)
	r.mu.Unlock()
	r.trace("address %s closed", c.peer)
}

// sendToServerLocked 官方 SendToServer@0x1005cc20：payload ≤0x1FE、
// 转义后 ≤0x400，超限 trace 拒发。调用方持有 r.mu（写经连接自锁）。
func (r *Robot) sendToServerLocked(c *connect, payload []byte) bool {
	if len(payload) > sendMaxPayload {
		r.trace("(SoloRobot::SendToServer)message too long")
		return false
	}
	if c.nc == nil {
		r.trace("(SoloRobot::SendToServer)no connect info")
		return false
	}
	wire := escapeFrame(payload)
	if len(wire) > sendStuffedLimit {
		r.trace("(SoloRobot::SendToServer)message too long")
		return false
	}
	if err := c.writeFrame(payload); err != nil {
		return false
	}
	return true
}

// SendToServer 对外单连接发送（状态查询/测试用）。
func (r *Robot) SendToServer(index int, payload []byte) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.connects {
		if c.index == index {
			return r.sendToServerLocked(c, payload)
		}
	}
	r.trace("(SoloRobot::SendToServer)index error")
	return false
}

// 统计四数（官方属性面 ConnectCount/InGameCount/FailedCount/State）。
func (r *Robot) stats() (totalRaised, connectCount, inGameCount, failedCount int, module ModuleState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.connects {
		if c.state == ConnInGame {
			inGameCount++
		}
	}
	return r.totalRaised, r.connectCount, inGameCount, r.failedCount, r.module
}

// GetConnectState 官方 GetConnectState@0x1005c970：按机器人序号查
// 连接级状态名（越界/空槽返回 ""）。
func (r *Robot) GetConnectState(index int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.connects {
		if c.index == index {
			return c.state.String()
		}
	}
	return ""
}

// echoLine 官方 main.lua echo_input：quit/end/其余。
func (r *Robot) echoLine(line string) (string, bool) {
	switch line {
	case "quit":
		r.trace("server will shutdown...")
		r.trace("game robot shutdown...")
		if r.onQuit != nil {
			go r.onQuit()
		}
		return "server will shutdown...", false
	case "end":
		r.echo.Close()
		return "", false
	case "status":
		total, ok, inGame, failed, module := r.stats()
		return fmt.Sprintf("state=%s total=%d connect=%d ingame=%d failed=%d", module, total, ok, inGame, failed), false
	default:
		return "undefined command: " + line, false
	}
}
