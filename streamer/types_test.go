package streamer

import "testing"

func TestDistanceSquared(t *testing.T) {
	tests := []struct {
		name string
		a, b Vector3
		want float32
	}{
		{
			name: "same point",
			a:    Vector3{0, 0, 0},
			b:    Vector3{0, 0, 0},
			want: 0,
		},
		{
			name: "unit distance X",
			a:    Vector3{0, 0, 0},
			b:    Vector3{1, 0, 0},
			want: 1,
		},
		{
			name: "3-4-5 triangle",
			a:    Vector3{0, 0, 0},
			b:    Vector3{3, 4, 0},
			want: 25,
		},
		{
			name: "3D distance",
			a:    Vector3{1, 2, 3},
			b:    Vector3{4, 6, 3},
			want: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DistanceSquared(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("DistanceSquared(%v, %v) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDistanceSquared2D(t *testing.T) {
	tests := []struct {
		name string
		a, b Vector2
		want float32
	}{
		{
			name: "same point",
			a:    Vector2{0, 0},
			b:    Vector2{0, 0},
			want: 0,
		},
		{
			name: "3-4-5",
			a:    Vector2{0, 0},
			b:    Vector2{3, 4},
			want: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DistanceSquared2D(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("DistanceSquared2D(%v, %v) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
