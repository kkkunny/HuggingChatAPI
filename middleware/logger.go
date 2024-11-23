package middleware

import (
	"github.com/labstack/echo/v4"

	"github.com/kkkunny/HuggingChatAPI/config"
)

func Logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(reqCtx echo.Context) error {
		_ = config.Logger.Infof("Method [%s] %s --> %s", reqCtx.Request().Method, reqCtx.RealIP(), reqCtx.Path())
		return next(reqCtx)
	}
}
