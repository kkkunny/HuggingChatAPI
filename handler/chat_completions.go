package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	stlbasic "github.com/kkkunny/stl/basic"
	stlslices "github.com/kkkunny/stl/container/slices"

	"github.com/satori/go.uuid"

	"github.com/sashabaranov/go-openai"

	"github.com/kkkunny/HuggingChatAPI/internal/api"
	"github.com/kkkunny/HuggingChatAPI/internal/config"
	"github.com/kkkunny/HuggingChatAPI/internal/consts"
)

func ChatCompletions(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	cli := api.NewAPI(consts.HuggingChatDomain, token, nil)

	var req openai.ChatCompletionRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(body, &req)
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

	msgID := stlbasic.Ternary(stlslices.Last(convInfo.Messages).From != "system", stlslices.Last(convInfo.Messages).ID, uuid.NewV4().String())
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

	var reply string
	for msg := range chatResp.Stream {
		switch msg.Type {
		case api.StreamMessageTypeError:
			config.Logger.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case api.StreamMessageTypeFinalAnswer:
			if msg.Text != nil {
				reply = *msg.Text
			}
			break
		case api.StreamMessageTypeStatus, api.StreamMessageTypeStream:
		default:
			config.Logger.Warnf("unknown stream msg type `%s`", msg.Type)
		}
	}

	data, err := json.Marshal(&openai.ChatCompletionResponse{
		ID:      msgID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: reply,
				},
				FinishReason: "stop",
			},
		},
	})
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = fmt.Fprint(w, string(data))
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}