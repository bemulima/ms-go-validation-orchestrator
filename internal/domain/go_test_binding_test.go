package domain

import (
	"encoding/json"
	"testing"
)

func TestMatchingSafeDivisionGoTestBindingRequiresExactEmptyDefinitionBinding(t *testing.T) {
	m := GoTestManifest{
		Schema:                 "go-test-capability.v1",
		ProfileID:              "go-safe-division.v1",
		BundleVersion:          "safe-division.v1",
		BundleDigest:           "sha256:bundle",
		RunnerDigest:           "sha256:runner",
		ExecutionProfileDigest: "sha256:execution",
		StarterDigest:          "sha256:starter",
		TestFileDigests:        map[string]string{"exercise_test.go": "sha256:tests"},
		CriterionTests: map[string]string{
			"positive_exact":     "TestPositiveExact",
			"quotient_truncates": "TestQuotientTruncates",
			"negative_numerator": "TestNegativeNumerator",
			"negative_divisor":   "TestNegativeDivisor",
			"both_negative":      "TestBothNegative",
			"zero_numerator":     "TestZeroNumerator",
			"zero_divisor_error": "TestZeroDivisorError",
		},
	}
	b := GoTestBinding{Schema: "go-test-execution.v1", ProfileID: m.ProfileID, BundleDigest: m.BundleDigest, RunnerDigest: m.RunnerDigest, ExecutionProfileDigest: m.ExecutionProfileDigest, StarterDigest: m.StarterDigest}
	if !MatchingGoTestBinding(b, m) {
		t.Fatal("valid SafeDivide binding rejected")
	}
	b.TestDefinitions = json.RawMessage(`{"unexpected":"profile-defined"}`)
	if MatchingGoTestBinding(b, m) {
		t.Fatal("SafeDivide binding accepted mutable test definitions")
	}
}
