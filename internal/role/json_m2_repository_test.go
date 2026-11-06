package role

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestJSONRepositoryImplementsM2DevelopmentContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roles.json")
	repository, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	accountA, err := repository.ProvisionLegacy(context.Background(), "account-a")
	if err != nil {
		t.Fatal(err)
	}
	accountB, err := repository.ProvisionLegacy(context.Background(), "account-b")
	if err != nil {
		t.Fatal(err)
	}
	draft := NewRole{
		Name:       "甲",
		Appearance: Appearance{FormatVersion: 1, Values: []string{"book5"}},
		Location:   Location{Scene: Scene{Config: "origin", Resource: "scene"}, Position: Position{}},
	}
	created, err := repository.Create(context.Background(), accountA.ID, draft)
	if err != nil {
		t.Fatal(err)
	}
	if created.Location.Position != (Position{}) {
		t.Fatalf("M2 JSON backend changed legal origin: %#v", created.Location.Position)
	}
	if _, err := repository.LoadOwned(context.Background(), accountB.ID, created.ID); !errors.Is(err, ErrNotOwned) {
		t.Fatalf("cross-account load error = %v", err)
	}

	newLocation := Location{Scene: Scene{Config: "next", Resource: "scene2"}, Position: Position{X: 4, Y: 5, Z: 6, Orient: 7}}
	version, err := repository.SaveLocation(context.Background(), created.ID, created.Location.Version, newLocation)
	if err != nil || version != created.Location.Version+1 {
		t.Fatalf("CAS save version=%d err=%v", version, err)
	}
	if current, err := repository.SaveLocation(context.Background(), created.ID, created.Location.Version, newLocation); !errors.Is(err, ErrVersionConflict) || current != version {
		t.Fatalf("stale CAS current=%d err=%v", current, err)
	}

	reopened, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	reloadedAccount, err := reopened.FindByKey(context.Background(), "account-a")
	if err != nil || reloadedAccount.ID != accountA.ID {
		t.Fatalf("stable account ID got=%#v err=%v", reloadedAccount, err)
	}
	reloaded, err := reopened.LoadOwned(context.Background(), accountA.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Location.Version != version || reloaded.Location.Scene != newLocation.Scene || reloaded.Location.Position != newLocation.Position {
		t.Fatalf("reloaded snapshot = %#v", reloaded)
	}
}

func TestJSONRepositoryM2NameAndSlotConflictsDoNotMutateWinner(t *testing.T) {
	repository, err := OpenJSONRepository(filepath.Join(t.TempDir(), "roles.json"))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := repository.ProvisionLegacy(context.Background(), "a")
	b, _ := repository.ProvisionLegacy(context.Background(), "b")
	draft := NewRole{Name: "same", Appearance: Appearance{FormatVersion: 1}, Location: Location{Scene: Scene{Config: "c", Resource: "r"}}}
	winner, err := repository.Create(context.Background(), a.ID, draft)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Create(context.Background(), b.ID, draft); !errors.Is(err, ErrRoleNameTaken) {
		t.Fatalf("name conflict = %v", err)
	}
	draft.Name = "different"
	if _, err := repository.Create(context.Background(), a.ID, draft); !errors.Is(err, ErrRoleSlotTaken) {
		t.Fatalf("slot conflict = %v", err)
	}
	loaded, err := repository.LoadOwned(context.Background(), a.ID, winner.ID)
	if err != nil || loaded.Name != "same" {
		t.Fatalf("winner changed: %#v err=%v", loaded, err)
	}
}
