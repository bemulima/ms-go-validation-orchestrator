package v2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

const practiceTestDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type practiceValidationExecutorStub struct {
	result domain.PracticeValidationResultV2
	called bool
}

func (executor *practiceValidationExecutorStub) ValidatePractice(_ context.Context, request domain.PracticeValidationRequestV2) (domain.PracticeValidationResultV2, error) {
	executor.called = true
	if executor.result.TaskInstanceID == "" {
		executor.result = domain.PracticeValidationResultV2{
			Schema:         domain.PracticeValidationResultSchemaV2,
			TaskInstanceID: request.TaskInstanceID, SubmissionID: request.SubmissionID, ValidationRunID: request.ValidationRunID,
			RevisionDigest: request.RevisionDigest, ValidationContractDigest: request.ValidationContractDigest,
			Snapshot: request.Snapshot, Outcome: domain.PracticeValidationError,
			ErrorClassification: domain.PracticeValidationStageExecutionError,
			Stages:              []domain.PracticeValidationStageSummaryV2{{StageID: "stage-1", Required: true, Outcome: domain.PracticeValidationError, Issues: []domain.PracticeValidationIssueV2{{Code: "STAGE_EXECUTION_ERROR", Message: "Validation could not complete."}}}},
		}
	}
	return executor.result, nil
}

func TestPracticeValidationHandlerReturnsCorrelatedTypedError(t *testing.T) {
	t.Parallel()
	executor := &practiceValidationExecutorStub{}
	handler := NewHandler(executor)
	request := httptest.NewRequest(http.MethodPost, "/practice-validations", strings.NewReader(validPracticeRequestJSON()))
	response := httptest.NewRecorder()
	handler.ValidatePractice(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", response.Header().Get("Cache-Control"))
	}
	var result domain.PracticeValidationResultV2
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !executor.called || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationStageExecutionError {
		t.Fatalf("result=%+v called=%t", result, executor.called)
	}
	if result.TaskInstanceID != "11111111-1111-4111-8111-111111111111" || result.SubmissionID != "22222222-2222-4222-8222-222222222222" || result.ValidationRunID != "33333333-3333-4333-8333-333333333333" {
		t.Fatalf("correlation identity not echoed: %+v", result)
	}
	if result.Snapshot.GenerationID != "gen-1" || result.ValidationContractDigest != practiceTestDigest {
		t.Fatalf("snapshot/digest not echoed: %+v", result)
	}
}

func TestPracticeValidationHandlerRejectsUnknownAndTrailingFields(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		strings.Replace(validPracticeRequestJSON(), `"scope":"MILESTONE"`, `"scope":"MILESTONE","unexpected":true`, 1),
		validPracticeRequestJSON() + ` {}`,
	} {
		executor := &practiceValidationExecutorStub{}
		handler := NewHandler(executor)
		request := httptest.NewRequest(http.MethodPost, "/practice-validations", strings.NewReader(body))
		response := httptest.NewRecorder()
		handler.ValidatePractice(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		if executor.called {
			t.Fatal("invalid command reached the use case")
		}
	}
}

func validPracticeRequestJSON() string {
	return `{"schema":"practice-validation-request.v2","task_instance_id":"11111111-1111-4111-8111-111111111111","submission_id":"22222222-2222-4222-8222-222222222222","validation_run_id":"33333333-3333-4333-8333-333333333333","revision_digest":"` + practiceTestDigest + `","validation_contract_digest":"` + practiceTestDigest + `","snapshot":{"sandbox_id":"44444444-4444-4444-8444-444444444444","snapshot_id":"snapshot-1","generation_id":"gen-1","workspace_digest":"` + practiceTestDigest + `"},"scope":"MILESTONE","milestone_id":"step-1","validation_specification":{"schema":"practice-validation-specification.v1","runtime_profile":{},"contracts":[{"milestone_id":"step-1","required":true,"validation_contract":{"version":1,"kind":"workspace_contract","stages":[{"id":"stage-1","engine":"practice.test"}]},"evaluation_assets":{"negative_fixture_refs":[]}}]}}`
}
