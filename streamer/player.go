package streamer

import (
	"github.com/ompgo-dev/ompgo/pkg/omp"
)

// playerState tracks per-player streaming state.
type playerState struct {
	PlayerID int32
	Player   *omp.Player
	Position Vector3
	WorldID  int32
	Interior int32

	// Per-player tick rate (0 = use global).
	TickRate  int32
	TickCount int32

	// Radius multipliers per entity type.
	RadiusMultiplier [MaxEntityTypes]float32

	// Per-player visible items overrides (0 = use global).
	VisibleObjects    int32
	VisibleMapIcons   int32
	VisibleTextLabels int32

	// Currently streamed-in per-player entities.
	// Maps dynamic entity ID -> server-side entity.
	Objects    map[int32]*omp.PlayerObject
	TextLabels map[int32]*omp.PlayerTextLabel
	MapIcons   map[int32]int32 // dynamic ID -> icon slot index

	// Current visible checkpoint / race checkpoint (only one each per player).
	VisibleCheckpoint     int32 // dynamic ID, or InvalidID
	VisibleRaceCheckpoint int32 // dynamic ID, or InvalidID

	// Areas the player is currently inside.
	InAreas map[int32]struct{}

	// Per-player toggle for items: entityType -> dynamicID -> enabled.
	DisabledItems [MaxEntityTypes]map[int32]struct{}

	// Next available map icon slot index.
	nextMapIconSlot  int32
	freeMapIconSlots []int32
}

func newPlayerState(playerID int32, player *omp.Player) *playerState {
	ps := &playerState{
		PlayerID:              playerID,
		Player:                player,
		VisibleCheckpoint:     InvalidID,
		VisibleRaceCheckpoint: InvalidID,
		Objects:               make(map[int32]*omp.PlayerObject),
		TextLabels:            make(map[int32]*omp.PlayerTextLabel),
		MapIcons:              make(map[int32]int32),
		InAreas:               make(map[int32]struct{}),
	}
	for i := range ps.RadiusMultiplier {
		ps.RadiusMultiplier[i] = 1.0
	}
	for i := range ps.DisabledItems {
		ps.DisabledItems[i] = make(map[int32]struct{})
	}
	return ps
}

// allocMapIconSlot returns the next available map icon slot.
func (ps *playerState) allocMapIconSlot() int32 {
	if len(ps.freeMapIconSlots) > 0 {
		slot := ps.freeMapIconSlots[len(ps.freeMapIconSlots)-1]
		ps.freeMapIconSlots = ps.freeMapIconSlots[:len(ps.freeMapIconSlots)-1]
		return slot
	}
	slot := ps.nextMapIconSlot
	ps.nextMapIconSlot++
	return slot
}

// freeMapIconSlot returns a slot to the pool for reuse.
func (ps *playerState) freeMapIconSlot(slot int32) {
	ps.freeMapIconSlots = append(ps.freeMapIconSlots, slot)
}

// isItemDisabled checks if a specific item is disabled for this player.
func (ps *playerState) isItemDisabled(entityType EntityType, dynamicID int32) bool {
	if entityType < 0 || int(entityType) >= MaxEntityTypes {
		return false
	}
	_, disabled := ps.DisabledItems[entityType][dynamicID]
	return disabled
}

// effectiveStreamDist returns the stream distance adjusted by the player's radius multiplier.
func (ps *playerState) effectiveStreamDist(entityType EntityType, baseDist float32) float32 {
	if entityType < 0 || int(entityType) >= MaxEntityTypes {
		return baseDist
	}
	return baseDist * ps.RadiusMultiplier[entityType]
}
