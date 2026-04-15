package streamer

import (
	"fmt"

	"github.com/ompgo-dev/ompgo/pkg/omp/core"
	"github.com/ompgo-dev/ompgo/pkg/omp/players"
)

// SetTickRate updates the global tick rate.
func (s *Streamer) SetTickRate(rate int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rate > 0 {
		s.cfg.TickRate = rate
	}
}

// GetTickRate returns the current global tick rate.
func (s *Streamer) GetTickRate() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.TickRate
}

// SetPlayerTickRate sets a per-player tick rate override.
func (s *Streamer) SetPlayerTickRate(playerID, rate int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ps, ok := s.playerStates[playerID]; ok {
		ps.TickRate = rate
	}
}

// SetRadiusMultiplier sets the radius multiplier for a player and entity type.
func (s *Streamer) SetRadiusMultiplier(playerID int32, entityType EntityType, multiplier float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ps, ok := s.playerStates[playerID]; ok {
		if entityType >= 0 && int(entityType) < MaxEntityTypes {
			ps.RadiusMultiplier[entityType] = multiplier
		}
	}
}

// GetRadiusMultiplier returns the radius multiplier for a player and entity type.
func (s *Streamer) GetRadiusMultiplier(playerID int32, entityType EntityType) float32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ps, ok := s.playerStates[playerID]; ok {
		if entityType >= 0 && int(entityType) < MaxEntityTypes {
			return ps.RadiusMultiplier[entityType]
		}
	}
	return 1.0
}

// TogglePlayerItem toggles a specific item type for a player.
func (s *Streamer) TogglePlayerItem(playerID int32, entityType EntityType, id int32, toggle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return
	}
	if entityType < 0 || int(entityType) >= MaxEntityTypes {
		return
	}
	if toggle {
		delete(ps.DisabledItems[entityType], id)
	} else {
		ps.DisabledItems[entityType][id] = struct{}{}
	}
}

// GetDistanceToItem returns the distance from a point to a dynamic entity.
func (s *Streamer) GetDistanceToItem(x, y, z float32, entityType EntityType, id int32) (float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pos Vector3
	switch entityType {
	case EntityTypeObject:
		if obj, ok := s.objects[id]; ok {
			pos = obj.Position
		} else {
			return 0, fmt.Errorf("streamer: object %d not found", id)
		}
	case EntityTypePickup:
		if pickupItem, ok := s.pickups[id]; ok {
			pos = pickupItem.Position
		} else {
			return 0, fmt.Errorf("streamer: pickup %d not found", id)
		}
	case EntityTypeCheckpoint:
		if checkpointItem, ok := s.checkpoints[id]; ok {
			pos = checkpointItem.Position
		} else {
			return 0, fmt.Errorf("streamer: checkpoint %d not found", id)
		}
	case EntityTypeRaceCheckpoint:
		if raceCheckpointItem, ok := s.raceCheckpoints[id]; ok {
			pos = raceCheckpointItem.Position
		} else {
			return 0, fmt.Errorf("streamer: race checkpoint %d not found", id)
		}
	case EntityTypeMapIcon:
		if mapIcon, ok := s.mapIcons[id]; ok {
			pos = mapIcon.Position
		} else {
			return 0, fmt.Errorf("streamer: map icon %d not found", id)
		}
	case EntityTypeTextLabel:
		if textLabel, ok := s.textLabels[id]; ok {
			pos = textLabel.Position
		} else {
			return 0, fmt.Errorf("streamer: text label %d not found", id)
		}
	case EntityTypeArea:
		if area, ok := s.areas[id]; ok {
			pos = area.Shape.Center()
		} else {
			return 0, fmt.Errorf("streamer: area %d not found", id)
		}
	case EntityTypeActor:
		if actorData, ok := s.actors[id]; ok {
			pos = actorData.Position
		} else {
			return 0, fmt.Errorf("streamer: actor %d not found", id)
		}
	default:
		return 0, fmt.Errorf("streamer: invalid entity type %d", entityType)
	}

	return distance3D(x, y, z, pos.X, pos.Y, pos.Z), nil
}

// UpdateEx forces an immediate position update and re-stream for a player.
func (s *Streamer) UpdateEx(playerID int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return
	}

	var x, y, z float32
	players.GetPos(ps.Player, &x, &y, &z)
	ps.Position = Vector3{X: x, Y: y, Z: z}
	ps.WorldID = players.GetVirtualWorld(ps.Player)
	ps.Interior = players.GetInterior(ps.Player)
	s.processPlayerUpdate(ps)
}

// Update forces an update for all players.
func (s *Streamer) Update() {
	s.mu.Lock()
	defer s.mu.Unlock()

	maxPlayers := core.MaxPlayers()
	for playerID := int32(0); playerID < maxPlayers; playerID++ {
		ps, ok := s.playerStates[playerID]
		if !ok {
			continue
		}
		var x, y, z float32
		players.GetPos(ps.Player, &x, &y, &z)
		ps.Position = Vector3{X: x, Y: y, Z: z}
		ps.WorldID = players.GetVirtualWorld(ps.Player)
		ps.Interior = players.GetInterior(ps.Player)
		s.processPlayerUpdate(ps)
	}
}
