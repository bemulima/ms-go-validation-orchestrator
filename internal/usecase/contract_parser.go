package usecase

import (
	"encoding/json"
	"fmt"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type ContractParser struct {
	legacyAdapter LegacyContractAdapter
}

func NewContractParser(legacyAdapter LegacyContractAdapter) ContractParser {
	return ContractParser{legacyAdapter: legacyAdapter}
}

func (parser ContractParser) Parse(request domain.ValidationRequest) (domain.ValidationContract, bool, error) {
	if len(request.CodeStructure) == 0 {
		return domain.ValidationContract{}, false, fmt.Errorf("%w: code_structure is required", domain.ErrInvalidRequest)
	}

	var contract domain.ValidationContract
	if err := json.Unmarshal(request.CodeStructure, &contract); err != nil {
		return domain.ValidationContract{}, false, fmt.Errorf("%w: parse contract: %w", domain.ErrInvalidRequest, err)
	}

	if contract.IsNewFormat() {
		return contract, false, nil
	}

	legacyContract, err := parser.legacyAdapter.Adapt(request)
	if err != nil {
		return domain.ValidationContract{}, false, err
	}

	return legacyContract, true, nil
}
