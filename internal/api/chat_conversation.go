package api

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"

	request "github.com/imroc/req/v3"
	"github.com/kkkunny/stl/container/tuple"
	stlerr "github.com/kkkunny/stl/error"

	"github.com/kkkunny/HuggingChatAPI/config"
)

type ChatConversationRequest struct {
	ConversationID string   `json:"-"`
	Files          []any    `json:"files,omitempty"`
	ID             string   `json:"id"`
	Inputs         string   `json:"inputs"`
	IsContinue     bool     `json:"is_continue"`
	IsRetry        bool     `json:"is_retry"`
	WebSearch      bool     `json:"web_search"`
	Tools          []string `json:"tools"`
}

// ChatConversation 对话
func ChatConversation(ctx context.Context, cookies []*http.Cookie, req *ChatConversationRequest) (chan tuple.Tuple2[string, error], error) {
	if len(req.Tools) == 0 {
		req.Tools = make([]string, 0)
	}
	reqBody, err := stlerr.ErrorWith(json.Marshal(req))
	if err != nil {
		return nil, err
	}
	resp, err := sendDefaultHttpRequest[request.Response](ctx, http.MethodPost, func(r *request.Request) *request.Request {
		return r.SetFormData(map[string]string{"data": string(reqBody)}).
			DisableAutoReadResponse()
	}, cookies, "/chat/conversation/%s", req.ConversationID)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(resp.Body)
	msgChan := make(chan tuple.Tuple2[string, error])

	go func() {
		defer func() {
			if err := recover(); err != nil {
				_ = config.Logger.Error(err)
			}
		}()

		defer func() {
			close(msgChan)
		}()

		for !resp.Close {
			msgChan <- tuple.Pack2(stlerr.ErrorWith(reader.ReadString('\n')))
		}
	}()

	return msgChan, nil
}
