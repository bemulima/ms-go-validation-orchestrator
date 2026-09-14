package usecase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type ContractParser struct {
	legacyAdapter LegacyContractAdapter
}

func NewContractParser(legacyAdapter LegacyContractAdapter) ContractParser {
	return ContractParser{legacyAdapter: legacyAdapter}
}

func (parser ContractParser) Parse(request domain.ValidationRequest) (domain.ValidationContract, bool, error) {
	if len(request.CodeStructure) == 0 {
		return domain.ValidationContract{}, false, fmt.Errorf("%w: code_structure is required", domain.ErrInvalidRequest)
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(request.CodeStructure, &envelope); err != nil {
		return domain.ValidationContract{}, false, fmt.Errorf("%w: parse contract: %w", domain.ErrInvalidRequest, err)
	}

	if looksLikeV1Contract(envelope) {
		contract, err := decodeStrictV1Contract(request.CodeStructure)
		if err != nil {
			return domain.ValidationContract{}, false, err
		}
		return contract, false, nil
	}

	legacyContract, err := parser.legacyAdapter.Adapt(request)
	if err != nil {
		return domain.ValidationContract{}, false, err
	}

	return legacyContract, true, nil
}

var contractIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

func looksLikeV1Contract(envelope map[string]json.RawMessage) bool {
	for _, field := range []string{"version", "kind", "stages", "profiles", "workspace", "links"} {
		if _, ok := envelope[field]; ok {
			return true
		}
	}
	return false
}

func decodeStrictV1Contract(raw json.RawMessage) (domain.ValidationContract, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var contract domain.ValidationContract
	if err := decoder.Decode(&contract); err != nil {
		return domain.ValidationContract{}, fmt.Errorf("%w: decode v1 contract: %v", domain.ErrInvalidContract, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return domain.ValidationContract{}, fmt.Errorf("%w: contract must contain exactly one JSON object", domain.ErrInvalidContract)
	}
	if err := validateV1Contract(contract); err != nil {
		return domain.ValidationContract{}, err
	}
	return contract, nil
}

func validateV1Contract(contract domain.ValidationContract) error {
	if contract.Version != 1 {
		return invalidContractf("version must be 1")
	}
	if contract.Kind != "workspace_contract" {
		return invalidContractf("kind must be workspace_contract")
	}
	if len(contract.Stages) == 0 || len(contract.Stages) > 100 {
		return invalidContractf("stages must contain between 1 and 100 items")
	}
	if err := validateWorkspacePaths(contract.Workspace.RequiredFiles, "workspace.required_files"); err != nil {
		return err
	}

	stageIDs := make(map[string]struct{}, len(contract.Stages))
	for _, stage := range contract.Stages {
		if !contractIDPattern.MatchString(stage.ID) || !contractIDPattern.MatchString(stage.Engine) {
			return invalidContractf("stage id and engine must be stable identifiers")
		}
		if _, duplicate := stageIDs[stage.ID]; duplicate {
			return invalidContractf("duplicate stage id %q", stage.ID)
		}
		stageIDs[stage.ID] = struct{}{}
		if stage.Mode != "" && stage.Mode != domain.ValidationModeLive && stage.Mode != domain.ValidationModeFinal && stage.Mode != domain.ValidationModeBoth {
			return invalidContractf("stage %q has unsupported mode", stage.ID)
		}
		if stage.TimeoutSeconds < 0 || stage.TimeoutSeconds > 300 {
			return invalidContractf("stage %q timeout_seconds is out of range", stage.ID)
		}
		if err := validateUniqueIDs(stage.DependsOn, "stage dependencies"); err != nil {
			return err
		}
		if err := validateWorkspacePaths(stage.Targets.Files, "stage targets.files"); err != nil {
			return err
		}
		if stage.Targets.Entrypoint != "" {
			if err := domain.ValidateWorkspaceFilePath(stage.Targets.Entrypoint); err != nil {
				return invalidContractf("stage %q has invalid entrypoint", stage.ID)
			}
		}
		if err := validateOptionalJSONObject(stage.Rules, "stage rules"); err != nil {
			return err
		}
		if err := validateOptionalJSONObject(stage.Checks, "stage checks"); err != nil {
			return err
		}
	}
	if _, err := orderStages(contract.Stages); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidContract, err)
	}

	linkIDs := make(map[string]struct{}, len(contract.Links))
	for _, link := range contract.Links {
		if !contractIDPattern.MatchString(link.ID) {
			return invalidContractf("link id must be a stable identifier")
		}
		if _, duplicate := linkIDs[link.ID]; duplicate {
			return invalidContractf("duplicate link id %q", link.ID)
		}
		linkIDs[link.ID] = struct{}{}
		if link.Kind != "workspace.file_contains" && link.Kind != "workspace.selector_exists" {
			return invalidContractf("link %q has unsupported kind", link.ID)
		}
		if err := validateUniqueIDs(link.DependsOn, "link dependencies"); err != nil {
			return err
		}
		for _, dependency := range link.DependsOn {
			if _, ok := stageIDs[dependency]; !ok {
				return invalidContractf("link %q references unknown dependency %q", link.ID, dependency)
			}
		}
		if err := validateLinkConfig(link); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkspacePaths(values []string, fieldName string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := domain.ValidateWorkspaceFilePath(value); err != nil {
			return invalidContractf("%s contains invalid path %q", fieldName, value)
		}
		if _, duplicate := seen[value]; duplicate {
			return invalidContractf("%s contains duplicate path %q", fieldName, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateUniqueIDs(values []string, fieldName string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !contractIDPattern.MatchString(value) {
			return invalidContractf("%s contains invalid id %q", fieldName, value)
		}
		if _, duplicate := seen[value]; duplicate {
			return invalidContractf("%s contains duplicate id %q", fieldName, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateOptionalJSONObject(raw json.RawMessage, fieldName string) error {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return invalidContractf("%s must be a JSON object", fieldName)
	}
	return nil
}

func validateLinkConfig(link domain.ValidationLink) error {
	switch link.Kind {
	case "workspace.file_contains":
		var config fileContainsConfig
		if err := decodeStrictJSONObject(link.Config, &config); err != nil {
			return invalidContractf("link %q has invalid config: %v", link.ID, err)
		}
		if err := domain.ValidateWorkspaceFilePath(config.File); err != nil || config.Needle == "" {
			return invalidContractf("link %q requires a safe file and non-empty needle", link.ID)
		}
	case "workspace.selector_exists":
		var config selectorExistsConfig
		if err := decodeStrictJSONObject(link.Config, &config); err != nil {
			return invalidContractf("link %q has invalid config: %v", link.ID, err)
		}
		if strings.TrimSpace(config.Selector) == "" {
			return invalidContractf("link %q requires a non-empty selector", link.ID)
		}
		if config.File != "" {
			if err := domain.ValidateWorkspaceFilePath(config.File); err != nil {
				return invalidContractf("link %q has invalid file", link.ID)
			}
		}
	}
	return nil
}

func decodeStrictJSONObject(raw json.RawMessage, output any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("config must contain exactly one JSON object")
	}
	return nil
}

func invalidContractf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalidContract, fmt.Sprintf(format, args...))
}
