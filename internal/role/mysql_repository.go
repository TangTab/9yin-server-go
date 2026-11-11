package role

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

const defaultMaxRoleSlots uint16 = 1

type MySQLConfig struct {
	MaxRoleSlots    uint16
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type MySQLRepository struct {
	db       *sql.DB
	maxSlots uint16
}

// ProductionDSN 应用仓库所需的连接不变量。
// 显式指定的非零超时会被保留；缺失的安全参数会得到
// 保守的默认值。
func ProductionDSN(raw string) (string, error) {
	config, err := mysql.ParseDSN(raw)
	if err != nil {
		return "", fmt.Errorf("%w: invalid MySQL DSN", ErrInvalidArgument)
	}
	if config.DBName == "" {
		return "", fmt.Errorf("%w: MySQL DSN has no database", ErrInvalidArgument)
	}
	if config.Params == nil {
		config.Params = make(map[string]string)
	}
	if _, exists := config.Params["charset"]; !exists {
		config.Params["charset"] = "utf8mb4"
	}
	config.ParseTime = true
	config.Loc = time.UTC
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Second
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 5 * time.Second
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 5 * time.Second
	}
	return config.FormatDSN(), nil
}

// OpenMySQLRepository 打开并校验一个生产 MySQL 仓库。
// 调用者拥有返回的仓库，并须在停机时将其 Close。
func OpenMySQLRepository(ctx context.Context, dsn string, config MySQLConfig) (*MySQLRepository, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("%w: empty MySQL DSN", ErrInvalidArgument)
	}
	normalizedDSN, err := ProductionDSN(dsn)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", normalizedDSN)
	if err != nil {
		return nil, storageError(err)
	}
	if config.MaxOpenConns <= 0 {
		config.MaxOpenConns = 16
	}
	if config.MaxIdleConns <= 0 {
		config.MaxIdleConns = 8
	}
	if config.ConnMaxLifetime <= 0 {
		config.ConnMaxLifetime = 5 * time.Minute
	}
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, storageError(err)
	}
	return NewMySQLRepository(db, config), nil
}

// NewMySQLRepository 绑定一个已打开的数据库。它主要用于
// 依赖注入和测试；它不会 ping，也不接管数据库的所有权。
func NewMySQLRepository(db *sql.DB, config MySQLConfig) *MySQLRepository {
	maxSlots := config.MaxRoleSlots
	if maxSlots == 0 {
		maxSlots = defaultMaxRoleSlots
	}
	return &MySQLRepository{db: db, maxSlots: maxSlots}
}

func (repository *MySQLRepository) Close() error { return repository.db.Close() }

// DB 暴露底层连接池，供 faculty 等同库存储共用（只读用途由调用方自律）。
func (repository *MySQLRepository) DB() *sql.DB { return repository.db }

func (repository *MySQLRepository) FindByKey(ctx context.Context, key AccountKey) (Account, error) {
	if err := ValidateAccountKey(key); err != nil {
		return Account{}, err
	}
	var account Account
	var stored []byte
	var passwordHash []byte
	err := repository.db.QueryRowContext(ctx, `
SELECT account_id, account_key, status, version, password_hash
FROM accounts WHERE account_key = ?`, []byte(key)).Scan(&account.ID, &stored, &account.Status, &account.Version, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, storageError(err)
	}
	account.Key = AccountKey(stored)
	account.PasswordHash = passwordHash
	if account.Status != AccountActive {
		return Account{}, ErrAccountDisabled
	}
	return account, nil
}

// RegisterAccount upsert 注册（fork /api/register）：不存在则建号落校验子，
// 已存在则覆写（改密语义）。accounts.password_hash 列在 0001 schema 即有，
// 无需迁移。
func (repository *MySQLRepository) RegisterAccount(ctx context.Context, key AccountKey, passwordHash []byte) (Account, error) {
	if err := ValidateAccountKey(key); err != nil {
		return Account{}, err
	}
	if len(passwordHash) == 0 || len(passwordHash) > 255 || len(passwordHash)%16 != 0 {
		return Account{}, fmt.Errorf("%w: invalid password verifier", ErrInvalidArgument)
	}
	if _, err := repository.db.ExecContext(ctx, `
INSERT INTO accounts(account_key, password_hash) VALUES (?, ?)
ON DUPLICATE KEY UPDATE password_hash = VALUES(password_hash)`, []byte(key), passwordHash); err != nil {
		return Account{}, storageError(err)
	}
	return repository.FindByKey(ctx, key)
}

func (repository *MySQLRepository) ProvisionLegacy(ctx context.Context, key AccountKey) (Account, error) {
	if err := ValidateAccountKey(key); err != nil {
		return Account{}, err
	}
	_, err := repository.db.ExecContext(ctx, `INSERT INTO accounts(account_key) VALUES (?)`, []byte(key))
	if err != nil && !isDuplicate(err) {
		return Account{}, storageError(err)
	}
	return repository.FindByKey(ctx, key)
}

func (repository *MySQLRepository) ListByAccount(ctx context.Context, accountID AccountID) ([]RoleSummary, error) {
	rows, err := repository.db.QueryContext(ctx, `
SELECT role_id, slot, name, delete_requested_at, version
FROM roles WHERE account_id = ? ORDER BY slot ASC`, accountID)
	if err != nil {
		return nil, storageError(err)
	}
	defer rows.Close()
	result := make([]RoleSummary, 0)
	for rows.Next() {
		var summary RoleSummary
		var deleted sql.NullTime
		if err := rows.Scan(&summary.ID, &summary.Slot, &summary.Name, &deleted, &summary.Version); err != nil {
			return nil, storageError(err)
		}
		summary.DeleteRequested = deleted.Valid
		result = append(result, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, storageError(err)
	}
	return result, nil
}

const snapshotColumns = `
SELECT r.role_id, r.account_id, r.slot, r.name, r.delete_requested_at, r.version,
       a.format_version, a.appearance, a.version,
       l.scene_config, l.scene_resource, l.pos_x, l.pos_y, l.pos_z, l.orient, l.version
FROM roles r
LEFT JOIN role_appearances a ON a.role_id = r.role_id
LEFT JOIN role_locations l ON l.role_id = r.role_id`

func (repository *MySQLRepository) LoadOwned(ctx context.Context, accountID AccountID, roleID RoleID) (RoleSnapshot, error) {
	row := repository.db.QueryRowContext(ctx, snapshotColumns+`
WHERE r.account_id = ? AND r.role_id = ?`, accountID, roleID)
	snapshot, err := scanSnapshot(row)
	if !errors.Is(err, sql.ErrNoRows) {
		return snapshot, classifyScan(err)
	}
	return RoleSnapshot{}, repository.classifyOwnership(ctx, accountID, roleID)
}

func (repository *MySQLRepository) LoadOwnedByName(ctx context.Context, accountID AccountID, name string) (RoleSnapshot, error) {
	row := repository.db.QueryRowContext(ctx, snapshotColumns+`
WHERE r.account_id = ? AND r.name = ?`, accountID, name)
	snapshot, err := scanSnapshot(row)
	if !errors.Is(err, sql.ErrNoRows) {
		return snapshot, classifyScan(err)
	}
	var owner AccountID
	err = repository.db.QueryRowContext(ctx, `SELECT account_id FROM roles WHERE name = ?`, name).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return RoleSnapshot{}, ErrNotFound
	}
	if err != nil {
		return RoleSnapshot{}, storageError(err)
	}
	return RoleSnapshot{}, ErrNotOwned
}

type rowScanner interface{ Scan(...any) error }

func scanSnapshot(row rowScanner) (RoleSnapshot, error) {
	var result RoleSnapshot
	var deleted sql.NullTime
	var format, appearanceVersion, locationVersion sql.NullInt64
	var appearance []byte
	var sceneConfig, sceneResource sql.NullString
	var x, y, z, orient sql.NullFloat64
	err := row.Scan(
		&result.ID, &result.AccountID, &result.Slot, &result.Name, &deleted, &result.RoleSummary.Version,
		&format, &appearance, &appearanceVersion,
		&sceneConfig, &sceneResource, &x, &y, &z, &orient, &locationVersion,
	)
	if err != nil {
		return RoleSnapshot{}, err
	}
	if !format.Valid || appearance == nil || !appearanceVersion.Valid ||
		!sceneConfig.Valid || !sceneResource.Valid || !x.Valid || !y.Valid || !z.Valid || !orient.Valid || !locationVersion.Valid {
		return RoleSnapshot{}, ErrCorruptSnapshot
	}
	var values []string
	if err := json.Unmarshal(appearance, &values); err != nil {
		return RoleSnapshot{}, ErrCorruptSnapshot
	}
	result.DeleteRequested = deleted.Valid
	result.Appearance = Appearance{FormatVersion: uint16(format.Int64), Values: values, Version: uint64(appearanceVersion.Int64)}
	result.Location = Location{
		Scene:    Scene{Config: sceneConfig.String, Resource: sceneResource.String},
		Position: Position{X: float32(x.Float64), Y: float32(y.Float64), Z: float32(z.Float64), Orient: float32(orient.Float64)},
		Version:  uint64(locationVersion.Int64),
	}
	if validateLocation(result.Location) != nil {
		return RoleSnapshot{}, ErrCorruptSnapshot
	}
	return result, nil
}

func classifyScan(err error) error {
	if err == nil || errors.Is(err, ErrCorruptSnapshot) {
		return err
	}
	return storageError(err)
}

func (repository *MySQLRepository) Create(ctx context.Context, accountID AccountID, draft NewRole) (RoleSnapshot, error) {
	if err := validateDraft(draft, repository.maxSlots); err != nil {
		return RoleSnapshot{}, err
	}
	if draft.Appearance.FormatVersion == 0 {
		draft.Appearance.FormatVersion = 1
	}
	values := draft.Appearance.Values
	if values == nil {
		values = []string{}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return RoleSnapshot{}, fmt.Errorf("%w: appearance JSON", ErrInvalidArgument)
	}
	tx, err := repository.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return RoleSnapshot{}, storageError(err)
	}
	defer tx.Rollback()

	var status AccountStatus
	err = tx.QueryRowContext(ctx, `SELECT status FROM accounts WHERE account_id = ? FOR UPDATE`, accountID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return RoleSnapshot{}, ErrNotFound
	}
	if err != nil {
		return RoleSnapshot{}, storageError(err)
	}
	if status != AccountActive {
		return RoleSnapshot{}, ErrAccountDisabled
	}

	created, err := tx.ExecContext(ctx, `INSERT INTO roles(account_id, slot, name) VALUES (?, ?, ?)`, accountID, draft.Slot, draft.Name)
	if err != nil {
		return RoleSnapshot{}, classifyCreateError(err)
	}
	id, err := created.LastInsertId()
	if err != nil {
		return RoleSnapshot{}, storageError(err)
	}
	roleID := RoleID(id)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO role_appearances(role_id, format_version, appearance) VALUES (?, ?, ?)`, roleID, draft.Appearance.FormatVersion, encoded); err != nil {
		return RoleSnapshot{}, storageError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO role_locations(role_id, scene_config, scene_resource, pos_x, pos_y, pos_z, orient)
VALUES (?, ?, ?, ?, ?, ?, ?)`, roleID, draft.Location.Scene.Config, draft.Location.Scene.Resource,
		draft.Location.Position.X, draft.Location.Position.Y, draft.Location.Position.Z, draft.Location.Position.Orient); err != nil {
		return RoleSnapshot{}, storageError(err)
	}
	if err := tx.Commit(); err != nil {
		return RoleSnapshot{}, storageError(err)
	}
	return RoleSnapshot{
		RoleSummary: RoleSummary{ID: roleID, Slot: draft.Slot, Name: draft.Name, Version: 1},
		AccountID:   accountID,
		Appearance:  Appearance{FormatVersion: draft.Appearance.FormatVersion, Values: append([]string(nil), values...), Version: 1},
		Location:    Location{Scene: draft.Location.Scene, Position: draft.Location.Position, Version: 1},
	}, nil
}

func (repository *MySQLRepository) RequestDelete(ctx context.Context, accountID AccountID, roleID RoleID) error {
	result, err := repository.db.ExecContext(ctx, `
UPDATE roles SET delete_requested_at = COALESCE(delete_requested_at, CURRENT_TIMESTAMP(6)), version = version + 1
WHERE account_id = ? AND role_id = ?`, accountID, roleID)
	if err != nil {
		return storageError(err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return storageError(err)
	}
	if count == 1 {
		return nil
	}
	return repository.classifyOwnership(ctx, accountID, roleID)
}

func (repository *MySQLRepository) SaveAppearance(ctx context.Context, roleID RoleID, expected uint64, value Appearance) (uint64, error) {
	if value.FormatVersion == 0 {
		return 0, fmt.Errorf("%w: zero appearance format", ErrInvalidArgument)
	}
	values := value.Values
	if values == nil {
		values = []string{}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return 0, fmt.Errorf("%w: appearance JSON", ErrInvalidArgument)
	}
	result, err := repository.db.ExecContext(ctx, `
UPDATE role_appearances
SET format_version = ?, appearance = ?, version = version + 1
WHERE role_id = ? AND version = ?`, value.FormatVersion, encoded, roleID, expected)
	if err != nil {
		return 0, storageError(err)
	}
	return repository.classifyCAS(ctx, "role_appearances", roleID, expected, result)
}

func (repository *MySQLRepository) SaveLocation(ctx context.Context, roleID RoleID, expected uint64, value Location) (uint64, error) {
	if err := validateLocation(value); err != nil {
		return 0, err
	}
	result, err := repository.db.ExecContext(ctx, `
UPDATE role_locations
SET scene_config = ?, scene_resource = ?, pos_x = ?, pos_y = ?, pos_z = ?, orient = ?, version = version + 1
WHERE role_id = ? AND version = ?`, value.Scene.Config, value.Scene.Resource,
		value.Position.X, value.Position.Y, value.Position.Z, value.Position.Orient, roleID, expected)
	if err != nil {
		return 0, storageError(err)
	}
	return repository.classifyCAS(ctx, "role_locations", roleID, expected, result)
}

func (repository *MySQLRepository) classifyCAS(ctx context.Context, table string, roleID RoleID, expected uint64, result sql.Result) (uint64, error) {
	count, err := result.RowsAffected()
	if err != nil {
		return 0, storageError(err)
	}
	if count == 1 {
		return expected + 1, nil
	}
	query := `SELECT version FROM ` + table + ` WHERE role_id = ?`
	var current uint64
	err = repository.db.QueryRowContext(ctx, query, roleID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, storageError(err)
	}
	return current, ErrVersionConflict
}

func (repository *MySQLRepository) classifyOwnership(ctx context.Context, accountID AccountID, roleID RoleID) error {
	var owner AccountID
	err := repository.db.QueryRowContext(ctx, `SELECT account_id FROM roles WHERE role_id = ?`, roleID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return storageError(err)
	}
	if owner != accountID {
		return ErrNotOwned
	}
	return ErrCorruptSnapshot
}

func classifyCreateError(err error) error {
	var mysqlError *mysql.MySQLError
	if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
		switch {
		case strings.Contains(mysqlError.Message, "uq_roles_name"):
			return ErrRoleNameTaken
		case strings.Contains(mysqlError.Message, "uq_roles_account_slot"):
			return ErrRoleSlotTaken
		}
	}
	return storageError(err)
}

func isDuplicate(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}

func storageError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrStorage, err)
}

var _ AccountRepository = (*MySQLRepository)(nil)
var _ RoleRepository = (*MySQLRepository)(nil)
