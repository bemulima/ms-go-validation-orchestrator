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
	result       domain.GoTestResult
	calls        int
	profileCalls []string
}

func (f *executableFake) Capability(context.Context) (domain.GoTestManifest, error) {
	return f.result.Manifest, nil
}
func (f *executableFake) CapabilityForProfile(_ context.Context, profile string) (domain.GoTestManifest, error) {
	f.profileCalls = append(f.profileCalls, profile)
	if profile != f.result.Manifest.ProfileID {
		return domain.GoTestManifest{}, domain.ErrValidatorProtocol
	}
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

func TestStrictExecutableContractAcceptsSafeDivisionProfileOnlyOnMainGo(t *testing.T) {
	valid := json.RawMessage(`{"version":1,"kind":"workspace_contract","workspace":{"required_files":["main.go"]},"stages":[{"id":"safe_division","engine":"go.tests.v1","mode":"final","targets":{"files":["main.go"]},"checks":{"profile_id":"go-safe-division.v1"}}]}`)
	if !strictExecutableContract(valid) {
		t.Fatal("versioned SafeDivide executable contract rejected")
	}
	for _, raw := range []string{
		`{"version":1,"kind":"workspace_contract","workspace":{"required_files":["solution.go"]},"stages":[{"id":"safe_division","engine":"go.tests.v1","mode":"final","targets":{"files":["solution.go"]},"checks":{"profile_id":"go-safe-division.v1"}}]}`,
		`{"version":1,"kind":"workspace_contract","workspace":{"required_files":["main.go"]},"stages":[{"id":"double_numbers","engine":"go.tests.v1","mode":"final","targets":{"files":["main.go"]},"checks":{"profile_id":"go-safe-division.v1"}}]}`,
		`{"version":1,"kind":"workspace_contract","workspace":{"required_files":["main.go"]},"stages":[{"id":"safe_division","engine":"go.tests.v1","mode":"final","targets":{"files":["main.go"]},"checks":{"profile_id":"go-double-numbers.v1"}}]}`,
	} {
		if strictExecutableContract(json.RawMessage(raw)) {
			t.Fatal("SafeDivide contract accepted wrong profile or workspace")
		}
	}
}

func TestSafeDivisionExecutableVerificationUsesExactCapabilityAndNegativeCriterion(t *testing.T) {
	criteria := map[string]string{
		"positive_exact": "TestPositiveExact", "quotient_truncates": "TestQuotientTruncates",
		"negative_numerator": "TestNegativeNumerator", "negative_divisor": "TestNegativeDivisor",
		"both_negative": "TestBothNegative", "zero_numerator": "TestZeroNumerator",
		"zero_divisor_error": "TestZeroDivisorError",
	}
	digest := func(value string) string { return domain.CanonicalDigestV2(value) }
	manifest := domain.GoTestManifest{
		Schema: "go-test-capability.v1", ProfileID: "go-safe-division.v1", BundleVersion: "safe-division.v1",
		BundleDigest: digest("bundle"), RunnerDigest: digest("runner"), ExecutionProfileDigest: digest("execution"),
		StarterDigest: digest("starter"), TestFileDigests: map[string]string{"exercise_test.go": digest("tests")},
		CriterionTests: criteria, FixtureDigests: map[string]string{"reference": digest("reference"), "starter": digest("starter"), "negative": digest("negative")},
	}
	statuses := map[string]domain.GoTestOutcome{}
	for name, status := range map[string]string{"reference": "PASS", "starter": "FAIL", "negative": "FAIL"} {
		criterionResults := map[string]string{}
		for criterion := range criteria {
			criterionResults[criterion] = "PASS"
		}
		if status == "FAIL" {
			criterionResults["positive_exact"] = "FAIL"
		}
		statuses[name] = domain.GoTestOutcome{Status: status, SourceDigest: manifest.FixtureDigests[name], Criteria: criterionResults}
	}
	fake := &executableFake{result: domain.GoTestResult{Schema: "go-test-result.v1", Manifest: manifest, Outcomes: statuses}}
	verification, err := NewExecutableVerification(fake, strings.Repeat("k", 64), "safe-division-key")
	if err != nil {
		t.Fatal(err)
	}
	contract := json.RawMessage(`{"version":1,"kind":"workspace_contract","workspace":{"required_files":["main.go"]},"stages":[{"id":"safe_division","engine":"go.tests.v1","mode":"final","targets":{"files":["main.go"]},"checks":{"profile_id":"go-safe-division.v1"}}]}`)
	request := domain.ExecutableVerificationRequest{
		Schema: "validation-contract-verification-request.v2", IdempotencyKey: "4658157e-0f3b-45cc-899b-d54f0ba71aab",
		CandidateDigest: digest("candidate"), RuntimeProfileDigest: digest("runtime"), ValidationContract: contract,
		Binding:        domain.GoTestBinding{Schema: "go-test-bundle-verification.v1", ProfileID: manifest.ProfileID, BundleDigest: manifest.BundleDigest, RunnerDigest: manifest.RunnerDigest, ExecutionProfileDigest: manifest.ExecutionProfileDigest, StarterDigest: manifest.StarterDigest},
		CriterionTests: criteria,
	}
	if _, err := verification.Verify(context.Background(), request); err != nil {
		t.Fatalf("SafeDivide executable verification failed: %v", err)
	}
	if len(fake.profileCalls) != 1 || fake.profileCalls[0] != "go-safe-division.v1" || fake.calls != 1 {
		t.Fatalf("exact capability was not used: profiles=%v executions=%d", fake.profileCalls, fake.calls)
	}
}
