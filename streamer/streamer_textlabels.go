package streamer

import (
	"fmt"

	"github.com/ompgo-dev/ompgo/pkg/omp/playertextlabel"
)

// CreateTextLabelParams holds parameters for creating a dynamic 3D text label.
type CreateTextLabelParams struct {
	Text              string
	Color             uint32
	X, Y, Z           float32
	DrawDistance      float32
	AttachedPlayerID  int32
	AttachedVehicleID int32
	TestLOS           bool
	StreamDistance    float32
	Worlds            []int32
	Interiors         []int32
	Players           []int32
	Areas             []int32
	Priority          int32
}

// CreateTextLabel creates a new dynamic 3D text label.
func (s *Streamer) CreateTextLabel(params CreateTextLabelParams) (*DynamicTextLabel, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultTextLabelStreamDistce
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeTextLabel].Next()
	tl := &DynamicTextLabel{
		baseEntity:        newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		Text:              params.Text,
		Color:             params.Color,
		DrawDistance:      params.DrawDistance,
		TestLOS:           params.TestLOS,
		AttachedPlayerID:  params.AttachedPlayerID,
		AttachedVehicleID: params.AttachedVehicleID,
	}

	applyFilters(&tl.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.textLabels[id] = tl
	s.grid.addTextLabel(tl)
	return tl, nil
}

// DestroyTextLabel removes a dynamic 3D text label.
func (s *Streamer) DestroyTextLabel(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tl, ok := s.textLabels[id]
	if !ok {
		return fmt.Errorf("streamer: text label %d not found", id)
	}

	for _, ps := range s.playerStates {
		if serverTL, exists := ps.TextLabels[id]; exists {
			playertextlabel.Destroy(ps.Player, serverTL)
			delete(ps.TextLabels, id)
		}
	}

	s.grid.removeTextLabel(tl)
	delete(s.textLabels, id)
	return nil
}

// IsValidTextLabel returns true if the text label ID exists.
func (s *Streamer) IsValidTextLabel(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.textLabels[id]
	return ok
}

// UpdateTextLabelText updates the text and color of a dynamic 3D text label.
func (s *Streamer) UpdateTextLabelText(id int32, color uint32, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tl, ok := s.textLabels[id]
	if !ok {
		return fmt.Errorf("streamer: text label %d not found", id)
	}

	tl.Color = color
	tl.Text = text

	for _, ps := range s.playerStates {
		if serverTL, exists := ps.TextLabels[id]; exists {
			playertextlabel.UpdateText(ps.Player, serverTL, color, text)
		}
	}
	return nil
}

// CountTextLabels returns the number of dynamic text labels.
func (s *Streamer) CountTextLabels() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.textLabels)
}

// DestroyAllTextLabels removes all dynamic text labels.
func (s *Streamer) DestroyAllTextLabels() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ps := range s.playerStates {
		for id, serverTL := range ps.TextLabels {
			playertextlabel.Destroy(ps.Player, serverTL)
			delete(ps.TextLabels, id)
		}
	}
	for id, tl := range s.textLabels {
		s.grid.removeTextLabel(tl)
		delete(s.textLabels, id)
	}
}
