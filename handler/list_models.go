package handler

import (
	"net/http"
	"strings"

	stlslices "github.com/kkkunny/stl/container/slices"
	stlerr "github.com/kkkunny/stl/error"
	"github.com/labstack/echo/v4"
	"github.com/sashabaranov/go-openai"

	"github.com/kkkunny/HuggingChatAPI/internal/api"
	"github.com/kkkunny/HuggingChatAPI/internal/config"
)

func ListModels(reqCtx echo.Context) error {
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

	models, err := cli.ListModels(reqCtx.Request().Context())
	if err != nil {
		return err
	}

	return stlerr.ErrorWrap(reqCtx.JSONPretty(http.StatusOK, &openai.ModelsList{
		Models: stlslices.Map(models, func(_ int, model *api.ModelInfo) openai.Model {
			return openai.Model{
				CreatedAt: 1692901427,
				ID:        model.ID,
				Object:    "model",
				OwnedBy:   "system",
			}
		}),
	}, ""))
}
