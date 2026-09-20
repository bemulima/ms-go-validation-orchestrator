package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/example/ms-validation-orchestrator-service/dto"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"github.com/example/ms-validation-orchestrator-service/mapper"
)

const maxValidationRequestBytes int64 = 2 << 20

type validationExecutor interface {
	Execute(ctx context.Context, request domain.ValidationRequest) (domain.ValidationResult, error)
	ConfiguredEngineIDs() []string
	ConfiguredEngineCapabilities() domain.EngineCapabilitiesV1
	InspectContract(request domain.ContractInspectionRequest) (domain.ContractInspectionResultV1, error)
	VerifyContract(ctx context.Context, request domain.ContractVerificationRequestV1) (domain.ContractVerificationReceiptV1, error)
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

func (handler Handler) ListEngines(responseWriter http.ResponseWriter, _ *http.Request) {
	writeJSON(responseWriter, http.StatusOK, map[string]any{
		"engines": handler.useCase.ConfiguredEngineIDs(),
	})
}

func (handler Handler) ListCapabilities(responseWriter http.ResponseWriter, _ *http.Request) {
	writeJSON(responseWriter, http.StatusOK, handler.useCase.ConfiguredEngineCapabilities())
}

func (handler Handler) InspectContract(responseWriter http.ResponseWriter, request *http.Request) {
	var input dto.InspectContractRequest
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxValidationRequestBytes)
	if err := decodeStrictJSON(request.Body, &input); err != nil {
		writeRequestDecodeError(responseWriter, err)
		return
	}
	domainRequest, err := mapper.ToDomainContractInspectionRequest(input)
	if err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, map[string]any{"error": "invalid_request", "message": err.Error()})
		return
	}
	result, err := handler.useCase.InspectContract(domainRequest)
	if err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, map[string]any{"error": "invalid_contract", "message": err.Error()})
		return
	}
	writeJSON(responseWriter, http.StatusOK, result)
}

func (handler Handler) VerifyContract(responseWriter http.ResponseWriter, request *http.Request) {
	var input dto.VerifyContractRequest
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxValidationRequestBytes)
	if err := decodeStrictJSON(request.Body, &input); err != nil {
		writeRequestDecodeError(responseWriter, err)
		return
	}
	domainRequest, err := mapper.ToDomainContractVerificationRequest(input)
	if err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, map[string]any{"error": "invalid_request", "message": err.Error()})
		return
	}
	result, err := handler.useCase.VerifyContract(request.Context(), domainRequest)
	if err != nil {
		writeJSON(responseWriter, http.StatusBadRequest, map[string]any{"error": "verification_failed", "message": err.Error()})
		return
	}
	writeJSON(responseWriter, http.StatusOK, result)
}

func (handler Handler) Validate(responseWriter http.ResponseWriter, request *http.Request) {
	var input dto.ValidateRequest
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxValidationRequestBytes)
	if err := decodeStrictJSON(request.Body, &input); err != nil {
		writeRequestDecodeError(responseWriter, err)
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

func writeRequestDecodeError(responseWriter http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		status = http.StatusRequestEntityTooLarge
	}
	writeJSON(responseWriter, status, map[string]any{
		"error":   "invalid_request",
		"message": err.Error(),
	})
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
			return errors.New("request must contain exactly one JSON object")
		}
		return err
	}
	return nil
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
