package streamer

import (
	"math"

	"github.com/ompgo-dev/ompgo/pkg/omp/checkpoint"
	"github.com/ompgo-dev/ompgo/pkg/omp/racecheckpoint"
)

// processCheckpoints shows the nearest checkpoint per player.
func (s *Streamer) processCheckpoints(ps *playerState, cells []*cell) {
	bestID := int32(InvalidID)
	bestDistSq := float32(math.MaxFloat32)
	bestPriority := int32(math.MinInt32)

	for _, currentCell := range cells {
		for id, cp := range currentCell.Checkpoints {
			if ps.isItemDisabled(EntityTypeCheckpoint, id) {
				continue
			}
			if !cp.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			if !s.passesAreaCheck(cp.baseEntity, ps.Position) {
				continue
			}
			distSq := DistanceSquared(ps.Position, cp.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypeCheckpoint, cp.StreamDistance)
			if distSq > effectiveDist*effectiveDist {
				continue
			}

			if cp.Priority > bestPriority || (cp.Priority == bestPriority && distSq < bestDistSq) {
				bestID = id
				bestDistSq = distSq
				bestPriority = cp.Priority
			}
		}
	}

	if bestID == ps.VisibleCheckpoint {
		return
	}
	if ps.VisibleCheckpoint != InvalidID {
		checkpoint.Disable(ps.Player)
	}
	if bestID != InvalidID {
		cp := s.checkpoints[bestID]
		checkpoint.Set(ps.Player, cp.Position.X, cp.Position.Y, cp.Position.Z, cp.Size)
	}
	ps.VisibleCheckpoint = bestID
}

// processRaceCheckpoints shows the nearest race checkpoint per player.
func (s *Streamer) processRaceCheckpoints(ps *playerState, cells []*cell) {
	bestID := int32(InvalidID)
	bestDistSq := float32(math.MaxFloat32)
	bestPriority := int32(math.MinInt32)

	for _, currentCell := range cells {
		for id, rcp := range currentCell.RaceCheckpoints {
			if ps.isItemDisabled(EntityTypeRaceCheckpoint, id) {
				continue
			}
			if !rcp.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			if !s.passesAreaCheck(rcp.baseEntity, ps.Position) {
				continue
			}
			distSq := DistanceSquared(ps.Position, rcp.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypeRaceCheckpoint, rcp.StreamDistance)
			if distSq > effectiveDist*effectiveDist {
				continue
			}

			if rcp.Priority > bestPriority || (rcp.Priority == bestPriority && distSq < bestDistSq) {
				bestID = id
				bestDistSq = distSq
				bestPriority = rcp.Priority
			}
		}
	}

	if bestID == ps.VisibleRaceCheckpoint {
		return
	}
	if ps.VisibleRaceCheckpoint != InvalidID {
		racecheckpoint.Disable(ps.Player)
	}
	if bestID != InvalidID {
		rcp := s.raceCheckpoints[bestID]
		racecheckpoint.Set(ps.Player, rcp.Type, rcp.Position.X, rcp.Position.Y, rcp.Position.Z, rcp.Next.X, rcp.Next.Y, rcp.Next.Z, rcp.Size)
	}
	ps.VisibleRaceCheckpoint = bestID
}
