package http

import (
	"net/http"

	apiv1 "github.com/example/ms-validation-orchestrator-service/internal/transport/http/api/v1"
	apiv2 "github.com/example/ms-validation-orchestrator-service/internal/transport/http/api/v2"
	privatehttp "github.com/example/ms-validation-orchestrator-service/internal/transport/http/private"
)

// Dependencies contains HTTP transport wiring.
type Dependencies struct {
	APIHandler                apiv1.Handler
	PracticeValidationHandler apiv2.Handler
	InternalToken             string
}

// NewRouter constructs the service HTTP router.
func NewRouter(deps Dependencies) http.Handler {
	root := http.NewServeMux()

	apiMux := http.NewServeMux()
	apiv1.RegisterRoutes(apiMux, deps.APIHandler)
	root.Handle("/api/v1/", requireInternalToken(deps.InternalToken, http.StripPrefix("/api/v1", apiMux)))

	practiceMux := http.NewServeMux()
	apiv2.RegisterRoutes(practiceMux, deps.PracticeValidationHandler)
	root.Handle("/api/v2/", requireInternalToken(deps.InternalToken, http.StripPrefix("/api/v2", practiceMux)))

	internalMux := http.NewServeMux()
	internalMux.Handle("/", privatehttp.HealthHandler())
	root.Handle("/internal/", http.StripPrefix("/internal", internalMux))

	return root
}
