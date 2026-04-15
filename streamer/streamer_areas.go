package streamer

import "fmt"

// CreateCircle creates a dynamic circular area.
func (s *Streamer) CreateCircle(x, y, radius float32, opts ...AreaOption) (*DynamicArea, error) {
	return s.createArea(AreaTypeCircle, &CircleShape{CenterX: x, CenterY: y, Radius: radius}, opts...)
}

// CreateCylinder creates a dynamic cylindrical area.
func (s *Streamer) CreateCylinder(x, y, minZ, maxZ, radius float32, opts ...AreaOption) (*DynamicArea, error) {
	return s.createArea(AreaTypeCylinder, &CylinderShape{CenterX: x, CenterY: y, Radius: radius, MinZ: minZ, MaxZ: maxZ}, opts...)
}

// CreateSphere creates a dynamic spherical area.
func (s *Streamer) CreateSphere(x, y, z, radius float32, opts ...AreaOption) (*DynamicArea, error) {
	return s.createArea(AreaTypeSphere, &SphereShape{CenterX: x, CenterY: y, CenterZ: z, Radius: radius}, opts...)
}

// CreateRectangle creates a dynamic rectangular area.
func (s *Streamer) CreateRectangle(minX, minY, maxX, maxY float32, opts ...AreaOption) (*DynamicArea, error) {
	return s.createArea(AreaTypeRectangle, &RectangleShape{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}, opts...)
}

// CreateCuboid creates a dynamic cuboid area.
func (s *Streamer) CreateCuboid(minX, minY, minZ, maxX, maxY, maxZ float32, opts ...AreaOption) (*DynamicArea, error) {
	return s.createArea(AreaTypeCuboid, &CuboidShape{MinX: minX, MinY: minY, MinZ: minZ, MaxX: maxX, MaxY: maxY, MaxZ: maxZ}, opts...)
}

// CreatePolygon creates a dynamic polygon area.
func (s *Streamer) CreatePolygon(points []Vector2, minZ, maxZ float32, opts ...AreaOption) (*DynamicArea, error) {
	return s.createArea(AreaTypePolygon, &PolygonShape{Points: points, MinZ: minZ, MaxZ: maxZ}, opts...)
}

func (s *Streamer) createArea(areaType AreaType, shape Shape, opts ...AreaOption) (*DynamicArea, error) {
	cfg := defaultAreaOptions()
	for _, opt := range opts {
		opt(cfg)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeArea].Next()
	center := shape.Center()
	a := &DynamicArea{
		baseEntity: newBaseEntity(id, center, cfg.StreamDistance),
		AreaType:   areaType,
		Shape:      shape,
	}

	applyFilters(&a.baseEntity, cfg.Worlds, cfg.Interiors, cfg.Players, nil, cfg.Priority)
	s.areas[id] = a
	s.grid.addArea(a)
	return a, nil
}

// DestroyArea removes a dynamic area.
func (s *Streamer) DestroyArea(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.areas[id]
	if !ok {
		return fmt.Errorf("streamer: area %d not found", id)
	}

	for _, ps := range s.playerStates {
		delete(ps.InAreas, id)
	}

	s.grid.removeArea(a)
	delete(s.areas, id)
	return nil
}

// IsValidArea returns true if the area ID exists.
func (s *Streamer) IsValidArea(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.areas[id]
	return ok
}

// GetAreaType returns the area type of a dynamic area.
func (s *Streamer) GetAreaType(id int32) (AreaType, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.areas[id]
	if !ok {
		return 0, fmt.Errorf("streamer: area %d not found", id)
	}
	return a.AreaType, nil
}

// IsPlayerInDynamicArea checks if a player is inside a specific area.
func (s *Streamer) IsPlayerInDynamicArea(playerID, areaID int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return false
	}
	_, inside := ps.InAreas[areaID]
	return inside
}

// IsPlayerInAnyDynamicArea returns true if the player is in any dynamic area.
func (s *Streamer) IsPlayerInAnyDynamicArea(playerID int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return false
	}
	return len(ps.InAreas) > 0
}

// GetPlayerDynamicAreas returns all area IDs the player is currently inside.
func (s *Streamer) GetPlayerDynamicAreas(playerID int32) []int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}
	result := make([]int32, 0, len(ps.InAreas))
	for id := range ps.InAreas {
		result = append(result, id)
	}
	return result
}

// IsPointInDynamicArea checks if a 3D point is within a specific area.
func (s *Streamer) IsPointInDynamicArea(areaID int32, x, y, z float32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.areas[areaID]
	if !ok {
		return false
	}
	return a.Shape.ContainsPoint(x, y, z)
}

// IsPointInAnyDynamicArea returns the first area ID containing the point, or InvalidID.
func (s *Streamer) IsPointInAnyDynamicArea(x, y, z float32) int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for id, a := range s.areas {
		if a.Shape.ContainsPoint(x, y, z) {
			return id
		}
	}
	return InvalidID
}

// GetDynamicAreasForPoint returns all area IDs containing the given point.
func (s *Streamer) GetDynamicAreasForPoint(x, y, z float32) []int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []int32
	for id, a := range s.areas {
		if a.Shape.ContainsPoint(x, y, z) {
			result = append(result, id)
		}
	}
	return result
}

// IsLineInDynamicArea checks if a line segment intersects a specific area.
func (s *Streamer) IsLineInDynamicArea(areaID int32, x1, y1, z1, x2, y2, z2 float32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.areas[areaID]
	if !ok {
		return false
	}
	return lineIntersectsShape(x1, y1, z1, x2, y2, z2, a.Shape)
}

// CountAreas returns the number of dynamic areas.
func (s *Streamer) CountAreas() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.areas)
}

// DestroyAllAreas removes all dynamic areas.
func (s *Streamer) DestroyAllAreas() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ps := range s.playerStates {
		for id := range ps.InAreas {
			delete(ps.InAreas, id)
		}
	}
	for id, area := range s.areas {
		s.grid.removeArea(area)
		delete(s.areas, id)
	}
}
