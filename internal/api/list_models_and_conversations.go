package api

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	request "github.com/imroc/req/v3"
	stlslices "github.com/kkkunny/stl/container/slices"
	stlerr "github.com/kkkunny/stl/error"
	stlval "github.com/kkkunny/stl/value"
)

type ModelInfo struct {
	ID           string
	Name         string
	Desc         string
	MaxNewTokens int64
	Active       bool
}

type SimpleConversationInfo struct {
	ID        string
	Model     string
	Title     string
	UpdatedAt time.Time
}

// ListModelsAndConversations 列出模型和会话
func ListModelsAndConversations(ctx context.Context, cookies []*http.Cookie) ([]*ModelInfo, []*SimpleConversationInfo, error) {
	httpResp, err := sendDefaultHttpRequest[string](ctx, http.MethodGet, func(r *request.Request) *request.Request {
		return r.SetQueryParam("x-sveltekit-invalidated", "10")
	}, cookies, "/chat/models/__data.json")
	if err != nil {
		return nil, nil, err
	}

	rawStr := "[" + regexp.MustCompile(`}\s*{`).ReplaceAllString(*httpResp, "},{") + "]"
	var rawResp []map[string]any
	err = stlerr.ErrorWrap(json.Unmarshal([]byte(rawStr), &rawResp))
	if err != nil {
		return nil, nil, err
	}
	nodes := stlval.IgnoreWith(stlslices.FindFirst(rawResp, func(_ int, kvs map[string]any) bool {
		return kvs["type"] == "data"
	}))["nodes"]
	data := stlval.IgnoreWith(stlslices.FindFirst(nodes.([]any), func(_ int, obj any) bool {
		return obj.(map[string]any)["type"] == "data"
	})).(map[string]any)["data"].([]any)

	var models []*ModelInfo
	{
		modelIndexObjList := data[int64(data[0].(map[string]any)["models"].(float64))].([]any)
		models = stlslices.Map(modelIndexObjList, func(_ int, modelIndexObj any) *ModelInfo {
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
				if _, existMaxNewTokensIndex := parameters["max_new_tokens"]; existMaxNewTokensIndex {
					maxNewTokensIndex := int64(parameters["max_new_tokens"].(float64))
					maxNewTokens = int64(data[maxNewTokensIndex].(float64))
				}
			}

			return &ModelInfo{
				ID:           id,
				Name:         name,
				Desc:         desc,
				MaxNewTokens: maxNewTokens,
				Active:       !unlisted,
			}
		})
	}

	var conversations []*SimpleConversationInfo
	{
		conversationIndexObjList := data[int64(data[0].(map[string]any)["conversations"].(float64))].([]any)
		conversations = stlslices.Map(conversationIndexObjList, func(_ int, conversationIndexObj any) *SimpleConversationInfo {
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
	}

	return models, conversations, nil
}
