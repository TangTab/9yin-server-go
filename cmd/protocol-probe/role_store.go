package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

const defaultRoleFaction = "force_yihua"

type roleStore struct {
	accounts role.AccountRepository
	roles    role.RoleRepository
	// db 为 MySQL 仓库的连接池，供 faculty 等同库存储共用；
	// JSON 仓库（仅测试）下为 nil。
	db    *sql.DB
	close func() error
}

// openRoleStore 打开 MySQL 角色仓库（唯一生产后端）。data/roles.json
// 已废弃：DSN 缺失、连不上或缺必需表都直接失败，提示配置 env.server
// 或重放 migrations/ 权威存档（upgrade-db.bat）。
func openRoleStore() (*roleStore, error) {
	dsn := strings.TrimSpace(os.Getenv("NINEYIN_MYSQL_DSN"))
	if dsn == "" {
		return nil, errors.New("NINEYIN_MYSQL_DSN 未配置：角色存储已切换为 MySQL（data/roles.json 已废弃）。" +
			"请在环境变量或 env.server 中配置，例如 nineyin:nineyin123@tcp(127.0.0.1:3306)/nineyin")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	repository, err := role.OpenMySQLRepository(ctx, dsn, role.MySQLConfig{})
	if err != nil {
		return nil, err
	}
	db := repository.DB()
	if err := verifyRequiredTables(ctx, db); err != nil {
		_ = repository.Close()
		return nil, err
	}
	return &roleStore{accounts: repository, roles: repository, db: db, close: repository.Close}, nil
}

// requiredStoreTables 是游戏服运行所需的全部表。accounts/roles/
// role_appearances/role_locations 属 M2 角色仓库；role_faculty 为内功
// 进度快照（Java FacultyStore 与新版服 data-migrate 共用同表）。
var requiredStoreTables = []string{
	"accounts",
	"roles",
	"role_appearances",
	"role_locations",
	"role_faculty",
}

// verifyRequiredTables 只读检查必需表是否齐全；缺表直接报错退出，
// 建表权威在 migrations/ 权威存档重放（upgrade-db.bat），服务器
// 启动绝不执行 DDL。
func verifyRequiredTables(ctx context.Context, db *sql.DB) error {
	placeholders := make([]string, 0, len(requiredStoreTables))
	args := make([]any, 0, len(requiredStoreTables)+1)
	for _, name := range requiredStoreTables {
		placeholders = append(placeholders, "?")
		args = append(args, name)
	}
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN (" +
		strings.Join(placeholders, ", ") + ")"
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("role store: inspect schema: %w", err)
	}
	defer rows.Close()
	present := make(map[string]bool, len(requiredStoreTables))
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("role store: inspect schema row: %w", err)
		}
		present[strings.ToLower(name)] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("role store: inspect schema: %w", err)
	}
	var missing []string
	for _, name := range requiredStoreTables {
		if !present[strings.ToLower(name)] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("role store: 数据库缺必需表 %s：请先运行 upgrade-db.bat 重放 migrations/ 权威存档",
			strings.Join(missing, ", "))
	}
	return nil
}

func (store *roleStore) Close() error {
	if store == nil || store.close == nil {
		return nil
	}
	return store.close()
}

// errPasswordMismatch / errAccountNotRegistered 是登录拒绝的两个业务因：
// 前者关闭连接由 main.go 日志区分，后者仅在要求注册模式下出现。
var (
	errPasswordMismatch    = errors.New("role store: password mismatch")
	errAccountNotRegistered = errors.New("role store: account not registered")
)

// requireRegistration 报告「登录必须先注册」模式（fork 默认行为；本服默认
// 关闭以兼容存量自动建号）。NINEYIN_REQUIRE_REGISTRATION=1/true 开启。
func requireRegistration() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("NINEYIN_REQUIRE_REGISTRATION"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// login 完成账号定位与口令校验：
//   - 账号已注册（password_hash 非空）→ 密码槽密文必须与校验子逐字节相等
//     （客户端/注册端同链确定性加密，比对即校验，不解密）；
//   - 存量账号（无校验子）→ 免口令直接放行，首次注册后开始强制；
//   - 未知账号 → 默认 ProvisionLegacy 自动建号；requireRegistration 模式下
//     拒绝（fork 行为：必须先经 GM 页注册）。
func (store *roleStore) login(ctx context.Context, key role.AccountKey, passwordCiphertext []byte) (role.Account, *role.RoleSnapshot, error) {
	account, err := store.accounts.FindByKey(ctx, key)
	if errors.Is(err, role.ErrNotFound) {
		if requireRegistration() {
			return role.Account{}, nil, errAccountNotRegistered
		}
		account, err = store.accounts.ProvisionLegacy(ctx, key)
	}
	if err != nil {
		return role.Account{}, nil, err
	}
	if len(account.PasswordHash) > 0 && !bytes.Equal(passwordCiphertext, account.PasswordHash) {
		return role.Account{}, nil, errPasswordMismatch
	}
	summaries, err := store.roles.ListByAccount(ctx, account.ID)
	if err != nil {
		return role.Account{}, nil, err
	}
	if len(summaries) == 0 {
		return account, nil, nil
	}
	if len(summaries) != 1 || summaries[0].Slot != 0 {
		return role.Account{}, nil, fmt.Errorf("role store: unsupported role layout for account %d", account.ID)
	}
	snapshot, err := store.roles.LoadOwned(ctx, account.ID, summaries[0].ID)
	if err != nil {
		return role.Account{}, nil, err
	}
	return account, &snapshot, nil
}

func (store *roleStore) create(ctx context.Context, account role.Account, name string, appearance []string) (*role.RoleSnapshot, error) {
	appearance = defaultRoleAppearance(appearance)
	created, err := store.roles.Create(ctx, account.ID, role.NewRole{
		Slot: 0,
		Name: name,
		Appearance: role.Appearance{
			FormatVersion: 1,
			Values:        append([]string(nil), appearance...),
		},
		Location: defaultRoleLocation(),
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func defaultRoleAppearance(appearance []string) []string {
	values := append([]string(nil), appearance...)
	for len(values) <= 8 {
		values = append(values, "")
	}
	values[8] = defaultRoleFaction
	return values
}

func roleFaction(appearance []string) string {
	if len(appearance) > 8 && appearance[8] != "" {
		return appearance[8]
	}
	return defaultRoleFaction
}

func (store *roleStore) saveLocation(ctx context.Context, snapshot *role.RoleSnapshot, location role.Location) error {
	if snapshot == nil {
		return role.ErrNotFound
	}
	next, err := store.roles.SaveLocation(ctx, snapshot.ID, snapshot.Location.Version, location)
	if err != nil {
		return err
	}
	location.Version = next
	snapshot.Location = location
	return nil
}

func (store *roleStore) saveFaction(ctx context.Context, snapshot *role.RoleSnapshot, faction string) error {
	if snapshot == nil {
		return role.ErrNotFound
	}
	values := append([]string(nil), snapshot.Appearance.Values...)
	for len(values) <= 8 {
		values = append(values, "")
	}
	values[8] = faction
	appearance := role.Appearance{FormatVersion: snapshot.Appearance.FormatVersion, Values: values}
	if appearance.FormatVersion == 0 {
		appearance.FormatVersion = 1
	}
	next, err := store.roles.SaveAppearance(ctx, snapshot.ID, snapshot.Appearance.Version, appearance)
	if err != nil {
		return err
	}
	appearance.Version = next
	snapshot.Appearance = appearance
	return nil
}

func defaultRoleLocation() role.Location {
	return role.Location{
		Scene: role.Scene{Config: `ini\scene\school09_YiHuaGong`, Resource: "school09"},
		// res/share/relive/relive_info.ini 将此处声明为 force_yihua 的
		// 权威落地出生点（HomePointschool09A）。
		Position: role.Position{X: 340.160, Y: 47.979, Z: 657.284, Orient: 2.588},
		Version:  1,
	}
}
