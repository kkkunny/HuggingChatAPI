package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	stlslices "github.com/kkkunny/stl/container/slices"
	stlerr "github.com/kkkunny/stl/error"
	stlval "github.com/kkkunny/stl/value"
	"github.com/labstack/echo/v4"

	"github.com/sashabaranov/go-openai"

	"github.com/kkkunny/HuggingChatAPI/internal/api"
	"github.com/kkkunny/HuggingChatAPI/internal/config"
)

func ChatCompletions(reqCtx echo.Context) error {
	token := strings.TrimPrefix(reqCtx.Request().Header.Get("Authorization"), "Bearer ")
	cli, err := api.NewAPI(config.HuggingChatDomain, token)
	if err != nil {
		_ = config.Logger.Error(err)
		return echo.ErrUnauthorized
	}
	err = cli.RefreshCookie(reqCtx.Request().Context())
	if err != nil {
		_ = config.Logger.Error(err)
		return echo.ErrUnauthorized
	}

	var req openai.ChatCompletionRequest
	if err = stlerr.ErrorWrap(reqCtx.Bind(&req)); err != nil {
		_ = config.Logger.Error(err)
		return echo.ErrBadRequest
	}

	_, convs, err := cli.ListModelsAndConversations(reqCtx.Request().Context())
	if err != nil {
		return err
	}
	_ = config.Logger.Debug(string(stlval.IgnoreWith(json.Marshal(convs))))
	convs = stlslices.Filter(convs, func(_ int, conv *api.SimpleConversationInfo) bool {
		// _ = config.Logger.Debug(string(stlval.IgnoreWith(json.Marshal(conv))))
		return conv.Model == req.Model
	})
	var convInfo *api.ConversationInfoResponse
	if len(convs) == 0 {
		createResp, err := cli.CreateConversation(reqCtx.Request().Context(), &api.CreateConversationRequest{
			Model: req.Model,
		})
		if err != nil {
			return err
		}
		convInfo, err = cli.ConversationInfoAfterCreate(reqCtx.Request().Context(), &api.ConversationInfoAfterCreateRequest{
			ConversationID: createResp.ConversationID,
		})
		if err != nil {
			return err
		}
	} else {
		convID := stlslices.Random(convs).ID
		convInfo, err = cli.ConversationInfo(reqCtx.Request().Context(), &api.ConversationInfoRequest{ConversationID: convID})
		if err != nil {
			return err
		}
	}

	// defer func() {
	// 	go func() {
	// 		delErr := cli.DeleteConversation(reqCtx.Request().Context(), &api.DeleteConversationRequest{ConversationID: convInfo.ConversationID})
	// 		if delErr != nil {
	// 			_ = config.Logger.Error(err)
	// 		}
	// 	}()
	// }()

	msgStrList := make([]string, len(req.Messages)+1)
	msgStrList[0] = "Forget previous messages and focus on the current message!\n"
	for i, msg := range req.Messages {
		msgStrList[i+1] = fmt.Sprintf("%s: %s", msg.Role, msg.Content)
	}
	prompt := fmt.Sprintf("%s\nassistant: ", strings.Join(msgStrList, ""))

	msgID := stlslices.Last(convInfo.Messages).ID
	chatResp, err := cli.ChatConversation(reqCtx.Request().Context(), &api.ChatConversationRequest{
		ConversationID: convInfo.ConversationID,
		ID:             msgID,
		Inputs:         prompt,
	})
	if err != nil {
		return err
	}

	handler := stlval.Ternary(req.Stream, chatCompletionsWithStream, chatCompletionsNoStream)
	return handler(reqCtx, msgID, convInfo, chatResp)
}

func chatCompletionsNoStream(reqCtx echo.Context, msgID string, convInfo *api.ConversationInfoResponse, resp *api.ChatConversationResponse) error {
	var tokenCount uint64
	var contents []openai.ChatMessagePart
	for msg := range resp.Stream {
		switch msg.Type {
		case api.StreamMessageTypeError:
			return msg.Error
		case api.StreamMessageTypeFinalAnswer:
			if stlval.DerefPtrOr(msg.Text) != "" {
				contents = append(contents, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: *msg.Text,
				})
			}
			break
		case api.StreamMessageTypeStream:
			tokenCount++
		case api.StreamMessageTypeFile:
			if stlval.DerefPtrOr(msg.MIME) == "image/webp" && stlval.DerefPtrOr(msg.SHA) != "" {
				contents = append(contents, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						Detail: openai.ImageURLDetailAuto,
						URL:    fmt.Sprintf("%s/chat/conversation/%s/output/%s", config.HuggingChatDomain, convInfo.ConversationID, *msg.SHA),
					},
				})
			}
		case api.StreamMessageTypeStatus, api.StreamMessageTypeTool, api.StreamMessageTypeTitle:
		default:
			_ = config.Logger.Warnf("unknown stream msg type `%s`", msg.Type)
		}
	}

	var reply string
	if len(contents) == 1 && contents[0].Type == openai.ChatMessagePartTypeText {
		reply = contents[0].Text
		contents = nil
	}

	return stlerr.ErrorWrap(reqCtx.JSONPretty(http.StatusOK, &openai.ChatCompletionResponse{
		ID:      msgID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   convInfo.Model,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:         "assistant",
					Content:      reply,
					MultiContent: contents,
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.Usage{
			PromptTokens:     0,
			CompletionTokens: int(tokenCount),
			TotalTokens:      int(tokenCount),
		},
	}, "  "))
}

func chatCompletionsWithStream(reqCtx echo.Context, msgID string, convInfo *api.ConversationInfoResponse, resp *api.ChatConversationResponse) error {
	writer := reqCtx.Response()
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("Transfer-Encoding", "chunked")

	for {
		select {
		case <-reqCtx.Request().Context().Done():
			return stlerr.Errorf("SSE client disconnected")
		case msg, ok := <-resp.Stream:
			if !ok {
				return nil
			}
			switch msg.Type {
			case api.StreamMessageTypeError:
				return msg.Error
			case api.StreamMessageTypeFinalAnswer:
				data, err := stlerr.ErrorWith(json.Marshal(&openai.ChatCompletionStreamResponse{
					ID:      msgID,
					Object:  "chat.completion",
					Created: time.Now().Unix(),
					Model:   convInfo.Model,
					Choices: []openai.ChatCompletionStreamChoice{
						{
							Index:        0,
							FinishReason: "stop",
						},
					},
				}))
				if err != nil {
					return err
				}
				_, err = stlerr.ErrorWith(fmt.Fprint(writer, "data: "+string(data)+"\n\n"))
				if err != nil {
					return err
				}
				writer.Flush()
				_, err = stlerr.ErrorWith(fmt.Fprint(writer, "data: [DONE]\n\n"))
				if err != nil {
					return err
				}
				writer.Flush()
			case api.StreamMessageTypeStream:
				var reply string
				if msg.Token != nil {
					reply = *msg.Token
				}
				data, err := stlerr.ErrorWith(json.Marshal(&openai.ChatCompletionStreamResponse{
					ID:      msgID,
					Object:  "chat.completion",
					Created: time.Now().Unix(),
					Model:   convInfo.Model,
					Choices: []openai.ChatCompletionStreamChoice{
						{
							Index: 0,
							Delta: openai.ChatCompletionStreamChoiceDelta{
								Role:    "assistant",
								Content: strings.TrimRight(reply, "\u0000"),
							},
						},
					},
				}))
				if err != nil {
					return err
				}
				_, err = stlerr.ErrorWith(fmt.Fprint(writer, "data: "+string(data)+"\n"))
				if err != nil {
					return err
				}
				writer.Flush()
			case api.StreamMessageTypeStatus, api.StreamMessageTypeTool, api.StreamMessageTypeFile, api.StreamMessageTypeTitle:
			default:
				_ = config.Logger.Warnf("unknown stream msg type `%s`", msg.Type)
			}
		}
	}
}
