package engines

import (
	"context"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestCSSClientUsesStylesValidatorHTTPContract(t *testing.T) {
	t.Parallel()

	httpClient := &fakeNodeHTTPClient{
		response: []byte(`{"isValid":true,"errors":[]}`),
	}
	client := NewCSSClient("http://css-validator", httpClient)

	result, err := client.Validate(context.Background(), domain.EngineValidationInput{
		Stage: domain.ValidationStage{
			ID:       "css",
			Engine:   "css.ast",
			Language: "css",
			Targets: domain.StageTargets{
				Files: []string{"styles.css"},
			},
		},
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{
				{Path: "styles.css", Content: ".catalog { display: grid; }"},
			},
		},
	})
	if err != nil {
		t.Fatalf("validate CSS: %v", err)
	}
	if !result.Passed {
		t.Fatalf("expected CSS validation to pass")
	}
	if httpClient.lastURL != "http://css-validator/validate" {
		t.Fatalf("unexpected CSS validator URL %q", httpClient.lastURL)
	}
}
