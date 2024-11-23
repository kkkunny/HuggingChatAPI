package main

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

	"github.com/kkkunny/HuggingChatAPI/config"
	"github.com/kkkunny/HuggingChatAPI/hugchat"
	"github.com/kkkunny/HuggingChatAPI/hugchat/dto"
)

func chatCompletions(reqCtx echo.Context) error {
	tokenProvider, err := parseAuthorization(strings.TrimPrefix(reqCtx.Request().Header.Get("Authorization"), "Bearer "))
	if err != nil {
		_ = config.Logger.Error(err)
		return echo.ErrUnauthorized
	}
	cli := hugchat.NewClient(tokenProvider)
	err = cli.CheckLogin(reqCtx.Request().Context())
	if err != nil {
		_ = config.Logger.Error(err)
		return echo.ErrUnauthorized
	}

	var req openai.ChatCompletionRequest
	if err = stlerr.ErrorWrap(reqCtx.Bind(&req)); err != nil {
		_ = config.Logger.Error(err)
		return echo.ErrBadRequest
	}

	convs, err := cli.ListConversations(reqCtx.Request().Context())
	if err != nil {
		return err
	}
	convs = stlslices.Filter(convs, func(_ int, conv *dto.SimpleConversationInfo) bool {
		return conv.Model == req.Model
	})
	var convInfo *dto.ConversationInfo
	if len(convs) == 0 {
		convInfo, err = cli.CreateConversation(reqCtx.Request().Context(), req.Model, "")
		if err != nil {
			return err
		}
	} else {
		convID := stlslices.Random(convs).ID
		convInfo, err = cli.ConversationInfo(reqCtx.Request().Context(), convID)
		if err != nil {
			return err
		}
	}

	msgStrList := make([]string, len(req.Messages)+1)
	msgStrList[0] = "Forget previous messages and focus on the current message!\n"
	for i, msg := range req.Messages {
		msgStrList[i+1] = fmt.Sprintf("%s: %s", msg.Role, msg.Content)
	}
	prompt := fmt.Sprintf("%s\nassistant: ", strings.Join(msgStrList, ""))

	msgID := stlslices.Last(convInfo.Messages).ID
	msgChan, err := cli.ChatConversation(reqCtx.Request().Context(), convInfo.ConversationID, &hugchat.ChatConversationParams{
		LastMsgID: msgID,
		Inputs:    prompt,
	})
	if err != nil {
		return err
	}

	handler := stlval.Ternary(req.Stream, chatCompletionsWithStream, chatCompletionsNoStream)
	return handler(reqCtx, msgID, convInfo, msgChan)
}

func chatCompletionsNoStream(reqCtx echo.Context, msgID string, convInfo *dto.ConversationInfo, msgChan chan *dto.StreamMessage) error {
	var tokenCount uint64
	var contents []openai.ChatMessagePart
	for msg := range msgChan {
		switch msg.Type {
		case dto.StreamMessageTypeError:
			return msg.Error
		case dto.StreamMessageTypeFinalAnswer:
			if stlval.DerefPtrOr(msg.Text) != "" {
				contents = append(contents, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: *msg.Text,
				})
			}
			break
		case dto.StreamMessageTypeStream:
			tokenCount++
		case dto.StreamMessageTypeFile:
			if stlval.DerefPtrOr(msg.MIME) == "image/webp" && stlval.DerefPtrOr(msg.SHA) != "" {
				contents = append(contents, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						Detail: openai.ImageURLDetailAuto,
						URL:    fmt.Sprintf("%s/chat/conversation/%s/output/%s", config.HuggingChatDomain, convInfo.ConversationID, *msg.SHA),
					},
				})
			}
		case dto.StreamMessageTypeStatus, dto.StreamMessageTypeTool, dto.StreamMessageTypeTitle:
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

func chatCompletionsWithStream(reqCtx echo.Context, msgID string, convInfo *dto.ConversationInfo, msgChan chan *dto.StreamMessage) error {
	writer := reqCtx.Response()
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("Transfer-Encoding", "chunked")

	for {
		select {
		case <-reqCtx.Request().Context().Done():
			return stlerr.Errorf("SSE client disconnected")
		case msg, ok := <-msgChan:
			if !ok {
				return nil
			}
			switch msg.Type {
			case dto.StreamMessageTypeError:
				return msg.Error
			case dto.StreamMessageTypeFinalAnswer:
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
			case dto.StreamMessageTypeStream:
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
			case dto.StreamMessageTypeStatus, dto.StreamMessageTypeTool, dto.StreamMessageTypeFile, dto.StreamMessageTypeTitle:
			default:
				_ = config.Logger.Warnf("unknown stream msg type `%s`", msg.Type)
			}
		}
	}
}
