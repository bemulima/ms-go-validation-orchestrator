package usecase

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"testing"
)

type httpExecutableFake struct{ *executableFake }

func (f httpExecutableFake) CapabilityForProfile(_ context.Context, p string) (domain.GoTestManifest, error) {
	return f.result.Manifest, nil
}
func TestHTTPGoExecutableVerificationExactPublicContract(t *testing.T) {
	_, base, req := executableFixture(t)
	f := httpExecutableFake{base}
	m := f.result.Manifest
	m.ProfileID = "go-http-health-main.v1"
	m.CriterionTests = map[string]string{"server_starts": "A", "health_status": "B", "health_body": "C", "health_content_type": "D", "unknown_path": "E", "method_rejected": "F", "repeat_request": "G"}
	f.result.Manifest = m
	for name, o := range f.result.Outcomes {
		out := map[string]string{}
		for k := range m.CriterionTests {
			out[k] = "PASS"
			if name == "starter" || name == "negative" && k == "health_body" {
				out[k] = "FAIL"
			}
		}
		o.Criteria = out
		f.result.Outcomes[name] = o
	}
	req.Binding.ProfileID = m.ProfileID
	req.CriterionTests = m.CriterionTests
	req.ValidationContract = json.RawMessage(`{"version":1,"kind":"workspace_contract","workspace":{"required_files":["main.go"]},"stages":[{"id":"health_http","engine":"go.tests.v1","mode":"final","targets":{"files":["main.go"]},"checks":{"profile_id":"go-http-health-main.v1"}}]}`)
	v, e := NewExecutableVerification(f, "unit-private-key-with-more-than-32-characters", "unit")
	if e != nil {
		t.Fatal(e)
	}
	receipt, e := v.Verify(context.Background(), req)
	if e != nil || !v.Validate(domain.VerificationReceiptValidation{Receipt: receipt, ExpectedRequest: req}) {
		t.Fatal("HTTP gate rejected", e)
	}
	o := f.result.Outcomes["negative"]
	o.Criteria["health_body"] = "PASS"
	o.Criteria["unknown_path"] = "FAIL"
	f.result.Outcomes["negative"] = o
	if _, e = v.Verify(context.Background(), req); e == nil {
		t.Fatal("wrong negative criterion accepted")
	}
	o.Status = "ERROR"
	f.result.Outcomes["negative"] = o
	if _, e = v.Verify(context.Background(), req); e == nil {
		t.Fatal("technical error accepted as expected FAIL")
	}
	if !strictExecutableContract(req.ValidationContract) {
		t.Fatal("HTTP public shape rejected")
	}
	var contract map[string]any
	json.Unmarshal(req.ValidationContract, &contract)
	contract["workspace"] = map[string]any{"required_files": []string{"solution.go"}}
	raw, _ := json.Marshal(contract)
	if strictExecutableContract(raw) {
		t.Fatal("legacy source substituted into HTTP profile")
	}
}
