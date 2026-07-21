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

func TestTypeScriptRuntimeExampleUsesLiveStaticAndFinalRuntime(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "ts-cli-runtime.json"))
	if err != nil {
		t.Fatalf("read TypeScript runtime example: %v", err)
	}

	var contract domain.ValidationContract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("unmarshal TypeScript runtime example: %v", err)
	}
	if len(contract.Stages) != 2 {
		t.Fatalf("expected two TypeScript stages, got %d", len(contract.Stages))
	}

	staticStage := contract.Stages[0]
	runtimeStage := contract.Stages[1]
	if staticStage.Engine != "ts.ast" || staticStage.Mode != domain.ValidationModeBoth {
		t.Fatalf("expected ts.ast in both mode, got %+v", staticStage)
	}
	if runtimeStage.Engine != "ts.runtime" || runtimeStage.Mode != domain.ValidationModeFinal {
		t.Fatalf("expected ts.runtime in final mode, got %+v", runtimeStage)
	}
	if len(runtimeStage.DependsOn) != 1 || runtimeStage.DependsOn[0] != staticStage.ID {
		t.Fatalf("expected runtime to depend on static stage, got %+v", runtimeStage.DependsOn)
	}
	if !strings.Contains(string(runtimeStage.Checks), `"kind": "cli"`) ||
		!strings.Contains(string(runtimeStage.Checks), `"stdoutEquals"`) {
		t.Fatalf("expected constrained CLI behavioral checks, got %s", runtimeStage.Checks)
	}
}

func TestJavaRuntimeExampleUsesLiveCompileAndFinalRuntime(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "java-cli-runtime.json"))
	if err != nil {
		t.Fatalf("read Java runtime example: %v", err)
	}

	var contract domain.ValidationContract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("unmarshal Java runtime example: %v", err)
	}
	if len(contract.Stages) != 2 {
		t.Fatalf("expected two Java stages, got %d", len(contract.Stages))
	}

	compileStage := contract.Stages[0]
	runtimeStage := contract.Stages[1]
	if compileStage.Engine != "java.compile" || compileStage.Mode != domain.ValidationModeBoth {
		t.Fatalf("expected java.compile in both mode, got %+v", compileStage)
	}
	if runtimeStage.Engine != "java.runtime" || runtimeStage.Mode != domain.ValidationModeFinal {
		t.Fatalf("expected java.runtime in final mode, got %+v", runtimeStage)
	}
	if compileStage.Targets.Entrypoint != "Main.java" || runtimeStage.Targets.Entrypoint != "Main.java" {
		t.Fatalf("expected Main.java entrypoints, got compile=%+v runtime=%+v", compileStage.Targets, runtimeStage.Targets)
	}
	if len(runtimeStage.DependsOn) != 1 || runtimeStage.DependsOn[0] != compileStage.ID {
		t.Fatalf("expected runtime to depend on compile stage, got %+v", runtimeStage.DependsOn)
	}
	if !strings.Contains(string(runtimeStage.Checks), `"kind": "cli"`) ||
		!strings.Contains(string(runtimeStage.Checks), `"stdoutEquals"`) {
		t.Fatalf("expected constrained Java CLI behavioral checks, got %s", runtimeStage.Checks)
	}
}

func TestKotlinRuntimeExampleUsesLiveCompileAndFinalRuntime(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "kotlin-cli-runtime.json"))
	if err != nil {
		t.Fatalf("read Kotlin runtime example: %v", err)
	}

	var contract domain.ValidationContract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("unmarshal Kotlin runtime example: %v", err)
	}
	if len(contract.Stages) != 2 {
		t.Fatalf("expected two Kotlin stages, got %d", len(contract.Stages))
	}

	compileStage := contract.Stages[0]
	runtimeStage := contract.Stages[1]
	if compileStage.Engine != "kotlin.compile" || compileStage.Mode != domain.ValidationModeBoth {
		t.Fatalf("expected kotlin.compile in both mode, got %+v", compileStage)
	}
	if runtimeStage.Engine != "kotlin.runtime" || runtimeStage.Mode != domain.ValidationModeFinal {
		t.Fatalf("expected kotlin.runtime in final mode, got %+v", runtimeStage)
	}
	if compileStage.Targets.Entrypoint != "Main.kt" || runtimeStage.Targets.Entrypoint != "Main.kt" {
		t.Fatalf("expected Main.kt entrypoints, got compile=%+v runtime=%+v", compileStage.Targets, runtimeStage.Targets)
	}
	if len(runtimeStage.DependsOn) != 1 || runtimeStage.DependsOn[0] != compileStage.ID {
		t.Fatalf("expected runtime to depend on compile stage, got %+v", runtimeStage.DependsOn)
	}
	if !strings.Contains(string(runtimeStage.Checks), `"kind": "cli"`) ||
		!strings.Contains(string(runtimeStage.Checks), `"stdoutEquals"`) {
		t.Fatalf("expected constrained Kotlin CLI behavioral checks, got %s", runtimeStage.Checks)
	}
}

func TestPHPExampleUsesLiveAndFinalStaticValidation(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "php-single-file.json"))
	if err != nil {
		t.Fatalf("read PHP example: %v", err)
	}

	var contract domain.ValidationContract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("unmarshal PHP example: %v", err)
	}
	if len(contract.Stages) != 1 {
		t.Fatalf("expected one PHP stage, got %d", len(contract.Stages))
	}

	stage := contract.Stages[0]
	if stage.Engine != "php.core" || stage.Mode != domain.ValidationModeBoth {
		t.Fatalf("expected php.core in both mode, got %+v", stage)
	}
	if len(stage.Targets.Files) != 1 || stage.Targets.Files[0] != "index.php" {
		t.Fatalf("expected single index.php target, got %+v", stage.Targets)
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
