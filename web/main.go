package main

import (
	stlerr "github.com/kkkunny/stl/error"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"github.com/kkkunny/HuggingChatAPI/config"
)

func main() {
	svr := echo.New()
	svr.HideBanner, svr.HidePort = true, true
	svr.Logger.SetLevel(log.OFF)
	svr.IPExtractor = echo.ExtractIPFromRealIPHeader()

	svr.Use(midErrorHandler, midLogger)

	svr.GET("/v1/models", listModels)
	svr.POST("/v1/chat/completions", chatCompletions)

	_ = config.Logger.Keywordf("listen http: 0.0.0.0:80")
	stlerr.Must(svr.Start(":80"))
}
