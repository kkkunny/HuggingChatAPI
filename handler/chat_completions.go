package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	stlslices "github.com/kkkunny/stl/container/slices"
	stlval "github.com/kkkunny/stl/value"

	"github.com/satori/go.uuid"

	"github.com/sashabaranov/go-openai"

	"github.com/kkkunny/HuggingChatAPI/internal/api"
	"github.com/kkkunny/HuggingChatAPI/internal/config"
)

func ChatCompletions(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	cli, err := api.NewAPI(config.HuggingChatDomain, token)
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	err = cli.RefreshCookie(r.Context())
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var req openai.ChatCompletionRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &req)
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// createConvResp, err := cli.CreateConversation(r.Context(), &api.CreateConversationRequest{
	// 	Model: req.Model,
	// })
	// if err != nil {
	// 	config.Logger.Error(err)
	// 	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	// 	return
	// }
	defer func() {
		// go func() {
		// 	delErr := cli.DeleteConversation(r.Context(), &api.DeleteConversationRequest{ConversationID: createConvResp.ConversationID})
		// 	if delErr != nil {
		// 		config.Logger.Error(err)
		// 	}
		// }()
	}()

	convs, err := cli.ListConversations(r.Context())
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	convs = stlslices.Filter(convs, func(_ int, conv *api.SimpleConversationInfo) bool {
		return conv.Model == req.Model
	})
	if len(convs) == 0 {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	convID := stlslices.Random(convs).ID

	convInfo, err := cli.ConversationInfo(r.Context(), &api.ConversationInfoRequest{ConversationID: convID})
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	msgStrList := make([]string, len(req.Messages)+1)
	msgStrList[0] = "Forget previous messages and focus on the current message!\n"
	for i, msg := range req.Messages {
		msgStrList[i+1] = fmt.Sprintf("%s: %s", msg.Role, msg.Content)
	}
	prompt := fmt.Sprintf("%s\nassistant: ", strings.Join(msgStrList, ""))

	msgID := stlval.Ternary(stlslices.Last(convInfo.Messages).From != "system", stlslices.Last(convInfo.Messages).ID, uuid.NewV4().String())
	chatResp, err := cli.ChatConversation(r.Context(), &api.ChatConversationRequest{
		ConversationID: convID,
		ID:             msgID,
		Inputs:         prompt,
	})
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	handler := stlval.Ternary(req.Stream, chatCompletionsWithStream, chatCompletionsNoStream)
	err = handler(w, msgID, convInfo, chatResp)
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func chatCompletionsNoStream(w http.ResponseWriter, msgID string, convInfo *api.ConversationInfoResponse, resp *api.ChatConversationResponse) error {
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
		case api.StreamMessageTypeStatus, api.StreamMessageTypeTool:
		default:
			config.Logger.Warnf("unknown stream msg type `%s`", msg.Type)
		}
	}

	var reply string
	if len(contents) == 1 && contents[0].Type == openai.ChatMessagePartTypeText {
		reply = contents[0].Text
		contents = nil
	}
	data, err := json.Marshal(&openai.ChatCompletionResponse{
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
	})
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = fmt.Fprint(w, string(data))
	return err
}

func chatCompletionsWithStream(w http.ResponseWriter, msgID string, convInfo *api.ConversationInfoResponse, resp *api.ChatConversationResponse) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	flusher := w.(http.Flusher)

	for msg := range resp.Stream {
		switch msg.Type {
		case api.StreamMessageTypeError:
			return msg.Error
		case api.StreamMessageTypeFinalAnswer:
			data, err := json.Marshal(&openai.ChatCompletionStreamResponse{
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
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(w, "data: "+string(data)+"\n\n")
			if err != nil {
				return err
			}
			flusher.Flush()
			_, err = fmt.Fprint(w, "data: [DONE]\n\n")
			if err != nil {
				return err
			}
			flusher.Flush()
		case api.StreamMessageTypeStream:
			var reply string
			if msg.Token != nil {
				reply = *msg.Token
			}
			data, err := json.Marshal(&openai.ChatCompletionStreamResponse{
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
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(w, "data: "+string(data)+"\n")
			if err != nil {
				return err
			}
			flusher.Flush()
		case api.StreamMessageTypeStatus, api.StreamMessageTypeTool, api.StreamMessageTypeFile:
		default:
			config.Logger.Warnf("unknown stream msg type `%s`", msg.Type)
		}
	}
	return nil
}
