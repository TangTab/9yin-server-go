package role

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestJSONRepositoryMissingFileIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "roles.json")
	repository, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	_, exists, err := repository.Load(context.Background(), "account-a")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("missing repository unexpectedly contained a role")
	}
}

func TestJSONRepositoryLoadsLegacySingleRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roles.json")
	legacy := `{
  "name": "邢军",
  "appearance": ["book5", "cloth_b0001"],
  "scene_config": "ini\\\\scene\\\\school07_WuDang",
  "scene_resource": "scene07",
  "PosX": 12.5,
  "PosY": 3.25,
  "PosZ": -8,
  "Orient": 1.5
}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	repository, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	got, exists, err := repository.Load(context.Background(), LegacyDefaultAccount)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("legacy role was not imported")
	}
	want := Role{
		Name:       "邢军",
		Appearance: []string{"book5", "cloth_b0001"},
		Scene:      Scene{Config: `ini\scene\school07_WuDang`, Resource: "school07"},
		Position:   Position{X: 12.5, Y: 3.25, Z: -8, Orient: 1.5},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy role mismatch\n got: %#v\nwant: %#v", got, want)
	}
	if _, exists, err := repository.Load(context.Background(), "another"); err != nil || exists {
		t.Fatalf("legacy role leaked to another account: exists=%v err=%v", exists, err)
	}
}

func TestJSONRepositoryLegacyEmptyNameMeansNoRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roles.json")
	if err := os.WriteFile(path, []byte(`{"name":"","PosX":10}`), 0o600); err != nil {
		t.Fatal(err)
	}
	repository, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	_, exists, err := repository.Load(context.Background(), LegacyDefaultAccount)
	if err != nil || exists {
		t.Fatalf("empty legacy role: exists=%v err=%v", exists, err)
	}
}

func TestJSONRepositorySavesMultipleAccountsAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "roles.json")
	repository, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	first := testRole("甲", 10)
	second := testRole("乙", 20)
	if err := repository.Save(context.Background(), "account-a", first); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), "account-b", second); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	assertLoadedRole(t, reopened, "account-a", first)
	assertLoadedRole(t, reopened, "account-b", second)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Version  int                        `json:"version"`
		Accounts map[string]json.RawMessage `json:"accounts"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("invalid persisted JSON: %v\n%s", err, data)
	}
	if document.Version != 1 || len(document.Accounts) != 2 {
		t.Fatalf("unexpected current format: %#v", document)
	}
	if strings.Contains(string(data), `"scene_config"`) || strings.Contains(string(data), `"PosX"`) {
		t.Fatalf("save did not migrate to nested current format:\n%s", data)
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".roles-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("temporary files left after atomic replacement: %v", leftovers)
	}
}

func TestJSONRepositoryUpdatePositionIsAccountScopedAndPersistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roles.json")
	repository, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	first, second := testRole("甲", 10), testRole("乙", 20)
	if err := repository.Save(context.Background(), "a", first); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), "b", second); err != nil {
		t.Fatal(err)
	}
	position := Position{X: 101.5, Y: 202.5, Z: 303.5, Orient: 2.75}
	if err := repository.UpdatePosition(context.Background(), "a", position); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	first.Position = position
	assertLoadedRole(t, reopened, "a", first)
	assertLoadedRole(t, reopened, "b", second)
	if err := repository.UpdatePosition(context.Background(), "missing", position); !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("missing role error = %v, want %v", err, ErrRoleNotFound)
	}
}

func TestJSONRepositoryClonesAppearanceAcrossBoundary(t *testing.T) {
	repository, err := OpenJSONRepository(filepath.Join(t.TempDir(), "roles.json"))
	if err != nil {
		t.Fatal(err)
	}
	value := testRole("甲", 10)
	if err := repository.Save(context.Background(), "a", value); err != nil {
		t.Fatal(err)
	}
	value.Appearance[0] = "caller-mutated"
	loaded, _, err := repository.Load(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	loaded.Appearance[0] = "load-mutated"
	reloaded, _, err := repository.Load(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Appearance[0] != "book5" {
		t.Fatalf("repository leaked appearance backing array: %q", reloaded.Appearance[0])
	}
}

func TestJSONRepositoryValidatesAccountAndContext(t *testing.T) {
	repository, err := OpenJSONRepository(filepath.Join(t.TempDir(), "roles.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), " ", testRole("甲", 10)); !errors.Is(err, ErrInvalidAccountKey) {
		t.Fatalf("empty key error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := repository.Load(ctx, "a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled load error = %v", err)
	}
}

func TestJSONRepositoryConcurrentSavesDoNotLoseAccounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roles.json")
	repository, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	const count = 24
	var wait sync.WaitGroup
	errorsSeen := make(chan error, count)
	for index := 0; index < count; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			account := AccountKey("account-" + string(rune('a'+index)))
			if err := repository.Save(context.Background(), account, testRole(string(rune('甲'+index)), float32(index+1))); err != nil {
				errorsSeen <- err
			}
		}(index)
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Error(err)
	}
	reopened, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < count; index++ {
		account := AccountKey("account-" + string(rune('a'+index)))
		if _, exists, err := reopened.Load(context.Background(), account); err != nil || !exists {
			t.Fatalf("account %q lost: exists=%v err=%v", account, exists, err)
		}
	}
}

func TestJSONRepositoryRejectsCorruptAndUnknownVersion(t *testing.T) {
	for name, content := range map[string]string{
		"corrupt": `{`,
		"version": `{"version":2,"accounts":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "roles.json")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := OpenJSONRepository(path); err == nil {
				t.Fatal("expected invalid repository to be rejected")
			}
		})
	}
}

func testRole(name string, x float32) Role {
	return Role{
		Name:       name,
		Appearance: []string{"book5", "cloth_b0001"},
		Scene:      Scene{Config: `ini\scene\school07_WuDang`, Resource: "school07"},
		Position:   Position{X: x, Y: 87.5, Z: 624.5, Orient: 3.12},
	}
}

func assertLoadedRole(t *testing.T, repository Repository, account AccountKey, want Role) {
	t.Helper()
	got, exists, err := repository.Load(context.Background(), account)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("account %q did not exist", account)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("account %q mismatch\n got: %#v\nwant: %#v", account, got, want)
	}
}
