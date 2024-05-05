package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kkkunny/HuggingChatAPI/handler"
	"github.com/kkkunny/HuggingChatAPI/internal/config"
)

func main() {
	svr := chi.NewRouter()

	svr.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			config.Logger.Infof("Method [%s] %s --> %s", r.Method, r.RemoteAddr, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	svr.Get("/v1/models", handler.ListModels)
	svr.Post("/v1/chat/completions", handler.ChatCompletions)

	config.Logger.Keywordf("listen http: 0.0.0.0:80")
	if err := http.ListenAndServe(":80", svr); err != nil {
		panic(err)
	}
}
