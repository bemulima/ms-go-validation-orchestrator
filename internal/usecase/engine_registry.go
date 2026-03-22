package usecase

import "github.com/example/ms-validation-orchestrator-service/internal/domain"

// EngineRegistry is an in-memory engine resolver.
type EngineRegistry struct {
	engines map[string]domain.EngineClient
}

// NewEngineRegistry creates a new registry.
func NewEngineRegistry(clients ...domain.EngineClient) *EngineRegistry {
	registry := &EngineRegistry{engines: make(map[string]domain.EngineClient, len(clients))}
	for _, client := range clients {
		if client == nil {
			continue
		}
		registry.engines[client.EngineID()] = client
	}
	return registry
}

// Resolve returns an engine by id.
func (r *EngineRegistry) Resolve(engineID string) domain.EngineClient {
	if r == nil {
		return nil
	}
	return r.engines[engineID]
}
