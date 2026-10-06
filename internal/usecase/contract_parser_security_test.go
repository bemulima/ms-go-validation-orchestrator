package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestContractParserRejectsUnknownV1Field(t *testing.T) {
	parser := NewContractParser(NewDefaultLegacyContractAdapter())
	_, _, err := parser.Parse(domain.ValidationRequest{
		CodeStructure: json.RawMessage(`{
			"version":1,
			"kind":"workspace_contract",
			"stages":[{"id":"go","engine":"go.core"}],
			"unexpected":true
		}`),
	})
	if !errors.Is(err, domain.ErrInvalidContract) {
		t.Fatalf("expected invalid contract, got %v", err)
	}
}

func TestContractParserRejectsIncompleteV1InsteadOfAdaptingAsLegacy(t *testing.T) {
	parser := NewContractParser(NewDefaultLegacyContractAdapter())
	_, legacy, err := parser.Parse(domain.ValidationRequest{
		CodeStructure: json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[]}`),
	})
	if !errors.Is(err, domain.ErrInvalidContract) {
		t.Fatalf("expected invalid contract, got %v", err)
	}
	if legacy {
		t.Fatal("incomplete v1 contract must not fall back to legacy")
	}
}

func TestContractParserRejectsUnsafeRequiredFile(t *testing.T) {
	parser := NewContractParser(NewDefaultLegacyContractAdapter())
	_, _, err := parser.Parse(domain.ValidationRequest{
		CodeStructure: json.RawMessage(`{
			"version":1,
			"kind":"workspace_contract",
			"workspace":{"required_files":["../solution.go"]},
			"stages":[{"id":"go","engine":"go.core"}]
		}`),
	})
	if !errors.Is(err, domain.ErrInvalidContract) {
		t.Fatalf("expected invalid contract, got %v", err)
	}
}

func TestExecuteRejectsMissingRequiredFileInInlineWorkspace(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(
		NewContractParser(NewDefaultLegacyContractAdapter()),
		[]domain.EngineClient{capabilityTestEngine{id: "go.core"}},
	)
	_, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		CodeStructure: json.RawMessage(`{
			"version":1,
			"kind":"workspace_contract",
			"workspace":{"required_files":["go.mod","main.go"]},
			"stages":[{"id":"go","engine":"go.core"}]
		}`),
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "main.go", Content: "package main"}},
		},
	})
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestContractParserAcceptsOpaqueRulesContainersWithoutChangingBytes(t *testing.T) {
	parser := NewContractParser(NewDefaultLegacyContractAdapter())
	for _, test := range []struct {
		name, engine, rules string
	}{
		{"React array", "react.ast", "[\n {\"kind\":\"component_exists\",\"name\":\"App\"},\n {\"kind\":\"jsx_tree\",\"selector\":\"button\",\"required\":true,\"attributesIncludes\":{\"id\":\"save-button\"}}\n]"},
		{"other engine object", "go.core", `{ "provider_owned" : {"nested":[true,1,"value"]} }`},
		{"generic opaque array", "external.engine", `[{"provider_owned":{"nested":[1,true]}},"member schema belongs to provider"]`},
		{"empty array", "react.ast", `[]`},
		{"empty object", "go.core", `{}`},
		{"null", "react.ast", `null`},
		{"missing", "react.ast", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			stage := `{"id":"check","engine":"` + test.engine + `","checks":{"provider_owned":true}`
			if test.rules != "" {
				stage += `,"rules":` + test.rules
			}
			contract, legacy, err := parser.Parse(domain.ValidationRequest{CodeStructure: json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[` + stage + `}]}`)})
			if err != nil || legacy {
				t.Fatalf("opaque container rejected or adapted: %v, legacy=%t", err, legacy)
			}
			if string(contract.Stages[0].Rules) != test.rules || string(contract.Stages[0].Checks) != `{"provider_owned":true}` {
				t.Fatal("engine-owned raw JSON was transformed")
			}
		})
	}
}

func TestContractParserAcceptsCheckedInReactRulesExample(t *testing.T) {
	raw, err := os.ReadFile("../../docs/examples/react-component.json")
	if err != nil {
		t.Fatal(err)
	}
	contract, legacy, err := NewContractParser(NewDefaultLegacyContractAdapter()).Parse(domain.ValidationRequest{CodeStructure: raw})
	if err != nil || legacy || len(contract.Stages) != 1 || contract.Stages[0].Engine != "react.ast" {
		t.Fatalf("checked-in React contract rejected or adapted: %v, legacy=%t", err, legacy)
	}
}

func TestContractParserKeepsScalarRulesAndNonObjectChecksInvalid(t *testing.T) {
	parser := NewContractParser(NewDefaultLegacyContractAdapter())
	for _, field := range []string{"rules", "checks"} {
		values := []string{`"opaque string"`, `42`, `true`, `false`}
		if field == "checks" {
			values = append(values, `[]`, `[{}]`)
		}
		for _, value := range values {
			t.Run(field+"/"+value, func(t *testing.T) {
				_, legacy, err := parser.Parse(domain.ValidationRequest{CodeStructure: json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"react.ast","` + field + `":` + value + `}]}`)})
				if !errors.Is(err, domain.ErrInvalidContract) || legacy {
					t.Fatalf("invalid %s shape accepted or adapted: %v", field, err)
				}
			})
		}
	}
}

func TestContractParserPreservesOptionalObjectChecks(t *testing.T) {
	for _, checks := range []string{"", `null`, `{ "provider_owned" : [true,1] }`} {
		t.Run(checks, func(t *testing.T) {
			stage := `{"id":"check","engine":"react.ast","rules":[]`
			if checks != "" {
				stage += `,"checks":` + checks
			}
			contract, legacy, err := NewContractParser(NewDefaultLegacyContractAdapter()).Parse(domain.ValidationRequest{CodeStructure: json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[` + stage + `}]}`)})
			if err != nil || legacy || string(contract.Stages[0].Checks) != checks {
				t.Fatalf("optional object Checks behavior changed: %v, legacy=%t", err, legacy)
			}
		})
	}
}

func TestContractParserKeepsMalformedAndTrailingJSONInvalid(t *testing.T) {
	for _, raw := range []string{
		`{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"react.ast","rules":[{"kind":"component_exists"},]}]}`,
		`{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"react.ast","rules":[]}]} {}`,
	} {
		_, legacy, err := NewContractParser(NewDefaultLegacyContractAdapter()).Parse(domain.ValidationRequest{CodeStructure: json.RawMessage(raw)})
		if !errors.Is(err, domain.ErrInvalidRequest) || legacy {
			t.Fatalf("malformed/trailing JSON accepted or adapted: %v", err)
		}
	}
}

func TestContractParserRulesArraysDoNotRelaxCoreOrLinkChecks(t *testing.T) {
	for _, test := range []struct{ name, raw string }{
		{"unknown stage field", `{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"react.ast","rules":[],"unknown":true}]}`},
		{"unsafe target", `{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"react.ast","rules":[],"targets":{"files":["../App.tsx"]}}]}`},
		{"invalid mode", `{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"react.ast","rules":[],"mode":"unexpected"}]}`},
		{"duplicate stages", `{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"react.ast","rules":[]},{"id":"check","engine":"react.ast","rules":[]}]}`},
		{"unknown dependency", `{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"react.ast","rules":[],"depends_on":["missing"]}]}`},
		{"link config unknown field", `{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"react.ast","rules":[]}],"links":[{"id":"link","kind":"workspace.file_contains","config":{"file":"src/App.tsx","needle":"App","unknown":true}}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, legacy, err := NewContractParser(NewDefaultLegacyContractAdapter()).Parse(domain.ValidationRequest{CodeStructure: json.RawMessage(test.raw)})
			if !errors.Is(err, domain.ErrInvalidContract) || legacy {
				t.Fatalf("strict core/link validation changed: %v", err)
			}
		})
	}
}
