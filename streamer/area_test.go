package streamer

import "testing"

func TestCircleShape(t *testing.T) {
	shape := &CircleShape{CenterX: 10, CenterY: 20, Radius: 5}

	tests := []struct {
		name    string
		x, y, z float32
		want    bool
	}{
		{"center", 10, 20, 0, true},
		{"edge", 15, 20, 0, true},
		{"inside", 12, 22, 0, true},
		{"outside", 20, 20, 0, false},
		{"Z ignored", 10, 20, 999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shape.ContainsPoint(tt.x, tt.y, tt.z)
			if got != tt.want {
				t.Errorf("ContainsPoint(%f,%f,%f) = %v, want %v", tt.x, tt.y, tt.z, got, tt.want)
			}
		})
	}

	center := shape.Center()
	if center.X != 10 || center.Y != 20 {
		t.Errorf("Center() = %v, want {10, 20, 0}", center)
	}
}

func TestCylinderShape(t *testing.T) {
	shape := &CylinderShape{CenterX: 0, CenterY: 0, Radius: 10, MinZ: 5, MaxZ: 15}

	tests := []struct {
		name    string
		x, y, z float32
		want    bool
	}{
		{"center middle", 0, 0, 10, true},
		{"center bottom", 0, 0, 5, true},
		{"center top", 0, 0, 15, true},
		{"below", 0, 0, 4, false},
		{"above", 0, 0, 16, false},
		{"outside radius", 11, 0, 10, false},
		{"edge", 10, 0, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shape.ContainsPoint(tt.x, tt.y, tt.z)
			if got != tt.want {
				t.Errorf("ContainsPoint(%f,%f,%f) = %v, want %v", tt.x, tt.y, tt.z, got, tt.want)
			}
		})
	}
}

func TestSphereShape(t *testing.T) {
	shape := &SphereShape{CenterX: 0, CenterY: 0, CenterZ: 0, Radius: 5}

	tests := []struct {
		name    string
		x, y, z float32
		want    bool
	}{
		{"center", 0, 0, 0, true},
		{"surface", 5, 0, 0, true},
		{"inside", 3, 4, 0, true},
		{"outside", 3, 4, 1, false},
		{"diagonal outside", 4, 4, 4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shape.ContainsPoint(tt.x, tt.y, tt.z)
			if got != tt.want {
				t.Errorf("ContainsPoint(%f,%f,%f) = %v, want %v", tt.x, tt.y, tt.z, got, tt.want)
			}
		})
	}
}

func TestRectangleShape(t *testing.T) {
	shape := &RectangleShape{MinX: -10, MinY: -10, MaxX: 10, MaxY: 10}

	tests := []struct {
		name    string
		x, y, z float32
		want    bool
	}{
		{"center", 0, 0, 0, true},
		{"corner", 10, 10, 0, true},
		{"edge", -10, 0, 0, true},
		{"outside", 11, 0, 0, false},
		{"Z ignored", 0, 0, 999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shape.ContainsPoint(tt.x, tt.y, tt.z)
			if got != tt.want {
				t.Errorf("ContainsPoint(%f,%f,%f) = %v, want %v", tt.x, tt.y, tt.z, got, tt.want)
			}
		})
	}
}

func TestCuboidShape(t *testing.T) {
	shape := &CuboidShape{MinX: 0, MinY: 0, MinZ: 0, MaxX: 10, MaxY: 10, MaxZ: 10}

	tests := []struct {
		name    string
		x, y, z float32
		want    bool
	}{
		{"center", 5, 5, 5, true},
		{"corner", 0, 0, 0, true},
		{"opposite corner", 10, 10, 10, true},
		{"outside X", 11, 5, 5, false},
		{"outside Y", 5, 11, 5, false},
		{"outside Z", 5, 5, 11, false},
		{"negative", -1, 5, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shape.ContainsPoint(tt.x, tt.y, tt.z)
			if got != tt.want {
				t.Errorf("ContainsPoint(%f,%f,%f) = %v, want %v", tt.x, tt.y, tt.z, got, tt.want)
			}
		})
	}

	center := shape.Center()
	if center.X != 5 || center.Y != 5 || center.Z != 5 {
		t.Errorf("Center() = %v, want {5, 5, 5}", center)
	}
}

func TestPolygonShape(t *testing.T) {
	// Square polygon: (0,0), (10,0), (10,10), (0,10)
	shape := &PolygonShape{
		Points: []Vector2{{0, 0}, {10, 0}, {10, 10}, {0, 10}},
		MinZ:   0,
		MaxZ:   10,
	}

	tests := []struct {
		name    string
		x, y, z float32
		want    bool
	}{
		{"center", 5, 5, 5, true},
		{"inside near edge", 1, 1, 5, true},
		{"outside", 15, 5, 5, false},
		{"below Z", 5, 5, -1, false},
		{"above Z", 5, 5, 11, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shape.ContainsPoint(tt.x, tt.y, tt.z)
			if got != tt.want {
				t.Errorf("ContainsPoint(%f,%f,%f) = %v, want %v", tt.x, tt.y, tt.z, got, tt.want)
			}
		})
	}
}

func TestPolygonShapeTriangle(t *testing.T) {
	// Right triangle
	shape := &PolygonShape{
		Points: []Vector2{{0, 0}, {10, 0}, {0, 10}},
		MinZ:   -100,
		MaxZ:   100,
	}

	tests := []struct {
		name    string
		x, y, z float32
		want    bool
	}{
		{"inside", 2, 2, 0, true},
		{"outside hypotenuse", 8, 8, 0, false},
		{"near origin", 1, 1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shape.ContainsPoint(tt.x, tt.y, tt.z)
			if got != tt.want {
				t.Errorf("ContainsPoint(%f,%f,%f) = %v, want %v", tt.x, tt.y, tt.z, got, tt.want)
			}
		})
	}
}

func TestPolygonShapeTooFewPoints(t *testing.T) {
	shape := &PolygonShape{
		Points: []Vector2{{0, 0}, {10, 0}},
		MinZ:   0,
		MaxZ:   10,
	}

	if shape.ContainsPoint(5, 5, 5) {
		t.Error("polygon with < 3 points should never contain a point")
	}
}

func TestLineIntersectsShape(t *testing.T) {
	shape := &SphereShape{CenterX: 50, CenterY: 50, CenterZ: 50, Radius: 10}

	tests := []struct {
		name                   string
		x1, y1, z1, x2, y2, z2 float32
		want                   bool
	}{
		{"through center", 0, 50, 50, 100, 50, 50, true},
		{"tangent miss", 0, 70, 50, 100, 70, 50, false},
		{"starts inside", 50, 50, 50, 100, 100, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lineIntersectsShape(tt.x1, tt.y1, tt.z1, tt.x2, tt.y2, tt.z2, shape)
			if got != tt.want {
				t.Errorf("lineIntersectsShape = %v, want %v", got, tt.want)
			}
		})
	}
}
