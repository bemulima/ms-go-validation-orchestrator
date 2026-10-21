package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"net/url"
	"testing"
)

type goBindingProviderStub struct {
	practiceEngineStub
	manifest                  domain.GoTestManifest
	capabilityErr, prepareErr error
}

func (g goBindingProviderStub) EngineID() string { return "go.tests.v1" }
func (g goBindingProviderStub) Capability(context.Context) (domain.GoTestManifest, error) {
	return g.manifest, g.capabilityErr
}
func (g goBindingProviderStub) Prepare(context.Context, json.RawMessage) (domain.GoTestManifest, error) {
	return g.manifest, g.prepareErr
}
func (g goBindingProviderStub) Run(context.Context, domain.GoTestBinding, bool) (domain.GoTestResult, error) {
	panic("binding rejection must not execute code")
}
func TestPracticeGoBindingProviderFailureIsNotAuthoringDefect(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		cause                      error
		prepare, drift, badProfile bool
		want                       string
	}{
		{name: "capability_unavailable", cause: &url.Error{Op: "Get", URL: "http://provider.invalid", Err: errors.New("refused")}, want: domain.PracticeValidationValidatorUnavailable},
		{name: "capability_timeout", cause: context.DeadlineExceeded, want: domain.PracticeValidationValidatorTimeout},
		{name: "protocol", cause: domain.ErrValidatorProtocol, want: domain.PracticeValidationValidatorProtocol},
		{name: "http_unavailable", cause: domain.ValidatorHTTPStatusError{StatusCode: 503}, want: domain.PracticeValidationValidatorUnavailable},
		{name: "prepare_unavailable", prepare: true, cause: &url.Error{Op: "Post", URL: "http://provider.invalid", Err: errors.New("refused")}, want: domain.PracticeValidationValidatorUnavailable},
		{name: "binding_drift", drift: true, want: domain.PracticeValidationAuthoringDefect},
		{name: "wrong_public_profile", badProfile: true, want: domain.PracticeValidationAuthoringDefect},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, fake, verified := executableFixture(t)
			manifest := fake.result.Manifest
			binding := verified.Binding
			profile := binding.ProfileID
			if tc.prepare {
				profile = "go-double-numbers-cases.v1"
				binding.ProfileID = profile
				binding.TestDefinitions = json.RawMessage(`{"schema":"unit-definition"}`)
				manifest.ProfileID = profile
			}
			if tc.drift {
				binding.RunnerDigest = practiceTestDigest
			}
			if tc.badProfile {
				profile = "foreign-profile"
			}
			engine := goBindingProviderStub{manifest: manifest}
			if tc.cause != nil {
				if tc.prepare {
					engine.prepareErr = domain.GoTestProviderError{Cause: tc.cause}
				} else {
					engine.capabilityErr = domain.GoTestProviderError{Cause: tc.cause}
				}
			}
			rawBinding, _ := json.Marshal(binding)
			contract, _ := json.Marshal(map[string]any{"version": 1, "kind": "workspace_contract", "stages": []any{map[string]any{"id": "double_numbers", "engine": "go.tests.v1", "mode": "final", "checks": map[string]string{"profile_id": profile}}}})
			request := practiceRequest(t, true, false)
			var spec practiceValidationSpecificationV1
			if json.Unmarshal(request.ValidationSpecification, &spec) != nil {
				t.Fatal("fixture")
			}
			spec.Contracts[0].ValidationContract = contract
			spec.Contracts[0].EvaluationAssets = &practiceEvaluationAssetsV1{VerificationRunnerRef: string(rawBinding), TestSuiteRef: "go-tests:" + manifest.BundleDigest, ReferenceSolutionRef: "go-fixture:" + manifest.FixtureDigests["reference"], NegativeFixtureRefs: []string{"go-fixture:" + manifest.FixtureDigests["negative"]}}
			request.ValidationSpecification, _ = json.Marshal(spec)
			request.ValidationContractDigest, _ = ComputePracticeValidationContractDigest(request.Scope, request.MilestoneID, request.ValidationSpecification)
			usecase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})
			result, err := usecase.ValidatePractice(context.Background(), request)
			if err != nil || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != tc.want {
				t.Fatalf("got %s/%s, error %v; want ERROR/%s", result.Outcome, result.ErrorClassification, err, tc.want)
			}
		})
	}
}
