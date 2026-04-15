package streamer

import "github.com/ompgo-dev/ompgo/pkg/omp/pickup"

// processPickups manages global pickup streaming.
func (s *Streamer) processPickups(ps *playerState, cells []*cell) {
	for _, currentCell := range cells {
		for id, item := range currentCell.Pickups {
			if ps.isItemDisabled(EntityTypePickup, id) {
				continue
			}
			if !item.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}

			distSq := DistanceSquared(ps.Position, item.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypePickup, item.StreamDistance)
			if distSq > effectiveDist*effectiveDist {
				continue
			}
			if _, exists := s.activePickups[id]; exists {
				continue
			}

			var serverID int32
			virtualWorld := int32(0)
			if _, anyWorld := item.Worlds[AnyWorld]; !anyWorld {
				for worldID := range item.Worlds {
					virtualWorld = worldID
					break
				}
			}

			serverPickup := pickup.Create(item.ModelID, item.Type, item.Position.X, item.Position.Y, item.Position.Z, virtualWorld, &serverID)
			if serverPickup != nil && serverPickup.Valid() {
				s.activePickups[id] = serverPickup
			}
		}
	}

	for dynamicID, serverPickup := range s.activePickups {
		dynamicPickup, ok := s.pickups[dynamicID]
		if !ok {
			pickup.Destroy(serverPickup)
			delete(s.activePickups, dynamicID)
			continue
		}

		anyNearby := false
		for _, otherPS := range s.playerStates {
			distSq := DistanceSquared(otherPS.Position, dynamicPickup.Position)
			effectiveDist := otherPS.effectiveStreamDist(EntityTypePickup, dynamicPickup.StreamDistance)
			if distSq <= effectiveDist*effectiveDist && dynamicPickup.isVisibleTo(otherPS.PlayerID, otherPS.WorldID, otherPS.Interior) {
				anyNearby = true
				break
			}
		}
		if anyNearby {
			continue
		}

		pickup.Destroy(serverPickup)
		delete(s.activePickups, dynamicID)
	}
}
