package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestExampleContractsAreWellFormed(t *testing.T) {
	t.Parallel()

	examplesDir := filepath.Join("..", "..", "docs", "examples")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("read examples dir: %v", err)
	}

	seenJSON := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		seenJSON++
		path := filepath.Join(examplesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read example %s: %v", entry.Name(), err)
		}

		var contract domain.ValidationContract
		if err := json.Unmarshal(data, &contract); err != nil {
			t.Fatalf("unmarshal example %s: %v", entry.Name(), err)
		}

		if !contract.IsNewFormat() {
			t.Fatalf("example %s is not recognized as ValidationContractV1", entry.Name())
		}

		validateContractInvariants(t, entry.Name(), contract)
	}

	if seenJSON == 0 {
		t.Fatalf("expected at least one example contract")
	}
}

func validateContractInvariants(t *testing.T, name string, contract domain.ValidationContract) {
	t.Helper()

	stageIDs := make(map[string]struct{}, len(contract.Stages))
	for _, stage := range contract.Stages {
		if stage.ID == "" {
			t.Fatalf("example %s contains stage without id", name)
		}
		if stage.Engine == "" {
			t.Fatalf("example %s contains stage without engine", name)
		}
		if _, exists := stageIDs[stage.ID]; exists {
			t.Fatalf("example %s contains duplicate stage id %q", name, stage.ID)
		}
		stageIDs[stage.ID] = struct{}{}
	}

	for _, stage := range contract.Stages {
		for _, dependency := range stage.DependsOn {
			if _, exists := stageIDs[dependency]; !exists {
				t.Fatalf("example %s stage %q depends on unknown stage %q", name, stage.ID, dependency)
			}
		}
	}

	linkIDs := make(map[string]struct{}, len(contract.Links))
	for _, link := range contract.Links {
		if link.ID == "" {
			t.Fatalf("example %s contains link without id", name)
		}
		if link.Kind == "" {
			t.Fatalf("example %s contains link without kind", name)
		}
		if _, exists := linkIDs[link.ID]; exists {
			t.Fatalf("example %s contains duplicate link id %q", name, link.ID)
		}
		linkIDs[link.ID] = struct{}{}

		for _, dependency := range link.DependsOn {
			if _, exists := stageIDs[dependency]; !exists {
				t.Fatalf("example %s link %q depends on unknown stage %q", name, link.ID, dependency)
			}
		}
	}
}
