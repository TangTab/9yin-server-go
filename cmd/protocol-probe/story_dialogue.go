package main

import (
	"fmt"
	"sort"
	"strings"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// These IDs are registered by the current FxGameLogic GameMessageHandler.
// They are distinct from the 4..8 face-to-face menu route: 43 is a short
// speaker bubble and 383 opens the client-owned drama/chapter prompt.
const (
	customNPCTalk    int32 = 43
	customBeginDrama int32 = 383
)

// dramaPrompt is deliberately data-only. Story definitions can later live in
// Creator/NpcFunc-derived data or a persisted quest table without changing the
// protocol encoder below.
type dramaPrompt struct {
	Chapter      int32
	ChapterName  string
	Explore      int32
	Challenge    int32
	Brief        string
	Detail       string
	Title        string
	PictureCount int32
}

func (p dramaPrompt) validate() error {
	if p.Chapter < 1 {
		return fmt.Errorf("drama chapter must be positive")
	}
	if p.Explore < 0 || p.Explore > 5 || p.Challenge < 0 || p.Challenge > 5 {
		return fmt.Errorf("drama ratings must be in [0,5]")
	}
	if p.PictureCount < 0 || p.PictureCount > 5 {
		return fmt.Errorf("drama picture count must be in [0,5]")
	}
	if strings.TrimSpace(p.ChapterName) == "" || strings.TrimSpace(p.Brief) == "" {
		return fmt.Errorf("drama chapter name and brief text IDs are required")
	}
	return nil
}

// encodeNPCBubble maps to custom_handler.on_npc_talk. Mode zero intentionally
// suppresses its secondary systems/chat routes while retaining the five-second
// HeadGame bubble. The scene object must be an existing "object-owner" key.
func encodeNPCBubble(objectID, ownerID uint32, textID string) ([]byte, error) {
	if objectID == 0 || strings.TrimSpace(textID) == "" {
		return nil, fmt.Errorf("NPC bubble requires object and text ID")
	}
	return serverCustomIntMessage(customNPCTalk,
		customString(sceneIdent(objectID, ownerID)),
		customString(textID),
		customInt(0), // stage
		customInt(0), // mode: bubble only
	)
}

// encodeDramaPrompt maps to custom_handler.on_begin_drama, then
// form_task/form_drama_tips.show_begin_drama_info. Empty Detail selects the
// no-art card layout; non-empty Detail additionally needs the matching client
// drama image assets named by Title.
func encodeDramaPrompt(prompt dramaPrompt) ([]byte, error) {
	if err := prompt.validate(); err != nil {
		return nil, err
	}
	return serverCustomIntMessage(customBeginDrama,
		customInt(prompt.Chapter),
		customString(prompt.ChapterName),
		customInt(prompt.Explore),
		customInt(prompt.Challenge),
		customString(prompt.Brief),
		customString(prompt.Detail),
		customString(prompt.Title),
		customInt(prompt.PictureCount),
	)
}

func (s *sceneLifecycle) sendNPCBubble(objectID, ownerID uint32, textID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entity, exists := s.entities[worldcore.EntityID(objectID)]
	if !exists || entity.ownerID != ownerID {
		return fmt.Errorf("NPC bubble target %s is not in this scene", sceneIdent(objectID, ownerID))
	}
	frame, err := encodeNPCBubble(objectID, ownerID, textID)
	if err != nil {
		return err
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("send NPC bubble for %s: %w", sceneIdent(objectID, ownerID), err)
	}
	return nil
}

// sendGMNPCBubble targets the first stable scene NPC. It is a GM-only
// protocol probe, not an NPC selection rule used by gameplay.
func (s *sceneLifecycle) sendGMNPCBubble(textID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]int, 0, len(s.entities))
	for id := range s.entities {
		ids = append(ids, int(id))
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no scene NPC is available for bubble test")
	}
	sort.Ints(ids)
	entity := s.entities[worldcore.EntityID(ids[0])]
	frame, err := encodeNPCBubble(entity.id, entity.ownerID, textID)
	if err != nil {
		return "", err
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return "", fmt.Errorf("send NPC bubble for %s: %w", sceneIdent(entity.id, entity.ownerID), err)
	}
	return sceneIdent(entity.id, entity.ownerID), nil
}

func (s *sceneLifecycle) sendDramaPrompt(prompt dramaPrompt) error {
	frame, err := encodeDramaPrompt(prompt)
	if err != nil {
		return err
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("send drama prompt: %w", err)
	}
	return nil
}

// gmDramaProbe deliberately uses installed UI text IDs only. It validates the
// protocol/form route without pretending to be authored quest content.
func gmDramaProbe() dramaPrompt {
	return dramaPrompt{
		Chapter: 1, ChapterName: "ui_shop", Explore: 1, Challenge: 1,
		Brief: "ui_shop", Detail: "", Title: "", PictureCount: 0,
	}
}
