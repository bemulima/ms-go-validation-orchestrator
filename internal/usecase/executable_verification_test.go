package usecase

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"os"
	"strings"
	"testing"
)

type executableFake struct {
	result domain.GoTestResult
	calls  int
}

func (f *executableFake) Capability(context.Context) (domain.GoTestManifest, error) {
	return f.result.Manifest, nil
}
func (f *executableFake) Run(context.Context, domain.GoTestBinding, bool) (domain.GoTestResult, error) {
	f.calls++
	return f.result, nil
}
func executableFixture(t *testing.T) (*ExecutableVerification, *executableFake, domain.ExecutableVerificationRequest) {
	t.Helper()
	b, e := os.ReadFile("testdata/go_executable_gate.json")
	if e != nil {
		t.Fatal(e)
	}
	f := &executableFake{}
	if json.Unmarshal(b, &f.result) != nil {
		t.Fatal("gate metadata fixture")
	}
	m := f.result.Manifest
	u, e := NewExecutableVerification(f, strings.Repeat("s", 64), "unit-key")
	if e != nil {
		t.Fatal(e)
	}
	r := domain.ExecutableVerificationRequest{Schema: "validation-contract-verification-request.v2", IdempotencyKey: "4658157e-0f3b-45cc-899b-d54f0ba71aab", CandidateDigest: "sha256:" + strings.Repeat("a", 64), RuntimeProfileDigest: "sha256:" + strings.Repeat("b", 64), ValidationContract: json.RawMessage(`{"version":1,"kind":"workspace_contract","workspace":{"required_files":["solution.go"]},"stages":[{"id":"double_numbers","engine":"go.tests.v1","mode":"final","targets":{"files":["solution.go"]},"checks":{"profile_id":"go-double-numbers.v1"}}]}`), Binding: domain.GoTestBinding{Schema: "go-test-bundle-verification.v1", ProfileID: m.ProfileID, BundleDigest: m.BundleDigest, RunnerDigest: m.RunnerDigest, ExecutionProfileDigest: m.ExecutionProfileDigest, StarterDigest: m.StarterDigest}, CriterionTests: m.CriterionTests}
	return u, f, r
}
func TestExecutableVerificationSealAndDrift(t *testing.T) {
	u, f, request := executableFixture(t)
	receipt, e := u.Verify(context.Background(), request)
	if e != nil {
		t.Fatal(e)
	}
	input := domain.VerificationReceiptValidation{Receipt: receipt, ExpectedRequest: request}
	if !u.Validate(input) || f.calls != 1 {
		t.Fatal("valid receipt rejected")
	}
	for _, change := range []func(*domain.VerificationReceiptValidation){
		func(v *domain.VerificationReceiptValidation) { v.Receipt.Seal = "forged" },
		func(v *domain.VerificationReceiptValidation) {
			v.ExpectedRequest.CandidateDigest = "sha256:" + strings.Repeat("c", 64)
		},
		func(v *domain.VerificationReceiptValidation) { v.Receipt.ContractDigest = "changed" },
		func(v *domain.VerificationReceiptValidation) { v.Receipt.KeyID = "other-owner" },
		func(v *domain.VerificationReceiptValidation) { v.Receipt.VerifiedAt = "changed" },
		func(v *domain.VerificationReceiptValidation) {
			v.ExpectedRequest.RuntimeProfileDigest = "sha256:" + strings.Repeat("c", 64)
		},
		func(v *domain.VerificationReceiptValidation) {
			v.ExpectedRequest.Binding.BundleDigest = "sha256:" + strings.Repeat("c", 64)
		},
		func(v *domain.VerificationReceiptValidation) {
			v.ExpectedRequest.Binding.RunnerDigest = "sha256:" + strings.Repeat("c", 64)
		},
		func(v *domain.VerificationReceiptValidation) {
			v.ExpectedRequest.Binding.ExecutionProfileDigest = "sha256:" + strings.Repeat("c", 64)
		},
		func(v *domain.VerificationReceiptValidation) {
			v.ExpectedRequest.Binding.StarterDigest = "sha256:" + strings.Repeat("c", 64)
		},
		func(v *domain.VerificationReceiptValidation) {
			v.ExpectedRequest.ValidationContract = json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[]}`)
		},
		func(v *domain.VerificationReceiptValidation) { v.ExpectedRequest.CriterionTests = map[string]string{} },
	} {
		b, _ := json.Marshal(input)
		var copy domain.VerificationReceiptValidation
		_ = json.Unmarshal(b, &copy)
		change(&copy)
		if u.Validate(copy) {
			t.Fatal("receipt drift accepted")
		}
	}
	if f.calls != 1 {
		t.Fatal("receipt validation re-executed code")
	}
}
func TestExecutableVerificationFailsClosed(t *testing.T) {
	for _, kind := range []string{"ERROR", "wrong_negative", "coverage", "binding", "contract", "reference_fail", "reference_assertion_fail", "starter_pass", "negative_error", "criterion_error", "no_semantic_failure", "fixture_swap", "missing_case", "zero_coverage", "optional_only"} {
		t.Run(kind, func(t *testing.T) {
			u, f, r := executableFixture(t)
			switch kind {
			case "ERROR":
				v := f.result.Outcomes["starter"]
				v.Status = "ERROR"
				f.result.Outcomes["starter"] = v
			case "wrong_negative":
				v := f.result.Outcomes["negative"]
				v.Status = "PASS"
				f.result.Outcomes["negative"] = v
			case "coverage":
				r.CriterionTests = map[string]string{"positive": "TestPositive"}
			case "binding":
				r.Binding.RunnerDigest = "changed"
			case "contract":
				r.ValidationContract = json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[]}`)
			case "reference_fail":
				v := f.result.Outcomes["reference"]
				v.Status = "FAIL"
				f.result.Outcomes["reference"] = v
			case "reference_assertion_fail":
				f.result.Outcomes["reference"].Criteria["positive"] = "FAIL"
			case "starter_pass":
				v := f.result.Outcomes["starter"]
				v.Status = "PASS"
				f.result.Outcomes["starter"] = v
			case "negative_error":
				v := f.result.Outcomes["negative"]
				v.Status = "ERROR"
				f.result.Outcomes["negative"] = v
			case "criterion_error":
				f.result.Outcomes["negative"].Criteria["zeros"] = "ERROR"
			case "no_semantic_failure":
				for name := range f.result.Outcomes["negative"].Criteria {
					f.result.Outcomes["negative"].Criteria[name] = "PASS"
				}
			case "fixture_swap":
				v := f.result.Outcomes["negative"]
				v.SourceDigest = "sha256:" + strings.Repeat("c", 64)
				f.result.Outcomes["negative"] = v
			case "missing_case":
				delete(f.result.Outcomes, "starter")
			case "zero_coverage":
				r.CriterionTests = map[string]string{}
			case "optional_only":
				r.ValidationContract = json.RawMessage(strings.Replace(string(r.ValidationContract), `"mode":"final"`, `"mode":"final","optional":true`, 1))
			}
			if _, e := u.Verify(context.Background(), r); e == nil {
				t.Fatal("invalid executable gate accepted")
			}
		})
	}
}
