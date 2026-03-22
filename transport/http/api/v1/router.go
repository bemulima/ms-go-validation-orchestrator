package v1

import "net/http"

func RegisterRoutes(mux *http.ServeMux, handler Handler) {
	mux.HandleFunc("POST /validate", handler.Validate)
}

func NewRouter(handler Handler, internalRouter http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/validate", handler.Validate)
	mux.Handle("/internal/", http.StripPrefix("/internal", internalRouter))
	return mux
}
