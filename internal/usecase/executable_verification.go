package usecase

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type ExecutableVerification struct {
	engine domain.GoTestPort
	key    []byte
	keyID  string
}

func NewExecutableVerification(engine domain.GoTestPort, key, keyID string) (*ExecutableVerification, error) {
	if engine == nil || len(key) < 32 || keyID == "" {
		return nil, errors.New("executable verification credentials required")
	}
	return &ExecutableVerification{engine: engine, key: []byte(key), keyID: keyID}, nil
}

func strictExecutableContract(raw json.RawMessage) bool {
	var contract domain.ValidationContract
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&contract) != nil {
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return false
	}
	if contract.Version != 1 || contract.Kind != "workspace_contract" || len(contract.Stages) != 1 || len(contract.Links) != 0 {
		return false
	}
	s := contract.Stages[0]
	if s.ID != "double_numbers" || s.Engine != "go.tests.v1" || s.Mode != "final" || s.Optional || len(s.DependsOn) != 0 || len(s.Rules) != 0 {
		return false
	}
	var checks map[string]string
	if json.Unmarshal(s.Checks, &checks) != nil || len(checks) != 1 || (checks["profile_id"] != "go-double-numbers.v1" && checks["profile_id"] != "go-double-numbers-cases.v1") {
		return false
	}
	return len(contract.Workspace.RequiredFiles) == 1 && contract.Workspace.RequiredFiles[0] == "solution.go" && len(s.Targets.Files) == 1 && s.Targets.Files[0] == "solution.go"
}
func validVerificationRequest(req domain.ExecutableVerificationRequest) bool {
	var c domain.ValidationContract
	if json.Unmarshal(req.ValidationContract, &c) != nil || len(c.Stages) != 1 {
		return false
	}
	var checks map[string]string
	if json.Unmarshal(c.Stages[0].Checks, &checks) != nil || checks["profile_id"] != req.Binding.ProfileID {
		return false
	}
	if req.Binding.ProfileID == "go-double-numbers-cases.v1" && len(req.Binding.TestDefinitions) == 0 {
		return false
	}

	return req.Schema == "validation-contract-verification-request.v2" && practiceUUIDPattern.MatchString(req.IdempotencyKey) && verificationDigestPattern.MatchString(req.CandidateDigest) && verificationDigestPattern.MatchString(req.RuntimeProfileDigest) && strictExecutableContract(req.ValidationContract) && req.Binding.Schema == "go-test-bundle-verification.v1" && len(req.Binding.Files) == 0
}
func matchingGoBinding(b domain.GoTestBinding, m domain.GoTestManifest) bool {
	if b.ProfileID == "go-double-numbers-cases.v1" && (len(b.TestDefinitions) == 0 || domain.CanonicalDigestV2(b.TestDefinitions) != m.DefinitionsDigest) {
		return false
	}
	if b.ProfileID == "go-double-numbers.v1" && (len(b.TestDefinitions) != 0 || m.DefinitionsDigest != "") {
		return false
	}

	return m.Schema == "go-test-capability.v1" && (b.ProfileID == "go-double-numbers.v1" || b.ProfileID == "go-double-numbers-cases.v1") && b.ProfileID == m.ProfileID && b.BundleDigest == m.BundleDigest && b.RunnerDigest == m.RunnerDigest && b.ExecutionProfileDigest == m.ExecutionProfileDigest && b.StarterDigest == m.StarterDigest && len(m.CriterionTests) == 7 && len(m.TestFileDigests) == 1
}
func validExecutableGate(req domain.ExecutableVerificationRequest, result domain.GoTestResult) bool {
	if result.Schema != "go-test-result.v1" || !matchingGoBinding(req.Binding, result.Manifest) || domain.CanonicalDigestV2(req.CriterionTests) != domain.CanonicalDigestV2(result.Manifest.CriterionTests) || len(result.Outcomes) != 3 {
		return false
	}
	for name, want := range map[string]string{"reference": "PASS", "starter": "FAIL", "negative": "FAIL"} {
		o, ok := result.Outcomes[name]
		if !ok || o.Status != want || o.SourceDigest != result.Manifest.FixtureDigests[name] || len(o.Criteria) != 7 {
			return false
		}
		failed := false
		for criterion := range result.Manifest.CriterionTests {
			s := o.Criteria[criterion]
			if s != "PASS" && s != "FAIL" {
				return false
			}
			failed = failed || s == "FAIL"
		}
		if want == "PASS" && failed || want == "FAIL" && !failed {
			return false
		}
	}
	return result.Outcomes["negative"].Criteria["zeros"] == "FAIL" && result.Manifest.FixtureDigests["starter"] == req.Binding.StarterDigest
}
func (v *ExecutableVerification) Verify(ctx context.Context, req domain.ExecutableVerificationRequest) (domain.ExecutableVerificationReceipt, error) {
	if !validVerificationRequest(req) {
		return domain.ExecutableVerificationReceipt{}, errors.New("invalid executable verification request")
	}
	m, err := v.engine.Capability(ctx)
	if req.Binding.ProfileID == "go-double-numbers-cases.v1" {
		m, err = v.Prepare(ctx, req.Binding.TestDefinitions)
	} else if len(req.Binding.TestDefinitions) != 0 {
		return domain.ExecutableVerificationReceipt{}, errors.New("legacy definitions forbidden")
	}
	if err != nil || !matchingGoBinding(req.Binding, m) || domain.CanonicalDigestV2(req.CriterionTests) != domain.CanonicalDigestV2(m.CriterionTests) {
		return domain.ExecutableVerificationReceipt{}, errors.New("capability binding mismatch")
	}
	result, err := v.engine.Run(ctx, req.Binding, true)
	if err != nil || !validExecutableGate(req, result) || domain.CanonicalDigestV2(m) != domain.CanonicalDigestV2(result.Manifest) {
		return domain.ExecutableVerificationReceipt{}, errors.New("executable gate failed")
	}
	r := domain.ExecutableVerificationReceipt{Schema: "validation-contract-verification-receipt.v2", Request: req, ContractDigest: domain.CanonicalDigestV2(req.ValidationContract), CapabilitiesDigest: domain.CanonicalDigestV2(m), Verification: result, Issuer: "ms-go-validation-orchestrator", KeyID: v.keyID, VerifiedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	r.ReceiptDigest = domain.CanonicalDigestV2(r)
	r.Seal = v.seal(r.ReceiptDigest)
	return r, nil
}
func (v *ExecutableVerification) seal(digest string) string {
	mac := hmac.New(sha256.New, v.key)
	_, _ = mac.Write([]byte(digest))
	return hex.EncodeToString(mac.Sum(nil))
}
func (v *ExecutableVerification) Validate(input domain.VerificationReceiptValidation) bool {
	r := input.Receipt
	if r.Schema != "validation-contract-verification-receipt.v2" || r.Issuer != "ms-go-validation-orchestrator" || r.KeyID != v.keyID || !validVerificationRequest(input.ExpectedRequest) || domain.CanonicalDigestV2(r.Request) != domain.CanonicalDigestV2(input.ExpectedRequest) || !validExecutableGate(r.Request, r.Verification) || r.ContractDigest != domain.CanonicalDigestV2(r.Request.ValidationContract) || r.CapabilitiesDigest != domain.CanonicalDigestV2(r.Verification.Manifest) {
		return false
	}
	digest, seal := r.ReceiptDigest, r.Seal
	r.ReceiptDigest = ""
	r.Seal = ""
	return digest == domain.CanonicalDigestV2(r) && hmac.Equal([]byte(seal), []byte(v.seal(digest)))
}

func (v *ExecutableVerification) Prepare(ctx context.Context, raw json.RawMessage) (domain.GoTestManifest, error) {
	p, ok := v.engine.(interface {
		Prepare(context.Context, json.RawMessage) (domain.GoTestManifest, error)
	})
	if !ok {
		return domain.GoTestManifest{}, errors.New("authored profile unavailable")
	}
	return p.Prepare(ctx, raw)
}
