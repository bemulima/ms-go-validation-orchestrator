package http

import (
	"net/http"

	apiv1 "github.com/example/ms-validation-orchestrator-service/transport/http/api/v1"
	internalhttp "github.com/example/ms-validation-orchestrator-service/transport/http/internal"
)

// Dependencies contains HTTP transport wiring.
type Dependencies struct {
	APIHandler apiv1.Handler
}

// NewRouter constructs the service HTTP router.
func NewRouter(deps Dependencies) http.Handler {
	root := http.NewServeMux()

	apiMux := http.NewServeMux()
	apiv1.RegisterRoutes(apiMux, deps.APIHandler)
	root.Handle("/api/v1/", http.StripPrefix("/api/v1", apiMux))

	internalMux := http.NewServeMux()
	internalMux.Handle("/", internalhttp.HealthHandler())
	root.Handle("/internal/", http.StripPrefix("/internal", internalMux))

	return root
}
