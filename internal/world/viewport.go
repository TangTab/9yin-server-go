package world

import (
	"errors"
	"sort"
	"sync"
)

var ErrStaleDelta = errors.New("world: stale viewport delta")

type Delta struct {
	PreviousGeneration Generation
	Generation         Generation
	Adds               []VersionedEntity
	Updates            []VersionedEntity
	Removes            []VersionedEntity

	baseEpoch uint64
	desired   map[EntityID]VersionedEntity
}

type Viewport struct {
	mu           sync.Mutex
	generation   Generation
	epoch        uint64
	materialized map[EntityID]VersionedEntity
}

func NewViewport() *Viewport {
	return &Viewport{materialized: make(map[EntityID]VersionedEntity)}
}

// Plan computes a delta without committing it. Callers commit only after all
// corresponding network writes succeed; a failed send leaves the old snapshot
// intact so the next reconcile can retry.
func (viewport *Viewport) Plan(snapshot Snapshot) Delta {
	viewport.mu.Lock()
	defer viewport.mu.Unlock()
	delta := Delta{
		PreviousGeneration: viewport.generation,
		Generation:         snapshot.Generation,
		baseEpoch:          viewport.epoch,
		desired:            make(map[EntityID]VersionedEntity, len(snapshot.Entities)),
	}
	for _, entry := range snapshot.Entities {
		delta.desired[entry.Entity.ID] = entry.clone()
	}

	if snapshot.Generation != viewport.generation {
		for _, entry := range viewport.materialized {
			delta.Removes = append(delta.Removes, entry.clone())
		}
		for _, entry := range snapshot.Entities {
			delta.Adds = append(delta.Adds, entry.clone())
		}
	} else {
		for _, entry := range snapshot.Entities {
			old, exists := viewport.materialized[entry.Entity.ID]
			if !exists {
				delta.Adds = append(delta.Adds, entry.clone())
			} else if old.Revision != entry.Revision {
				delta.Updates = append(delta.Updates, entry.clone())
			}
		}
		for id, entry := range viewport.materialized {
			if _, exists := delta.desired[id]; !exists {
				delta.Removes = append(delta.Removes, entry.clone())
			}
		}
	}
	sort.Slice(delta.Removes, func(i, j int) bool { return delta.Removes[i].Entity.ID < delta.Removes[j].Entity.ID })
	return delta
}

func (viewport *Viewport) Commit(delta Delta) error {
	viewport.mu.Lock()
	defer viewport.mu.Unlock()
	if delta.baseEpoch != viewport.epoch || delta.PreviousGeneration != viewport.generation {
		return ErrStaleDelta
	}
	viewport.generation = delta.Generation
	viewport.materialized = make(map[EntityID]VersionedEntity, len(delta.desired))
	for id, entry := range delta.desired {
		viewport.materialized[id] = entry.clone()
	}
	viewport.epoch++
	return nil
}

func (viewport *Viewport) Generation() Generation {
	viewport.mu.Lock()
	defer viewport.mu.Unlock()
	return viewport.generation
}
