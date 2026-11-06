package main

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

func TestFacultyProgressStorePersistsRoleSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "faculty.json")
	store, err := openFacultyProgressStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	progress := newPlayerProgress()
	value := progress.snapshot()
	value.LastAdvanceUTC = time.Date(2026, 7, 23, 12, 34, 0, 0, time.UTC)
	if err := store.Save(role.RoleID(7), value); err != nil {
		t.Fatal(err)
	}
	value.Books[0].Fill = 321
	if err := store.Save(role.RoleID(7), value); err != nil {
		t.Fatal(err)
	}

	reopened, err := openFacultyProgressStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	got, exists := reopened.Load(role.RoleID(7))
	if !exists {
		t.Fatal("saved faculty role missing")
	}
	if !reflect.DeepEqual(got, value) {
		t.Fatalf("faculty snapshot\n got: %#v\nwant: %#v", got, value)
	}
	if _, exists := reopened.Load(role.RoleID(8)); exists {
		t.Fatal("different role unexpectedly shared faculty state")
	}
}
