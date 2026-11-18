package login

import (
	"fmt"
	gonet "net"
	"sync"
	"sync/atomic"

	"github.com/lizhanyin/9yin-server-go/internal/net"
	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

// Gateway 是客户端线路网关：TCP accept → net.Connection（EE 转义 + XOR
// 初始键）→ 每连接一个会话循环，对应原版 GameEntry 的 UserSock 层职责。
//
// 会话分派当前只认引擎级心跳（0x18→0x38 整帧回显，golden 钉死）；
// 其余帧交给 OnFrame 钩子——login 主干（认证/选角/进场景）从这里接入。
type Gateway struct {
	// Addr 监听地址；":0" 由测试取随机端口。
	Addr string
	// NetOptions 线路装配（零值=EE+XOR(InitialKey) 默认客户端线路；
	// 填 IdentityCipher 可切 CJ 明文对拍基线）。
	NetOptions net.Options
	// OnFrame 心跳之外的分派钩子（nil=忽略）。不得阻塞过久——
	// 它在同一会话循环里串行执行。
	OnFrame func(*Session, []byte)
	// Logf 可选日志钩子（连接建立/断开/异常帧）。
	Logf func(format string, args ...any)

	mu       sync.Mutex
	listener gonet.Listener
	sessions map[uint64]*Session
	closed   bool

	nextID atomic.Uint64
	wg     sync.WaitGroup
}

// Session 单条客户端线路的会话态宿主：账号身份、角色选择、
// 键轮换结果都挂这里。
type Session struct {
	ID      uint64
	Conn    *net.Connection
	Account string // 登录后的账号身份键（acct:v1:…，空=未登录）
	Role    *Role  // 已建/已选角色（空=无角色）
}

// Listen 绑定端口并返回实际地址（solo 的 OnStart 调用一次；
// ":0" 由测试取随机端口）。
func (g *Gateway) Listen() (gonet.Addr, error) {
	ln, err := gonet.Listen("tcp", g.Addr)
	if err != nil {
		return nil, fmt.Errorf("login gateway listen %s: %w", g.Addr, err)
	}
	g.mu.Lock()
	g.listener = ln
	if g.sessions == nil {
		g.sessions = make(map[uint64]*Session)
	}
	g.mu.Unlock()
	return ln.Addr(), nil
}

// Serve 阻塞 accept 循环，直到 Close 或 listener 出错。
// 返回的 error 仅供诊断；正常 Close 路径返回 nil。
func (g *Gateway) Serve() error {
	for {
		g.mu.Lock()
		ln, closed := g.listener, g.closed
		g.mu.Unlock()
		if ln == nil || closed {
			return nil
		}
		stream, err := ln.Accept()
		if err != nil {
			g.mu.Lock()
			closed := g.closed
			g.mu.Unlock()
			if closed {
				return nil // Close 竞态下的 Accept 报错视为正常关停
			}
			return fmt.Errorf("login gateway accept: %w", err)
		}
		g.wg.Add(1)
		go g.serveConn(stream)
	}
}

// serveConn 为单条流装配 Connection 并进入会话循环。
func (g *Gateway) serveConn(stream gonet.Conn) {
	defer g.wg.Done()
	conn, err := net.NewConnection(stream, g.NetOptions)
	if err != nil {
		g.logf("gateway: 装配连接失败: %v", err)
		stream.Close()
		return
	}
	sess := &Session{ID: g.nextID.Add(1), Conn: conn}
	g.mu.Lock()
	if g.closed { // Close 与 accept 竞态：直接丢弃这条新会话
		g.mu.Unlock()
		conn.Close()
		return
	}
	g.sessions[sess.ID] = sess
	g.mu.Unlock()

	g.logf("gateway: 会话 %d 建立 %s", sess.ID, stream.RemoteAddr())
	g.sessionLoop(sess)
	g.mu.Lock()
	delete(g.sessions, sess.ID)
	g.mu.Unlock()
	conn.Close()
	g.logf("gateway: 会话 %d 关闭", sess.ID)
}

// sessionLoop 读帧分派：心跳直接回显，其余交给 OnFrame。
func (g *Gateway) sessionLoop(sess *Session) {
	for {
		frame, err := sess.Conn.ReadMsg()
		if err != nil {
			return // 连接关闭或线路错误，会话结束
		}
		if len(frame) == 0 {
			g.logf("gateway: 会话 %d 空帧忽略", sess.ID)
			continue
		}
		switch frame[0] {
		case proto.MsgPing:
			if err := sess.Conn.WriteMsg(proto.HeartbeatReply(frame)); err != nil {
				return
			}
		default:
			if g.OnFrame != nil {
				g.OnFrame(sess, frame)
			} else {
				g.logf("gateway: 会话 %d 未处理帧 op=%02X len=%d", sess.ID, frame[0], len(frame))
			}
		}
	}
}

// Close 关闭 listener 与全部会话，等待会话循环退出（幂等）。
// solo 的 OnStop 调用；也用于测试清理。
func (g *Gateway) Close() error {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return nil
	}
	g.closed = true
	var lnErr error
	if g.listener != nil {
		lnErr = g.listener.Close()
	}
	sessions := make([]*Session, 0, len(g.sessions))
	for _, s := range g.sessions {
		sessions = append(sessions, s)
	}
	g.mu.Unlock()

	for _, s := range sessions {
		s.Conn.Close() // 会话循环随即读到错误退出
	}
	g.wg.Wait()
	return lnErr
}

func (g *Gateway) logf(format string, args ...any) {
	if g.Logf != nil {
		g.Logf(format, args...)
	}
}
