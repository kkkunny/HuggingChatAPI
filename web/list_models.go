package main

import (
	"net/http"
	"strings"

	stlslices "github.com/kkkunny/stl/container/slices"
	stlerr "github.com/kkkunny/stl/error"
	"github.com/labstack/echo/v4"
	"github.com/sashabaranov/go-openai"

	"github.com/kkkunny/HuggingChatAPI/config"
	"github.com/kkkunny/HuggingChatAPI/hugchat"
	"github.com/kkkunny/HuggingChatAPI/hugchat/dto"
)

func listModels(reqCtx echo.Context) error {
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

	models, err := cli.ListModels(reqCtx.Request().Context())
	if err != nil {
		return err
	}

	return stlerr.ErrorWrap(reqCtx.JSONPretty(http.StatusOK, &openai.ModelsList{
		Models: stlslices.Map(models, func(_ int, model *dto.ModelInfo) openai.Model {
			return openai.Model{
				CreatedAt: 1692901427,
				ID:        model.ID,
				Object:    "model",
				OwnedBy:   "system",
			}
		}),
	}, ""))
}
