package usecase

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

const practiceValidationContractDigestSchemaV1 = "practice-validation-contract-digest.v1"

// ComputePracticeValidationContractDigest implements the canonical digest
// preimage from the frozen Practice Runtime V2 contract.
func ComputePracticeValidationContractDigest(scope, milestoneID string, specification json.RawMessage) (string, error) {
	decoded, err := decodeCanonicalJSON(specification)
	if err != nil {
		return "", fmt.Errorf("decode validation specification: %w", err)
	}
	preimage := map[string]any{
		"schema":                   practiceValidationContractDigestSchemaV1,
		"scope":                    scope,
		"validation_specification": decoded,
	}
	if scope == domain.PracticeValidationScopeMilestone {
		preimage["milestone_id"] = milestoneID
	}
	canonical, err := marshalCanonicalJSON(preimage)
	if err != nil {
		return "", fmt.Errorf("marshal validation contract digest preimage: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func decodeCanonicalJSON(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}

func marshalCanonicalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}
