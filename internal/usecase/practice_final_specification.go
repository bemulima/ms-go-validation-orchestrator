package usecase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

// The execution form has no milestone identity. V1 validates authored milestone
// identities before conversion; V2 admits one full FINAL contract without one.
type practiceExecutableContract struct {
	Required           bool
	ValidationContract json.RawMessage
	EvaluationAssets   *practiceEvaluationAssetsV1
}

type practiceExecutionSpecification struct {
	Schema         string
	RuntimeProfile json.RawMessage
	Contracts      []practiceExecutableContract
}

type practiceValidationSpecificationV2 struct {
	Schema         string                             `json:"schema"`
	RuntimeProfile json.RawMessage                    `json:"runtime_profile"`
	FinalContract  *practiceValidationFinalContractV2 `json:"final_contract"`
}

type practiceValidationFinalContractV2 struct {
	Required           *bool                       `json:"required"`
	ValidationContract json.RawMessage             `json:"validation_contract"`
	EvaluationAssets   *practiceEvaluationAssetsV1 `json:"evaluation_assets"`
}

func decodePracticeExecutionSpecification(request domain.PracticeValidationRequestV2) (practiceExecutionSpecification, error) {
	var header struct {
		Schema string `json:"schema"`
	}
	if json.Unmarshal(request.ValidationSpecification, &header) == nil && header.Schema == domain.PracticeValidationSpecificationSchemaV2 {
		return decodePracticeFinalSpecification(request)
	}

	// Keep the frozen V1 parser and validation rules, including unsupported
	// version and malformed-input classifications, for every existing caller.
	specification, err := decodePracticeValidationSpecification(request.ValidationSpecification)
	if err != nil {
		return practiceExecutionSpecification{}, err
	}
	execution := practiceExecutionSpecification{Schema: specification.Schema, RuntimeProfile: specification.RuntimeProfile}
	if specification.Schema != domain.PracticeValidationSpecificationSchemaV1 {
		return execution, nil
	}
	contracts, err := validatePracticeSpecification(request, specification)
	if err != nil {
		return practiceExecutionSpecification{}, err
	}
	for _, contract := range contracts {
		execution.Contracts = append(execution.Contracts, practiceExecutableContract{
			Required: *contract.Required, ValidationContract: contract.ValidationContract, EvaluationAssets: contract.EvaluationAssets,
		})
	}
	return execution, nil
}

func decodePracticeFinalSpecification(request domain.PracticeValidationRequestV2) (practiceExecutionSpecification, error) {
	if err := validatePracticeV2ObjectFields(request.ValidationSpecification); err != nil {
		return practiceExecutionSpecification{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(request.ValidationSpecification))
	decoder.DisallowUnknownFields()
	var specification practiceValidationSpecificationV2
	if err := decoder.Decode(&specification); err != nil {
		return practiceExecutionSpecification{}, fmt.Errorf("%w: invalid validation_specification fields", domain.ErrInvalidRequest)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return practiceExecutionSpecification{}, fmt.Errorf("%w: validation_specification must be one JSON object", domain.ErrInvalidRequest)
	}
	if request.Scope != domain.PracticeValidationScopeFinal {
		return practiceExecutionSpecification{}, fmt.Errorf("%w: specification v2 requires FINAL scope", domain.ErrInvalidRequest)
	}
	profile, err := decodeCanonicalJSON(specification.RuntimeProfile)
	if err != nil {
		return practiceExecutionSpecification{}, fmt.Errorf("%w: runtime_profile must be a JSON object", domain.ErrInvalidRequest)
	}
	if _, ok := profile.(map[string]any); !ok {
		return practiceExecutionSpecification{}, fmt.Errorf("%w: runtime_profile must be a JSON object", domain.ErrInvalidRequest)
	}
	contract := specification.FinalContract
	if contract == nil || contract.Required == nil || !*contract.Required || contract.EvaluationAssets == nil {
		return practiceExecutionSpecification{}, fmt.Errorf("%w: FINAL requires one required executable contract and evaluation_assets", domain.ErrInvalidContract)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(contract.ValidationContract, &object); err != nil || object == nil {
		return practiceExecutionSpecification{}, fmt.Errorf("%w: validation_contract must be a JSON object", domain.ErrInvalidContract)
	}
	return practiceExecutionSpecification{
		Schema: specification.Schema, RuntimeProfile: specification.RuntimeProfile,
		Contracts: []practiceExecutableContract{{
			Required: true, ValidationContract: contract.ValidationContract, EvaluationAssets: contract.EvaluationAssets,
		}},
	}, nil
}

// The V2 envelope has exact field names. encoding/json's struct decoder alone
// also accepts case aliases and repeated members, while the canonical digest
// treats case-distinct object keys separately. Check only owner-owned objects;
// runtime_profile and engine rules/checks retain their opaque JSON contract.
func validatePracticeV2ObjectFields(raw json.RawMessage) error {
	fields, err := decodePracticeV2ExactObject(raw, []string{"schema", "runtime_profile", "final_contract"})
	if err != nil {
		return err
	}
	final := fields["final_contract"]
	if len(final) == 0 || bytes.Equal(bytes.TrimSpace(final), []byte("null")) {
		return nil // Missing immutable values retain AUTHORING_DEFECT below.
	}
	fields, err = decodePracticeV2ExactObject(final, []string{"required", "validation_contract", "evaluation_assets"})
	if err != nil {
		return err
	}
	assets := fields["evaluation_assets"]
	if len(assets) == 0 || bytes.Equal(bytes.TrimSpace(assets), []byte("null")) {
		return nil
	}
	_, err = decodePracticeV2ExactObject(assets, []string{"test_suite_ref", "reference_solution_ref", "negative_fixture_refs", "verification_runner_ref"})
	return err
}

func decodePracticeV2ExactObject(raw json.RawMessage, allowed []string) (map[string]json.RawMessage, error) {
	invalid := func() (map[string]json.RawMessage, error) {
		return nil, fmt.Errorf("%w: invalid or ambiguous specification v2 fields", domain.ErrInvalidRequest)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	open, err := decoder.Token()
	if err != nil || open != json.Delim('{') {
		return invalid()
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return invalid()
		}
		known := false
		for _, field := range allowed {
			known = known || key == field
		}
		if _, duplicate := fields[key]; duplicate || !known {
			return invalid()
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return invalid()
		}
		fields[key] = value
	}
	close, err := decoder.Token()
	if err != nil || close != json.Delim('}') {
		return invalid()
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return invalid()
	}
	return fields, nil
}
