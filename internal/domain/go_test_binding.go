package domain

// MatchingGoTestBinding checks the immutable executable identities before outcomes.
func MatchingGoTestBinding(b GoTestBinding, m GoTestManifest) bool {
	if b.ProfileID == "go-double-numbers-cases.v1" && (len(b.TestDefinitions) == 0 || CanonicalDigestV2(b.TestDefinitions) != m.DefinitionsDigest) {
		return false
	}
	if b.ProfileID == "go-double-numbers.v1" && (len(b.TestDefinitions) != 0 || m.DefinitionsDigest != "") {
		return false
	}

	return m.Schema == "go-test-capability.v1" && (b.ProfileID == "go-double-numbers.v1" || b.ProfileID == "go-double-numbers-cases.v1") && b.ProfileID == m.ProfileID && b.BundleDigest == m.BundleDigest && b.RunnerDigest == m.RunnerDigest && b.ExecutionProfileDigest == m.ExecutionProfileDigest && b.StarterDigest == m.StarterDigest && len(m.CriterionTests) == 7 && len(m.TestFileDigests) == 1
}
