package role

import (
	"context"
	"fmt"
)

// 本文件中的方法使 JSONRepository 成为 M2 契约的完整开发后端。
// 其稳定 ID 与 CAS 版本号随兼容账号映射一起持久化；
// MySQL 仍是唯一的生产后端。

func (repository *JSONRepository) FindByKey(ctx context.Context, key AccountKey) (Account, error) {
	if err := ctx.Err(); err != nil {
		return Account{}, err
	}
	if err := ValidateAccountKey(key); err != nil {
		return Account{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	identity, exists := repository.identities[string(key)]
	if !exists {
		return Account{}, ErrNotFound
	}
	if identity.Status != AccountActive {
		return Account{}, ErrAccountDisabled
	}
	return accountFromJSON(key, identity), nil
}

// RegisterAccount upsert 注册（fork /api/register）：账号不存在则建号并落
// 口令校验子；已存在则覆写校验子（改密语义，fork 实证 updated:true）。
func (repository *JSONRepository) RegisterAccount(ctx context.Context, key AccountKey, passwordHash []byte) (Account, error) {
	if err := ctx.Err(); err != nil {
		return Account{}, err
	}
	if err := ValidateAccountKey(key); err != nil {
		return Account{}, err
	}
	if len(passwordHash) == 0 || len(passwordHash) > 255 || len(passwordHash)%16 != 0 {
		return Account{}, fmt.Errorf("%w: invalid password verifier", ErrInvalidArgument)
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	text := string(key)
	identity, exists := repository.identities[text]
	if exists && identity.Status != AccountActive {
		return Account{}, ErrAccountDisabled
	}
	if !exists {
		identity = repository.ensureIdentityLocked(text, false)
	}
	previous := identity.PasswordHash
	identity.PasswordHash = append([]byte(nil), passwordHash...)
	identity.AccountVersion++
	repository.identities[text] = identity
	if err := repository.persistLocked(); err != nil {
		identity.PasswordHash = previous
		identity.AccountVersion--
		repository.identities[text] = identity
		if !exists {
			delete(repository.identities, text)
		}
		return Account{}, err
	}
	return accountFromJSON(key, identity), nil
}

func (repository *JSONRepository) ProvisionLegacy(ctx context.Context, key AccountKey) (Account, error) {
	if err := ctx.Err(); err != nil {
		return Account{}, err
	}
	if err := ValidateAccountKey(key); err != nil {
		return Account{}, err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	text := string(key)
	if existing, exists := repository.identities[text]; exists {
		if existing.Status != AccountActive {
			return Account{}, ErrAccountDisabled
		}
		return accountFromJSON(key, existing), nil
	}
	identity := repository.ensureIdentityLocked(text, false)
	if err := repository.persistLocked(); err != nil {
		delete(repository.identities, text)
		return Account{}, err
	}
	return accountFromJSON(key, identity), nil
}

func accountFromJSON(key AccountKey, identity jsonIdentity) Account {
	return Account{
		ID:           identity.AccountID,
		Key:          key,
		Status:       identity.Status,
		Version:      identity.AccountVersion,
		PasswordHash: identity.PasswordHash,
	}
}

func (repository *JSONRepository) ListByAccount(ctx context.Context, accountID AccountID) ([]RoleSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	key, identity, exists := repository.findAccountLocked(accountID)
	if !exists {
		return []RoleSummary{}, nil
	}
	value, hasRole := repository.accounts[key]
	if !hasRole || identity.RoleID == 0 {
		return []RoleSummary{}, nil
	}
	return []RoleSummary{{
		ID: identity.RoleID, Slot: identity.Slot, Name: value.Name,
		DeleteRequested: identity.DeleteRequested, Version: identity.RoleVersion,
	}}, nil
}

func (repository *JSONRepository) LoadOwned(ctx context.Context, accountID AccountID, roleID RoleID) (RoleSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return RoleSnapshot{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	key, identity, exists := repository.findRoleLocked(roleID)
	if !exists {
		return RoleSnapshot{}, ErrNotFound
	}
	if identity.AccountID != accountID {
		return RoleSnapshot{}, ErrNotOwned
	}
	value, exists := repository.accounts[key]
	if !exists {
		return RoleSnapshot{}, ErrCorruptSnapshot
	}
	return snapshotFromJSON(value, identity), nil
}

func (repository *JSONRepository) LoadOwnedByName(ctx context.Context, accountID AccountID, name string) (RoleSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return RoleSnapshot{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	for key, value := range repository.accounts {
		if value.Name != name {
			continue
		}
		identity, exists := repository.identities[key]
		if !exists || identity.RoleID == 0 {
			return RoleSnapshot{}, ErrCorruptSnapshot
		}
		if identity.AccountID != accountID {
			return RoleSnapshot{}, ErrNotOwned
		}
		return snapshotFromJSON(value, identity), nil
	}
	return RoleSnapshot{}, ErrNotFound
}

func (repository *JSONRepository) Create(ctx context.Context, accountID AccountID, draft NewRole) (RoleSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return RoleSnapshot{}, err
	}
	if err := validateDraft(draft, defaultMaxRoleSlots); err != nil {
		return RoleSnapshot{}, err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key, identity, exists := repository.findAccountLocked(accountID)
	if !exists {
		return RoleSnapshot{}, ErrNotFound
	}
	if identity.Status != AccountActive {
		return RoleSnapshot{}, ErrAccountDisabled
	}
	if identity.RoleID != 0 || repository.accounts[key].Name != "" {
		return RoleSnapshot{}, ErrRoleSlotTaken
	}
	for _, value := range repository.accounts {
		if value.Name == draft.Name {
			return RoleSnapshot{}, ErrRoleNameTaken
		}
	}
	previousIdentity := identity
	identity = repository.ensureIdentityLocked(key, true)
	identity.Slot = draft.Slot
	repository.identities[key] = identity
	value := Role{Name: draft.Name, Appearance: append([]string(nil), draft.Appearance.Values...), Scene: draft.Location.Scene, Position: draft.Location.Position}
	repository.accounts[key] = value
	if err := repository.persistLocked(); err != nil {
		delete(repository.accounts, key)
		repository.identities[key] = previousIdentity
		return RoleSnapshot{}, err
	}
	return snapshotFromJSON(value, identity), nil
}

func (repository *JSONRepository) RequestDelete(ctx context.Context, accountID AccountID, roleID RoleID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key, identity, exists := repository.findRoleLocked(roleID)
	if !exists {
		return ErrNotFound
	}
	if identity.AccountID != accountID {
		return ErrNotOwned
	}
	previous := identity
	identity.DeleteRequested = true
	identity.RoleVersion++
	repository.identities[key] = identity
	if err := repository.persistLocked(); err != nil {
		repository.identities[key] = previous
		return err
	}
	return nil
}

func (repository *JSONRepository) SaveAppearance(ctx context.Context, roleID RoleID, expected uint64, value Appearance) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if value.FormatVersion == 0 {
		return 0, fmt.Errorf("%w: zero appearance format", ErrInvalidArgument)
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key, identity, exists := repository.findRoleLocked(roleID)
	if !exists {
		return 0, ErrNotFound
	}
	if identity.AppearanceVersion != expected {
		return identity.AppearanceVersion, ErrVersionConflict
	}
	roleValue, exists := repository.accounts[key]
	if !exists {
		return 0, ErrCorruptSnapshot
	}
	previous := append([]string(nil), roleValue.Appearance...)
	roleValue.Appearance = append([]string(nil), value.Values...)
	repository.accounts[key] = roleValue
	identity.AppearanceVersion++
	repository.identities[key] = identity
	if err := repository.persistLocked(); err != nil {
		roleValue.Appearance = previous
		repository.accounts[key] = roleValue
		identity.AppearanceVersion--
		repository.identities[key] = identity
		return 0, err
	}
	return identity.AppearanceVersion, nil
}

func (repository *JSONRepository) SaveLocation(ctx context.Context, roleID RoleID, expected uint64, value Location) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := validateLocation(value); err != nil {
		return 0, err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key, identity, exists := repository.findRoleLocked(roleID)
	if !exists {
		return 0, ErrNotFound
	}
	if identity.LocationVersion != expected {
		return identity.LocationVersion, ErrVersionConflict
	}
	roleValue, exists := repository.accounts[key]
	if !exists {
		return 0, ErrCorruptSnapshot
	}
	previousScene, previousPosition := roleValue.Scene, roleValue.Position
	roleValue.Scene, roleValue.Position = value.Scene, value.Position
	repository.accounts[key] = roleValue
	identity.LocationVersion++
	repository.identities[key] = identity
	if err := repository.persistLocked(); err != nil {
		roleValue.Scene, roleValue.Position = previousScene, previousPosition
		repository.accounts[key] = roleValue
		identity.LocationVersion--
		repository.identities[key] = identity
		return 0, err
	}
	return identity.LocationVersion, nil
}

func (repository *JSONRepository) findAccountLocked(id AccountID) (string, jsonIdentity, bool) {
	for key, identity := range repository.identities {
		if identity.AccountID == id {
			return key, identity, true
		}
	}
	return "", jsonIdentity{}, false
}

func (repository *JSONRepository) findRoleLocked(id RoleID) (string, jsonIdentity, bool) {
	if id == 0 {
		return "", jsonIdentity{}, false
	}
	for key, identity := range repository.identities {
		if identity.RoleID == id {
			return key, identity, true
		}
	}
	return "", jsonIdentity{}, false
}

func snapshotFromJSON(value Role, identity jsonIdentity) RoleSnapshot {
	return RoleSnapshot{
		RoleSummary: RoleSummary{ID: identity.RoleID, Slot: identity.Slot, Name: value.Name, DeleteRequested: identity.DeleteRequested, Version: identity.RoleVersion},
		AccountID:   identity.AccountID,
		Appearance:  Appearance{FormatVersion: 1, Values: append([]string(nil), value.Appearance...), Version: identity.AppearanceVersion},
		Location:    Location{Scene: value.Scene, Position: value.Position, Version: identity.LocationVersion},
	}
}

var _ AccountRepository = (*JSONRepository)(nil)
var _ RoleRepository = (*JSONRepository)(nil)
