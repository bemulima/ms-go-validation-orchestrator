package mapper

import (
	"testing"

	"github.com/example/ms-validation-orchestrator-service/dto"
)

func TestToDomainValidationRequestAcceptsSandboxWorkspaceRoot(t *testing.T) {
	request, err := ToDomainValidationRequest(dto.ValidateRequest{
		CodeStructure: map[string]any{"rules": map[string]any{}},
		Workspace:     dto.WorkspaceInput{RootPath: "/workspaces/7de8aa06-c42f-41ae-9ae0-e39f28522ce5"},
	})
	if err != nil {
		t.Fatalf("map request: %v", err)
	}
	if request.Workspace.RootPath != "/workspaces/7de8aa06-c42f-41ae-9ae0-e39f28522ce5" {
		t.Fatalf("unexpected root path %q", request.Workspace.RootPath)
	}
}

func TestToDomainValidationRequestRejectsHostWorkspaceRoot(t *testing.T) {
	_, err := ToDomainValidationRequest(dto.ValidateRequest{
		CodeStructure: map[string]any{"rules": map[string]any{}},
		Workspace:     dto.WorkspaceInput{RootPath: "/tmp/student-workspace"},
	})
	if err == nil {
		t.Fatal("expected host workspace root to be rejected")
	}
}

func TestToDomainValidationRequestRejectsWorkspaceTraversal(t *testing.T) {
	_, err := ToDomainValidationRequest(dto.ValidateRequest{
		CodeStructure: map[string]any{"rules": map[string]any{}},
		Workspace:     dto.WorkspaceInput{RootPath: "/workspaces/sandbox-a/../sandbox-b"},
	})
	if err == nil {
		t.Fatal("expected traversing workspace root to be rejected")
	}
}
