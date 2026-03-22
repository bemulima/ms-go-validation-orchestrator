package public

import (
	"context"
	"net/http"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type Server interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type ValidationClient interface {
	Validate(ctx context.Context, request domain.ValidationRequest) (domain.ValidationResult, error)
}

func AdaptServer(server *http.Server) Server {
	return server
}
