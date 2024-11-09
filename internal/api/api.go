package api

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	stlslices "github.com/kkkunny/stl/container/slices"
	stlerr "github.com/kkkunny/stl/error"
)

type Api struct {
	domain    string
	client    *req.Client
	cookieMgr cookieMgr
}

func NewAPI(domain string, token string) (*Api, error) {
	api := &Api{
		domain: domain,
		client: globalClient.Clone().
			SetCommonHeader("origin", domain),
	}
	err := api.SetToken(token)
	return api, err
}

func (api *Api) SetToken(token string) error {
	account, err := base64.StdEncoding.DecodeString(token)
	if err == nil {
		res := regexp.MustCompile(`username=(.+?)&password=(.+)`).FindStringSubmatch(string(account))
		if len(res) != 3 {
			return stlerr.Errorf("invalid token")
		}
		api.cookieMgr = newAccountCookieMgr(res[1], res[2])
		return nil
	}
	api.cookieMgr = newTokenCookieMgr(token)
	return nil
}

func (api *Api) RefreshCookie(ctx context.Context) error {
	cookies, err := api.cookieMgr.Cookies(ctx)
	if err != nil {
		return err
	}
	api.client.ClearCookies().SetCommonCookies(cookies...)

	isLogin, err := api.CheckLogin(ctx)
	if err != nil {
		return err
	} else if isLogin {
		return nil
	}

	cookies, err = api.cookieMgr.Refresh(ctx)
	if err != nil {
		return err
	}
	api.client.ClearCookies().SetCommonCookies(cookies...)

	isLogin, err = api.CheckLogin(ctx)
	if err != nil {
		return err
	} else if isLogin {
		return nil
	}
	return stlerr.Errorf("login failed")
}

func (api *Api) CheckLogin(ctx context.Context) (bool, error) {
	httpResp, err := stlerr.ErrorWith(api.client.R().
		SetContext(ctx).
		SetHeaders(map[string]string{
			"accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		}).
		Get(fmt.Sprintf("%s/chat/", api.domain)))
	if err != nil {
		return false, err
	} else if httpResp.GetStatusCode() != http.StatusOK {
		return false, stlerr.Errorf("http error: code=%d, status=%s", httpResp.GetStatusCode(), httpResp.GetStatus())
	}
	return !strings.Contains(httpResp.String(), "action=\"/chat/login\""), nil
}

type ModelInfo struct {
	ID           string
	Name         string
	Desc         string
	MaxNewTokens int64
	Active       bool
}

func (api *Api) ListModels(ctx context.Context) (resp []*ModelInfo, err error) {
	httpResp, err := stlerr.ErrorWith(api.client.R().
		SetContext(ctx).
		SetSuccessResult(make(map[string]any)).
		Get(fmt.Sprintf("%s/chat/models/__data.json?x-sveltekit-invalidated=10", api.domain)))
	if err != nil {
		return nil, err
	} else if httpResp.GetStatusCode() != http.StatusOK {
		return nil, stlerr.Errorf("http error: code=%d, status=%s", httpResp.GetStatusCode(), httpResp.String())
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = stlerr.Errorf("parse resp error: err=%+v", panicErr)
		}
	}()

	rawResp := *httpResp.SuccessResult().(*map[string]any)
	data := rawResp["nodes"].([]any)[0].(map[string]any)["data"].([]any)

	indexMap := data[0].(map[string]any)
	modelsIndex := int64(indexMap["models"].(float64))
	modelIndexObjList := data[modelsIndex].([]any)
	models := stlslices.Map(modelIndexObjList, func(_ int, modelIndexObj any) *ModelInfo {
		modelIdx := int64(modelIndexObj.(float64))
		modelMetaInfo := data[modelIdx].(map[string]any)

		idIndex := int64(modelMetaInfo["id"].(float64))
		nameIndex := int64(modelMetaInfo["name"].(float64))
		descriptionIndex := int64(modelMetaInfo["description"].(float64))
		parametersIndex := int64(modelMetaInfo["parameters"].(float64))
		unlistedIndex := int64(modelMetaInfo["unlisted"].(float64))

		id := data[idIndex].(string)
		name := data[nameIndex].(string)
		unlisted := data[unlistedIndex].(bool)
		var desc string
		var maxNewTokens int64
		if !unlisted {
			desc = data[descriptionIndex].(string)
			parameters := data[parametersIndex].(map[string]any)
			max_new_tokensIndex := int64(parameters["max_new_tokens"].(float64))
			maxNewTokens = int64(data[max_new_tokensIndex].(float64))
		}

		return &ModelInfo{
			ID:           id,
			Name:         name,
			Desc:         desc,
			MaxNewTokens: maxNewTokens,
			Active:       !unlisted,
		}
	})
	return models, nil
}

type CreateConversationRequest struct {
	Model     string `json:"model"`
	PrePrompt string `json:"preprompt"`
}

type CreateConversationResponse struct {
	ConversationID string `json:"conversationId"`
}

func (api *Api) CreateConversation(ctx context.Context, req *CreateConversationRequest) (*CreateConversationResponse, error) {
	httpResp, err := stlerr.ErrorWith(api.client.R().
		SetContext(ctx).
		SetBodyJsonMarshal(req).
		SetSuccessResult(CreateConversationResponse{}).
		Post(fmt.Sprintf("%s/chat/conversation", api.domain)))
	if err != nil {
		return nil, err
	} else if httpResp.GetStatusCode() != http.StatusOK {
		return nil, stlerr.Errorf("http error: code=%d, status=%s", httpResp.GetStatusCode(), httpResp.String())
	}
	conversation := httpResp.SuccessResult().(*CreateConversationResponse)
	return conversation, nil
}

type DeleteConversationRequest struct {
	ConversationID string
}

func (api *Api) DeleteConversation(ctx context.Context, req *DeleteConversationRequest) error {
	httpResp, err := stlerr.ErrorWith(api.client.R().
		SetContext(ctx).
		Delete(fmt.Sprintf("%s/chat/conversation/%s", api.domain, req.ConversationID)))
	if err != nil {
		return err
	} else if httpResp.GetStatusCode() != http.StatusOK {
		return stlerr.Errorf("http error: code=%d, status=%s", httpResp.GetStatusCode(), httpResp.String())
	}
	return nil
}

type SimpleConversationInfo struct {
	ID        string
	Model     string
	Title     string
	UpdatedAt time.Time
}

func (api *Api) ListConversations(ctx context.Context) (resp []*SimpleConversationInfo, err error) {
	httpResp, err := stlerr.ErrorWith(api.client.R().
		SetContext(ctx).
		Get(fmt.Sprintf("%s/chat/models/__data.json?x-sveltekit-invalidated=10", api.domain)))
	if err != nil {
		return nil, err
	} else if httpResp.GetStatusCode() != http.StatusOK {
		return nil, stlerr.Errorf("http error: code=%d, status=%s", httpResp.GetStatusCode(), httpResp.String())
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = stlerr.Errorf("parse resp error: err=%+v", panicErr)
		}
	}()

	rawStr := "[" + regexp.MustCompile(`}\s*{`).ReplaceAllString(httpResp.String(), "},{") + "]"
	var rawResp []map[string]any
	err = stlerr.ErrorWrap(json.Unmarshal([]byte(rawStr), &rawResp))
	if err != nil {
		return nil, err
	}
	for _, rawRespItem := range rawResp {
		if rawRespItem["type"] != "chunk" || fmt.Sprintf("%v", rawRespItem["id"]) != "1" {
			continue
		}

		data := rawRespItem["data"].([]any)
		conversationIndexObjList := data[0].([]any)
		conversationInfos := stlslices.Map(conversationIndexObjList, func(_ int, conversationIndexObj any) *SimpleConversationInfo {
			conversationIdx := int64(conversationIndexObj.(float64))
			conversationMetaInfo := data[conversationIdx].(map[string]any)

			idIndex := int64(conversationMetaInfo["id"].(float64))
			titleIndex := int64(conversationMetaInfo["title"].(float64))
			modelIndex := int64(conversationMetaInfo["model"].(float64))
			updatedAtIndex := int64(conversationMetaInfo["updatedAt"].(float64))

			id := data[idIndex].(string)
			title := data[titleIndex].(string)
			model := data[modelIndex].(string)
			updatedAtStr := data[updatedAtIndex].([]any)[1].(string)
			updatedAt, _ := time.Parse(time.RFC3339, updatedAtStr)

			return &SimpleConversationInfo{
				ID:        id,
				Title:     title,
				Model:     model,
				UpdatedAt: updatedAt,
			}
		})
		return conversationInfos, nil
	}
	return nil, stlerr.Errorf("not found conversion data")
}

type ConversationInfoRequest struct {
	ConversationID string `json:"-"`
}

type ConversationInfoResponse struct {
	ConversationID string
	Model          string
	Title          string
	PrePrompt      string
	Messages       []*Message
}

type Message struct {
	ID       string
	From     string
	Content  string
	Children []string
	CreateAt time.Time
	UpdateAt time.Time
}

func (api *Api) ConversationInfo(ctx context.Context, req *ConversationInfoRequest) (resp *ConversationInfoResponse, err error) {
	httpResp, err := stlerr.ErrorWith(api.client.R().
		SetContext(ctx).
		SetSuccessResult(make(map[string]any)).
		Get(fmt.Sprintf("%s/chat/conversation/%s/__data.json?x-sveltekit-invalidated=01", api.domain, req.ConversationID)))
	if err != nil {
		return nil, err
	} else if httpResp.GetStatusCode() != http.StatusOK {
		return nil, stlerr.Errorf("http error: code=%d, status=%s", httpResp.GetStatusCode(), httpResp.String())
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = stlerr.Errorf("parse resp error: err=%+v", panicErr)
		}
	}()

	rawResp := *httpResp.SuccessResult().(*map[string]any)
	data := rawResp["nodes"].([]any)[1].(map[string]any)["data"].([]any)

	indexMap := data[0].(map[string]any)
	messagesIndex := int64(indexMap["messages"].(float64))
	titleIndex := int64(indexMap["title"].(float64))
	modelIndex := int64(indexMap["model"].(float64))
	prepromptIndex := int64(indexMap["preprompt"].(float64))

	messagesIndexList := data[messagesIndex].([]any)
	model := data[modelIndex].(string)
	title := data[titleIndex].(string)
	prePrompt := data[prepromptIndex].(string)

	messages := stlslices.Map(messagesIndexList, func(_ int, msgIdxObj any) *Message {
		msgIdx := int64(msgIdxObj.(float64))
		msgMetaInfo := data[msgIdx].(map[string]any)

		idIndex := int64(msgMetaInfo["id"].(float64))
		fromIndex := int64(msgMetaInfo["from"].(float64))
		contentIndex := int64(msgMetaInfo["content"].(float64))
		childrenIndex := int64(msgMetaInfo["children"].(float64))
		createdAtIndex := int64(msgMetaInfo["createdAt"].(float64))
		updatedAtIndex := int64(msgMetaInfo["updatedAt"].(float64))

		id := data[idIndex].(string)
		from := data[fromIndex].(string)
		content := data[contentIndex].(string)
		children := stlslices.Map(data[childrenIndex].([]any), func(_ int, childIdxObj any) string {
			childIdx := int64(childIdxObj.(float64))
			return data[childIdx].(string)
		})
		createdAtStr := data[createdAtIndex].([]any)[1].(string)
		createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
		updatedAtStr := data[updatedAtIndex].([]any)[1].(string)
		updatedAt, _ := time.Parse(time.RFC3339, updatedAtStr)
		return &Message{
			ID:       id,
			From:     from,
			Content:  content,
			Children: children,
			CreateAt: createdAt,
			UpdateAt: updatedAt,
		}
	})
	return &ConversationInfoResponse{
		ConversationID: req.ConversationID,
		Model:          model,
		Title:          title,
		PrePrompt:      prePrompt,
		Messages:       messages,
	}, nil
}

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

type ChatConversationResponse struct {
	Stream chan StreamMessage
}

type StreamMessageType string

const (
	StreamMessageTypeStatus      StreamMessageType = "status"
	StreamMessageTypeStream      StreamMessageType = "stream"
	StreamMessageTypeFinalAnswer StreamMessageType = "finalAnswer"
	StreamMessageTypeError       StreamMessageType = "error"
	StreamMessageTypeTool        StreamMessageType = "tool"
	StreamMessageTypeFile        StreamMessageType = "file"
)

type StreamMessageSubType string

const (
	StreamMessageSubTypeCall   StreamMessageSubType = "call"
	StreamMessageSubTypeEta    StreamMessageSubType = "eta"
	StreamMessageSubTypeResult StreamMessageSubType = "result"
)

type StreamMessageStatus string

const (
	StreamMessageStatusStarted   StreamMessageStatus = "started"
	StreamMessageStatusTitle     StreamMessageStatus = "title"
	StreamMessageStatusSuccess   StreamMessageStatus = "success"
	StreamMessageStatusKeepAlive StreamMessageStatus = "keepAlive"
)

type StreamMessage struct {
	Type    StreamMessageType      `json:"type"`
	SubType *StreamMessageSubType  `json:"subtype,omitempty"` // only StreamMessageTypeTool
	UUID    *string                `json:"uuid,omitempty"`    // only StreamMessageTypeTool
	Eta     *float64               `json:"eta,omitempty"`     // only StreamMessageTypeTool && StreamMessageSubTypeEta
	Call    *StreamMessageToolCall `json:"call,omitempty"`    // only StreamMessageTypeTool && (StreamMessageSubTypeCall || StreamMessageSubTypeResult)
	Status  *StreamMessageStatus   `json:"status,omitempty"`  // only StreamMessageTypeStatus || (StreamMessageTypeTool && StreamMessageSubTypeResult)
	Token   *string                `json:"token,omitempty"`   // only StreamMessageTypeStream
	Text    *string                `json:"text,omitempty"`    // only StreamMessageTypeFinalAnswer
	Message *string                `json:"message,omitempty"` // only StreamMessageTypeStatus && StreamMessageStatusTitle
	Error   error                  `json:"-"`                 // only StreamMessageTypeError
	Name    *string                `json:"name,omitempty"`    // only StreamMessageTypeFile
	SHA     *string                `json:"sha,omitempty"`     // only StreamMessageTypeFile
	MIME    *string                `json:"mime,omitempty"`    // only StreamMessageTypeFile
}

type StreamMessageToolCall struct {
	Name       string                     `json:"name"`
	Parameters StreamMessageToolParameter `json:"parameters"`
}

type StreamMessageToolParameter struct {
	Prompt string `json:"prompt"`
	Width  string `json:"width"`
	Height string `json:"height"`
}

func (api *Api) ChatConversation(ctx context.Context, req *ChatConversationRequest) (*ChatConversationResponse, error) {
	if len(req.Tools) == 0 {
		req.Tools = make([]string, 0)
	}
	reqBody, err := stlerr.ErrorWith(json.Marshal(req))
	if err != nil {
		return nil, err
	}

	resp, err := stlerr.ErrorWith(api.client.R().
		SetContext(ctx).
		SetHeaders(map[string]string{
			"authority":          "huggingface.co",
			"accept":             "*/*",
			"accept-language":    "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6",
			"origin":             "https://huggingface.co",
			"sec-ch-ua":          "\"Not/A)Brand\";v=\"8\", \"Chromium\";v=\"126\", \"Microsoft Edge\";v=\"126\"",
			"sec-ch-ua-mobile":   "?0",
			"sec-ch-ua-platform": "\"Windows\"",
			"sec-fetch-dest":     "empty",
			"sec-fetch-mode":     "cors",
			"sec-fetch-site":     "same-origin",
		}).
		SetFormData(map[string]string{"data": string(reqBody)}).
		DisableAutoReadResponse().
		Post(fmt.Sprintf("%s/chat/conversation/%s", api.domain, req.ConversationID)))
	if err != nil {
		return nil, err
	} else if resp.GetStatusCode() != http.StatusOK {
		return nil, stlerr.Errorf("http error: code=%d, status=%s", resp.GetStatusCode(), resp.String())
	}

	reader := bufio.NewReader(resp.Body)
	msgChan := make(chan StreamMessage)

	go func() {
		defer func() {
			close(msgChan)
		}()

		for !resp.Close {
			line, err := stlerr.ErrorWith(reader.ReadString('\n'))
			if err != nil && errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				msgChan <- StreamMessage{Type: StreamMessageTypeError, Error: err}
				break
			}
			data := strings.TrimSpace(line)
			if data == "" {
				continue
			}

			var msg StreamMessage
			err = stlerr.ErrorWrap(json.Unmarshal([]byte(data), &msg))
			if err != nil {
				msgChan <- StreamMessage{Type: StreamMessageTypeError, Error: err}
				break
			}

			msgChan <- msg
		}
	}()

	return &ChatConversationResponse{Stream: msgChan}, nil
}
