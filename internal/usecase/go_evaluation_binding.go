package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func (u PracticeValidationUseCase) bindGoEvaluation(ctx context.Context, c *domain.ValidationContract, assets *practiceEvaluationAssetsV1) error {
	for i, s := range c.Stages {
		if s.Engine != "go.tests.v1" {
			continue
		}
		engine, ok := u.orchestrator.engines[s.Engine].(domain.GoTestPort)
		if !ok || assets == nil {
			return errors.New("Go evaluation unavailable")
		}
		var checks map[string]string
		if json.Unmarshal(s.Checks, &checks) != nil || len(checks) != 1 || (checks["profile_id"] != "go-double-numbers.v1" && checks["profile_id"] != "go-double-numbers-cases.v1") {
			return errors.New("invalid public Go profile")
		}
		m, e := engine.Capability(ctx)
		if e != nil {
			return e
		}
		var b domain.GoTestBinding
		if json.Unmarshal([]byte(assets.VerificationRunnerRef), &b) != nil {
			return errors.New("invalid frozen runner binding")
		}
		if !samePublicGoProfile(checks["profile_id"], b.ProfileID) {
			return errors.New("public/private Go profile mismatch")
		}
		if b.ProfileID == "go-double-numbers-cases.v1" {
			p, ok := engine.(interface {
				Prepare(context.Context, json.RawMessage) (domain.GoTestManifest, error)
			})
			if !ok {
				return errors.New("authored preparation unavailable")
			}
			m, e = p.Prepare(ctx, b.TestDefinitions)
			if e != nil {
				return e
			}
		}
		b.Schema = "go-test-execution.v1"
		if !matchingGoBinding(b, m) || len(b.Files) != 0 || assets.TestSuiteRef != "go-tests:"+m.BundleDigest || assets.ReferenceSolutionRef != "go-fixture:"+m.FixtureDigests["reference"] || len(assets.NegativeFixtureRefs) != 1 || assets.NegativeFixtureRefs[0] != "go-fixture:"+m.FixtureDigests["negative"] {
			return errors.New("Go evaluation drift")
		}
		c.Stages[i].Checks, _ = json.Marshal(map[string]any{"profile_id": m.ProfileID, "private_binding": b})
	}
	return nil
}

func samePublicGoProfile(public, private string) bool {
	return public == private && (public == "go-double-numbers.v1" || public == "go-double-numbers-cases.v1")
}
