package usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type OrchestrateValidationUseCase struct {
	parser  ContractParser
	engines map[string]domain.EngineClient
}

func NewOrchestrateValidationUseCase(
	parser ContractParser,
	engineClients []domain.EngineClient,
) OrchestrateValidationUseCase {
	engines := make(map[string]domain.EngineClient, len(engineClients))
	for _, client := range engineClients {
		engines[client.EngineID()] = client
	}

	return OrchestrateValidationUseCase{
		parser:  parser,
		engines: engines,
	}
}

func (useCase OrchestrateValidationUseCase) Execute(
	ctx context.Context,
	request domain.ValidationRequest,
) (domain.ValidationResult, error) {
	contract, legacy, err := useCase.parser.Parse(request)
	if err != nil {
		return domain.ValidationResult{}, err
	}

	filteredStages := filterStagesByMode(contract.Stages, request.Mode)
	orderedStages, err := orderStages(filteredStages)
	if err != nil {
		return domain.ValidationResult{}, err
	}
	filteredLinks := filterLinksByStageIDs(contract.Links, collectStageIDs(orderedStages))

	result := domain.ValidationResult{
		ContractKind:    contract.Kind,
		ContractVersion: contract.Version,
		Legacy:          legacy,
		Passed:          true,
		Stages:          make([]domain.StageReport, 0, len(orderedStages)),
		Links:           make([]domain.LinkReport, 0, len(filteredLinks)),
		Errors:          []domain.ValidationIssue{},
	}

	stageIndex := make(map[string]domain.StageReport, len(orderedStages))

	for _, stage := range orderedStages {
		report, execErr := useCase.executeStage(ctx, request, stage, stageIndex)
		result.Stages = append(result.Stages, report)
		stageIndex[stage.ID] = report

		if execErr != nil {
			result.Passed = false
			result.Errors = append(result.Errors, domain.ValidationIssue{
				Code:     "STAGE_EXECUTION_ERROR",
				Message:  execErr.Error(),
				Severity: "error",
				StageID:  stage.ID,
				Engine:   stage.Engine,
			})
			if !stage.Optional {
				continue
			}
		}

		if !report.Passed && !stage.Optional {
			result.Passed = false
		}

		result.Errors = append(result.Errors, report.Errors...)
	}

	linkReports := executeLinks(filteredLinks, request.Workspace, stageIndex)
	result.Links = append(result.Links, linkReports...)
	for _, linkReport := range linkReports {
		if !linkReport.Passed && !linkReport.Optional {
			result.Passed = false
		}
		result.Errors = append(result.Errors, linkReport.Errors...)
	}

	return result, nil
}

func filterStagesByMode(stages []domain.ValidationStage, mode string) []domain.ValidationStage {
	if mode == "" {
		return stages
	}

	filtered := make([]domain.ValidationStage, 0, len(stages))
	for _, stage := range stages {
		if stageMatchesMode(stage.Mode, mode) {
			filtered = append(filtered, stage)
		}
	}

	return filtered
}

func filterLinksByStageIDs(links []domain.ValidationLink, stageIDs map[string]struct{}) []domain.ValidationLink {
	filtered := make([]domain.ValidationLink, 0, len(links))
	for _, link := range links {
		if allDependenciesAvailable(link.DependsOn, stageIDs) {
			filtered = append(filtered, link)
		}
	}

	return filtered
}

func collectStageIDs(stages []domain.ValidationStage) map[string]struct{} {
	result := make(map[string]struct{}, len(stages))
	for _, stage := range stages {
		result[stage.ID] = struct{}{}
	}

	return result
}

func allDependenciesAvailable(dependencies []string, stageIDs map[string]struct{}) bool {
	for _, dependency := range dependencies {
		if _, exists := stageIDs[dependency]; !exists {
			return false
		}
	}

	return true
}

func stageMatchesMode(stageMode string, requestMode string) bool {
	switch stageMode {
	case "", domain.ValidationModeBoth:
		return true
	case requestMode:
		return true
	default:
		return false
	}
}

func (useCase OrchestrateValidationUseCase) executeStage(
	ctx context.Context,
	request domain.ValidationRequest,
	stage domain.ValidationStage,
	stageIndex map[string]domain.StageReport,
) (domain.StageReport, error) {
	if dependencyFailed(stage.DependsOn, stageIndex) {
		return domain.StageReport{
			StageID:  stage.ID,
			Engine:   stage.Engine,
			Status:   "skipped",
			Passed:   stage.Optional,
			Optional: stage.Optional,
			Errors: []domain.ValidationIssue{
				{
					Code:     "DEPENDENCY_FAILED",
					Message:  "stage skipped because dependency failed",
					Severity: "error",
					StageID:  stage.ID,
					Engine:   stage.Engine,
				},
			},
		}, nil
	}

	engine, ok := useCase.engines[stage.Engine]
	if !ok {
		return domain.StageReport{
			StageID:  stage.ID,
			Engine:   stage.Engine,
			Status:   "failed",
			Passed:   false,
			Optional: stage.Optional,
			Errors: []domain.ValidationIssue{
				{
					Code:     "UNSUPPORTED_ENGINE",
					Message:  fmt.Sprintf("engine %q is not configured", stage.Engine),
					Severity: "error",
					StageID:  stage.ID,
					Engine:   stage.Engine,
				},
			},
		}, fmt.Errorf("%w: %s", domain.ErrUnsupportedEngine, stage.Engine)
	}

	startedAt := time.Now()
	executionResult, err := engine.Validate(ctx, domain.EngineValidationInput{
		TaskID:       request.TaskID,
		Stage:        stage,
		Workspace:    request.Workspace,
		TaskMetadata: request.TaskMetadata,
		Locale:       request.Locale,
		Mode:         request.Mode,
	})

	report := domain.StageReport{
		StageID:   stage.ID,
		Engine:    stage.Engine,
		Status:    "passed",
		Passed:    executionResult.Passed,
		Optional:  stage.Optional,
		Duration:  time.Since(startedAt).Milliseconds(),
		Evidence:  executionResult.Evidence,
		Errors:    executionResult.Errors,
		Warnings:  executionResult.Warnings,
		RawResult: executionResult.RawResult,
	}

	if err != nil {
		report.Status = "failed"
		report.Passed = false
		return report, fmt.Errorf("%w: %s: %v", domain.ErrStageExecutionFailed, stage.ID, err)
	}

	if !executionResult.Passed {
		report.Status = "failed"
	}

	return report, nil
}

func orderStages(stages []domain.ValidationStage) ([]domain.ValidationStage, error) {
	stageByID := make(map[string]domain.ValidationStage, len(stages))
	inDegree := make(map[string]int, len(stages))
	dependents := make(map[string][]string, len(stages))
	queue := make([]string, 0, len(stages))
	ordered := make([]domain.ValidationStage, 0, len(stages))

	for _, stage := range stages {
		if stage.ID == "" || stage.Engine == "" {
			return nil, fmt.Errorf("%w: stage id and engine are required", domain.ErrInvalidContract)
		}

		if _, exists := stageByID[stage.ID]; exists {
			return nil, fmt.Errorf("%w: duplicate stage id %q", domain.ErrInvalidContract, stage.ID)
		}

		stageByID[stage.ID] = stage
		inDegree[stage.ID] = 0
	}

	for _, stage := range stages {
		for _, dependency := range stage.DependsOn {
			if _, exists := stageByID[dependency]; !exists {
				return nil, fmt.Errorf("%w: unknown dependency %q for stage %q", domain.ErrInvalidContract, dependency, stage.ID)
			}

			inDegree[stage.ID]++
			dependents[dependency] = append(dependents[dependency], stage.ID)
		}
	}

	for _, stage := range stages {
		if inDegree[stage.ID] == 0 {
			queue = append(queue, stage.ID)
		}
	}

	slices.Sort(queue)

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		ordered = append(ordered, stageByID[currentID])

		for _, dependentID := range dependents[currentID] {
			inDegree[dependentID]--
			if inDegree[dependentID] == 0 {
				queue = append(queue, dependentID)
				slices.Sort(queue)
			}
		}
	}

	if len(ordered) != len(stages) {
		return nil, domain.ErrDependencyCycle
	}

	return ordered, nil
}

func dependencyFailed(dependencies []string, stageIndex map[string]domain.StageReport) bool {
	for _, dependency := range dependencies {
		report, exists := stageIndex[dependency]
		if !exists {
			return true
		}
		if !report.Passed {
			return true
		}
	}

	return false
}
