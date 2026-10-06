package engines

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestFileOnlyAdapterVerificationInputDeclarations(t *testing.T) {
	for _, client := range []domain.VerificationFileInput{NewHTMLClient("", HTTPClient{}), NewCSSClient("", HTTPClient{}), NewSCSSClient("", HTTPClient{}), NewReactClient("", HTTPClient{}), NewPHPClient("", HTTPClient{}), NewNodeClient("", nil, "js.ast")} {
		if !client.NeedsWorkspaceFiles(domain.ValidationStage{}) {
			t.Fatalf("file-only adapter %T declared root sufficient", client)
		}
	}
}

func TestFoundationVerificationInputDeclarationsMatchProviderInputs(t *testing.T) {
	for _, engine := range []string{"golang", "go.core", "go.gin", "go.echo", "java.compile", "java.runtime", "kotlin.compile", "kotlin.runtime", "php.laravel", "php.yii2", "php.yii3", "php.symfony", "python.core", "python.django", "nextjs.app", "browser.runtime", "docker.dockerfile", "docker.compose", "db.postgres.schema", "db.mysql.schema", "db.tarantool.schema"} {
		if !NewWorkspaceFoundationClient("", nil, engine).NeedsWorkspaceFiles(domain.ValidationStage{}) {
			t.Fatalf("%s file input lost", engine)
		}
	}
	for _, engine := range []string{"git.core", "http.runtime", "linux.fs", "linux.cli", "linux.runtime", "python.django.runtime", "go.gin.runtime", "go.echo.runtime", "php.laravel.runtime", "php.symfony.runtime", "php.yii2.runtime", "php.yii3.runtime", "cache.redis.runtime", "search.elasticsearch.runtime", "db.tarantool.runtime", "unclassified.external"} {
		if NewWorkspaceFoundationClient("", nil, engine).NeedsWorkspaceFiles(domain.ValidationStage{}) {
			t.Fatalf("%s gained unrelated mount dependency", engine)
		}
	}
	for _, engine := range []string{"cache.redis.config", "search.elasticsearch.mapping", "search.manticore", "search.sphinx"} {
		client := NewWorkspaceFoundationClient("", nil, engine)
		if !client.NeedsWorkspaceFiles(domain.ValidationStage{}) || client.NeedsWorkspaceFiles(domain.ValidationStage{Targets: domain.StageTargets{Files: []string{"config.txt"}}}) {
			t.Fatalf("%s explicit target/root fallback differs", engine)
		}
	}
	for _, engine := range []string{"db.postgres.runtime", "db.mysql.runtime"} {
		client := NewWorkspaceFoundationClient("", nil, engine)
		if client.NeedsWorkspaceFiles(domain.ValidationStage{}) || !client.NeedsWorkspaceFiles(domain.ValidationStage{Checks: json.RawMessage(`{"setupFiles":["schema.sql"]}`)}) {
			t.Fatalf("%s setupFiles input differs", engine)
		}
	}
}

type declarationEngine struct {
	files bool
	calls int
}

func (*declarationEngine) EngineID() string { return "http.runtime" }
func (engine *declarationEngine) NeedsWorkspaceFiles(domain.ValidationStage) bool {
	return engine.files
}
func (engine *declarationEngine) Validate(context.Context, domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	engine.calls++
	return domain.StageExecutionResult{Passed: true}, nil
}

func TestHTTPRuntimeInputDeclarationUsesExactValidateBranch(t *testing.T) {
	for _, test := range []struct {
		name          string
		checks        string
		generic, node bool
		files         bool
	}{
		{"dedicated command", `{"command":["test"]}`, true, true, false},
		{"command presence", `{"command":null}`, true, true, false},
		{"node fallback", `{}`, true, true, true},
		{"node only command", `{"command":["test"]}`, false, true, true},
		{"generic only", `{}`, true, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			generic, node := &declarationEngine{}, &declarationEngine{files: true}
			var g, n domain.EngineClient
			if test.generic {
				g = generic
			}
			if test.node {
				n = node
			}
			client := NewHTTPRuntimeDispatchClient(g, n)
			stage := domain.ValidationStage{Checks: json.RawMessage(test.checks)}
			if client.NeedsWorkspaceFiles(stage) != test.files {
				t.Fatal("input declaration selected wrong provider")
			}
			_, err := client.Validate(context.Background(), domain.EngineValidationInput{Stage: stage})
			if err != nil || (node.calls == 1) != test.files || generic.calls+node.calls != 1 {
				t.Fatal("declaration and Validate diverged")
			}
		})
	}
}

func TestFoundationRejectsMissingStrictRootPrerequisiteBeforeHTTP(t *testing.T) {
	for _, engine := range []string{"git.core", "http.runtime", "python.django.runtime", "go.gin.runtime", "go.echo.runtime", "php.laravel.runtime", "php.symfony.runtime", "php.yii2.runtime", "php.yii3.runtime"} {
		for _, root := range []string{"", " \t\n"} {
			t.Run(engine+"/"+root, func(t *testing.T) {
				poster := &fakeFoundationHTTPClient{response: []byte(`{"ok":false,"errors":[{"code":"WORKSPACE_ROOT_REQUIRED","message":"root required"}]}`)}
				client := NewWorkspaceFoundationClient("http://provider.test", poster, engine)
				_, err := client.Validate(context.Background(), domain.EngineValidationInput{
					Stage:     domain.ValidationStage{ID: "check", Engine: engine},
					Workspace: domain.ValidationWorkspace{RootPath: root, Files: []domain.WorkspaceFile{{Path: "index.txt", Content: "synthetic pinned text"}}},
				})
				if !errors.Is(err, domain.ErrUnsupportedEngine) {
					t.Fatalf("missing execution prerequisite error = %v, want ErrUnsupportedEngine", err)
				}
				if poster.lastURL != "" {
					t.Fatal("unsupported artifact reached provider HTTP")
				}
			})
		}
	}
}

func TestFoundationPreservesRootSuppliedInputAndFilesCapableProviders(t *testing.T) {
	root := "/workspaces/11111111-1111-4111-8111-111111111111"
	for _, test := range []struct {
		engine string
		root   string
	}{
		{"git.core", root}, {"http.runtime", root}, {"python.django.runtime", root},
		{"go.gin.runtime", root}, {"go.echo.runtime", root}, {"php.laravel.runtime", root},
		{"php.symfony.runtime", root}, {"php.yii2.runtime", root}, {"php.yii3.runtime", root},
		{"linux.fs", ""}, {"linux.cli", ""}, {"linux.runtime", ""}, {"browser.runtime", ""},
		{"go.core", ""}, {"cache.redis.runtime", ""}, {"db.postgres.runtime", ""}, {"unclassified.external", ""},
	} {
		t.Run(test.engine, func(t *testing.T) {
			poster := &fakeFoundationHTTPClient{response: []byte(`{"ok":true}`)}
			input := domain.EngineValidationInput{
				Stage:     domain.ValidationStage{ID: "check", Engine: test.engine, Rules: json.RawMessage(`{"authored":true}`), Checks: json.RawMessage(`{"command":["synthetic"]}`)},
				Workspace: domain.ValidationWorkspace{RootPath: test.root, Files: []domain.WorkspaceFile{{Path: "index.txt", Content: "synthetic text"}}},
			}
			result, err := NewWorkspaceFoundationClient("http://provider.test", poster, test.engine).Validate(context.Background(), input)
			if err != nil || !result.Passed || poster.lastURL != "http://provider.test/api/v1/validate" {
				t.Fatalf("existing dispatch changed: passed=%t error=%v", result.Passed, err)
			}
			workspace := poster.lastPayload["workspace"].(map[string]any)
			if workspace["root_path"] != test.root || !reflect.DeepEqual(workspace["files"], workspaceFilesAsMaps(input.Workspace.Files)) {
				t.Fatal("workspace changed during existing dispatch")
			}
			stage := poster.lastPayload["stage"].(map[string]any)
			if !reflect.DeepEqual(stage["rules"], rawJSONOrValue(input.Stage.Rules)) || !reflect.DeepEqual(stage["checks"], rawJSONOrValue(input.Stage.Checks)) {
				t.Fatal("opaque authored rules/checks changed")
			}
		})
	}
}

func TestHTTPRuntimePrerequisiteUsesActualSelectedProvider(t *testing.T) {
	for _, test := range []struct {
		name          string
		checks        string
		generic, node bool
		root          string
		unsupported   bool
		selectedNode  bool
	}{
		{name: "dedicated command", checks: `{"command":["synthetic"]}`, generic: true, node: true, unsupported: true},
		{name: "null command retains dedicated branch", checks: `{"command":null}`, generic: true, node: true, unsupported: true},
		{name: "Node fallback", checks: `{}`, generic: true, node: true, selectedNode: true},
		{name: "Node only command", checks: `{"command":["synthetic"]}`, node: true, selectedNode: true},
		{name: "dedicated only", checks: `{}`, generic: true, unsupported: true},
		{name: "root supplied dedicated", checks: `{"command":["synthetic"]}`, generic: true, node: true, root: "/workspaces/11111111-1111-4111-8111-111111111111"},
	} {
		t.Run(test.name, func(t *testing.T) {
			genericPoster := &fakeFoundationHTTPClient{response: []byte(`{"ok":true}`)}
			nodePoster := &fakeNodeHTTPClient{response: []byte(`{"ok":true}`)}
			var generic, node domain.EngineClient
			if test.generic {
				generic = NewWorkspaceFoundationClient("http://dedicated.test", genericPoster, "http.runtime")
			}
			if test.node {
				node = NewNodeClient("http://node.test", nodePoster, "http.runtime")
			}
			result, err := NewHTTPRuntimeDispatchClient(generic, node).Validate(context.Background(), domain.EngineValidationInput{
				Stage:     domain.ValidationStage{ID: "check", Engine: "http.runtime", Checks: json.RawMessage(test.checks)},
				Workspace: domain.ValidationWorkspace{RootPath: test.root, Files: []domain.WorkspaceFile{{Path: "index.js", Content: "synthetic text"}}},
			})
			if test.unsupported {
				if !errors.Is(err, domain.ErrUnsupportedEngine) || genericPoster.lastURL != "" || nodePoster.lastURL != "" {
					t.Fatalf("unsupported selected provider was not rejected before HTTP: %v", err)
				}
				return
			}
			if err != nil || !result.Passed || (nodePoster.lastURL != "") != test.selectedNode || (genericPoster.lastURL != "") == test.selectedNode {
				t.Fatalf("selected provider changed: passed=%t error=%v", result.Passed, err)
			}
		})
	}
}
