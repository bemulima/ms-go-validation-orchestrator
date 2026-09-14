package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

var verificationDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// VerifyContract executes the final validation contract against isolated
// starter, reference, and negative-fixture sandboxes and returns a content-
// addressed receipt. A structurally valid inspection is necessary but not
// sufficient; every expected outcome must match for Passed to be true.
func (useCase OrchestrateValidationUseCase) VerifyContract(
	ctx context.Context,
	request domain.ContractVerificationRequestV1,
) (domain.ContractVerificationReceiptV1, error) {
	if request.Schema != domain.ContractVerificationRequestSchemaV1 {
		return domain.ContractVerificationReceiptV1{}, verificationRequestErrorf("schema is unsupported")
	}
	if !verificationDigestPattern.MatchString(request.BlueprintDigest) ||
		!verificationDigestPattern.MatchString(request.RuntimeProfileDigest) {
		return domain.ContractVerificationReceiptV1{}, verificationRequestErrorf("blueprint and runtime profile digests are required")
	}
	if err := validateVerificationCases(request.Cases); err != nil {
		return domain.ContractVerificationReceiptV1{}, err
	}
	inspection, err := useCase.InspectContract(domain.ContractInspectionRequest{
		CodeStructure: request.CodeStructure,
		Mode:          domain.ValidationModeFinal,
	})
	if err != nil {
		return domain.ContractVerificationReceiptV1{}, err
	}
	if !inspection.Runnable {
		return domain.ContractVerificationReceiptV1{}, verificationRequestErrorf("contract requires unavailable engines: %v", inspection.MissingEngines)
	}

	receipt := domain.ContractVerificationReceiptV1{
		Schema:               domain.ContractVerificationReceiptSchemaV1,
		BlueprintDigest:      request.BlueprintDigest,
		RuntimeProfileDigest: request.RuntimeProfileDigest,
		ContractDigest:       inspection.ContractDigest,
		CapabilitiesDigest:   inspection.CapabilitiesDigest,
		Passed:               true,
		Cases:                make([]domain.ContractVerificationCaseResultV1, 0, len(request.Cases)),
		VerifiedAt:           time.Now().UTC(),
	}
	for _, verificationCase := range request.Cases {
		result, err := useCase.Execute(ctx, domain.ValidationRequest{
			TaskID:        "contract-verification:" + verificationCase.ID,
			Mode:          domain.ValidationModeFinal,
			CodeStructure: request.CodeStructure,
			Workspace: domain.ValidationWorkspace{
				RootPath: verificationCase.WorkspaceRoot,
			},
		})
		if err != nil {
			return domain.ContractVerificationReceiptV1{}, fmt.Errorf("verify case %q: %w", verificationCase.ID, err)
		}
		expectedPassed := verificationCase.Kind == domain.VerificationCaseReference
		if result.Passed != expectedPassed {
			receipt.Passed = false
		}
		encodedResult, err := json.Marshal(result)
		if err != nil {
			return domain.ContractVerificationReceiptV1{}, fmt.Errorf("encode verification result: %w", err)
		}
		resultDigest := sha256.Sum256(encodedResult)
		receipt.Cases = append(receipt.Cases, domain.ContractVerificationCaseResultV1{
			ID:             verificationCase.ID,
			Kind:           verificationCase.Kind,
			ExpectedPassed: expectedPassed,
			ActualPassed:   result.Passed,
			ResultDigest:   "sha256:" + hex.EncodeToString(resultDigest[:]),
		})
	}
	encodedReceipt, err := json.Marshal(receipt)
	if err != nil {
		return domain.ContractVerificationReceiptV1{}, fmt.Errorf("encode verification receipt: %w", err)
	}
	receiptDigest := sha256.Sum256(encodedReceipt)
	receipt.ReceiptDigest = "sha256:" + hex.EncodeToString(receiptDigest[:])
	return receipt, nil
}

func validateVerificationCases(cases []domain.ContractVerificationCaseV1) error {
	if len(cases) < 3 || len(cases) > 100 {
		return verificationRequestErrorf("cases must contain starter, reference, and at least one negative fixture")
	}
	caseIDs := make(map[string]struct{}, len(cases))
	workspaceRoots := make(map[string]struct{}, len(cases))
	kindCounts := make(map[string]int, 3)
	for _, verificationCase := range cases {
		if !contractIDPattern.MatchString(verificationCase.ID) {
			return verificationRequestErrorf("case id %q is invalid", verificationCase.ID)
		}
		if _, duplicate := caseIDs[verificationCase.ID]; duplicate {
			return verificationRequestErrorf("duplicate case id %q", verificationCase.ID)
		}
		caseIDs[verificationCase.ID] = struct{}{}
		switch verificationCase.Kind {
		case domain.VerificationCaseStarter, domain.VerificationCaseReference, domain.VerificationCaseNegative:
		default:
			return verificationRequestErrorf("case %q has unsupported kind", verificationCase.ID)
		}
		kindCounts[verificationCase.Kind]++
		if err := domain.ValidateSandboxWorkspaceRoot(verificationCase.WorkspaceRoot); err != nil || verificationCase.WorkspaceRoot == "" {
			return verificationRequestErrorf("case %q must reference an isolated sandbox workspace", verificationCase.ID)
		}
		if _, duplicate := workspaceRoots[verificationCase.WorkspaceRoot]; duplicate {
			return verificationRequestErrorf("verification cases must use distinct sandbox workspaces")
		}
		workspaceRoots[verificationCase.WorkspaceRoot] = struct{}{}
	}
	if kindCounts[domain.VerificationCaseStarter] != 1 {
		return verificationRequestErrorf("exactly one starter case is required")
	}
	if kindCounts[domain.VerificationCaseReference] != 1 {
		return verificationRequestErrorf("exactly one reference case is required")
	}
	if kindCounts[domain.VerificationCaseNegative] < 1 {
		return verificationRequestErrorf("at least one negative case is required")
	}
	return nil
}

func verificationRequestErrorf(format string, args ...any) error {
	return fmt.Errorf("%w: contract verification: %s", domain.ErrInvalidRequest, fmt.Sprintf(format, args...))
}
