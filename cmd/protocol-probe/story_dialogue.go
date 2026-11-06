package main

import (
	"fmt"
	"sort"
	"strings"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// 这些 ID 由当前 FxGameLogic 的 GameMessageHandler 注册。
// 它们与 4..8 的面对面菜单路线不同：43 是短暂的
// 说话者气泡，383 打开客户端自带的剧情/章节提示。
const (
	customNPCTalk    int32 = 43
	customBeginDrama int32 = 383
)

// dramaPrompt 刻意只包含数据。剧情定义之后可以放在
// 由 Creator/NpcFunc 派生的数据或持久化的任务表中，
// 而无需改动下面的协议编码器。
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

// encodeNPCBubble 对应 custom_handler.on_npc_talk。模式零刻意
// 抑制其次要的系统/聊天路线，同时保留五秒的
// HeadGame 气泡。场景对象必须是已存在的 "object-owner" 键。
func encodeNPCBubble(objectID, ownerID uint32, textID string) ([]byte, error) {
	if objectID == 0 || strings.TrimSpace(textID) == "" {
		return nil, fmt.Errorf("NPC bubble requires object and text ID")
	}
	return serverCustomIntMessage(customNPCTalk,
		customString(sceneIdent(objectID, ownerID)),
		customString(textID),
		customInt(0), // stage
		customInt(0), // mode：仅气泡
	)
}

// encodeDramaPrompt 对应 custom_handler.on_begin_drama，再到
// form_task/form_drama_tips.show_begin_drama_info。空的 Detail 会选择
// 无插图的卡片布局；非空 Detail 还需要 Title 所指的
// 相应客户端剧情图片资源。
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

// sendGMNPCBubble 以第一个稳定的场景 NPC 为目标。它是仅供 GM 使用的
// 协议探针，不是游戏玩法使用的 NPC 选择规则。
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

// gmDramaProbe 刻意只使用已安装的 UI 文本 ID。它验证
// 协议/表单路线，而不会伪装成正式编写的任务内容。
func gmDramaProbe() dramaPrompt {
	return dramaPrompt{
		Chapter: 1, ChapterName: "ui_shop", Explore: 1, Challenge: 1,
		Brief: "ui_shop", Detail: "", Title: "", PictureCount: 0,
	}
}
