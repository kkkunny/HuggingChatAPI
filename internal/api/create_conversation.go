package api

import (
	"context"
	"net/http"

	request "github.com/imroc/req/v3"
)

type CreateConversationRequest struct {
	Model     string `json:"model"`
	PrePrompt string `json:"preprompt"`
}

type CreateConversationResponse struct {
	ConversationID string `json:"conversationId"`
}

// CreateConversation 创建会话
func CreateConversation(ctx context.Context, cookies []*http.Cookie, req *CreateConversationRequest) (*CreateConversationResponse, error) {
	return sendDefaultHttpRequest[CreateConversationResponse](ctx, http.MethodPost, func(r *request.Request) *request.Request {
		return r.SetBodyJsonMarshal(req)
	}, cookies, "/chat/conversation")
}
