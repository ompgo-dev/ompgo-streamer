package streamer

import (
	"sort"

	"github.com/ompgo-dev/ompgo/pkg/omp/actor"
)

// processAreas checks which areas the player is inside and fires enter or leave events.
func (s *Streamer) processAreas(ps *playerState, cells []*cell) {
	currentAreas := make(map[int32]struct{})

	for _, currentCell := range cells {
		for id, area := range currentCell.Areas {
			if ps.isItemDisabled(EntityTypeArea, id) {
				continue
			}
			if !area.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			if area.Shape.ContainsPoint(ps.Position.X, ps.Position.Y, ps.Position.Z) {
				currentAreas[id] = struct{}{}
			}
		}
	}

	for areaID := range ps.InAreas {
		if _, stillIn := currentAreas[areaID]; !stillIn {
			s.cfg.EventHandler.OnPlayerLeaveDynamicArea(ps.PlayerID, areaID)
		}
	}
	for areaID := range currentAreas {
		if _, wasIn := ps.InAreas[areaID]; !wasIn {
			s.cfg.EventHandler.OnPlayerEnterDynamicArea(ps.PlayerID, areaID)
		}
	}

	ps.InAreas = currentAreas
}

// processActors manages global actor streaming.
func (s *Streamer) processActors(ps *playerState, cells []*cell) {
	for _, currentCell := range cells {
		for id, actorData := range currentCell.Actors {
			if ps.isItemDisabled(EntityTypeActor, id) {
				continue
			}
			if !actorData.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}

			distSq := DistanceSquared(ps.Position, actorData.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypeActor, actorData.StreamDistance)
			if distSq > effectiveDist*effectiveDist {
				continue
			}
			if _, exists := s.activeActors[id]; exists {
				continue
			}

			var serverID int32
			serverActor := actor.Create(actorData.ModelID, actorData.Position.X, actorData.Position.Y, actorData.Position.Z, actorData.Rotation, &serverID)
			if serverActor == nil || !serverActor.Valid() {
				continue
			}

			s.activeActors[id] = serverActor
			actor.SetInvulnerable(serverActor, actorData.Invulnerable)
			actor.SetHealth(serverActor, actorData.Health)
			if actorData.Animation != nil {
				actor.ApplyAnimation(serverActor, actorData.Animation.Name, actorData.Animation.Library, actorData.Animation.Delta, actorData.Animation.Loop, actorData.Animation.LockX, actorData.Animation.LockY, actorData.Animation.Freeze, actorData.Animation.Time)
			}
			s.cfg.EventHandler.OnDynamicActorStreamIn(id, ps.PlayerID)
		}
	}

	for dynamicID, serverActor := range s.activeActors {
		dynamicActor, ok := s.actors[dynamicID]
		if !ok {
			actor.Destroy(serverActor)
			delete(s.activeActors, dynamicID)
			continue
		}

		anyNearby := false
		for _, otherPS := range s.playerStates {
			distSq := DistanceSquared(otherPS.Position, dynamicActor.Position)
			effectiveDist := otherPS.effectiveStreamDist(EntityTypeActor, dynamicActor.StreamDistance)
			if distSq <= effectiveDist*effectiveDist && dynamicActor.isVisibleTo(otherPS.PlayerID, otherPS.WorldID, otherPS.Interior) {
				anyNearby = true
				break
			}
		}
		if anyNearby {
			continue
		}

		s.cfg.EventHandler.OnDynamicActorStreamOut(dynamicID, ps.PlayerID)
		actor.Destroy(serverActor)
		delete(s.activeActors, dynamicID)
	}
}

// passesAreaCheck verifies that the player is in or not in the entity's required areas.
func (s *Streamer) passesAreaCheck(e baseEntity, playerPos Vector3) bool {
	if len(e.Areas) == 0 {
		return true
	}

	for areaID := range e.Areas {
		area, ok := s.areas[areaID]
		if !ok {
			continue
		}
		inArea := area.Shape.ContainsPoint(playerPos.X, playerPos.Y, playerPos.Z)
		if e.InverseAreaCheck {
			if inArea {
				return false
			}
			continue
		}
		if inArea {
			return true
		}
	}

	return e.InverseAreaCheck
}

// sortCandidates sorts by priority descending then distance ascending.
func sortCandidates(candidates []candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].DistSq < candidates[j].DistSq
	})
}
