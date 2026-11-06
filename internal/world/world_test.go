package world

import (
	"errors"
	"math"
	"testing"
)

func testEntity(id EntityID, name string) Entity {
	properties := PropertyArchive{}
	_ = properties.Set("Name", WideString(name))
	return Entity{ID: id, OwnerID: 1, Kind: EntityNPC, Transform: Transform{X: float32(id)}, Properties: properties}
}

func TestSceneRejectsDuplicateWithoutPanicking(t *testing.T) {
	scene := NewScene()
	_, _ = scene.Begin(`ini\scene\school07_WuDang`, "school07")
	if err := scene.Add(testEntity(2, "first")); err != nil {
		t.Fatal(err)
	}
	if err := scene.Add(testEntity(2, "duplicate")); !errors.Is(err, ErrDuplicateEntity) {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestSnapshotDoesNotExposeMutableProperties(t *testing.T) {
	scene := NewScene()
	_, _ = scene.Begin("scene", "resource")
	_ = scene.Add(testEntity(2, "original"))
	snapshot := scene.Snapshot()
	snapshot.Entities[0].Entity.Properties["Name"] = WideString("changed")
	again := scene.Snapshot()
	if got := again.Entities[0].Entity.Properties["Name"].String; got != "original" {
		t.Fatalf("stored name = %q", got)
	}
}

func TestViewportDoesNotCommitFailedPlan(t *testing.T) {
	scene := NewScene()
	_, _ = scene.Begin("scene", "resource")
	_ = scene.Add(testEntity(2, "npc"))
	viewport := NewViewport()
	first := viewport.Plan(scene.Snapshot())
	if len(first.Adds) != 1 {
		t.Fatalf("adds = %d", len(first.Adds))
	}
	// Simulate a network failure by not committing first.
	retry := viewport.Plan(scene.Snapshot())
	if len(retry.Adds) != 1 {
		t.Fatalf("retry adds = %d, want 1", len(retry.Adds))
	}
	if err := viewport.Commit(retry); err != nil {
		t.Fatal(err)
	}
	if got := viewport.Plan(scene.Snapshot()); len(got.Adds)+len(got.Updates)+len(got.Removes) != 0 {
		t.Fatalf("clean delta = %#v", got)
	}
}

func TestGenerationChangeRemovesOldAndAddsNew(t *testing.T) {
	scene := NewScene()
	_, _ = scene.Begin("old", "old-resource")
	_ = scene.Add(testEntity(2, "old"))
	viewport := NewViewport()
	initial := viewport.Plan(scene.Snapshot())
	_ = viewport.Commit(initial)

	_, _ = scene.Begin("new", "new-resource")
	_ = scene.Add(testEntity(3, "new"))
	delta := viewport.Plan(scene.Snapshot())
	if len(delta.Removes) != 1 || delta.Removes[0].Entity.ID != 2 || len(delta.Adds) != 1 || delta.Adds[0].Entity.ID != 3 {
		t.Fatalf("generation delta = %#v", delta)
	}
}

func TestEntityReplacementProducesUpdate(t *testing.T) {
	scene := NewScene()
	_, _ = scene.Begin("scene", "resource")
	_ = scene.Add(testEntity(2, "before"))
	viewport := NewViewport()
	initial := viewport.Plan(scene.Snapshot())
	_ = viewport.Commit(initial)
	if err := scene.Replace(testEntity(2, "after")); err != nil {
		t.Fatal(err)
	}
	delta := viewport.Plan(scene.Snapshot())
	if len(delta.Updates) != 1 || delta.Updates[0].Entity.Properties["Name"].String != "after" {
		t.Fatalf("update delta = %#v", delta)
	}
}

func TestEntityRemovalCarriesOwnerIdentity(t *testing.T) {
	scene := NewScene()
	_, _ = scene.Begin("scene", "resource")
	entity := testEntity(2, "npc")
	entity.OwnerID = 77
	_ = scene.Add(entity)
	viewport := NewViewport()
	initial := viewport.Plan(scene.Snapshot())
	_ = viewport.Commit(initial)
	if err := scene.Remove(2); err != nil {
		t.Fatal(err)
	}
	delta := viewport.Plan(scene.Snapshot())
	if len(delta.Removes) != 1 || delta.Removes[0].Entity.ID != 2 || delta.Removes[0].Entity.OwnerID != 77 {
		t.Fatalf("remove delta = %#v", delta)
	}
}

func TestStaleDeltaCannotOverwriteNewerCommit(t *testing.T) {
	scene := NewScene()
	_, _ = scene.Begin("scene", "resource")
	_ = scene.Add(testEntity(2, "npc"))
	viewport := NewViewport()
	first := viewport.Plan(scene.Snapshot())
	second := viewport.Plan(scene.Snapshot())
	if err := viewport.Commit(first); err != nil {
		t.Fatal(err)
	}
	if err := viewport.Commit(second); !errors.Is(err, ErrStaleDelta) {
		t.Fatalf("stale commit error = %v", err)
	}
}

func TestPropertyAndTransformValidation(t *testing.T) {
	entity := testEntity(2, "npc")
	entity.Transform.X = float32(math.NaN())
	if err := entity.Validate(); err == nil {
		t.Fatal("non-finite transform accepted")
	}
}
