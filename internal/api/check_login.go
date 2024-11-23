package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/imroc/req/v3"
)

// CheckLogin 检查是否登录
func CheckLogin(ctx context.Context, cookies []*http.Cookie) (bool, error) {
	res, err := sendDefaultHttpRequest[string](ctx, http.MethodGet, func(r *req.Request) *req.Request {
		return r.SetHeader("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	}, cookies, "/chat/")
	if err != nil {
		return false, err
	}
	return !strings.Contains(*res, "action=\"/chat/login\""), nil
}
