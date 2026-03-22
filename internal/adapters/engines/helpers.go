package engines

import (
	"encoding/json"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

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
