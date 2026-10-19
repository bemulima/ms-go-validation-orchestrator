package usecase

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"strings"
	"testing"
)

func (f *executableFake) Prepare(context.Context, json.RawMessage) (domain.GoTestManifest, error) {
	return f.result.Manifest, nil
}

// Authored negative receipt fixtures exercise the gate; actual execution is separately required.
func TestAuthoredGateCannotAcceptWrongOutcome(t *testing.T) {
	for _, kind := range []string{"positive", "reference_fail", "starter_pass", "negative_pass", "infrastructure_error", "definition_drift", "public_profile_drift"} {
		t.Run(kind, func(t *testing.T) {
			u, f, r := executableFixture(t)
			profile := "go-double-numbers-cases.v1"
			r.ValidationContract = json.RawMessage(strings.ReplaceAll(string(r.ValidationContract), "go-double-numbers.v1", profile))
			r.Binding.ProfileID = profile
			r.Binding.TestDefinitions = json.RawMessage(`{"schema":"controlled-negative-fixture"}`)
			f.result.Manifest.ProfileID = profile
			f.result.Manifest.DefinitionsDigest = domain.CanonicalDigestV2(r.Binding.TestDefinitions)
			switch kind {
			case "reference_fail":
				o := f.result.Outcomes["reference"]
				o.Status = "FAIL"
				o.Code = "semantic_assertion_failed"
				o.Criteria["positive"] = "FAIL"
				f.result.Outcomes["reference"] = o
			case "starter_pass":
				o := f.result.Outcomes["starter"]
				o.Status = "PASS"
				o.Code = "all_criteria_passed"
				for k := range o.Criteria {
					o.Criteria[k] = "PASS"
				}
				f.result.Outcomes["starter"] = o
			case "negative_pass":
				o := f.result.Outcomes["negative"]
				o.Status = "PASS"
				o.Code = "all_criteria_passed"
				for k := range o.Criteria {
					o.Criteria[k] = "PASS"
				}
				f.result.Outcomes["negative"] = o
			case "infrastructure_error":
				f.result.Outcomes["reference"] = domain.GoTestOutcome{Status: "ERROR"}
			case "definition_drift":
				r.Binding.TestDefinitions = json.RawMessage(`{"changed":true}`)
			case "public_profile_drift":
				r.ValidationContract = json.RawMessage(strings.ReplaceAll(string(r.ValidationContract), profile, "go-double-numbers.v1"))
			}
			receipt, e := u.Verify(context.Background(), r)
			if kind == "positive" {
				if e != nil || !u.Validate(domain.VerificationReceiptValidation{Receipt: receipt, ExpectedRequest: r}) {
					t.Fatal("fixture success rejected", e)
				}
			} else if e == nil {
				t.Fatal("unsafe gate accepted")
			}
		})
	}
}

func TestFrozenGoPublicPrivateProfileCannotMix(t *testing.T) {
	for _, profiles := range [][2]string{{"go-double-numbers.v1", "go-double-numbers-cases.v1"}, {"go-double-numbers-cases.v1", "go-double-numbers.v1"}, {"foreign", "foreign"}} {
		if samePublicGoProfile(profiles[0], profiles[1]) {
			t.Fatal("mixed frozen profile accepted")
		}
	}
}
