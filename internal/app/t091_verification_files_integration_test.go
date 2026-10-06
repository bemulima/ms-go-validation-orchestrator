//go:build integration

package app

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/config"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

// This opt-in proof uses the actual application composition and an external
// listening HTML owner. Its fixture roots contain only test-owned text files.
func TestT091ProductionVerificationUsesRealHTMLProvider(t *testing.T) {
	provider := os.Getenv("T091_HTML_PROVIDER_URL")
	if provider == "" {
		t.Skip("requires explicitly launched isolated real HTML provider")
	}
	physicalRoot, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VERIFICATION_WORKSPACES_DIR", physicalRoot)
	t.Setenv("INTERNAL_API_TOKEN", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	var entropy [24]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal("generate isolated caller identity")
	}
	cfg.InternalAPIToken = hex.EncodeToString(entropy[:])
	cfg.Engines = config.EngineEndpoints{HTML: provider}
	cfg.SandboxServiceBaseURL, cfg.SandboxServiceInternalToken = "", ""
	application := New(cfg)
	server := httptest.NewServer(application.Server().Handler)
	defer server.Close()
	contents := []string{
		"<!DOCTYPE html><html lang=\"en\"><body><h1 class=\"title\">Starter</h1></body></html>",
		"<!DOCTYPE html><html lang=\"en\"><body><h1 class=\"title\">Hello World!</h1></body></html>",
		"<!DOCTYPE html><html lang=\"en\"><body><h1 class=\"title\">Wrong output</h1></body></html>",
	}
	kinds := []string{domain.VerificationCaseStarter, domain.VerificationCaseReference, domain.VerificationCaseNegative}
	request := domain.ContractVerificationRequestV1{
		Schema:               domain.ContractVerificationRequestSchemaV1,
		BlueprintDigest:      "sha256:" + strings.Repeat("1", 64),
		RuntimeProfileDigest: "sha256:" + strings.Repeat("2", 64),
		CodeStructure:        json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"html","engine":"html.dom","mode":"final","targets":{"files":["index.html"]},"rules":{"version":2,"head":{"doctype":true,"htmlLang":"en"},"body":{"roots":[{"tag":{"value":"h1"},"attributes":[{"name":"class","value":"title"}],"text":{"value":"Hello World!"}}]}}}]}`),
	}
	for i, kind := range kinds {
		name := "t091-" + kind
		directory := filepath.Join(physicalRoot, name)
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte(contents[i]), 0o400); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(directory, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(directory, 0o700) })
		request.Cases = append(request.Cases, domain.ContractVerificationCaseV1{ID: kind, Kind: kind, WorkspaceRoot: "/workspaces/" + name})
	}
	call := func(input domain.ContractVerificationRequestV1, authorized bool) (int, []byte) {
		t.Helper()
		body, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		httpRequest, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/contracts/verify", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		httpRequest.Header.Set("Content-Type", "application/json")
		if authorized {
			httpRequest.Header.Set("X-Internal-Token", cfg.InternalAPIToken)
		}
		response, err := server.Client().Do(httpRequest)
		if err != nil {
			t.Fatal("call actual composition")
		}
		defer response.Body.Close()
		data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, data
	}
	status, data := call(request, true)
	if status != http.StatusOK {
		t.Fatalf("actual verification status=%d", status)
	}
	var receipt domain.ContractVerificationReceiptV1
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Cases) != 3 || receipt.Cases[0].ActualPassed || !receipt.Cases[1].ActualPassed || receipt.Cases[2].ActualPassed {
		t.Fatalf("actual file-dependent HTML owner failed reference contract: passed=%v cases=%+v", receipt.Passed, receipt.Cases)
	}
	for _, content := range contents {
		if bytes.Contains(data, []byte(content)) {
			t.Fatal("private fixture content leaked into receipt")
		}
	}
	if bytes.Contains(data, []byte(physicalRoot)) {
		t.Fatal("physical host root leaked into receipt")
	}
	unauthenticatedStatus, _ := call(request, false)
	if unauthenticatedStatus != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", unauthenticatedStatus)
	}
	unsafe := request
	unsafe.Cases = append([]domain.ContractVerificationCaseV1(nil), request.Cases...)
	unsafe.Cases[0].WorkspaceRoot = "/tmp/t091-unsafe"
	unsafeStatus, _ := call(unsafe, true)
	if unsafeStatus != http.StatusBadRequest {
		t.Fatalf("unsafe root status=%d", unsafeStatus)
	}
	missing := request
	missing.Cases = append([]domain.ContractVerificationCaseV1(nil), request.Cases...)
	missing.Cases[0].WorkspaceRoot = "/workspaces/t091-not-created"
	missingStatus, _ := call(missing, true)
	if missingStatus != http.StatusBadRequest {
		t.Fatalf("missing mount/case must fail technical request, status=%d", missingStatus)
	}
	// Unsafe material must fail as a technical request, never count as a
	// successful negative fixture. Both root and descendant symlinks are tested.
	unsafeMaterialStatuses := map[string]int{}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "index.html"), []byte("test-only outside root"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"root-link", "file-link", "binary-text"} {
		directory := filepath.Join(physicalRoot, name)
		if name == "root-link" {
			if err := os.Symlink(outside, directory); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			if name == "file-link" {
				if err := os.Symlink(filepath.Join(outside, "index.html"), filepath.Join(directory, "index.html")); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte{0xff, 0xfe}, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		input := request
		input.Cases = append([]domain.ContractVerificationCaseV1(nil), request.Cases...)
		input.Cases[0].WorkspaceRoot = "/workspaces/" + name
		status, data := call(input, true)
		if status != http.StatusBadRequest || bytes.Contains(data, []byte(outside)) || bytes.Contains(data, []byte("test-only outside root")) {
			t.Fatalf("unsafe material case=%s status=%d", name, status)
		}
		unsafeMaterialStatuses[name] = status
	}
	if resultPath := os.Getenv("T091_RESULT_FILE"); resultPath != "" {
		proof := map[string]any{"classification": "actual application factory/registered HTTP → actual HTML production DI/registered HTTP; read-only test text roots; not full live E2E", "receipt": receipt, "unauthenticated_status": unauthenticatedStatus, "unsafe_root_status": unsafeStatus, "missing_case_technical_status": missingStatus, "unsafe_material_technical_statuses": unsafeMaterialStatuses, "private_content_in_receipt": false, "full_live_e2e": false}
		encoded, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(resultPath, append(encoded, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
