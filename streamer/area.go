package streamer

import "math"

// DynamicArea represents a streamed area with one of several shape types.
type DynamicArea struct {
	baseEntity
	AreaType     AreaType
	Shape        Shape
	SpectateMode bool
	AttachedTo   *AttachData
}

// Shape is the interface for area shape containment testing.
type Shape interface {
	ContainsPoint(x, y, z float32) bool
	Center() Vector3
}

// Compile-time interface checks.
var (
	_ Shape = (*CircleShape)(nil)
	_ Shape = (*CylinderShape)(nil)
	_ Shape = (*SphereShape)(nil)
	_ Shape = (*RectangleShape)(nil)
	_ Shape = (*CuboidShape)(nil)
	_ Shape = (*PolygonShape)(nil)
)

// CircleShape is a 2D circle (ignores Z).
type CircleShape struct {
	CenterX float32
	CenterY float32
	Radius  float32
}

func (s *CircleShape) ContainsPoint(x, y, _ float32) bool {
	dx := x - s.CenterX
	dy := y - s.CenterY
	return dx*dx+dy*dy <= s.Radius*s.Radius
}

func (s *CircleShape) Center() Vector3 {
	return Vector3{X: s.CenterX, Y: s.CenterY}
}

// CylinderShape is a 2D circle with min/max Z bounds.
type CylinderShape struct {
	CenterX float32
	CenterY float32
	Radius  float32
	MinZ    float32
	MaxZ    float32
}

func (s *CylinderShape) ContainsPoint(x, y, z float32) bool {
	if z < s.MinZ || z > s.MaxZ {
		return false
	}
	dx := x - s.CenterX
	dy := y - s.CenterY
	return dx*dx+dy*dy <= s.Radius*s.Radius
}

func (s *CylinderShape) Center() Vector3 {
	return Vector3{X: s.CenterX, Y: s.CenterY, Z: (s.MinZ + s.MaxZ) / 2}
}

// SphereShape is a 3D sphere.
type SphereShape struct {
	CenterX float32
	CenterY float32
	CenterZ float32
	Radius  float32
}

func (s *SphereShape) ContainsPoint(x, y, z float32) bool {
	dx := x - s.CenterX
	dy := y - s.CenterY
	dz := z - s.CenterZ
	return dx*dx+dy*dy+dz*dz <= s.Radius*s.Radius
}

func (s *SphereShape) Center() Vector3 {
	return Vector3{X: s.CenterX, Y: s.CenterY, Z: s.CenterZ}
}

// RectangleShape is a 2D axis-aligned rectangle (ignores Z).
type RectangleShape struct {
	MinX float32
	MinY float32
	MaxX float32
	MaxY float32
}

func (s *RectangleShape) ContainsPoint(x, y, _ float32) bool {
	return x >= s.MinX && x <= s.MaxX && y >= s.MinY && y <= s.MaxY
}

func (s *RectangleShape) Center() Vector3 {
	return Vector3{X: (s.MinX + s.MaxX) / 2, Y: (s.MinY + s.MaxY) / 2}
}

// CuboidShape is a 3D axis-aligned box.
type CuboidShape struct {
	MinX float32
	MinY float32
	MinZ float32
	MaxX float32
	MaxY float32
	MaxZ float32
}

func (s *CuboidShape) ContainsPoint(x, y, z float32) bool {
	return x >= s.MinX && x <= s.MaxX &&
		y >= s.MinY && y <= s.MaxY &&
		z >= s.MinZ && z <= s.MaxZ
}

func (s *CuboidShape) Center() Vector3 {
	return Vector3{
		X: (s.MinX + s.MaxX) / 2,
		Y: (s.MinY + s.MaxY) / 2,
		Z: (s.MinZ + s.MaxZ) / 2,
	}
}

// PolygonShape is a 2D polygon with Z bounds using the ray-casting algorithm.
type PolygonShape struct {
	Points []Vector2
	MinZ   float32
	MaxZ   float32
}

func (s *PolygonShape) ContainsPoint(x, y, z float32) bool {
	if z < s.MinZ || z > s.MaxZ {
		return false
	}
	return pointInPolygon(x, y, s.Points)
}

func (s *PolygonShape) Center() Vector3 {
	if len(s.Points) == 0 {
		return Vector3{}
	}
	var sumX, sumY float32
	for _, p := range s.Points {
		sumX += p.X
		sumY += p.Y
	}
	n := float32(len(s.Points))
	return Vector3{X: sumX / n, Y: sumY / n, Z: (s.MinZ + s.MaxZ) / 2}
}

// pointInPolygon uses the ray-casting algorithm to determine point-in-polygon.
func pointInPolygon(px, py float32, points []Vector2) bool {
	n := len(points)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := float64(points[i].X), float64(points[i].Y)
		xj, yj := float64(points[j].X), float64(points[j].Y)
		testX, testY := float64(px), float64(py)

		if ((yi > testY) != (yj > testY)) &&
			(testX < (xj-xi)*(testY-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
}

// pointInLineSegment checks if a point is close to a line segment.
// Used for IsLineInDynamicArea.
func lineIntersectsShape(x1, y1, z1, x2, y2, z2 float32, shape Shape) bool {
	steps := int(math.Ceil(float64(distance3D(x1, y1, z1, x2, y2, z2)) / 5.0))
	if steps < 2 {
		steps = 2
	}
	for i := 0; i <= steps; i++ {
		t := float32(i) / float32(steps)
		px := x1 + t*(x2-x1)
		py := y1 + t*(y2-y1)
		pz := z1 + t*(z2-z1)
		if shape.ContainsPoint(px, py, pz) {
			return true
		}
	}
	return false
}

func distance3D(x1, y1, z1, x2, y2, z2 float32) float32 {
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}
