package streamer

import (
	"fmt"

	"github.com/ompgo-dev/ompgo/pkg/omp/playerobject"
)

// CreateObjectParams holds parameters for creating a dynamic object.
type CreateObjectParams struct {
	ModelID          int32
	X, Y, Z          float32
	RotX, RotY, RotZ float32
	StreamDistance   float32
	DrawDistance     float32
	Worlds           []int32
	Interiors        []int32
	Players          []int32
	Areas            []int32
	Priority         int32
}

// CreateObject creates a new dynamic object.
func (s *Streamer) CreateObject(params CreateObjectParams) (*DynamicObject, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultObjectStreamDistance
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeObject].Next()
	obj := &DynamicObject{
		baseEntity:    newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		ModelID:       params.ModelID,
		Rotation:      Vector3{X: params.RotX, Y: params.RotY, Z: params.RotZ},
		DrawDistance:  params.DrawDistance,
		Materials:     make(map[int32]*ObjectMaterial),
		MaterialTexts: make(map[int32]*ObjectMaterialText),
	}

	applyFilters(&obj.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.objects[id] = obj
	s.grid.addObject(obj)
	return obj, nil
}

// DestroyObject removes a dynamic object.
func (s *Streamer) DestroyObject(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return fmt.Errorf("streamer: object %d not found", id)
	}

	for _, ps := range s.playerStates {
		if serverObj, exists := ps.Objects[id]; exists {
			playerobject.Destroy(ps.Player, serverObj)
			delete(ps.Objects, id)
		}
	}

	s.grid.removeObject(obj)
	delete(s.objects, id)
	return nil
}

// IsValidObject returns true if the object ID exists.
func (s *Streamer) IsValidObject(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.objects[id]
	return ok
}

// GetObjectPos returns the position of a dynamic object.
func (s *Streamer) GetObjectPos(id int32) (Vector3, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, ok := s.objects[id]
	if !ok {
		return Vector3{}, fmt.Errorf("streamer: object %d not found", id)
	}
	return obj.Position, nil
}

// SetObjectPos updates the position of a dynamic object.
func (s *Streamer) SetObjectPos(id int32, x, y, z float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return fmt.Errorf("streamer: object %d not found", id)
	}

	s.grid.removeObject(obj)
	obj.Position = Vector3{X: x, Y: y, Z: z}
	s.grid.addObject(obj)
	return nil
}

// GetObjectRot returns the rotation of a dynamic object.
func (s *Streamer) GetObjectRot(id int32) (Vector3, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, ok := s.objects[id]
	if !ok {
		return Vector3{}, fmt.Errorf("streamer: object %d not found", id)
	}
	return obj.Rotation, nil
}

// SetObjectRot updates the rotation of a dynamic object.
func (s *Streamer) SetObjectRot(id int32, rx, ry, rz float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return fmt.Errorf("streamer: object %d not found", id)
	}
	obj.Rotation = Vector3{X: rx, Y: ry, Z: rz}
	return nil
}

// SetObjectMaterial sets a texture material on a dynamic object.
func (s *Streamer) SetObjectMaterial(id int32, index int32, modelID int32, txdName, textureName string, color uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return fmt.Errorf("streamer: object %d not found", id)
	}
	obj.Materials[index] = &ObjectMaterial{
		ModelID:     modelID,
		TXDName:     txdName,
		TextureName: textureName,
		Color:       color,
	}
	return nil
}

// SetObjectMaterialText sets material text on a dynamic object.
func (s *Streamer) SetObjectMaterialText(id int32, params ObjectMaterialText, index int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return fmt.Errorf("streamer: object %d not found", id)
	}
	obj.MaterialTexts[index] = &params
	return nil
}

// SetObjectNoCameraCol disables camera collision on a dynamic object.
func (s *Streamer) SetObjectNoCameraCol(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[id]
	if !ok {
		return fmt.Errorf("streamer: object %d not found", id)
	}
	obj.NoCameraCollision = true
	return nil
}

// CountObjects returns the number of dynamic objects.
func (s *Streamer) CountObjects() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.objects)
}

// DestroyAllObjects removes all dynamic objects.
func (s *Streamer) DestroyAllObjects() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ps := range s.playerStates {
		for id, serverObj := range ps.Objects {
			playerobject.Destroy(ps.Player, serverObj)
			delete(ps.Objects, id)
		}
	}
	for id, obj := range s.objects {
		s.grid.removeObject(obj)
		delete(s.objects, id)
	}
}
