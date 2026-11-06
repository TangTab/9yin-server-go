// Package role contains the account-scoped role model and its persistence
// contract. It deliberately has no dependency on protocol payloads or the
// executable that hosts the server.
package role

import (
	"context"
	"errors"
)

// AccountKey is the stable persistence key of an account. It is intentionally
// separate from a role's display name.
type AccountKey string

// Scene identifies the client scene configuration and resource bundle.
type Scene struct {
	Config   string `json:"config"`
	Resource string `json:"resource"`
}

// Position is a role's last persistent position in a scene.
type Position struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Z      float32 `json:"z"`
	Orient float32 `json:"orient"`
}

// Role is the account's selected role. Appearance remains an ordered list
// because its entries are positional in the modern client's role archive.
type Role struct {
	Name       string   `json:"name"`
	Appearance []string `json:"appearance,omitempty"`
	Scene      Scene    `json:"scene"`
	Position   Position `json:"position"`
}

var (
	// ErrInvalidAccountKey is returned before any persistence operation when
	// an empty account key is supplied.
	ErrInvalidAccountKey = errors.New("role: empty account key")
	// ErrRoleNotFound is returned when updating a role that does not exist.
	ErrRoleNotFound = errors.New("role: account has no role")
)

// Repository persists roles independently for each account.
type Repository interface {
	Load(ctx context.Context, account AccountKey) (Role, bool, error)
	Save(ctx context.Context, account AccountKey, value Role) error
	UpdatePosition(ctx context.Context, account AccountKey, position Position) error
}
