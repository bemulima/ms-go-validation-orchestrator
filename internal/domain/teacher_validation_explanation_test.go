package domain

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

func TestTeacherValidationIssueJSONHasOnlySafeAllowlistedFields(t *testing.T) {
	payload, err := json.Marshal(TeacherValidationIssueV1{
		Code: "REQUIRED_FILE", Message: "Create the required file.", Hint: "Add the file.",
		File: "src/main.go", Line: 4, Column: 2, StageID: "compile", Engine: "go.core", Severity: "error",
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	want := []string{"code", "column", "engine", "file", "hint", "line", "message", "severity", "stage_id"}
	got := make([]string, 0, len(fields))
	for field := range fields {
		got = append(got, field)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unsafe teacher issue fields: got=%v want=%v json=%s", got, want, payload)
	}
}

func TestValidationResultWithoutTeacherProjectionPreservesLegacyJSON(t *testing.T) {
	payload, err := json.Marshal(ValidationResult{
		ContractKind: "workspace_contract", ContractVersion: 1, Passed: true, Stages: []StageReport{},
	})
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"contract_kind":"workspace_contract","contract_version":1,"legacy":false,"passed":true,"stages":[]}`
	if string(payload) != want {
		t.Fatalf("legacy response changed: got=%s want=%s", payload, want)
	}
}
