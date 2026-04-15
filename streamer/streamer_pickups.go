package streamer

import (
	"fmt"

	"github.com/ompgo-dev/ompgo/pkg/omp/pickup"
)

// CreatePickupParams holds parameters for creating a dynamic pickup.
type CreatePickupParams struct {
	ModelID        int32
	Type           int32
	X, Y, Z        float32
	StreamDistance float32
	Worlds         []int32
	Interiors      []int32
	Players        []int32
	Areas          []int32
	Priority       int32
}

// CreatePickup creates a new dynamic pickup.
func (s *Streamer) CreatePickup(params CreatePickupParams) (*DynamicPickup, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultPickupStreamDistance
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypePickup].Next()
	p := &DynamicPickup{
		baseEntity: newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		ModelID:    params.ModelID,
		Type:       params.Type,
	}

	applyFilters(&p.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.pickups[id] = p
	s.grid.addPickup(p)
	return p, nil
}

// DestroyPickup removes a dynamic pickup.
func (s *Streamer) DestroyPickup(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pickups[id]
	if !ok {
		return fmt.Errorf("streamer: pickup %d not found", id)
	}

	if serverPickup, exists := s.activePickups[id]; exists {
		pickup.Destroy(serverPickup)
		delete(s.activePickups, id)
	}

	s.grid.removePickup(p)
	delete(s.pickups, id)
	return nil
}

// IsValidPickup returns true if the pickup ID exists.
func (s *Streamer) IsValidPickup(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.pickups[id]
	return ok
}

// CountPickups returns the number of dynamic pickups.
func (s *Streamer) CountPickups() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pickups)
}

// DestroyAllPickups removes all dynamic pickups.
func (s *Streamer) DestroyAllPickups() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, serverPickup := range s.activePickups {
		pickup.Destroy(serverPickup)
		delete(s.activePickups, id)
	}
	for id, p := range s.pickups {
		s.grid.removePickup(p)
		delete(s.pickups, id)
	}
}
