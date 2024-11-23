package api

import (
	"context"
	"net/http"

	request "github.com/imroc/req/v3"
)

// DeleteConversation 删除会话
func DeleteConversation(ctx context.Context, cookies []*http.Cookie, convID string) error {
	_, err := sendDefaultHttpRequest[request.Response](ctx, http.MethodDelete, nil, cookies, "/chat/conversation/%s", convID)
	return err
}
