package api

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	request "github.com/imroc/req/v3"
	stlslices "github.com/kkkunny/stl/container/slices"
	stlerr "github.com/kkkunny/stl/error"
)

type DetailConversationInfo struct {
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

func parseDetailConversationInfo(convID string, data []any) (*DetailConversationInfo, error) {
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
	return &DetailConversationInfo{
		ConversationID: convID,
		Model:          model,
		Title:          title,
		PrePrompt:      prePrompt,
		Messages:       messages,
	}, nil
}

// ConversationInfoAfterCreate 在创建会话后获取会话信息
func ConversationInfoAfterCreate(ctx context.Context, cookies []*http.Cookie, convID string) (*DetailConversationInfo, error) {
	httpResp, err := sendDefaultHttpRequest[string](ctx, http.MethodGet, func(r *request.Request) *request.Request {
		return r.SetQueryParam("x-sveltekit-invalidated", "11")
	}, cookies, "/chat/conversation/%s/__data.json", convID)
	if err != nil {
		return nil, err
	}

	rawStr := "[" + regexp.MustCompile(`}\s*\{`).ReplaceAllString(*httpResp, "},{") + "]"
	var rawResp []map[string]any
	err = stlerr.ErrorWrap(json.Unmarshal([]byte(rawStr), &rawResp))
	if err != nil {
		return nil, err
	}

	for _, rawRespItem := range rawResp {
		if rawRespItem["type"] != "data" {
			continue
		}
		nodes := rawRespItem["nodes"].([]any)
		for _, node := range nodes {
			if stlslices.First(node.(map[string]any)["uses"].(map[string]any)["dependencies"].([]any)).(string) != "https://huggingface.co/chat/conversation/conversation" {
				continue
			}
			return parseDetailConversationInfo(convID, node.(map[string]any)["data"].([]any))
		}
	}
	return nil, stlerr.Errorf("not found conversation, id=%s", convID)
}

// ConversationInfo 获取会话信息
func ConversationInfo(ctx context.Context, cookies []*http.Cookie, convID string) (resp *DetailConversationInfo, err error) {
	httpResp, err := sendDefaultHttpRequest[map[string]any](ctx, http.MethodGet, func(r *request.Request) *request.Request {
		return r.SetQueryParam("x-sveltekit-invalidated", "01")
	}, cookies, "/chat/conversation/%s/__data.json", convID)
	if err != nil {
		return nil, err
	}
	node := (*httpResp)["nodes"].([]any)[1].(map[string]any)
	if node["type"] == "error" && strings.Contains(node["error"].(map[string]any)["message"].(string), "access to") {
		return nil, stlerr.ErrorWrap(ErrUnauthorized)
	}
	return parseDetailConversationInfo(convID, node["data"].([]any))
}
