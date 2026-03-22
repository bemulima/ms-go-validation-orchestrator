package engines

import (
	"context"
	"encoding/json"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type HTMLClient struct {
	baseURL string
	http    HTTPClient
}

func NewHTMLClient(baseURL string, httpClient HTTPClient) HTMLClient {
	return HTMLClient{
		baseURL: baseURL,
		http:    httpClient,
	}
}

func (client HTMLClient) EngineID() string {
	return "html.dom"
}

func (client HTMLClient) Validate(
	ctx context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	body, ok := firstTargetContent(input.Stage, input.Workspace)
	if !ok {
		return domain.StageExecutionResult{
			Passed: false,
			Errors: []domain.ValidationIssue{
				{
					Code:     "HTML_TARGET_FILE_MISSING",
					Message:  "no target html file found in workspace",
					Severity: "error",
					StageID:  input.Stage.ID,
					Engine:   input.Stage.Engine,
				},
			},
		}, nil
	}

	requestPayload := map[string]any{
		"code":   body,
		"locale": input.Locale,
	}
	if input.TaskID != "" {
		requestPayload["taskId"] = input.TaskID
	}
	if hasPayload(input.Stage.Rules) {
		requestPayload["rules"] = rawJSONOrEmptyObject(input.Stage.Rules)
	}

	responseBody, err := client.http.PostJSON(ctx, client.baseURL+"/validate", requestPayload)
	if err != nil {
		return domain.StageExecutionResult{}, err
	}

	return parseCommonValidationResponse(responseBody, input.Stage)
}

func firstTargetContent(stage domain.ValidationStage, workspace domain.ValidationWorkspace) (string, bool) {
	for _, target := range stage.Targets.Files {
		for _, file := range workspace.Files {
			if file.Path == target {
				return file.Content, true
			}
		}
	}

	if len(workspace.Files) == 0 {
		return "", false
	}

	return workspace.Files[0].Content, true
}

func parseCommonValidationResponse(
	body []byte,
	stage domain.ValidationStage,
) (domain.StageExecutionResult, error) {
	type validationError struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Detail   string `json:"detail"`
		File     string `json:"file"`
		Path     string `json:"path"`
		Line     int    `json:"line"`
		Column   int    `json:"column"`
		Selector string `json:"selector"`
		Symbol   string `json:"symbol"`
		Property string `json:"property"`
		Route    string `json:"route"`
		Hint     string `json:"hint"`
		Position struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"position"`
	}

	type response struct {
		OK       bool              `json:"ok"`
		IsValid  bool              `json:"isValid"`
		Valid    bool              `json:"valid"`
		Warnings []validationError `json:"warnings"`
		Evidence []validationError `json:"evidence"`
		Errors   []validationError `json:"errors"`
	}

	var payload response
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.StageExecutionResult{}, err
	}

	passed := payload.OK || payload.IsValid || payload.Valid
	if !payload.OK && !payload.IsValid && !payload.Valid && len(payload.Errors) == 0 {
		passed = false
	}
	if payload.OK || (payload.IsValid && payload.Valid) {
		passed = true
	}

	errors := make([]domain.ValidationIssue, 0, len(payload.Errors))
	for _, item := range payload.Errors {
		errors = append(errors, domain.ValidationIssue{
			Code:     defaultString(item.Code, "VALIDATION_FAILED"),
			Message:  item.Message,
			Severity: "error",
			StageID:  stage.ID,
			Engine:   stage.Engine,
			File:     defaultString(item.File, item.Path),
			Line:     firstPositive(item.Line, item.Position.Line),
			Column:   firstPositive(item.Column, item.Position.Column),
			Selector: item.Selector,
			Symbol:   defaultString(item.Symbol, item.Detail),
			Property: item.Property,
			Route:    item.Route,
			Hint:     item.Hint,
		})
	}

	warnings := make([]domain.ValidationIssue, 0, len(payload.Warnings))
	for _, item := range payload.Warnings {
		warnings = append(warnings, domain.ValidationIssue{
			Code:     defaultString(item.Code, "VALIDATION_WARNING"),
			Message:  item.Message,
			Severity: defaultString(item.Severity, "warning"),
			StageID:  stage.ID,
			Engine:   stage.Engine,
			File:     defaultString(item.File, item.Path),
			Line:     firstPositive(item.Line, item.Position.Line),
			Column:   firstPositive(item.Column, item.Position.Column),
			Selector: item.Selector,
			Symbol:   defaultString(item.Symbol, item.Detail),
			Property: item.Property,
			Route:    item.Route,
			Hint:     item.Hint,
		})
	}

	evidence := make([]domain.ValidationPoint, 0, len(payload.Evidence))
	for _, item := range payload.Evidence {
		evidence = append(evidence, domain.ValidationPoint{
			File:     defaultString(item.File, item.Path),
			Selector: item.Selector,
			Route:    item.Route,
			Symbol:   defaultString(item.Symbol, item.Detail),
			Property: item.Property,
			Message:  item.Message,
		})
	}

	return domain.StageExecutionResult{
		Passed:    passed,
		Evidence:  evidence,
		Errors:    errors,
		Warnings:  warnings,
		RawResult: body,
	}, nil
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func firstPositive(value int, fallback int) int {
	if value > 0 {
		return value
	}

	return fallback
}
