package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type stubValidationExecutor struct {
	engineIDs    []string
	capabilities domain.EngineCapabilitiesV1
	verification domain.ContractVerificationReceiptV1
	result       domain.ValidationResult
}

func (executor stubValidationExecutor) Execute(
	context.Context,
	domain.ValidationRequest,
) (domain.ValidationResult, error) {
	return executor.result, nil
}

func (executor stubValidationExecutor) ConfiguredEngineIDs() []string {
	return executor.engineIDs
}

func (executor stubValidationExecutor) ConfiguredEngineCapabilities() domain.EngineCapabilitiesV1 {
	return executor.capabilities
}

func (stubValidationExecutor) InspectContract(
	domain.ContractInspectionRequest,
) (domain.ContractInspectionResultV1, error) {
	return domain.ContractInspectionResultV1{}, nil
}

func (executor stubValidationExecutor) VerifyContract(
	context.Context,
	domain.ContractVerificationRequestV1,
) (domain.ContractVerificationReceiptV1, error) {
	return executor.verification, nil
}

type stubLogger struct{}

func (stubLogger) Info(string, map[string]string)  {}
func (stubLogger) Error(string, map[string]string) {}

func TestListEngines(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubValidationExecutor{
		engineIDs: []string{"go.core", "java.compile", "linux.runtime"},
	}, stubLogger{})
	router := http.NewServeMux()
	RegisterRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/engines", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload struct {
		Engines []string `json:"engines"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := []string{"go.core", "java.compile", "linux.runtime"}
	if !reflect.DeepEqual(payload.Engines, want) {
		t.Fatalf("expected engines %v, got %v", want, payload.Engines)
	}
}

func TestListEnginesRejectsOtherMethods(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubValidationExecutor{}, stubLogger{})
	router := http.NewServeMux()
	RegisterRoutes(router, handler)

	request := httptest.NewRequest(http.MethodPost, "/engines", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
}

func TestListCapabilitiesReturnsVersionedContract(t *testing.T) {
	t.Parallel()

	want := domain.EngineCapabilitiesV1{
		Schema: domain.EngineCapabilitiesSchemaV1,
		Digest: "sha256:test",
		Engines: []domain.EngineCapabilityV1{
			{ID: "go.core", ContractVersions: []int{1}, Modes: []string{"live", "final"}},
		},
	}
	handler := NewHandler(stubValidationExecutor{capabilities: want}, stubLogger{})
	request := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	response := httptest.NewRecorder()

	handler.ListCapabilities(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var got domain.EngineCapabilitiesV1
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected capabilities: %+v", got)
	}
}

func TestVerifyContractReturnsExecutableReceipt(t *testing.T) {
	t.Parallel()

	want := domain.ContractVerificationReceiptV1{
		Schema:        domain.ContractVerificationReceiptSchemaV1,
		Passed:        true,
		ReceiptDigest: "sha256:receipt",
	}
	handler := NewHandler(stubValidationExecutor{verification: want}, stubLogger{})
	request := httptest.NewRequest(http.MethodPost, "/contracts/verify", strings.NewReader(`{
		"schema":"validation-contract-verification-request.v1",
		"blueprint_digest":"sha256:blueprint",
		"runtime_profile_digest":"sha256:runtime",
		"code_structure":{"version":1},
		"cases":[]
	}`))
	response := httptest.NewRecorder()

	handler.VerifyContract(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var got domain.ContractVerificationReceiptV1
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected receipt: %+v", got)
	}
}

func TestValidateRejectsUnknownRequestFields(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubValidationExecutor{}, stubLogger{})
	request := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(`{
		"task_id":"task-1",
		"code_structure":{"rules":{}},
		"workspace":{},
		"unexpected":true
	}`))
	response := httptest.NewRecorder()

	handler.Validate(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}

func TestValidateRejectsTrailingJSONValue(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubValidationExecutor{}, stubLogger{})
	request := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(
		`{"task_id":"task-1","code_structure":{"rules":{}}} {}`,
	))
	response := httptest.NewRecorder()

	handler.Validate(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}

func TestValidateRejectsOversizedRequest(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubValidationExecutor{}, stubLogger{})
	request := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(
		`{"task_id":"`+strings.Repeat("a", int(maxValidationRequestBytes))+`","code_structure":{}}`,
	))
	response := httptest.NewRecorder()

	handler.Validate(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", response.Code)
	}
}

func TestValidateReturnsAdditiveSafeTeacherExplanationAndLegacyClientsIgnoreIt(t *testing.T) {
	t.Parallel()

	explanation := &domain.TeacherValidationExplanationV1{
		Schema: domain.TeacherValidationExplanationSchemaV1, Passed: false,
		BlockingIssues: []domain.TeacherValidationIssueV1{{Code: "FAIL", Message: "Fix the required file.", File: "main.go", Severity: "error"}},
		SourceDigest:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	handler := NewHandler(stubValidationExecutor{result: domain.ValidationResult{
		ContractKind: "workspace_contract", ContractVersion: 1, Passed: false,
		Stages: []domain.StageReport{}, Links: []domain.LinkReport{}, Errors: []domain.ValidationIssue{},
		TeacherExplanation: explanation,
	}}, stubLogger{})
	request := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(`{
		"task_id":"task-1","code_structure":{"rules":{}},"workspace":{}
	}`))
	response := httptest.NewRecorder()
	handler.Validate(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	for _, legacyField := range []string{"contract_kind", "contract_version", "legacy", "passed", "stages"} {
		if _, exists := envelope[legacyField]; !exists {
			t.Fatalf("legacy field %q disappeared: %s", legacyField, response.Body.String())
		}
	}
	if _, exists := envelope["teacher_explanation"]; !exists {
		t.Fatalf("teacher explanation missing: %s", response.Body.String())
	}
	var safeEnvelope struct {
		Schema         string                       `json:"schema"`
		Passed         bool                         `json:"passed"`
		BlockingIssues []map[string]json.RawMessage `json:"blocking_issues"`
		Truncated      bool                         `json:"truncated"`
		SourceDigest   string                       `json:"source_digest"`
	}
	if err := json.Unmarshal(envelope["teacher_explanation"], &safeEnvelope); err != nil || len(safeEnvelope.BlockingIssues) != 1 {
		t.Fatalf("decode safe explanation: envelope=%+v err=%v", safeEnvelope, err)
	}
	allowedIssueFields := map[string]bool{"code": true, "message": true, "hint": true, "file": true, "line": true, "column": true, "stage_id": true, "engine": true, "severity": true}
	for field := range safeEnvelope.BlockingIssues[0] {
		if !allowedIssueFields[field] {
			t.Fatalf("non-allowlisted teacher issue field %q: %s", field, response.Body.String())
		}
	}
	var legacyClient struct {
		ContractKind    string `json:"contract_kind"`
		ContractVersion int    `json:"contract_version"`
		Passed          bool   `json:"passed"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &legacyClient); err != nil || legacyClient.ContractKind != "workspace_contract" || legacyClient.ContractVersion != 1 || legacyClient.Passed {
		t.Fatalf("legacy Go client could not ignore additive field: client=%+v err=%v", legacyClient, err)
	}
	for _, forbidden := range []string{"raw_result", "evidence", "command", "selector", "route", "symbol", "property", "fixture", "rubric"} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("HTTP teacher projection leaked forbidden field %q: %s", forbidden, response.Body.String())
		}
	}
}
