package streamer

import "github.com/ompgo-dev/ompgo/pkg/omp/playerobject"

// processObjects streams per-player objects in and out.
func (s *Streamer) processObjects(ps *playerState, cells []*cell) {
	maxVisible := s.cfg.VisibleObjects
	if ps.VisibleObjects > 0 {
		maxVisible = ps.VisibleObjects
	}

	candidates := s.collectObjectCandidates(ps, cells)
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

	for dynamicID, serverObj := range ps.Objects {
		if _, keep := visible[dynamicID]; keep {
			continue
		}
		playerobject.Destroy(ps.Player, serverObj)
		delete(ps.Objects, dynamicID)
		if s.objects[dynamicID] != nil && s.objects[dynamicID].StreamCallbacks {
			s.cfg.EventHandler.OnDynamicObjectStreamOut(dynamicID, ps.PlayerID)
		}
	}

	for dynamicID := range visible {
		if _, exists := ps.Objects[dynamicID]; exists {
			continue
		}
		obj := s.objects[dynamicID]
		if obj == nil {
			continue
		}

		var serverID int32
		serverObj := playerobject.Create(
			ps.Player,
			obj.ModelID,
			obj.Position.X, obj.Position.Y, obj.Position.Z,
			obj.Rotation.X, obj.Rotation.Y, obj.Rotation.Z,
			obj.DrawDistance,
			&serverID,
		)
		if serverObj == nil || !serverObj.Valid() {
			continue
		}

		for idx, mat := range obj.Materials {
			playerobject.SetMaterial(ps.Player, serverObj, idx, mat.ModelID, mat.TXDName, mat.TextureName, mat.Color)
		}
		for idx, materialText := range obj.MaterialTexts {
			playerobject.SetMaterialText(ps.Player, serverObj, materialText.Text, idx, materialText.MaterialSize, materialText.FontFace, materialText.FontSize, materialText.Bold, materialText.FontColor, materialText.BackColor, materialText.TextAlignment)
		}
		if obj.NoCameraCollision {
			playerobject.SetNoCameraCollision(ps.Player, serverObj)
		}

		ps.Objects[dynamicID] = serverObj
		if obj.StreamCallbacks {
			s.cfg.EventHandler.OnDynamicObjectStreamIn(dynamicID, ps.PlayerID)
		}
	}
}

func (s *Streamer) collectObjectCandidates(ps *playerState, cells []*cell) []candidate {
	var candidates []candidate
	for _, currentCell := range cells {
		for id, obj := range currentCell.Objects {
			if ps.isItemDisabled(EntityTypeObject, id) {
				continue
			}
			if !obj.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			if !s.passesAreaCheck(obj.baseEntity, ps.Position) {
				continue
			}
			distSq := DistanceSquared(ps.Position, obj.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypeObject, obj.StreamDistance)
			if distSq > effectiveDist*effectiveDist {
				continue
			}
			candidates = append(candidates, candidate{ID: id, DistSq: distSq, Priority: obj.Priority})
		}
	}
	return candidates
}
