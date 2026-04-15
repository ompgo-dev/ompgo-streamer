package streamer

import (
	"math"
	"sort"

	"github.com/ompgo-dev/ompgo/pkg/handle"
	"github.com/ompgo-dev/ompgo/pkg/omp"
	"github.com/ompgo-dev/ompgo/pkg/omp/actor"
	"github.com/ompgo-dev/ompgo/pkg/omp/checkpoint"
	"github.com/ompgo-dev/ompgo/pkg/omp/pickup"
	"github.com/ompgo-dev/ompgo/pkg/omp/playerobject"
	"github.com/ompgo-dev/ompgo/pkg/omp/players"
	"github.com/ompgo-dev/ompgo/pkg/omp/playertextlabel"
	"github.com/ompgo-dev/ompgo/pkg/omp/racecheckpoint"
)

// candidate is an entity with a computed distance from the player.
type candidate struct {
	ID       int32
	DistSq   float32
	Priority int32
}

// processPlayerUpdate runs the streaming logic for a single player.
// Must be called with s.mu held.
func (s *Streamer) processPlayerUpdate(ps *playerState) {
	cells := s.grid.nearbyCells(ps.Position.X, ps.Position.Y)

	for _, entityType := range s.cfg.TypePriority {
		switch entityType {
		case EntityTypeObject:
			s.processObjects(ps, cells)
		case EntityTypePickup:
			s.processPickups(ps, cells)
		case EntityTypeCheckpoint:
			s.processCheckpoints(ps, cells)
		case EntityTypeRaceCheckpoint:
			s.processRaceCheckpoints(ps, cells)
		case EntityTypeMapIcon:
			s.processMapIcons(ps, cells)
		case EntityTypeTextLabel:
			s.processTextLabels(ps, cells)
		case EntityTypeArea:
			s.processAreas(ps, cells)
		case EntityTypeActor:
			s.processActors(ps, cells)
		}
	}
}

// processObjects streams per-player objects in and out.
func (s *Streamer) processObjects(ps *playerState, cells []*cell) {
	maxVisible := s.cfg.VisibleObjects
	if ps.VisibleObjects > 0 {
		maxVisible = ps.VisibleObjects
	}

	candidates := s.collectObjectCandidates(ps, cells)
	sortCandidates(candidates)

	// Determine which dynamic IDs should be visible.
	visible := make(map[int32]struct{}, len(candidates))
	count := int32(0)
	for _, c := range candidates {
		if count >= maxVisible {
			break
		}
		visible[c.ID] = struct{}{}
		count++
	}

	// Stream out objects no longer visible.
	for dynamicID, serverObj := range ps.Objects {
		if _, keep := visible[dynamicID]; !keep {
			playerobject.Destroy(ps.Player, serverObj)
			delete(ps.Objects, dynamicID)
			if s.objects[dynamicID] != nil && s.objects[dynamicID].StreamCallbacks {
				s.cfg.EventHandler.OnDynamicObjectStreamOut(dynamicID, ps.PlayerID)
			}
		}
	}

	// Stream in new visible objects.
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

		// Apply materials.
		for idx, mat := range obj.Materials {
			playerobject.SetMaterial(ps.Player, serverObj, idx, mat.ModelID, mat.TXDName, mat.TextureName, mat.Color)
		}
		for idx, mt := range obj.MaterialTexts {
			playerobject.SetMaterialText(ps.Player, serverObj, mt.Text, idx, mt.MaterialSize, mt.FontFace, mt.FontSize, mt.Bold, mt.FontColor, mt.BackColor, mt.TextAlignment)
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
	for _, c := range cells {
		for id, obj := range c.Objects {
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

// processPickups manages global pickup streaming.
func (s *Streamer) processPickups(ps *playerState, cells []*cell) {
	for _, c := range cells {
		for id, p := range c.Pickups {
			if ps.isItemDisabled(EntityTypePickup, id) {
				continue
			}
			if !p.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			distSq := DistanceSquared(ps.Position, p.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypePickup, p.StreamDistance)

			if distSq <= effectiveDist*effectiveDist {
				// Should be active - create if not.
				if _, exists := s.activePickups[id]; !exists {
					var serverID int32
					vw := int32(0)
					if _, anyWorld := p.Worlds[AnyWorld]; !anyWorld {
						for w := range p.Worlds {
							vw = w
							break
						}
					}
					sp := pickup.Create(p.ModelID, p.Type, p.Position.X, p.Position.Y, p.Position.Z, vw, &serverID)
					if sp != nil && sp.Valid() {
						s.activePickups[id] = sp
					}
				}
			}
		}
	}

	// Check if any active pickup should be destroyed (no player nearby).
	for dynamicID, sp := range s.activePickups {
		dp, ok := s.pickups[dynamicID]
		if !ok {
			pickup.Destroy(sp)
			delete(s.activePickups, dynamicID)
			continue
		}

		anyNearby := false
		for _, otherPS := range s.playerStates {
			distSq := DistanceSquared(otherPS.Position, dp.Position)
			effectiveDist := otherPS.effectiveStreamDist(EntityTypePickup, dp.StreamDistance)
			if distSq <= effectiveDist*effectiveDist && dp.isVisibleTo(otherPS.PlayerID, otherPS.WorldID, otherPS.Interior) {
				anyNearby = true
				break
			}
		}
		if !anyNearby {
			pickup.Destroy(sp)
			delete(s.activePickups, dynamicID)
		}
	}
}

// processCheckpoints shows the nearest checkpoint per player.
func (s *Streamer) processCheckpoints(ps *playerState, cells []*cell) {
	var bestID int32 = InvalidID
	var bestDistSq float32 = float32(math.MaxFloat32)
	var bestPriority int32 = math.MinInt32

	for _, c := range cells {
		for id, cp := range c.Checkpoints {
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

	if bestID != ps.VisibleCheckpoint {
		if ps.VisibleCheckpoint != InvalidID {
			checkpoint.Disable(ps.Player)
		}
		if bestID != InvalidID {
			cp := s.checkpoints[bestID]
			checkpoint.Set(ps.Player, cp.Position.X, cp.Position.Y, cp.Position.Z, cp.Size)
		}
		ps.VisibleCheckpoint = bestID
	}
}

// processRaceCheckpoints shows the nearest race checkpoint per player.
func (s *Streamer) processRaceCheckpoints(ps *playerState, cells []*cell) {
	var bestID int32 = InvalidID
	var bestDistSq float32 = float32(math.MaxFloat32)
	var bestPriority int32 = math.MinInt32

	for _, c := range cells {
		for id, rcp := range c.RaceCheckpoints {
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

	if bestID != ps.VisibleRaceCheckpoint {
		if ps.VisibleRaceCheckpoint != InvalidID {
			racecheckpoint.Disable(ps.Player)
		}
		if bestID != InvalidID {
			rcp := s.raceCheckpoints[bestID]
			racecheckpoint.Set(ps.Player, rcp.Type, rcp.Position.X, rcp.Position.Y, rcp.Position.Z, rcp.Next.X, rcp.Next.Y, rcp.Next.Z, rcp.Size)
		}
		ps.VisibleRaceCheckpoint = bestID
	}
}

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
	for _, c := range candidates {
		if count >= maxVisible {
			break
		}
		visible[c.ID] = struct{}{}
		count++
	}

	// Stream out.
	for dynamicID, slot := range ps.MapIcons {
		if _, keep := visible[dynamicID]; !keep {
			players.RemoveMapIcon(ps.Player, slot)
			ps.freeMapIconSlot(slot)
			delete(ps.MapIcons, dynamicID)
		}
	}

	// Stream in.
	for dynamicID := range visible {
		if _, exists := ps.MapIcons[dynamicID]; exists {
			continue
		}
		mi := s.mapIcons[dynamicID]
		if mi == nil {
			continue
		}

		slot := ps.allocMapIconSlot()
		if slot >= maxVisible {
			ps.freeMapIconSlot(slot)
			break
		}

		players.SetMapIcon(ps.Player, slot, mi.Position.X, mi.Position.Y, mi.Position.Z, mi.Type, mi.Color, mi.Style)
		ps.MapIcons[dynamicID] = slot
	}
}

func (s *Streamer) collectMapIconCandidates(ps *playerState, cells []*cell) []candidate {
	var candidates []candidate
	for _, c := range cells {
		for id, mi := range c.MapIcons {
			if ps.isItemDisabled(EntityTypeMapIcon, id) {
				continue
			}
			if !mi.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			if !s.passesAreaCheck(mi.baseEntity, ps.Position) {
				continue
			}
			distSq := DistanceSquared(ps.Position, mi.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypeMapIcon, mi.StreamDistance)
			if distSq > effectiveDist*effectiveDist {
				continue
			}
			candidates = append(candidates, candidate{ID: id, DistSq: distSq, Priority: mi.Priority})
		}
	}
	return candidates
}

// processTextLabels streams per-player text labels in and out.
func (s *Streamer) processTextLabels(ps *playerState, cells []*cell) {
	maxVisible := s.cfg.VisibleTextLabels
	if ps.VisibleTextLabels > 0 {
		maxVisible = ps.VisibleTextLabels
	}

	candidates := s.collectTextLabelCandidates(ps, cells)
	sortCandidates(candidates)

	visible := make(map[int32]struct{}, len(candidates))
	count := int32(0)
	for _, c := range candidates {
		if count >= maxVisible {
			break
		}
		visible[c.ID] = struct{}{}
		count++
	}

	// Stream out.
	for dynamicID, serverTL := range ps.TextLabels {
		if _, keep := visible[dynamicID]; !keep {
			playertextlabel.Destroy(ps.Player, serverTL)
			delete(ps.TextLabels, dynamicID)
		}
	}

	// Stream in.
	for dynamicID := range visible {
		if _, exists := ps.TextLabels[dynamicID]; exists {
			continue
		}
		tl := s.textLabels[dynamicID]
		if tl == nil {
			continue
		}

		var attachedPlayer *omp.Player
		var attachedVehicle *omp.Vehicle

		if tl.AttachedPlayerID >= 0 {
			attachedPlayer = players.FromID(tl.AttachedPlayerID)
		}
		// Vehicle attachment can be added when vehicle helpers are available.

		if attachedPlayer == nil {
			attachedPlayer = omp.NewPlayer(handle.Handle(0))
		}
		if attachedVehicle == nil {
			attachedVehicle = omp.NewVehicle(handle.Handle(0))
		}

		var serverID int32
		serverTL := playertextlabel.Create(
			ps.Player,
			tl.Text,
			tl.Color,
			tl.Position.X, tl.Position.Y, tl.Position.Z,
			tl.DrawDistance,
			attachedPlayer,
			attachedVehicle,
			tl.TestLOS,
			&serverID,
		)
		if serverTL == nil || !serverTL.Valid() {
			continue
		}

		ps.TextLabels[dynamicID] = serverTL
	}
}

func (s *Streamer) collectTextLabelCandidates(ps *playerState, cells []*cell) []candidate {
	var candidates []candidate
	for _, c := range cells {
		for id, tl := range c.TextLabels {
			if ps.isItemDisabled(EntityTypeTextLabel, id) {
				continue
			}
			if !tl.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			if !s.passesAreaCheck(tl.baseEntity, ps.Position) {
				continue
			}
			distSq := DistanceSquared(ps.Position, tl.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypeTextLabel, tl.StreamDistance)
			if distSq > effectiveDist*effectiveDist {
				continue
			}
			candidates = append(candidates, candidate{ID: id, DistSq: distSq, Priority: tl.Priority})
		}
	}
	return candidates
}

// processAreas checks which areas the player is inside and fires enter/leave events.
func (s *Streamer) processAreas(ps *playerState, cells []*cell) {
	currentAreas := make(map[int32]struct{})

	for _, c := range cells {
		for id, area := range c.Areas {
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

	// Fire leave events for areas the player is no longer in.
	for areaID := range ps.InAreas {
		if _, stillIn := currentAreas[areaID]; !stillIn {
			s.cfg.EventHandler.OnPlayerLeaveDynamicArea(ps.PlayerID, areaID)
		}
	}

	// Fire enter events for areas the player just entered.
	for areaID := range currentAreas {
		if _, wasIn := ps.InAreas[areaID]; !wasIn {
			s.cfg.EventHandler.OnPlayerEnterDynamicArea(ps.PlayerID, areaID)
		}
	}

	ps.InAreas = currentAreas
}

// processActors manages global actor streaming.
func (s *Streamer) processActors(ps *playerState, cells []*cell) {
	for _, c := range cells {
		for id, a := range c.Actors {
			if ps.isItemDisabled(EntityTypeActor, id) {
				continue
			}
			if !a.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			distSq := DistanceSquared(ps.Position, a.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypeActor, a.StreamDistance)

			if distSq <= effectiveDist*effectiveDist {
				if _, exists := s.activeActors[id]; !exists {
					var serverID int32
					sa := actor.Create(a.ModelID, a.Position.X, a.Position.Y, a.Position.Z, a.Rotation, &serverID)
					if sa != nil && sa.Valid() {
						s.activeActors[id] = sa
						actor.SetInvulnerable(sa, a.Invulnerable)
						actor.SetHealth(sa, a.Health)
						if a.Animation != nil {
							actor.ApplyAnimation(sa, a.Animation.Name, a.Animation.Library, a.Animation.Delta, a.Animation.Loop, a.Animation.LockX, a.Animation.LockY, a.Animation.Freeze, a.Animation.Time)
						}
						s.cfg.EventHandler.OnDynamicActorStreamIn(id, ps.PlayerID)
					}
				}
			}
		}
	}

	// Check if any active actor should be destroyed.
	for dynamicID, sa := range s.activeActors {
		da, ok := s.actors[dynamicID]
		if !ok {
			actor.Destroy(sa)
			delete(s.activeActors, dynamicID)
			continue
		}

		anyNearby := false
		for _, otherPS := range s.playerStates {
			distSq := DistanceSquared(otherPS.Position, da.Position)
			effectiveDist := otherPS.effectiveStreamDist(EntityTypeActor, da.StreamDistance)
			if distSq <= effectiveDist*effectiveDist && da.isVisibleTo(otherPS.PlayerID, otherPS.WorldID, otherPS.Interior) {
				anyNearby = true
				break
			}
		}
		if !anyNearby {
			s.cfg.EventHandler.OnDynamicActorStreamOut(dynamicID, ps.PlayerID)
			actor.Destroy(sa)
			delete(s.activeActors, dynamicID)
		}
	}
}

// passesAreaCheck verifies that the player is in (or not in) the entity's required areas.
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
		} else {
			if inArea {
				return true
			}
		}
	}

	return e.InverseAreaCheck
}

// sortCandidates sorts by priority (descending) then distance (ascending).
func sortCandidates(candidates []candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].DistSq < candidates[j].DistSq
	})
}
