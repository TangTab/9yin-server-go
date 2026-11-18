package proc

// Role 进程角色，对应原版 game_*.exe 家族的职责边界。
// solo 形态下所有角色同进程，仍以 Role 标记消息来源。
type Role string

const (
	RoleEntry    Role = "entry"    // 登录网关
	RoleLister   Role = "lister"   // 区服列表
	RoleMember   Role = "member"   // 角色服务
	RoleWorld    Role = "world"    // 场景服
	RolePublic   Role = "public"   // 公共数据
	RoleStore    Role = "store"    // 商城
	RoleAuction  Role = "auction"  // 拍卖行
	RoleLog      Role = "log"      // 日志
	RoleClient   Role = "client"   // 客户端连接会话（entry 侧）
)

// Envelope 跨角色消息信封。Payload 为 proto 层序列化产物，
// Bus 不解释内容；Topic 用于订阅路由（如 "world.scene.1"）。
type Envelope struct {
	From    Role
	Topic   string
	Payload []byte
}

// Bus 是进程拓扑的核心抽象：
//   - solo 形态：inproc 总线，进程内同步直连（本包 InprocBus）
//   - 分布式形态：TCP 实现替换，业务层零改动
//
// 语义约定：同 Topic 内投递顺序与发布顺序一致（同步实现天然满足）。
type Bus interface {
	// Publish 向 Topic 的全部订阅者（含 "*" 广播订阅者）投递一条消息。
	Publish(env Envelope) error
	// Subscribe 注册订阅；topic 为 "*" 表示接收全部消息（审计/日志用）。
	Subscribe(role Role, topic string, handler func(Envelope)) error
	// Close 关闭总线并释放资源；之后的 Publish 返回错误。
	Close()
}
