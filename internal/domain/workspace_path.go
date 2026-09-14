package domain

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

const SandboxWorkspaceRoot = "/workspaces"

var sandboxIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ValidateSandboxWorkspaceRoot accepts only the portable validator namespace
// /workspaces/<sandbox-id>. Host paths and nested/traversing paths are never
// valid service inputs.
func ValidateSandboxWorkspaceRoot(value string) error {
	if value == "" {
		return nil
	}
	if strings.TrimSpace(value) != value || path.Clean(value) != value {
		return fmt.Errorf("%w: workspace.root_path is not canonical", ErrInvalidRequest)
	}
	prefix := SandboxWorkspaceRoot + "/"
	if !strings.HasPrefix(value, prefix) {
		return fmt.Errorf("%w: workspace.root_path must use %s/<sandbox-id>", ErrInvalidRequest, SandboxWorkspaceRoot)
	}
	sandboxID := strings.TrimPrefix(value, prefix)
	if strings.Contains(sandboxID, "/") || !sandboxIDPattern.MatchString(sandboxID) {
		return fmt.Errorf("%w: workspace.root_path has an invalid sandbox id", ErrInvalidRequest)
	}
	return nil
}

// ValidateWorkspaceFilePath validates a portable workspace-relative path.
func ValidateWorkspaceFilePath(value string) error {
	if value == "" || strings.TrimSpace(value) != value || strings.Contains(value, "\\") || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%w: invalid workspace file path %q", ErrInvalidRequest, value)
	}
	if strings.HasPrefix(value, "/") || value == "." || path.Clean(value) != value || strings.HasPrefix(value, "../") {
		return fmt.Errorf("%w: invalid workspace file path %q", ErrInvalidRequest, value)
	}
	return nil
}
