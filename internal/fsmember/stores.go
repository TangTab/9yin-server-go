package fsmember

import (
	"fmt"
	"sort"
	"sync"
)

// Chunk store 存档块状态（§10：块=场景分片存档单位）。
type Chunk struct {
	SceneID uint32
	ChunkID uint32
	Loaded  bool // PublishLoadChunk 后置位
	Saved   bool // PublishSaveChunk 应答后置位
}

// ChunkStore 复刻 store 面四型的本地状态（清单→加载→保存→删除）。
type ChunkStore struct {
	mu     sync.Mutex
	chunks map[uint32]map[uint32]*Chunk // sceneID → chunkID → 状态
}

// NewChunkStore 建库。
func NewChunkStore() *ChunkStore {
	return &ChunkStore{chunks: make(map[uint32]map[uint32]*Chunk)}
}

// SetInfo 复刻 PublishChunkInfo(0x9)：整场景块清单（覆盖旧清单）。
// 场景无块报 "no scene chunks"、场景 id 0 报 "scene id error"。
func (c *ChunkStore) SetInfo(sceneID uint32, chunkIDs []uint32) error {
	if sceneID == 0 {
		return fmt.Errorf("(GameMember::PublishChunkInfo)scene id error")
	}
	if len(chunkIDs) == 0 {
		return fmt.Errorf("(GameMember::PublishChunkInfo)no scene chunks: %d", sceneID)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	m := make(map[uint32]*Chunk, len(chunkIDs))
	for _, id := range chunkIDs {
		m[id] = &Chunk{SceneID: sceneID, ChunkID: id}
	}
	c.chunks[sceneID] = m
	return nil
}

// Load 复刻 PublishLoadChunk(0xA)：数量与清单不符报 "chunks amount
// error"，未登记报 "set chunk data failed"。
func (c *ChunkStore) Load(sceneID uint32, chunkIDs []uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.chunks[sceneID]
	if !ok || len(m) == 0 {
		return fmt.Errorf("(GameMember::PublishLoadChunk)no scene chunks: %d", sceneID)
	}
	if len(chunkIDs) != len(m) {
		return fmt.Errorf("(GameMember::PublishLoadChunk)chunks amount error: %d != %d", len(chunkIDs), len(m))
	}
	for _, id := range chunkIDs {
		ch, ok := m[id]
		if !ok {
			return fmt.Errorf("(GameMember::PublishLoadChunk)set chunk data failed: %d", id)
		}
		ch.Loaded = true
	}
	return nil
}

// Save 复刻 PublishSaveChunk(0xB)：保存应答落账。
func (c *ChunkStore) Save(sceneID, chunkID uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.chunks[sceneID]
	if !ok {
		return fmt.Errorf("(GameMember::PublishSaveChunk)failed: scene %d", sceneID)
	}
	ch, ok := m[chunkID]
	if !ok {
		return fmt.Errorf("(GameMember::PublishSaveChunk)failed: chunk %d", chunkID)
	}
	ch.Saved = true
	return nil
}

// Delete 复刻 PublishDeleteChunk(0xC)。
func (c *ChunkStore) Delete(sceneID, chunkID uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.chunks[sceneID]
	if !ok {
		return fmt.Errorf("(GameMember::PublishDeleteChunk)failed: scene %d", sceneID)
	}
	if _, ok := m[chunkID]; !ok {
		return fmt.Errorf("(GameMember::PublishDeleteChunk)failed: chunk %d", chunkID)
	}
	delete(m, chunkID)
	if len(m) == 0 {
		delete(c.chunks, sceneID)
	}
	return nil
}

// Snapshot 场景块清单（升序）。
func (c *ChunkStore) Snapshot(sceneID uint32) []Chunk {
	c.mu.Lock()
	defer c.mu.Unlock()
	m, ok := c.chunks[sceneID]
	if !ok {
		return nil
	}
	out := make([]Chunk, 0, len(m))
	for _, ch := range m {
		out = append(out, *ch)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChunkID < out[j].ChunkID })
	return out
}
