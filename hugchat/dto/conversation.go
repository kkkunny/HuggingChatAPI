package dto

import (
	"time"

	stlslices "github.com/kkkunny/stl/container/slices"

	"github.com/kkkunny/HuggingChatAPI/internal/api"
)

// SimpleConversationInfo 会话简单信息
type SimpleConversationInfo struct {
	ID        string
	Model     string
	Title     string
	UpdatedAt time.Time
}

func NewSimpleConversationInfoFromAPI(conv *api.SimpleConversationInfo) *SimpleConversationInfo {
	if conv == nil {
		return nil
	}
	return &SimpleConversationInfo{
		ID:        conv.ID,
		Model:     conv.Model,
		Title:     conv.Title,
		UpdatedAt: conv.UpdatedAt,
	}
}

// ConversationInfo 会话详细信息
type ConversationInfo struct {
	ConversationID string
	Model          string
	Title          string
	PrePrompt      string
	Messages       []*Message
}

func NewConversationInfoFromAPI(conv *api.DetailConversationInfo) *ConversationInfo {
	if conv == nil {
		return nil
	}
	return &ConversationInfo{
		ConversationID: conv.ConversationID,
		Model:          conv.Model,
		Title:          conv.Title,
		PrePrompt:      conv.PrePrompt,
		Messages: stlslices.Map(conv.Messages, func(_ int, msg *api.Message) *Message {
			return NewMessageFromAPI(msg)
		}),
	}
}
