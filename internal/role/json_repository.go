package role

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	jsonFormatVersion = 1
	// LegacyDefaultAccount is the account used when importing the former
	// process-wide, single-role roles.json document.
	LegacyDefaultAccount AccountKey = "local_default"
)

type roleFile struct {
	Version       int                     `json:"version"`
	Accounts      map[string]Role         `json:"accounts"`
	Identities    map[string]jsonIdentity `json:"identities,omitempty"`
	NextAccountID AccountID               `json:"next_account_id,omitempty"`
	NextRoleID    RoleID                  `json:"next_role_id,omitempty"`
}

type jsonIdentity struct {
	AccountID         AccountID     `json:"account_id"`
	RoleID            RoleID        `json:"role_id,omitempty"`
	Status            AccountStatus `json:"status"`
	AccountVersion    uint64        `json:"account_version"`
	Slot              uint16        `json:"slot,omitempty"`
	RoleVersion       uint64        `json:"role_version,omitempty"`
	AppearanceVersion uint64        `json:"appearance_version,omitempty"`
	LocationVersion   uint64        `json:"location_version,omitempty"`
	DeleteRequested   bool          `json:"delete_requested,omitempty"`
}

// legacyRole is the exact on-disk shape written by cmd/protocol-probe before
// roles became account-scoped. PosX through Orient intentionally use their old
// exported Go field names.
type legacyRole struct {
	Name          string   `json:"name"`
	Appearance    []string `json:"appearance"`
	SceneConfig   string   `json:"scene_config"`
	SceneResource string   `json:"scene_resource"`
	PosX          float32  `json:"PosX"`
	PosY          float32  `json:"PosY"`
	PosZ          float32  `json:"PosZ"`
	Orient        float32  `json:"Orient"`
}

// JSONRepository stores all account roles in one JSON document. Operations on
// the in-memory snapshot and its corresponding disk replacement are serialized
// so concurrent callers cannot overwrite a newer account snapshot.
type JSONRepository struct {
	mu            sync.RWMutex
	path          string
	accounts      map[string]Role
	identities    map[string]jsonIdentity
	nextAccountID AccountID
	nextRoleID    RoleID
}

// OpenJSONRepository opens path. A missing file represents an empty repository.
// Both the current account map format and the old single-role format are read.
func OpenJSONRepository(path string) (*JSONRepository, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("role: empty repository path")
	}

	repository := &JSONRepository{
		path:          filepath.Clean(path),
		accounts:      make(map[string]Role),
		identities:    make(map[string]jsonIdentity),
		nextAccountID: 1,
		nextRoleID:    1,
	}
	data, err := os.ReadFile(repository.path)
	if errors.Is(err, os.ErrNotExist) {
		return repository, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read role repository: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, errors.New("decode role repository: empty document")
	}
	if err := repository.decode(data); err != nil {
		return nil, err
	}
	return repository, nil
}

func (repository *JSONRepository) decode(data []byte) error {
	var discriminator struct {
		Accounts json.RawMessage `json:"accounts"`
	}
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return fmt.Errorf("decode role repository: %w", err)
	}

	if discriminator.Accounts != nil {
		var document roleFile
		if err := json.Unmarshal(data, &document); err != nil {
			return fmt.Errorf("decode role repository: %w", err)
		}
		if document.Version != jsonFormatVersion {
			return fmt.Errorf("decode role repository: unsupported version %d", document.Version)
		}
		for account, value := range document.Accounts {
			if strings.TrimSpace(account) == "" {
				return errors.New("decode role repository: empty account key")
			}
			repository.accounts[account] = normalize(value)
		}
		for account, identity := range document.Identities {
			repository.identities[account] = identity
		}
		repository.nextAccountID = document.NextAccountID
		repository.nextRoleID = document.NextRoleID
		if repository.nextAccountID == 0 {
			repository.nextAccountID = 1
		}
		if repository.nextRoleID == 0 {
			repository.nextRoleID = 1
		}
		repository.rebuildIdentitiesLocked()
		return nil
	}

	var old legacyRole
	if err := json.Unmarshal(data, &old); err != nil {
		return fmt.Errorf("decode legacy role repository: %w", err)
	}
	// The old store considered Name != "" to be the sole existence flag.
	if old.Name != "" {
		repository.accounts[string(LegacyDefaultAccount)] = normalize(Role{
			Name:       old.Name,
			Appearance: old.Appearance,
			Scene: Scene{
				Config:   old.SceneConfig,
				Resource: old.SceneResource,
			},
			Position: Position{X: old.PosX, Y: old.PosY, Z: old.PosZ, Orient: old.Orient},
		})
		repository.rebuildIdentitiesLocked()
	}
	return nil
}

func (repository *JSONRepository) Load(ctx context.Context, account AccountKey) (Role, bool, error) {
	if err := validate(ctx, account); err != nil {
		return Role{}, false, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	value, exists := repository.accounts[string(account)]
	return clone(value), exists, nil
}

func (repository *JSONRepository) Save(ctx context.Context, account AccountKey, value Role) error {
	if err := validate(ctx, account); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()

	key := string(account)
	previous, existed := repository.accounts[key]
	previousIdentity, identityExisted := repository.identities[key]
	repository.ensureIdentityLocked(key, true)
	identity := repository.identities[key]
	if existed {
		identity.RoleVersion++
		identity.AppearanceVersion++
		identity.LocationVersion++
	}
	repository.identities[key] = identity
	repository.accounts[key] = normalize(clone(value))
	if err := repository.persistLocked(); err != nil {
		if existed {
			repository.accounts[key] = previous
		} else {
			delete(repository.accounts, key)
		}
		if identityExisted {
			repository.identities[key] = previousIdentity
		} else {
			delete(repository.identities, key)
		}
		return err
	}
	return nil
}

func (repository *JSONRepository) UpdatePosition(ctx context.Context, account AccountKey, position Position) error {
	if err := validate(ctx, account); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()

	key := string(account)
	value, exists := repository.accounts[key]
	if !exists {
		return ErrRoleNotFound
	}
	previous := value.Position
	identity := repository.identities[key]
	previousVersion := identity.LocationVersion
	value.Position = position
	repository.accounts[key] = value
	identity.LocationVersion++
	repository.identities[key] = identity
	if err := repository.persistLocked(); err != nil {
		value.Position = previous
		repository.accounts[key] = value
		identity.LocationVersion = previousVersion
		repository.identities[key] = identity
		return err
	}
	return nil
}

func (repository *JSONRepository) persistLocked() error {
	document := roleFile{
		Version: jsonFormatVersion, Accounts: repository.accounts, Identities: repository.identities,
		NextAccountID: repository.nextAccountID, NextRoleID: repository.nextRoleID,
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode role repository: %w", err)
	}
	data = append(data, '\n')

	directory := filepath.Dir(repository.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create role repository directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".roles-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary role repository: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set temporary role repository permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary role repository: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary role repository: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary role repository: %w", err)
	}
	if err := os.Rename(temporaryPath, repository.path); err != nil {
		return fmt.Errorf("replace role repository: %w", err)
	}
	committed = true
	return nil
}

func (repository *JSONRepository) rebuildIdentitiesLocked() {
	keys := make([]string, 0, len(repository.accounts))
	for key := range repository.accounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		repository.ensureIdentityLocked(key, true)
	}
	for _, identity := range repository.identities {
		if identity.AccountID >= repository.nextAccountID {
			repository.nextAccountID = identity.AccountID + 1
		}
		if identity.RoleID >= repository.nextRoleID {
			repository.nextRoleID = identity.RoleID + 1
		}
	}
	if repository.nextAccountID == 0 {
		repository.nextAccountID = 1
	}
	if repository.nextRoleID == 0 {
		repository.nextRoleID = 1
	}
}

func (repository *JSONRepository) ensureIdentityLocked(key string, withRole bool) jsonIdentity {
	identity, exists := repository.identities[key]
	if !exists {
		identity = jsonIdentity{AccountID: repository.nextAccountID, Status: AccountActive, AccountVersion: 1}
		repository.nextAccountID++
	}
	if identity.AccountVersion == 0 {
		identity.AccountVersion = 1
	}
	if withRole && identity.RoleID == 0 {
		identity.RoleID = repository.nextRoleID
		repository.nextRoleID++
		identity.RoleVersion = 1
		identity.AppearanceVersion = 1
		identity.LocationVersion = 1
	}
	repository.identities[key] = identity
	return identity
}

func validate(ctx context.Context, account AccountKey) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(string(account)) == "" {
		return ErrInvalidAccountKey
	}
	return nil
}

func clone(value Role) Role {
	value.Appearance = append([]string(nil), value.Appearance...)
	return value
}

// normalize preserves the compatibility defaults of the former roleStore.
func normalize(value Role) Role {
	for strings.Contains(value.Scene.Config, `\\`) {
		value.Scene.Config = strings.ReplaceAll(value.Scene.Config, `\\`, `\`)
	}
	if value.Scene.Config == "" {
		value.Scene.Config = `ini\scene\school07_WuDang`
	}
	if value.Scene.Resource == "" || value.Scene.Resource == "scene07" {
		value.Scene.Resource = "school07"
	}
	if value.Position.X == 0 && value.Position.Y == 0 && value.Position.Z == 0 {
		value.Position = Position{X: 479.75, Y: 87.5, Z: 624.5, Orient: 3.12}
	}
	return value
}

var _ Repository = (*JSONRepository)(nil)
