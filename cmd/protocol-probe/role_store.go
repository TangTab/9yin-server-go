package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

const defaultRoleFaction = "force_yihua"

type roleStore struct {
	accounts role.AccountRepository
	roles    role.RoleRepository
	close    func() error
}

func openRoleStore() (*roleStore, error) {
	if dsn := os.Getenv("NINEYIN_MYSQL_DSN"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		repository, err := role.OpenMySQLRepository(ctx, dsn, role.MySQLConfig{})
		if err != nil {
			return nil, err
		}
		return &roleStore{accounts: repository, roles: repository, close: repository.Close}, nil
	}
	path := runtimeProjectPath("data", "roles.json")
	repository, err := role.OpenJSONRepository(path)
	if err != nil {
		return nil, err
	}
	return &roleStore{accounts: repository, roles: repository, close: func() error { return nil }}, nil
}

func (store *roleStore) Close() error {
	if store == nil || store.close == nil {
		return nil
	}
	return store.close()
}

func (store *roleStore) login(ctx context.Context, key role.AccountKey) (role.Account, *role.RoleSnapshot, error) {
	account, err := store.accounts.FindByKey(ctx, key)
	if errors.Is(err, role.ErrNotFound) {
		account, err = store.accounts.ProvisionLegacy(ctx, key)
	}
	if err != nil {
		return role.Account{}, nil, err
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
		// res/share/relive/relive_info.ini declares this as force_yihua's
		// authoritative land spawn (HomePointschool09A).
		Position: role.Position{X: 340.160, Y: 47.979, Z: 657.284, Orient: 2.588},
		Version:  1,
	}
}
