package dto

import (
	"github.com/kkkunny/HuggingChatAPI/internal/api"
)

// ModelInfo 模型信息
type ModelInfo struct {
	ID           string
	Name         string
	Desc         string
	MaxNewTokens int64
	Active       bool
}

func NewModelInfoFromAPI(model *api.ModelInfo) *ModelInfo {
	if model == nil {
		return nil
	}
	return &ModelInfo{
		ID:           model.ID,
		Name:         model.Name,
		Desc:         model.Desc,
		MaxNewTokens: model.MaxNewTokens,
		Active:       model.Active,
	}
}
