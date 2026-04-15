package streamer

import (
	"github.com/ompgo-dev/ompgo/pkg/handle"
	"github.com/ompgo-dev/ompgo/pkg/omp"
	"github.com/ompgo-dev/ompgo/pkg/omp/players"
	"github.com/ompgo-dev/ompgo/pkg/omp/playertextlabel"
)

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
	for _, candidate := range candidates {
		if count >= maxVisible {
			break
		}
		visible[candidate.ID] = struct{}{}
		count++
	}

	for dynamicID, serverTL := range ps.TextLabels {
		if _, keep := visible[dynamicID]; keep {
			continue
		}
		playertextlabel.Destroy(ps.Player, serverTL)
		delete(ps.TextLabels, dynamicID)
	}

	for dynamicID := range visible {
		if _, exists := ps.TextLabels[dynamicID]; exists {
			continue
		}
		textLabel := s.textLabels[dynamicID]
		if textLabel == nil {
			continue
		}

		var attachedPlayer *omp.Player
		var attachedVehicle *omp.Vehicle
		if textLabel.AttachedPlayerID >= 0 {
			attachedPlayer = players.FromID(textLabel.AttachedPlayerID)
		}
		if attachedPlayer == nil {
			attachedPlayer = omp.NewPlayer(handle.Handle(0))
		}
		if attachedVehicle == nil {
			attachedVehicle = omp.NewVehicle(handle.Handle(0))
		}

		var serverID int32
		serverTL := playertextlabel.Create(
			ps.Player,
			textLabel.Text,
			textLabel.Color,
			textLabel.Position.X, textLabel.Position.Y, textLabel.Position.Z,
			textLabel.DrawDistance,
			attachedPlayer,
			attachedVehicle,
			textLabel.TestLOS,
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
	for _, currentCell := range cells {
		for id, textLabel := range currentCell.TextLabels {
			if ps.isItemDisabled(EntityTypeTextLabel, id) {
				continue
			}
			if !textLabel.isVisibleTo(ps.PlayerID, ps.WorldID, ps.Interior) {
				continue
			}
			if !s.passesAreaCheck(textLabel.baseEntity, ps.Position) {
				continue
			}
			distSq := DistanceSquared(ps.Position, textLabel.Position)
			effectiveDist := ps.effectiveStreamDist(EntityTypeTextLabel, textLabel.StreamDistance)
			if distSq > effectiveDist*effectiveDist {
				continue
			}
			candidates = append(candidates, candidate{ID: id, DistSq: distSq, Priority: textLabel.Priority})
		}
	}
	return candidates
}
