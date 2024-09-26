package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (app *Config) Routes() http.Handler {
	mux := chi.NewRouter()

	//specify who is allowed to connect
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"https://*",
			"http://*",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	//broker-service
	mux.Use(middleware.Heartbeat("/ping"))
	mux.Get("/", app.Broker)
	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		payload := jsonResponse{
			Error:   false,
			Message: "healthy",
		}
		app.writeJSON(w, http.StatusOK, payload)
	})

	//authentication service
	mux.Post("/auth-service/authenticate", app.Authenticate)

	//Snippet Service
	mux.Post("/snippet-service/snippet", app.CreateSnippet)
	mux.Get("/snippet-service/search", app.SearchSnippet)
	mux.Get("/snippet-service/user/snippets", app.GetUserSnippets)

	//Logger Service
	mux.Post("/logger-service/log", app.LogItem)

	//Communication Service
	mux.Post("/comm-service/mail", app.SendMail)

	mux.Post("/logger-service/log", app.LogItem)

	return mux
}
