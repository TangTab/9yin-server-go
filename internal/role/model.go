// Package role 包含账号作用域的角色模型及其持久化
// 契约。它刻意不依赖协议负载，也不依赖承载服务器的
// 可执行程序。
package role

import (
	"context"
	"errors"
)

// AccountKey 是账号稳定的持久化键。它有意与角色的
// 显示名称分开。
type AccountKey string

// Scene 标识客户端的场景配置和资源包。
type Scene struct {
	Config   string `json:"config"`
	Resource string `json:"resource"`
}

// Position 是角色在场景中最后保存的位置。
type Position struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Z      float32 `json:"z"`
	Orient float32 `json:"orient"`
}

// Role 是账号当前选中的角色。Appearance 仍保持有序列表，
// 因为其条目在现代客户端的角色档案中按位置存储。
type Role struct {
	Name       string   `json:"name"`
	Appearance []string `json:"appearance,omitempty"`
	Scene      Scene    `json:"scene"`
	Position   Position `json:"position"`
}

var (
	// ErrInvalidAccountKey 在提供空账号键时于任何持久化操作
	// 之前返回。
	ErrInvalidAccountKey = errors.New("role: empty account key")
	// ErrRoleNotFound 在更新不存在的角色时返回。
	ErrRoleNotFound = errors.New("role: account has no role")
)

// Repository 为每个账号独立持久化角色。
type Repository interface {
	Load(ctx context.Context, account AccountKey) (Role, bool, error)
	Save(ctx context.Context, account AccountKey, value Role) error
	UpdatePosition(ctx context.Context, account AccountKey, position Position) error
}
