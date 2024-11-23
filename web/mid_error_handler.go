package main

import (
	"errors"
	"net/http"

	stlerr "github.com/kkkunny/stl/error"
	"github.com/labstack/echo/v4"

	"github.com/kkkunny/HuggingChatAPI/config"
)

func midErrorHandler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(reqCtx echo.Context) (err error) {
		var isPanic bool

		defer func() {
			if err != nil {
				if !isPanic {
					_ = config.Logger.Error(err)
				}
				var httpErr *echo.HTTPError
				if errors.As(err, &httpErr) {
					err = httpErr
				} else {
					err = echo.NewHTTPError(http.StatusInternalServerError)
				}
			}
		}()

		defer func() {
			if errObj := recover(); errObj != nil {
				isPanic = true
				_ = config.Logger.Panic(errObj)
				var ok bool
				err, ok = errObj.(error)
				if !ok {
					err = stlerr.Errorf("%v", errObj)
				}
			}
		}()

		return next(reqCtx)
	}
}
