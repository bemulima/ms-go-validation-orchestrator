package usecase

import (
	"encoding/json"
	"strings"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type selectorExistsConfig struct {
	Selector string `json:"selector"`
	File     string `json:"file,omitempty"`
}

type fileContainsConfig struct {
	File    string `json:"file"`
	Needle  string `json:"needle"`
	Message string `json:"message,omitempty"`
}

func executeLinks(
	links []domain.ValidationLink,
	workspace domain.ValidationWorkspace,
	stageIndex map[string]domain.StageReport,
) []domain.LinkReport {
	reports := make([]domain.LinkReport, 0, len(links))

	for _, link := range links {
		if dependencyFailed(link.DependsOn, stageIndex) {
			reports = append(reports, domain.LinkReport{
				LinkID:   link.ID,
				Kind:     link.Kind,
				Status:   "skipped",
				Passed:   link.Optional,
				Optional: link.Optional,
				Errors: []domain.ValidationIssue{
					{
						Code:     "LINK_DEPENDENCY_FAILED",
						Message:  "link skipped because dependency failed",
						Severity: "error",
					},
				},
			})
			continue
		}

		reports = append(reports, evaluateLink(link, workspace))
	}

	return reports
}

func evaluateLink(link domain.ValidationLink, workspace domain.ValidationWorkspace) domain.LinkReport {
	switch link.Kind {
	case "workspace.file_contains":
		return evaluateFileContainsLink(link, workspace)
	case "workspace.selector_exists":
		return evaluateSelectorExistsLink(link, workspace)
	default:
		return domain.LinkReport{
			LinkID:   link.ID,
			Kind:     link.Kind,
			Status:   "failed",
			Passed:   false,
			Optional: link.Optional,
			Errors: []domain.ValidationIssue{
				{
					Code:     "UNSUPPORTED_LINK_KIND",
					Message:  "link kind is not implemented yet",
					Severity: "error",
				},
			},
		}
	}
}

func evaluateFileContainsLink(link domain.ValidationLink, workspace domain.ValidationWorkspace) domain.LinkReport {
	var config fileContainsConfig
	if err := json.Unmarshal(link.Config, &config); err != nil {
		return invalidLinkConfigReport(link)
	}

	for _, file := range workspace.Files {
		if file.Path != config.File {
			continue
		}

		if strings.Contains(file.Content, config.Needle) {
			return domain.LinkReport{
				LinkID:   link.ID,
				Kind:     link.Kind,
				Status:   "passed",
				Passed:   true,
				Optional: link.Optional,
			}
		}

		message := config.Message
		if message == "" {
			message = "required content not found in file"
		}

		return domain.LinkReport{
			LinkID:   link.ID,
			Kind:     link.Kind,
			Status:   "failed",
			Passed:   false,
			Optional: link.Optional,
			Errors: []domain.ValidationIssue{
				{
					Code:     "LINK_FILE_CONTENT_MISSING",
					Message:  message,
					Severity: "error",
					File:     config.File,
				},
			},
		}
	}

	return domain.LinkReport{
		LinkID:   link.ID,
		Kind:     link.Kind,
		Status:   "failed",
		Passed:   false,
		Optional: link.Optional,
		Errors: []domain.ValidationIssue{
			{
				Code:     "LINK_FILE_NOT_FOUND",
				Message:  "target file not found in workspace",
				Severity: "error",
				File:     config.File,
			},
		},
	}
}

func evaluateSelectorExistsLink(link domain.ValidationLink, workspace domain.ValidationWorkspace) domain.LinkReport {
	var config selectorExistsConfig
	if err := json.Unmarshal(link.Config, &config); err != nil {
		return invalidLinkConfigReport(link)
	}

	files := workspace.Files
	if config.File != "" {
		files = filterFilesByPath(workspace.Files, config.File)
	}

	for _, file := range files {
		if strings.Contains(file.Content, config.Selector) {
			return domain.LinkReport{
				LinkID:   link.ID,
				Kind:     link.Kind,
				Status:   "passed",
				Passed:   true,
				Optional: link.Optional,
			}
		}
	}

	return domain.LinkReport{
		LinkID:   link.ID,
		Kind:     link.Kind,
		Status:   "failed",
		Passed:   false,
		Optional: link.Optional,
		Errors: []domain.ValidationIssue{
			{
				Code:     "LINK_SELECTOR_NOT_FOUND",
				Message:  "selector not found in workspace content",
				Severity: "error",
				File:     config.File,
				Selector: config.Selector,
			},
		},
	}
}

func invalidLinkConfigReport(link domain.ValidationLink) domain.LinkReport {
	return domain.LinkReport{
		LinkID:   link.ID,
		Kind:     link.Kind,
		Status:   "failed",
		Passed:   false,
		Optional: link.Optional,
		Errors: []domain.ValidationIssue{
			{
				Code:     "INVALID_LINK_CONFIG",
				Message:  "link config could not be parsed",
				Severity: "error",
			},
		},
	}
}

func filterFilesByPath(files []domain.WorkspaceFile, path string) []domain.WorkspaceFile {
	result := make([]domain.WorkspaceFile, 0, len(files))
	for _, file := range files {
		if file.Path == path {
			result = append(result, file)
		}
	}

	return result
}
