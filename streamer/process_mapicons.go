package streamer

import "github.com/ompgo-dev/ompgo/pkg/omp/players"

// processMapIcons streams per-player map icons in and out.
func (s *Streamer) processMapIcons(ps *playerState, cells []*cell) {
	maxVisible := s.cfg.VisibleMapIcons
	if int32(len(ps.MapIcons)) >= maxVisible && len(s.mapIcons) == 0 {
		return
	}

	candidates := s.collectMapIconCandidates(ps, cells)
	sortCandidates(candidates)

	visible := make(map[int32]struct{}, len(candidates))
	count := int32(0)
	for _, candidate := range candidates {
		if count >= maxVisible {
			break
		}
		visible[candidate.ID] = struct{}{}
		count++
	}

	for dynamicID, slot := range ps.MapIcons {
		if _, keep := visible[dynamicID]; keep {
			continue
		}
		players.RemoveMapIcon(ps.Player, slot)
		ps.freeMapIconSlot(slot)
		delete(ps.MapIcons, dynamicID)
	}

	for dynamicID := range visible {
		if _, exists := ps.MapIcons[dynamicID]; exists {
			continue
		}
		mapIcon := s.mapIcons[dynamicID]
		if mapIcon == nil {
			continue
		}

		slot := ps.allocMapIconSlot()
		if slot >= maxVisible {
			ps.freeMapIconSlot(slot)
			break
		}

		players.SetMapIcon(ps.Player, slot, mapIcon.Position.X, mapIcon.Position.Y, mapIcon.Position.Z, mapIcon.Type, mapIcon.Color, mapIcon.Style)
		ps.MapIcons[dynamicID] = slot
	}
}

func (s *Streamer) collectMapIconCandidates(ps *playerState, cells []*cell) []candidate {
	var candidates []candidate
	for _, currentCell := range cells {
		for id, mapIcon := range currentCell.MapIcons {
			if ps.isItemDisabled(EntityTypeMapIcon, id) {
				continue
			}
			if !mapIcon.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			if !s.passesAreaCheck(mapIcon.baseEntity, ps.Position) {
				continue
			}
			distSq := DistanceSquared(ps.Position, mapIcon.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypeMapIcon, mapIcon.StreamDistance)
			if distSq > effectiveDist*effectiveDist {
				continue
			}
			candidates = append(candidates, candidate{ID: id, DistSq: distSq, Priority: mapIcon.Priority})
		}
	}
	return candidates
}
