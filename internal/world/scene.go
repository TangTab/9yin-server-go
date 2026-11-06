package world

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Generation uint64
type Revision uint64

var (
	ErrDuplicateEntity = errors.New("world: duplicate entity")
	ErrEntityNotFound  = errors.New("world: entity not found")
)

type VersionedEntity struct {
	Entity   Entity
	Revision Revision
}

func (entry VersionedEntity) clone() VersionedEntity {
	entry.Entity = entry.Entity.Clone()
	return entry
}

type Snapshot struct {
	Generation Generation
	Config     string
	Resource   string
	Entities   []VersionedEntity
}

type Scene struct {
	mu         sync.RWMutex
	generation Generation
	revision   Revision
	config     string
	resource   string
	entities   map[EntityID]VersionedEntity
}

func NewScene() *Scene {
	return &Scene{entities: make(map[EntityID]VersionedEntity)}
}

// Begin starts a new scene generation. Old entities cannot leak into the new
// generation even if a client sends delayed activity from the previous scene.
func (scene *Scene) Begin(config, resource string) (Generation, error) {
	config, resource = strings.TrimSpace(config), strings.TrimSpace(resource)
	if config == "" || resource == "" {
		return 0, errors.New("world: scene config and resource are required")
	}
	scene.mu.Lock()
	defer scene.mu.Unlock()
	scene.generation++
	if scene.generation == 0 {
		scene.generation++
	}
	scene.config, scene.resource = config, resource
	clear(scene.entities)
	return scene.generation, nil
}

func (scene *Scene) Add(entity Entity) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	scene.mu.Lock()
	defer scene.mu.Unlock()
	if scene.generation == 0 {
		return errors.New("world: scene generation has not begun")
	}
	if _, exists := scene.entities[entity.ID]; exists {
		return fmt.Errorf("%w: %d", ErrDuplicateEntity, entity.ID)
	}
	scene.revision++
	scene.entities[entity.ID] = VersionedEntity{Entity: entity.Clone(), Revision: scene.revision}
	return nil
}

func (scene *Scene) Replace(entity Entity) error {
	if err := entity.Validate(); err != nil {
		return err
	}
	scene.mu.Lock()
	defer scene.mu.Unlock()
	if _, exists := scene.entities[entity.ID]; !exists {
		return fmt.Errorf("%w: %d", ErrEntityNotFound, entity.ID)
	}
	scene.revision++
	scene.entities[entity.ID] = VersionedEntity{Entity: entity.Clone(), Revision: scene.revision}
	return nil
}

// Get returns the authoritative entity snapshot for a targeted action or a
// server-side simulation tick. Callers receive a clone and cannot mutate scene
// state without going through Replace.
func (scene *Scene) Get(id EntityID) (VersionedEntity, error) {
	scene.mu.RLock()
	defer scene.mu.RUnlock()
	entry, exists := scene.entities[id]
	if !exists {
		return VersionedEntity{}, fmt.Errorf("%w: %d", ErrEntityNotFound, id)
	}
	return entry.clone(), nil
}

func (scene *Scene) Remove(id EntityID) error {
	scene.mu.Lock()
	defer scene.mu.Unlock()
	if _, exists := scene.entities[id]; !exists {
		return fmt.Errorf("%w: %d", ErrEntityNotFound, id)
	}
	delete(scene.entities, id)
	return nil
}

func (scene *Scene) Snapshot() Snapshot {
	scene.mu.RLock()
	defer scene.mu.RUnlock()
	ids := make([]int, 0, len(scene.entities))
	for id := range scene.entities {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	result := Snapshot{Generation: scene.generation, Config: scene.config, Resource: scene.resource, Entities: make([]VersionedEntity, 0, len(ids))}
	for _, id := range ids {
		result.Entities = append(result.Entities, scene.entities[EntityID(id)].clone())
	}
	return result
}
