package v1

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/example/ms-validation-orchestrator-service/dto"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"github.com/example/ms-validation-orchestrator-service/mapper"
)

type validationExecutor interface {
	Execute(ctx context.Context, request domain.ValidationRequest) (domain.ValidationResult, error)
}

type logger interface {
	Info(message string, fields map[string]string)
	Error(message string, fields map[string]string)
}

type Handler struct {
	useCase validationExecutor
	logger  logger
}

func NewHandler(useCase validationExecutor, logger logger) Handler {
	return Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (handler Handler) Validate(responseWriter http.ResponseWriter, request *http.Request) {
	var input dto.ValidateRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, map[string]any{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	domainRequest, err := mapper.ToDomainValidationRequest(input)
	if err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, map[string]any{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	result, err := handler.useCase.Execute(request.Context(), domainRequest)
	if err != nil {
		handler.logger.Error("validation failed", map[string]string{
			"task_id": domainRequest.TaskID,
			"error":   err.Error(),
		})

		writeJSON(responseWriter, http.StatusBadRequest, map[string]any{
			"error":   "validation_failed",
			"message": err.Error(),
		})
		return
	}

	handler.logger.Info("validation completed", map[string]string{
		"task_id": domainRequest.TaskID,
		"passed":  boolString(result.Passed),
	})

	writeJSON(responseWriter, http.StatusOK, result)
}

func writeJSON(responseWriter http.ResponseWriter, statusCode int, payload any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(statusCode)
	_ = json.NewEncoder(responseWriter).Encode(payload)
}

func boolString(value bool) string {
	if value {
		return "true"
	}

	return "false"
}
