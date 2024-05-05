package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	stlslices "github.com/kkkunny/stl/container/slices"

	"github.com/kkkunny/HuggingChatAPI/internal/api"
	"github.com/kkkunny/HuggingChatAPI/internal/config"
	"github.com/kkkunny/HuggingChatAPI/internal/consts"
)

func ListModels(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	cli := api.NewAPI(consts.HuggingChatDomain, token, nil)
	models, err := cli.ListModels(r.Context())
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(map[string]any{
		"object": "list",
		"data": stlslices.Map(models, func(_ int, model *api.ModelInfo) map[string]any {
			return map[string]any{
				"id":       model.ID,
				"object":   "model",
				"created":  1692901427,
				"owned_by": "system",
			}
		}),
	})
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	_, err = fmt.Fprint(w, string(data))
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}
