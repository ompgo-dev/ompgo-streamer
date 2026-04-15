package streamer

import "testing"

func TestBaseEntityIsVisibleTo_AllDefaults(t *testing.T) {
	e := newBaseEntity(1, Vector3{}, 200)
	// Default entity visible to any player/world/interior.
	if !e.isVisibleTo(0, 0, 0) {
		t.Error("default entity should be visible to any player")
	}
}

func TestBaseEntityIsVisibleTo_WorldFilter(t *testing.T) {
	e := newBaseEntity(1, Vector3{}, 200)
	// Restrict to world 5 only.
	e.Worlds = map[int32]struct{}{5: {}}

	if e.isVisibleTo(0, 3, 0) {
		t.Error("should not be visible in world 3")
	}
	if !e.isVisibleTo(0, 5, 0) {
		t.Error("should be visible in world 5")
	}
}

func TestBaseEntityIsVisibleTo_InteriorFilter(t *testing.T) {
	e := newBaseEntity(1, Vector3{}, 200)
	e.Interiors = map[int32]struct{}{2: {}}

	if e.isVisibleTo(0, 0, 0) {
		t.Error("should not be visible in interior 0")
	}
	if !e.isVisibleTo(0, 0, 2) {
		t.Error("should be visible in interior 2")
	}
}

func TestBaseEntityIsVisibleTo_PlayerFilter(t *testing.T) {
	e := newBaseEntity(1, Vector3{}, 200)
	e.Players = map[int32]struct{}{42: {}}

	if e.isVisibleTo(10, 0, 0) {
		t.Error("should not be visible to player 10")
	}
	if !e.isVisibleTo(42, 0, 0) {
		t.Error("should be visible to player 42")
	}
}

func TestBaseEntityIsVisibleTo_CombinedFilters(t *testing.T) {
	e := newBaseEntity(1, Vector3{}, 200)
	e.Worlds = map[int32]struct{}{1: {}}
	e.Interiors = map[int32]struct{}{3: {}}
	e.Players = map[int32]struct{}{7: {}}

	tests := []struct {
		name     string
		playerID int32
		worldID  int32
		interior int32
		want     bool
	}{
		{"all match", 7, 1, 3, true},
		{"wrong world", 7, 2, 3, false},
		{"wrong interior", 7, 1, 0, false},
		{"wrong player", 0, 1, 3, false},
		{"all wrong", 0, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.isVisibleTo(tt.playerID, tt.worldID, tt.interior)
			if got != tt.want {
				t.Errorf("isVisibleTo(%d, %d, %d) = %v, want %v",
					tt.playerID, tt.worldID, tt.interior, got, tt.want)
			}
		})
	}
}

func TestBaseEntityStreamDistSq(t *testing.T) {
	e := newBaseEntity(1, Vector3{}, 200)
	got := e.streamDistSq()
	want := float32(40000)
	if got != want {
		t.Errorf("streamDistSq() = %f, want %f", got, want)
	}
}
