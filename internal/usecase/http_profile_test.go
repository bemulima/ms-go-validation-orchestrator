package usecase

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"os"
	"testing"
)

func httpVerificationFixture(t *testing.T) (*ExecutableVerification, *executableFake, domain.ExecutableVerificationRequest) {
	u, f, r := executableFixture(t)
	b, e := os.ReadFile("testdata/http_handler_capability.json")
	if e != nil {
		t.Fatal(e)
	}
	var m domain.GoTestManifest
	if json.Unmarshal(b, &m) != nil {
		t.Fatal("manifest")
	}
	f.result.Manifest = m
	f.result.Outcomes = map[string]domain.GoTestOutcome{}
	for _, name := range []string{"reference", "negative", "starter"} {
		criteria := map[string]string{}
		status := "PASS"
		for c := range m.CriterionTests {
			criteria[c] = "PASS"
		}
		if name != "reference" {
			status = "FAIL"
			criteria["method"] = "FAIL"
		}
		f.result.Outcomes[name] = domain.GoTestOutcome{Status: status, Code: "fixture", SourceDigest: m.FixtureDigests[name], Criteria: criteria}
	}
	r.ValidationContract = json.RawMessage(`{"version":1,"kind":"workspace_contract","workspace":{"required_files":["go.mod","handler.go","main.go"]},"stages":[{"id":"http_handler","engine":"go.tests.v1","mode":"final","targets":{"files":["go.mod","handler.go","main.go"]},"checks":{"profile_id":"go-http-handler.v1"}}]}`)
	r.Binding = domain.GoTestBinding{Schema: "go-test-bundle-verification.v1", ProfileID: m.ProfileID, BundleDigest: m.BundleDigest, RunnerDigest: m.RunnerDigest, ExecutionProfileDigest: m.ExecutionProfileDigest, StarterDigest: m.StarterDigest}
	r.CriterionTests = m.CriterionTests
	return u, f, r
}
func TestHTTPProfileSealedGateAndErrorRejection(t *testing.T) {
	for _, mode := range []string{"valid", "starter_error", "negative_error", "reference_fail", "zero_checks", "missing_file", "runner_drift"} {
		t.Run(mode, func(t *testing.T) {
			u, f, r := httpVerificationFixture(t)
			switch mode {
			case "starter_error", "negative_error":
				name := "starter"
				if mode == "negative_error" {
					name = "negative"
				}
				o := f.result.Outcomes[name]
				o.Status = "ERROR"
				f.result.Outcomes[name] = o
			case "reference_fail":
				o := f.result.Outcomes["reference"]
				o.Status = "FAIL"
				f.result.Outcomes["reference"] = o
			case "zero_checks":
				r.CriterionTests = map[string]string{}
			case "missing_file":
				r.ValidationContract = json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[]}`)
			case "runner_drift":
				r.Binding.RunnerDigest = "wrong"
			}
			receipt, e := u.Verify(context.Background(), r)
			if mode != "valid" {
				if e == nil {
					t.Fatal("invalid gate accepted")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if !u.Validate(domain.VerificationReceiptValidation{Receipt: receipt, ExpectedRequest: r}) || f.calls != 1 {
				t.Fatal("receipt reread executes or rejects")
			}
			receipt.Seal = "forged"
			if u.Validate(domain.VerificationReceiptValidation{Receipt: receipt, ExpectedRequest: r}) {
				t.Fatal("forged seal")
			}
		})
	}
}
func TestHTTPProfileRejectsSwappedContract(t *testing.T) {
	u, _, r := httpVerificationFixture(t)
	_, _, old := executableFixture(t)
	r.ValidationContract = old.ValidationContract
	if _, e := u.Verify(context.Background(), r); e == nil {
		t.Fatal("HTTP binding sealed for single-file contract")
	}
	u, _, r = httpVerificationFixture(t)
	receipt, e := u.Verify(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	receipt.Request.ValidationContract = old.ValidationContract
	if u.Validate(domain.VerificationReceiptValidation{Receipt: receipt, ExpectedRequest: receipt.Request}) {
		t.Fatal("swapped contract receipt validated")
	}
}
