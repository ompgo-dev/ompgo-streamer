package streamer

import (
	"fmt"

	"github.com/ompgo-dev/ompgo/pkg/omp/players"
)

// CreateMapIconParams holds parameters for creating a dynamic map icon.
type CreateMapIconParams struct {
	X, Y, Z        float32
	Type           int32
	Color          uint32
	Style          int32
	StreamDistance float32
	Worlds         []int32
	Interiors      []int32
	Players        []int32
	Areas          []int32
	Priority       int32
}

// CreateMapIcon creates a new dynamic map icon.
func (s *Streamer) CreateMapIcon(params CreateMapIconParams) (*DynamicMapIcon, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultMapIconStreamDistance
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeMapIcon].Next()
	mi := &DynamicMapIcon{
		baseEntity: newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		Type:       params.Type,
		Color:      params.Color,
		Style:      params.Style,
	}

	applyFilters(&mi.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.mapIcons[id] = mi
	s.grid.addMapIcon(mi)
	return mi, nil
}

// DestroyMapIcon removes a dynamic map icon.
func (s *Streamer) DestroyMapIcon(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mi, ok := s.mapIcons[id]
	if !ok {
		return fmt.Errorf("streamer: map icon %d not found", id)
	}

	for _, ps := range s.playerStates {
		if slot, exists := ps.MapIcons[id]; exists {
			players.RemoveMapIcon(ps.Player, slot)
			ps.freeMapIconSlot(slot)
			delete(ps.MapIcons, id)
		}
	}

	s.grid.removeMapIcon(mi)
	delete(s.mapIcons, id)
	return nil
}

// IsValidMapIcon returns true if the map icon ID exists.
func (s *Streamer) IsValidMapIcon(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.mapIcons[id]
	return ok
}

// CountMapIcons returns the number of dynamic map icons.
func (s *Streamer) CountMapIcons() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.mapIcons)
}

// DestroyAllMapIcons removes all dynamic map icons.
func (s *Streamer) DestroyAllMapIcons() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ps := range s.playerStates {
		for id, slot := range ps.MapIcons {
			players.RemoveMapIcon(ps.Player, slot)
			ps.freeMapIconSlot(slot)
			delete(ps.MapIcons, id)
		}
	}
	for id, mi := range s.mapIcons {
		s.grid.removeMapIcon(mi)
		delete(s.mapIcons, id)
	}
}
