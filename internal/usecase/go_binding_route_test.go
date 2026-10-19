package usecase

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"testing"
)

type boundExecutableFake struct {
	*executableFake
	selected int
}

func (f *boundExecutableFake) CapabilityForBinding(_ context.Context, b domain.GoTestBinding) (domain.GoTestManifest, error) {
	f.selected++
	return f.result.Manifest, nil
}
func TestExecutableVerificationUsesBoundCapabilityAndReceiptReadIsOffline(t *testing.T) {
	_, f, r := executableFixture(t)
	bound := &boundExecutableFake{executableFake: f}
	u, e := NewExecutableVerification(bound, "12345678901234567890123456789012", "unit-key")
	if e != nil {
		t.Fatal(e)
	}
	receipt, e := u.Verify(context.Background(), r)
	if e != nil || bound.selected != 1 {
		t.Fatal("did not select trusted binding", e)
	}
	calls := bound.selected
	if !u.Validate(domain.VerificationReceiptValidation{Receipt: receipt, ExpectedRequest: r}) || bound.selected != calls || f.calls != 1 {
		t.Fatal("receipt reread performed routing or execution")
	}
	receipt.Seal = "forged"
	if u.Validate(domain.VerificationReceiptValidation{Receipt: receipt, ExpectedRequest: r}) {
		t.Fatal("forged receipt")
	}
}
func (f *boundExecutableFake) EngineID() string { return "go.tests.v1" }
func (f *boundExecutableFake) Validate(context.Context, domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	panic("must not execute during binding")
}
func TestBindGoEvaluationSelectsHistoricalPrivateBinding(t *testing.T) {
	_, f, r := executableFixture(t)
	bound := &boundExecutableFake{executableFake: f}
	u := PracticeValidationUseCase{orchestrator: NewOrchestrateValidationUseCase(ContractParser{}, []domain.EngineClient{bound})}
	var c domain.ValidationContract
	json.Unmarshal(r.ValidationContract, &c)
	b := r.Binding
	b.Schema = ""
	raw, _ := json.Marshal(b)
	m := f.result.Manifest
	assets := &practiceEvaluationAssetsV1{VerificationRunnerRef: string(raw), TestSuiteRef: "go-tests:" + m.BundleDigest, ReferenceSolutionRef: "go-fixture:" + m.FixtureDigests["reference"], NegativeFixtureRefs: []string{"go-fixture:" + m.FixtureDigests["negative"]}}
	if e := u.bindGoEvaluation(context.Background(), &c, assets); e != nil || bound.selected != 1 {
		t.Fatal("historical tuple not selected", e)
	}
	var checks struct {
		Binding domain.GoTestBinding `json:"private_binding"`
	}
	json.Unmarshal(c.Stages[0].Checks, &checks)
	if checks.Binding.RunnerDigest != r.Binding.RunnerDigest {
		t.Fatal("frozen digest rewritten")
	}
	json.Unmarshal(r.ValidationContract, &c)
	b.RunnerDigest = "forged"
	raw, _ = json.Marshal(b)
	assets.VerificationRunnerRef = string(raw)
	if e := u.bindGoEvaluation(context.Background(), &c, assets); e == nil {
		t.Fatal("changed frozen binding accepted")
	}
}
