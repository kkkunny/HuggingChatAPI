package api

import (
	"github.com/imroc/req/v3"

	"github.com/kkkunny/HuggingChatAPI/config"
)

var globalHttpClient *req.Client

func init() {
	globalHttpClient = req.C().
		SetProxy(config.Proxy).
		SetRedirectPolicy(req.NoRedirectPolicy()).
		SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0")
	if config.Debug {
		globalHttpClient = globalHttpClient.DevMode()
	}
}
