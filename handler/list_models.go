package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	stlslices "github.com/kkkunny/stl/container/slices"
	"github.com/sashabaranov/go-openai"

	"github.com/kkkunny/HuggingChatAPI/internal/api"
	"github.com/kkkunny/HuggingChatAPI/internal/config"
)

func ListModels(w http.ResponseWriter, r *http.Request) {
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

	models, err := cli.ListModels(r.Context())
	if err != nil {
		config.Logger.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(&openai.ModelsList{
		Models: stlslices.Map(models, func(_ int, model *api.ModelInfo) openai.Model {
			return openai.Model{
				CreatedAt: 1692901427,
				ID:        model.ID,
				Object:    "model",
				OwnedBy:   "system",
			}
		}),
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
