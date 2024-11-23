package dto

import (
	"time"

	"github.com/kkkunny/HuggingChatAPI/internal/api"
)

// Message 消息
type Message struct {
	ID       string
	From     string
	Content  string
	Children []string
	CreateAt time.Time
	UpdateAt time.Time
}

func NewMessageFromAPI(msg *api.Message) *Message {
	if msg == nil {
		return nil
	}
	return &Message{
		ID:       msg.ID,
		From:     msg.From,
		Content:  msg.Content,
		Children: msg.Children,
		CreateAt: msg.CreateAt,
		UpdateAt: msg.UpdateAt,
	}
}
