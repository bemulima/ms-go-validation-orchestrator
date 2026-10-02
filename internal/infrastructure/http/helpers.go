package engines

import (
	"encoding/json"
	"fmt"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func validateValidatorResponseEnvelope(body []byte, acceptedFields ...string) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidatorProtocol, err)
	}
	if envelope == nil {
		return fmt.Errorf("%w: response must be a JSON object", domain.ErrValidatorProtocol)
	}
	for _, field := range acceptedFields {
		if _, exists := envelope[field]; exists {
			return nil
		}
	}
	return fmt.Errorf("%w: response contains no recognized result fields", domain.ErrValidatorProtocol)
}

func validateValidatorBooleanFields(body []byte, fields ...string) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidatorProtocol, err)
	}
	for _, field := range fields {
		raw, exists := envelope[field]
		if !exists {
			continue
		}
		var value bool
		if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &value) != nil {
			return fmt.Errorf("%w: %q must be a boolean", domain.ErrValidatorProtocol, field)
		}
	}
	return nil
}

func validateValidatorRequiredBooleanFields(body []byte, fields ...string) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidatorProtocol, err)
	}
	for _, field := range fields {
		raw, exists := envelope[field]
		if !exists {
			return fmt.Errorf("%w: required field %q is missing", domain.ErrValidatorProtocol, field)
		}
		var value bool
		if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &value) != nil {
			return fmt.Errorf("%w: %q must be a boolean", domain.ErrValidatorProtocol, field)
		}
	}
	return nil
}

func validateOptionalValidatorObject(body []byte, field string) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidatorProtocol, err)
	}
	raw, exists := envelope[field]
	if !exists {
		return nil
	}
	var object map[string]json.RawMessage
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &object) != nil || object == nil {
		return fmt.Errorf("%w: %q must be an object", domain.ErrValidatorProtocol, field)
	}
	return nil
}

func validateOptionalNestedBooleanFields(body []byte, objectField string, fields ...string) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidatorProtocol, err)
	}
	rawObject, exists := envelope[objectField]
	if !exists {
		return nil
	}
	var object map[string]json.RawMessage
	if len(rawObject) == 0 || string(rawObject) == "null" || json.Unmarshal(rawObject, &object) != nil || object == nil {
		return fmt.Errorf("%w: %q must be an object", domain.ErrValidatorProtocol, objectField)
	}
	for _, field := range fields {
		raw, exists := object[field]
		if !exists {
			continue
		}
		var value bool
		if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &value) != nil {
			return fmt.Errorf("%w: %q.%q must be a boolean", domain.ErrValidatorProtocol, objectField, field)
		}
	}
	return nil
}

func wrapValidatorProtocolError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", domain.ErrValidatorProtocol, err)
}

func rawJSONOrEmptyObject(value json.RawMessage) any {
	if len(value) == 0 {
		return map[string]any{}
	}

	var result any
	if err := json.Unmarshal(value, &result); err != nil {
		return map[string]any{}
	}

	return result
}

func rawJSONOrEmptyArray(value json.RawMessage) any {
	if len(value) == 0 {
		return []any{}
	}

	var result any
	if err := json.Unmarshal(value, &result); err != nil {
		return []any{}
	}

	return result
}

func rawJSONOrValue(value json.RawMessage) any {
	if len(value) == 0 || string(value) == "null" {
		return nil
	}

	var result any
	if err := json.Unmarshal(value, &result); err != nil {
		return nil
	}

	return result
}

func hasPayload(value json.RawMessage) bool {
	return len(value) > 0 && string(value) != "null"
}

func workspaceFilesAsMaps(files []domain.WorkspaceFile) []map[string]string {
	result := make([]map[string]string, 0, len(files))
	for _, file := range files {
		result = append(result, map[string]string{
			"path":    file.Path,
			"content": file.Content,
		})
	}

	return result
}
