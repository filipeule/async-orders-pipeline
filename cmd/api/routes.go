package main

import "net/http"

func (app *application) mount() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /webhooks/{gateway}", app.webhookHandler)
	mux.HandleFunc("GET /health", app.healthHandler)

	return mux
}
