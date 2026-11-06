package role

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

type AccountID uint64
type RoleID uint64

type AccountStatus uint8

const (
	AccountDisabled AccountStatus = iota
	AccountActive
	AccountLocked
	MaxRoleNameRunes = 64
)

type Account struct {
	ID      AccountID
	Key     AccountKey
	Status  AccountStatus
	Version uint64
}

type Appearance struct {
	FormatVersion uint16
	Values        []string
	Version       uint64
}

type Location struct {
	Scene    Scene
	Position Position
	Version  uint64
}

type RoleSummary struct {
	ID              RoleID
	Slot            uint16
	Name            string
	DeleteRequested bool
	Version         uint64
}

type RoleSnapshot struct {
	RoleSummary
	AccountID  AccountID
	Appearance Appearance
	Location   Location
}

type NewRole struct {
	Slot       uint16
	Name       string
	Appearance Appearance
	Location   Location
}

var (
	ErrNotFound        = errors.New("role: not found")
	ErrNotOwned        = errors.New("role: not owned by account")
	ErrAccountDisabled = errors.New("role: account disabled")
	ErrRoleNameTaken   = errors.New("role: name already taken")
	ErrRoleSlotTaken   = errors.New("role: account slot already taken")
	ErrVersionConflict = errors.New("role: version conflict")
	ErrCorruptSnapshot = errors.New("role: corrupt snapshot")
	ErrInvalidArgument = errors.New("role: invalid argument")
	ErrStorage         = errors.New("role: storage unavailable")
)

type AccountRepository interface {
	FindByKey(context.Context, AccountKey) (Account, error)
	ProvisionLegacy(context.Context, AccountKey) (Account, error)
}

type RoleRepository interface {
	ListByAccount(context.Context, AccountID) ([]RoleSummary, error)
	LoadOwned(context.Context, AccountID, RoleID) (RoleSnapshot, error)
	LoadOwnedByName(context.Context, AccountID, string) (RoleSnapshot, error)
	Create(context.Context, AccountID, NewRole) (RoleSnapshot, error)
	RequestDelete(context.Context, AccountID, RoleID) error
	SaveAppearance(context.Context, RoleID, uint64, Appearance) (uint64, error)
	SaveLocation(context.Context, RoleID, uint64, Location) (uint64, error)
}

func ValidateAccountKey(key AccountKey) error {
	s := string(key)
	if s == "" || len(s) > 128 || !utf8.ValidString(s) {
		return ErrInvalidAccountKey
	}
	for _, r := range s {
		if r == 0 || r < 0x20 || r == 0x7f {
			return ErrInvalidAccountKey
		}
	}
	return nil
}

func validateDraft(draft NewRole, maxSlots uint16) error {
	if draft.Slot >= maxSlots || strings.TrimSpace(draft.Name) == "" || utf8.RuneCountInString(draft.Name) > MaxRoleNameRunes || !utf8.ValidString(draft.Name) || hasControl(draft.Name) {
		return fmt.Errorf("%w: invalid role slot or name", ErrInvalidArgument)
	}
	if draft.Appearance.FormatVersion == 0 {
		draft.Appearance.FormatVersion = 1
	}
	return validateLocation(draft.Location)
}

func validateLocation(location Location) error {
	if location.Scene.Config == "" || location.Scene.Resource == "" || !utf8.ValidString(location.Scene.Config) || !utf8.ValidString(location.Scene.Resource) ||
		utf8.RuneCountInString(location.Scene.Config) > 255 || utf8.RuneCountInString(location.Scene.Resource) > 64 || hasControl(location.Scene.Config) || hasControl(location.Scene.Resource) {
		return fmt.Errorf("%w: empty scene", ErrInvalidArgument)
	}
	values := [...]float32{location.Position.X, location.Position.Y, location.Position.Z, location.Position.Orient}
	for _, value := range values {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return fmt.Errorf("%w: non-finite location", ErrInvalidArgument)
		}
	}
	return nil
}

func hasControl(value string) bool {
	for _, r := range value {
		if r == 0 || r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}
