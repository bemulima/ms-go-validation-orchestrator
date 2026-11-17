package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

const maxPracticeValidationRequestBytes int64 = 2 << 20

type practiceValidationExecutor interface {
	ValidatePractice(context.Context, domain.PracticeValidationRequestV2) (domain.PracticeValidationResultV2, error)
}

type Handler struct {
	useCase practiceValidationExecutor
}

func NewHandler(useCase practiceValidationExecutor) Handler {
	return Handler{useCase: useCase}
}

func RegisterRoutes(mux *http.ServeMux, handler Handler) {
	mux.HandleFunc("POST /practice-validations", handler.ValidatePractice)
}

func (handler Handler) ValidatePractice(responseWriter http.ResponseWriter, request *http.Request) {
	responseWriter.Header().Set("Cache-Control", "no-store")
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxPracticeValidationRequestBytes)
	var input domain.PracticeValidationRequestV2
	if err := decodeStrictJSON(request.Body, &input); err != nil {
		status := http.StatusBadRequest
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			status = http.StatusRequestEntityTooLarge
		}
		writeJSON(responseWriter, status, map[string]any{"error": "invalid_request", "message": err.Error()})
		return
	}
	// Official bounded compilation may exceed the shared 30-second socket
	// deadline. Extend only this already-authenticated, body-bounded route.
	const budget = 2 * time.Minute
	if err := http.NewResponseController(responseWriter).SetWriteDeadline(time.Now().Add(budget)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		writeJSON(responseWriter, http.StatusServiceUnavailable, map[string]string{"error": "validation_deadline_unavailable"})
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), budget)
	defer cancel()
	result, err := handler.useCase.ValidatePractice(ctx, input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRequest) {
			writeJSON(responseWriter, http.StatusBadRequest, map[string]any{"error": "invalid_request", "message": err.Error()})
			return
		}
		writeJSON(responseWriter, http.StatusInternalServerError, map[string]string{"error": "validation_unavailable"})
		return
	}
	writeJSON(responseWriter, http.StatusOK, result)
}

func decodeStrictJSON(reader io.Reader, output any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request must contain exactly one JSON object")
		}
		return err
	}
	return nil
}

func writeJSON(responseWriter http.ResponseWriter, status int, payload any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(status)
	_ = json.NewEncoder(responseWriter).Encode(payload)
}
