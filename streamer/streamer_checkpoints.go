package streamer

import (
	"fmt"

	"github.com/ompgo-dev/ompgo/pkg/omp/checkpoint"
	"github.com/ompgo-dev/ompgo/pkg/omp/racecheckpoint"
)

// CreateCheckpointParams holds parameters for creating a dynamic checkpoint.
type CreateCheckpointParams struct {
	X, Y, Z        float32
	Size           float32
	StreamDistance float32
	Worlds         []int32
	Interiors      []int32
	Players        []int32
	Areas          []int32
	Priority       int32
}

// CreateCheckpoint creates a new dynamic checkpoint.
func (s *Streamer) CreateCheckpoint(params CreateCheckpointParams) (*DynamicCheckpoint, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultCPStreamDistance
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeCheckpoint].Next()
	cp := &DynamicCheckpoint{
		baseEntity: newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		Size:       params.Size,
	}

	applyFilters(&cp.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.checkpoints[id] = cp
	s.grid.addCheckpoint(cp)
	return cp, nil
}

// DestroyCheckpoint removes a dynamic checkpoint.
func (s *Streamer) DestroyCheckpoint(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, ok := s.checkpoints[id]
	if !ok {
		return fmt.Errorf("streamer: checkpoint %d not found", id)
	}

	for _, ps := range s.playerStates {
		if ps.VisibleCheckpoint == id {
			checkpoint.Disable(ps.Player)
			ps.VisibleCheckpoint = InvalidID
		}
	}

	s.grid.removeCheckpoint(cp)
	delete(s.checkpoints, id)
	return nil
}

// IsValidCheckpoint returns true if the checkpoint ID exists.
func (s *Streamer) IsValidCheckpoint(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.checkpoints[id]
	return ok
}

// IsPlayerInDynamicCP returns true if the player is in the specified checkpoint.
func (s *Streamer) IsPlayerInDynamicCP(playerID, cpID int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return false
	}
	if ps.VisibleCheckpoint != cpID {
		return false
	}
	return checkpoint.IsPlayerIn(ps.Player)
}

// GetPlayerVisibleDynamicCP returns the currently visible dynamic checkpoint for a player.
func (s *Streamer) GetPlayerVisibleDynamicCP(playerID int32) int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return InvalidID
	}
	return ps.VisibleCheckpoint
}

// CountCheckpoints returns the number of dynamic checkpoints.
func (s *Streamer) CountCheckpoints() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.checkpoints)
}

// TogglePlayerDynamicCP toggles a dynamic checkpoint for a specific player.
func (s *Streamer) TogglePlayerDynamicCP(playerID, cpID int32, toggle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return
	}
	if toggle {
		delete(ps.DisabledItems[EntityTypeCheckpoint], cpID)
	} else {
		ps.DisabledItems[EntityTypeCheckpoint][cpID] = struct{}{}
	}
}

// DestroyAllCheckpoints removes all dynamic checkpoints.
func (s *Streamer) DestroyAllCheckpoints() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ps := range s.playerStates {
		if ps.VisibleCheckpoint != InvalidID {
			checkpoint.Disable(ps.Player)
			ps.VisibleCheckpoint = InvalidID
		}
	}
	for id, cp := range s.checkpoints {
		s.grid.removeCheckpoint(cp)
		delete(s.checkpoints, id)
	}
}

// CreateRaceCheckpointParams holds parameters for creating a dynamic race checkpoint.
type CreateRaceCheckpointParams struct {
	Type                int32
	X, Y, Z             float32
	NextX, NextY, NextZ float32
	Size                float32
	StreamDistance      float32
	Worlds              []int32
	Interiors           []int32
	Players             []int32
	Areas               []int32
	Priority            int32
}

// CreateRaceCheckpoint creates a new dynamic race checkpoint.
func (s *Streamer) CreateRaceCheckpoint(params CreateRaceCheckpointParams) (*DynamicRaceCheckpoint, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultRaceCPStreamDistance
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeRaceCheckpoint].Next()
	rcp := &DynamicRaceCheckpoint{
		baseEntity: newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		Type:       params.Type,
		Next:       Vector3{X: params.NextX, Y: params.NextY, Z: params.NextZ},
		Size:       params.Size,
	}

	applyFilters(&rcp.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.raceCheckpoints[id] = rcp
	s.grid.addRaceCheckpoint(rcp)
	return rcp, nil
}

// DestroyRaceCheckpoint removes a dynamic race checkpoint.
func (s *Streamer) DestroyRaceCheckpoint(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rcp, ok := s.raceCheckpoints[id]
	if !ok {
		return fmt.Errorf("streamer: race checkpoint %d not found", id)
	}

	for _, ps := range s.playerStates {
		if ps.VisibleRaceCheckpoint == id {
			racecheckpoint.Disable(ps.Player)
			ps.VisibleRaceCheckpoint = InvalidID
		}
	}

	s.grid.removeRaceCheckpoint(rcp)
	delete(s.raceCheckpoints, id)
	return nil
}

// IsValidRaceCheckpoint returns true if the race checkpoint ID exists.
func (s *Streamer) IsValidRaceCheckpoint(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.raceCheckpoints[id]
	return ok
}

// IsPlayerInDynamicRaceCP returns true if the player is in the specified race checkpoint.
func (s *Streamer) IsPlayerInDynamicRaceCP(playerID, rcpID int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return false
	}
	if ps.VisibleRaceCheckpoint != rcpID {
		return false
	}
	return racecheckpoint.IsPlayerIn(ps.Player)
}

// GetPlayerVisibleDynamicRaceCP returns the currently visible dynamic race checkpoint.
func (s *Streamer) GetPlayerVisibleDynamicRaceCP(playerID int32) int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return InvalidID
	}
	return ps.VisibleRaceCheckpoint
}

// CountRaceCheckpoints returns the number of dynamic race checkpoints.
func (s *Streamer) CountRaceCheckpoints() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.raceCheckpoints)
}

// TogglePlayerDynamicRaceCP toggles a dynamic race checkpoint for a specific player.
func (s *Streamer) TogglePlayerDynamicRaceCP(playerID, rcpID int32, toggle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return
	}
	if toggle {
		delete(ps.DisabledItems[EntityTypeRaceCheckpoint], rcpID)
	} else {
		ps.DisabledItems[EntityTypeRaceCheckpoint][rcpID] = struct{}{}
	}
}

// DestroyAllRaceCheckpoints removes all dynamic race checkpoints.
func (s *Streamer) DestroyAllRaceCheckpoints() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ps := range s.playerStates {
		if ps.VisibleRaceCheckpoint != InvalidID {
			racecheckpoint.Disable(ps.Player)
			ps.VisibleRaceCheckpoint = InvalidID
		}
	}
	for id, rcp := range s.raceCheckpoints {
		s.grid.removeRaceCheckpoint(rcp)
		delete(s.raceCheckpoints, id)
	}
}
