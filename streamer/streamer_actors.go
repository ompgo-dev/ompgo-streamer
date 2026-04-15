package streamer

import (
	"fmt"

	"github.com/ompgo-dev/ompgo/pkg/omp/actor"
)

// CreateActorParams holds parameters for creating a dynamic actor.
type CreateActorParams struct {
	ModelID        int32
	X, Y, Z        float32
	Rotation       float32
	Invulnerable   bool
	Health         float32
	StreamDistance float32
	Worlds         []int32
	Interiors      []int32
	Players        []int32
	Areas          []int32
	Priority       int32
}

// CreateActor creates a new dynamic actor.
func (s *Streamer) CreateActor(params CreateActorParams) (*DynamicActor, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultActorStreamDistance
	}
	if params.Health <= 0 {
		params.Health = 100.0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeActor].Next()
	a := &DynamicActor{
		baseEntity:   newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		ModelID:      params.ModelID,
		Rotation:     params.Rotation,
		Invulnerable: params.Invulnerable,
		Health:       params.Health,
	}

	applyFilters(&a.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.actors[id] = a
	s.grid.addActor(a)
	return a, nil
}

// DestroyActor removes a dynamic actor.
func (s *Streamer) DestroyActor(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.actors[id]
	if !ok {
		return fmt.Errorf("streamer: actor %d not found", id)
	}

	if serverActor, exists := s.activeActors[id]; exists {
		actor.Destroy(serverActor)
		delete(s.activeActors, id)
	}

	s.grid.removeActor(a)
	delete(s.actors, id)
	return nil
}

// IsValidActor returns true if the actor ID exists.
func (s *Streamer) IsValidActor(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.actors[id]
	return ok
}

// GetActorPos returns the position of a dynamic actor.
func (s *Streamer) GetActorPos(id int32) (Vector3, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.actors[id]
	if !ok {
		return Vector3{}, fmt.Errorf("streamer: actor %d not found", id)
	}
	return a.Position, nil
}

// SetActorPos sets the position of a dynamic actor.
func (s *Streamer) SetActorPos(id int32, x, y, z float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.actors[id]
	if !ok {
		return fmt.Errorf("streamer: actor %d not found", id)
	}

	s.grid.removeActor(a)
	a.Position = Vector3{X: x, Y: y, Z: z}
	s.grid.addActor(a)

	if serverActor, exists := s.activeActors[id]; exists {
		actor.SetPos(serverActor, x, y, z)
	}
	return nil
}

// ApplyDynamicActorAnimation applies an animation to a dynamic actor.
func (s *Streamer) ApplyDynamicActorAnimation(id int32, anim ActorAnimation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.actors[id]
	if !ok {
		return fmt.Errorf("streamer: actor %d not found", id)
	}
	a.Animation = &anim

	if serverActor, exists := s.activeActors[id]; exists {
		actor.ApplyAnimation(serverActor, anim.Name, anim.Library, anim.Delta, anim.Loop, anim.LockX, anim.LockY, anim.Freeze, anim.Time)
	}
	return nil
}

// ClearDynamicActorAnimations clears animations on a dynamic actor.
func (s *Streamer) ClearDynamicActorAnimations(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.actors[id]
	if !ok {
		return fmt.Errorf("streamer: actor %d not found", id)
	}
	a.Animation = nil

	if serverActor, exists := s.activeActors[id]; exists {
		actor.ClearAnimations(serverActor)
	}
	return nil
}

// CountActors returns the number of dynamic actors.
func (s *Streamer) CountActors() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.actors)
}

// DestroyAllActors removes all dynamic actors.
func (s *Streamer) DestroyAllActors() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, serverActor := range s.activeActors {
		actor.Destroy(serverActor)
		delete(s.activeActors, id)
	}
	for id, actorData := range s.actors {
		s.grid.removeActor(actorData)
		delete(s.actors, id)
	}
}
