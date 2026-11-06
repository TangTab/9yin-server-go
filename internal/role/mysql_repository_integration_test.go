package role

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// This contract test is opt-in because it writes test rows. Point it only at a
// disposable, already-migrated database; never at the user's game database.
func TestMySQLRepositoryIntegrationContract(t *testing.T) {
	dsn := os.Getenv("NINEYIN_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NINEYIN_TEST_MYSQL_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repository, err := OpenMySQLRepository(ctx, dsn, MySQLConfig{MaxOpenConns: 8, MaxIdleConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	accounts := []AccountKey{AccountKey("it-a-" + suffix), AccountKey("it-b-" + suffix), AccountKey("it-c-" + suffix)}
	for _, key := range accounts {
		key := key
		t.Cleanup(func() { _, _ = repository.db.Exec(`DELETE FROM accounts WHERE account_key=?`, []byte(key)) })
	}
	a, err := repository.ProvisionLegacy(ctx, accounts[0])
	if err != nil {
		t.Fatal(err)
	}
	b, err := repository.ProvisionLegacy(ctx, accounts[1])
	if err != nil {
		t.Fatal(err)
	}
	c, err := repository.ProvisionLegacy(ctx, accounts[2])
	if err != nil {
		t.Fatal(err)
	}
	draft := NewRole{
		Name:       "integration-" + suffix,
		Appearance: Appearance{FormatVersion: 1, Values: []string{"book5"}},
		Location:   Location{Scene: Scene{Config: "origin", Resource: "scene"}, Position: Position{}},
	}

	type outcome struct {
		snapshot RoleSnapshot
		err      error
	}
	results := make(chan outcome, 2)
	var wait sync.WaitGroup
	for _, account := range []Account{a, b} {
		wait.Add(1)
		go func(account Account) {
			defer wait.Done()
			snapshot, err := repository.Create(ctx, account.ID, draft)
			results <- outcome{snapshot, err}
		}(account)
	}
	wait.Wait()
	close(results)
	var winner RoleSnapshot
	successes, nameConflicts := 0, 0
	for result := range results {
		switch {
		case result.err == nil:
			successes++
			winner = result.snapshot
		case errors.Is(result.err, ErrRoleNameTaken):
			nameConflicts++
		default:
			t.Fatalf("unexpected name race error: %v", result.err)
		}
	}
	if successes != 1 || nameConflicts != 1 {
		t.Fatalf("name race successes=%d conflicts=%d", successes, nameConflicts)
	}
	other := a.ID
	if winner.AccountID == a.ID {
		other = b.ID
	}
	if _, err := repository.LoadOwned(ctx, other, winner.ID); !errors.Is(err, ErrNotOwned) {
		t.Fatalf("cross-account load error = %v", err)
	}
	if winner.Location.Position != (Position{}) {
		t.Fatalf("legal origin changed: %#v", winner.Location.Position)
	}

	casDraft := draft
	casDraft.Name = "cas-" + suffix
	casRole, err := repository.Create(ctx, c.ID, casDraft)
	if err != nil {
		t.Fatal(err)
	}
	locations := []Location{
		{Scene: Scene{Config: "one", Resource: "scene"}, Position: Position{X: 1}},
		{Scene: Scene{Config: "two", Resource: "scene"}, Position: Position{X: 2}},
	}
	casResults := make(chan error, 2)
	for _, location := range locations {
		wait.Add(1)
		go func(location Location) {
			defer wait.Done()
			_, err := repository.SaveLocation(ctx, casRole.ID, casRole.Location.Version, location)
			casResults <- err
		}(location)
	}
	wait.Wait()
	close(casResults)
	casSuccesses, casConflicts := 0, 0
	for err := range casResults {
		if err == nil {
			casSuccesses++
		} else if errors.Is(err, ErrVersionConflict) {
			casConflicts++
		} else {
			t.Fatalf("unexpected CAS race error: %v", err)
		}
	}
	if casSuccesses != 1 || casConflicts != 1 {
		t.Fatalf("CAS race successes=%d conflicts=%d", casSuccesses, casConflicts)
	}
}
