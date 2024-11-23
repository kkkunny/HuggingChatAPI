package hugchat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	stlslices "github.com/kkkunny/stl/container/slices"
	stlerr "github.com/kkkunny/stl/error"

	"github.com/kkkunny/HuggingChatAPI/config"
	"github.com/kkkunny/HuggingChatAPI/hugchat/dto"
	"github.com/kkkunny/HuggingChatAPI/internal/api"
)

type Client struct {
	tokenProvider TokenProvider
}

func NewClient(tokenProvider TokenProvider) *Client {
	return &Client{
		tokenProvider: tokenProvider,
	}
}

// CheckLogin 检查并刷新登录
func (c *Client) CheckLogin(ctx context.Context) error {
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return err
	}
	login, err := api.CheckLogin(ctx, token)
	if err != nil {
		return err
	}
	if login {
		return nil
	}

	token, err = c.tokenProvider.RefreshToken(ctx)
	if err != nil {
		return err
	}
	login, err = api.CheckLogin(ctx, token)
	if err != nil {
		return err
	} else if !login {
		return stlerr.Errorf("not login")
	}
	return nil
}

// ListModels 列出模型
func (c *Client) ListModels(ctx context.Context) ([]*dto.ModelInfo, error) {
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, err
	}
	models, _, err := api.ListModelsAndConversations(ctx, token)
	if err != nil {
		return nil, err
	}
	return stlslices.Map(models, func(_ int, model *api.ModelInfo) *dto.ModelInfo {
		return dto.NewModelInfoFromAPI(model)
	}), nil
}

// ListConversations 列出会话
func (c *Client) ListConversations(ctx context.Context) ([]*dto.SimpleConversationInfo, error) {
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, err
	}
	_, convs, err := api.ListModelsAndConversations(ctx, token)
	if err != nil {
		return nil, err
	}
	return stlslices.Map(convs, func(_ int, conv *api.SimpleConversationInfo) *dto.SimpleConversationInfo {
		return dto.NewSimpleConversationInfoFromAPI(conv)
	}), nil
}

// ConversationInfo 获取会话信息
func (c *Client) ConversationInfo(ctx context.Context, convID string) (*dto.ConversationInfo, error) {
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, err
	}
	conv, err := api.ConversationInfo(ctx, token, convID)
	return dto.NewConversationInfoFromAPI(conv), err
}

// CreateConversation 创建会话
func (c *Client) CreateConversation(ctx context.Context, model string, systemPrompt string) (*dto.ConversationInfo, error) {
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, err
	}
	createResp, err := api.CreateConversation(ctx, token, &api.CreateConversationRequest{
		Model:     model,
		PrePrompt: systemPrompt,
	})
	if err != nil {
		return nil, err
	}

	info, err := api.ConversationInfoAfterCreate(ctx, token, createResp.ConversationID)
	return dto.NewConversationInfoFromAPI(info), err
}

// DeleteConversation 删除会话
func (c *Client) DeleteConversation(ctx context.Context, convID string) error {
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return err
	}
	return api.DeleteConversation(ctx, token, convID)
}

type ChatConversationParams struct {
	LastMsgID string
	Inputs    string
	WebSearch bool
	Tools     []string
}

func (c *Client) ChatConversation(ctx context.Context, convID string, params *ChatConversationParams) (chan *dto.StreamMessage, error) {
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, err
	}
	msgDataChan, err := api.ChatConversation(ctx, token, &api.ChatConversationRequest{
		ConversationID: convID,
		ID:             params.LastMsgID,
		Inputs:         params.Inputs,
		WebSearch:      params.WebSearch,
		Tools:          params.Tools,
	})
	if err != nil {
		return nil, err
	}

	msgChan := make(chan *dto.StreamMessage)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				_ = config.Logger.Error(err)
			}
		}()

		defer func() {
			close(msgChan)
		}()

		for msgData := range msgDataChan {
			data, err := msgData.Unpack()
			if err != nil && errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				msgChan <- &dto.StreamMessage{Type: dto.StreamMessageTypeError, Error: err}
				break
			}
			data = strings.TrimSpace(data)
			if data == "" {
				continue
			}

			var msg dto.StreamMessage
			err = stlerr.ErrorWrap(json.Unmarshal([]byte(data), &msg))
			if err != nil {
				msgChan <- &dto.StreamMessage{Type: dto.StreamMessageTypeError, Error: err}
				break
			}

			msgChan <- &msg
		}
	}()
	return msgChan, nil
}
