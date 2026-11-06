package usecase

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestPracticeValidationOwnerDigestGoldenVectors(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/practice_validation_digest_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Vectors []struct {
			Name, Scope    string
			MilestoneID    string          `json:"milestone_id"`
			Specification  json.RawMessage `json:"validation_specification"`
			ExpectedDigest string          `json:"expected_digest"`
		}
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Vectors) != 2 {
		t.Fatal("both frozen v1 milestone and full v2 final vectors are required")
	}
	for _, vector := range fixture.Vectors {
		t.Run(vector.Name, func(t *testing.T) {
			got, err := ComputePracticeValidationContractDigest(vector.Scope, vector.MilestoneID, vector.Specification)
			if err != nil || got != vector.ExpectedDigest {
				t.Fatalf("digest %s error %v, want independently pinned %s", got, err, vector.ExpectedDigest)
			}
			// JSON escaping is insignificant; numeric spellings stay significant.
			escaped := strings.ReplaceAll(string(vector.Specification), "<main>&", `\u003cmain\u003e\u0026`)
			equivalent, err := ComputePracticeValidationContractDigest(vector.Scope, vector.MilestoneID, json.RawMessage(escaped))
			if err != nil || equivalent != got {
				t.Fatal("HTML string escaping changed canonical value")
			}
			for _, replacement := range []struct{ old, new string }{{"1e-7", "0.0000001"}, {"9007199254740993", "9007199254740992"}} {
				rewritten := strings.Replace(string(vector.Specification), replacement.old, replacement.new, 1)
				different, err := ComputePracticeValidationContractDigest(vector.Scope, vector.MilestoneID, json.RawMessage(rewritten))
				if err != nil || different == got {
					t.Fatal("numeric lexeme or integer precision was lost")
				}
			}
		})
	}
}
