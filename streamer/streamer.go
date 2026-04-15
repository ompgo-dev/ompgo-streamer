package streamer

import (
	"context"
	"fmt"
	"sync"

	"github.com/ompgo-dev/ompgo/pkg/omp"
	"github.com/ompgo-dev/ompgo/pkg/omp/actor"
	"github.com/ompgo-dev/ompgo/pkg/omp/checkpoint"
	"github.com/ompgo-dev/ompgo/pkg/omp/core"
	"github.com/ompgo-dev/ompgo/pkg/omp/pickup"
	"github.com/ompgo-dev/ompgo/pkg/omp/playerobject"
	"github.com/ompgo-dev/ompgo/pkg/omp/players"
	"github.com/ompgo-dev/ompgo/pkg/omp/playertextlabel"
	"github.com/ompgo-dev/ompgo/pkg/omp/racecheckpoint"
	"github.com/ompgo-dev/ompgo/pkg/runtime"
)

// Streamer manages dynamic entity streaming for open.mp via ompgo.
type Streamer struct {
	mu      sync.RWMutex
	cfg     *config
	grid    *grid
	idAlloc [MaxEntityTypes]*IDAllocator

	// Entity storage.
	objects         map[int32]*DynamicObject
	pickups         map[int32]*DynamicPickup
	checkpoints     map[int32]*DynamicCheckpoint
	raceCheckpoints map[int32]*DynamicRaceCheckpoint
	mapIcons        map[int32]*DynamicMapIcon
	textLabels      map[int32]*DynamicTextLabel
	areas           map[int32]*DynamicArea
	actors          map[int32]*DynamicActor

	// Per-player state.
	playerStates map[int32]*playerState

	// Global entity tracking (pickups/actors are server-side, shared).
	activePickups map[int32]*omp.Pickup // dynamic ID -> server pickup
	activeActors  map[int32]*omp.Actor  // dynamic ID -> server actor

	// Unregister functions for event handlers.
	unregisterFns []func()

	// Tick counter for rate limiting.
	globalTickCount int32
}

// New creates a new Streamer with the given options.
func New(opts ...Option) *Streamer {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	s := &Streamer{
		cfg:             cfg,
		grid:            newGrid(cfg.CellSize, cfg.CellDistance),
		objects:         make(map[int32]*DynamicObject),
		pickups:         make(map[int32]*DynamicPickup),
		checkpoints:     make(map[int32]*DynamicCheckpoint),
		raceCheckpoints: make(map[int32]*DynamicRaceCheckpoint),
		mapIcons:        make(map[int32]*DynamicMapIcon),
		textLabels:      make(map[int32]*DynamicTextLabel),
		areas:           make(map[int32]*DynamicArea),
		actors:          make(map[int32]*DynamicActor),
		playerStates:    make(map[int32]*playerState),
		activePickups:   make(map[int32]*omp.Pickup),
		activeActors:    make(map[int32]*omp.Actor),
	}

	for i := range s.idAlloc {
		s.idAlloc[i] = NewIDAllocator()
	}

	return s
}

// Start registers event handlers with the ompgo runtime and begins streaming.
func (s *Streamer) Start() {
	s.unregisterFns = append(s.unregisterFns,
		runtime.RegisterOnTick(s.onTick),
		runtime.RegisterOnPlayerConnect(s.onPlayerConnect),
		runtime.RegisterOnPlayerDisconnect(s.onPlayerDisconnect),
		runtime.RegisterOnPlayerEnterCheckpoint(s.onPlayerEnterCheckpoint),
		runtime.RegisterOnPlayerLeaveCheckpoint(s.onPlayerLeaveCheckpoint),
		runtime.RegisterOnPlayerEnterRaceCheckpoint(s.onPlayerEnterRaceCheckpoint),
		runtime.RegisterOnPlayerLeaveRaceCheckpoint(s.onPlayerLeaveRaceCheckpoint),
		runtime.RegisterOnPlayerPickUpPickup(s.onPlayerPickUpPickup),
		runtime.RegisterOnPlayerGiveDamageActor(s.onPlayerGiveDamageActor),
	)
}

// Stop unregisters all event handlers and cleans up.
func (s *Streamer) Stop() {
	for _, unreg := range s.unregisterFns {
		unreg()
	}
	s.unregisterFns = nil
}

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

	// Remove from all players.
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

// CreatePickupParams holds parameters for creating a dynamic pickup.
type CreatePickupParams struct {
	ModelID        int32
	Type           int32
	X, Y, Z        float32
	StreamDistance float32
	Worlds         []int32
	Interiors      []int32
	Players        []int32
	Areas          []int32
	Priority       int32
}

// CreatePickup creates a new dynamic pickup.
func (s *Streamer) CreatePickup(params CreatePickupParams) (*DynamicPickup, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultPickupStreamDistance
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypePickup].Next()
	p := &DynamicPickup{
		baseEntity: newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		ModelID:    params.ModelID,
		Type:       params.Type,
	}

	applyFilters(&p.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.pickups[id] = p
	s.grid.addPickup(p)
	return p, nil
}

// DestroyPickup removes a dynamic pickup.
func (s *Streamer) DestroyPickup(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pickups[id]
	if !ok {
		return fmt.Errorf("streamer: pickup %d not found", id)
	}

	if serverPickup, exists := s.activePickups[id]; exists {
		pickup.Destroy(serverPickup)
		delete(s.activePickups, id)
	}

	s.grid.removePickup(p)
	delete(s.pickups, id)
	return nil
}

// IsValidPickup returns true if the pickup ID exists.
func (s *Streamer) IsValidPickup(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.pickups[id]
	return ok
}

// CountPickups returns the number of dynamic pickups.
func (s *Streamer) CountPickups() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pickups)
}

// DestroyAllPickups removes all dynamic pickups.
func (s *Streamer) DestroyAllPickups() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, serverPickup := range s.activePickups {
		pickup.Destroy(serverPickup)
		delete(s.activePickups, id)
	}
	for id, p := range s.pickups {
		s.grid.removePickup(p)
		delete(s.pickups, id)
	}
}

// CreateCheckpointParams holds parameters for creating a dynamic checkpoint.
type CreateCheckpointParams struct {
	X, Y, Z        float32
	Size           float32
	StreamDistance float32
	Worlds         []int32
	Interiors      []int32
	Players        []int32
	Areas          []int32
	Priority       int32
}

// CreateCheckpoint creates a new dynamic checkpoint.
func (s *Streamer) CreateCheckpoint(params CreateCheckpointParams) (*DynamicCheckpoint, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultCPStreamDistance
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeCheckpoint].Next()
	cp := &DynamicCheckpoint{
		baseEntity: newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		Size:       params.Size,
	}

	applyFilters(&cp.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.checkpoints[id] = cp
	s.grid.addCheckpoint(cp)
	return cp, nil
}

// DestroyCheckpoint removes a dynamic checkpoint.
func (s *Streamer) DestroyCheckpoint(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, ok := s.checkpoints[id]
	if !ok {
		return fmt.Errorf("streamer: checkpoint %d not found", id)
	}

	for _, ps := range s.playerStates {
		if ps.VisibleCheckpoint == id {
			checkpoint.Disable(ps.Player)
			ps.VisibleCheckpoint = InvalidID
		}
	}

	s.grid.removeCheckpoint(cp)
	delete(s.checkpoints, id)
	return nil
}

// IsValidCheckpoint returns true if the checkpoint ID exists.
func (s *Streamer) IsValidCheckpoint(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.checkpoints[id]
	return ok
}

// IsPlayerInDynamicCP returns true if the player is in the specified checkpoint.
func (s *Streamer) IsPlayerInDynamicCP(playerID, cpID int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return false
	}
	if ps.VisibleCheckpoint != cpID {
		return false
	}
	return checkpoint.IsPlayerIn(ps.Player)
}

// GetPlayerVisibleDynamicCP returns the currently visible dynamic checkpoint for a player.
func (s *Streamer) GetPlayerVisibleDynamicCP(playerID int32) int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return InvalidID
	}
	return ps.VisibleCheckpoint
}

// CountCheckpoints returns the number of dynamic checkpoints.
func (s *Streamer) CountCheckpoints() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.checkpoints)
}

// TogglePlayerDynamicCP toggles a dynamic checkpoint for a specific player.
func (s *Streamer) TogglePlayerDynamicCP(playerID, cpID int32, toggle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return
	}
	if toggle {
		delete(ps.DisabledItems[EntityTypeCheckpoint], cpID)
	} else {
		ps.DisabledItems[EntityTypeCheckpoint][cpID] = struct{}{}
	}
}

// DestroyAllCheckpoints removes all dynamic checkpoints.
func (s *Streamer) DestroyAllCheckpoints() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ps := range s.playerStates {
		if ps.VisibleCheckpoint != InvalidID {
			checkpoint.Disable(ps.Player)
			ps.VisibleCheckpoint = InvalidID
		}
	}
	for id, cp := range s.checkpoints {
		s.grid.removeCheckpoint(cp)
		delete(s.checkpoints, id)
	}
}

// CreateRaceCheckpointParams holds parameters for creating a dynamic race checkpoint.
type CreateRaceCheckpointParams struct {
	Type                int32
	X, Y, Z             float32
	NextX, NextY, NextZ float32
	Size                float32
	StreamDistance      float32
	Worlds              []int32
	Interiors           []int32
	Players             []int32
	Areas               []int32
	Priority            int32
}

// CreateRaceCheckpoint creates a new dynamic race checkpoint.
func (s *Streamer) CreateRaceCheckpoint(params CreateRaceCheckpointParams) (*DynamicRaceCheckpoint, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultRaceCPStreamDistance
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeRaceCheckpoint].Next()
	rcp := &DynamicRaceCheckpoint{
		baseEntity: newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		Type:       params.Type,
		Next:       Vector3{X: params.NextX, Y: params.NextY, Z: params.NextZ},
		Size:       params.Size,
	}

	applyFilters(&rcp.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.raceCheckpoints[id] = rcp
	s.grid.addRaceCheckpoint(rcp)
	return rcp, nil
}

// DestroyRaceCheckpoint removes a dynamic race checkpoint.
func (s *Streamer) DestroyRaceCheckpoint(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rcp, ok := s.raceCheckpoints[id]
	if !ok {
		return fmt.Errorf("streamer: race checkpoint %d not found", id)
	}

	for _, ps := range s.playerStates {
		if ps.VisibleRaceCheckpoint == id {
			racecheckpoint.Disable(ps.Player)
			ps.VisibleRaceCheckpoint = InvalidID
		}
	}

	s.grid.removeRaceCheckpoint(rcp)
	delete(s.raceCheckpoints, id)
	return nil
}

// IsValidRaceCheckpoint returns true if the race checkpoint ID exists.
func (s *Streamer) IsValidRaceCheckpoint(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.raceCheckpoints[id]
	return ok
}

// IsPlayerInDynamicRaceCP returns true if the player is in the specified race checkpoint.
func (s *Streamer) IsPlayerInDynamicRaceCP(playerID, rcpID int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return false
	}
	if ps.VisibleRaceCheckpoint != rcpID {
		return false
	}
	return racecheckpoint.IsPlayerIn(ps.Player)
}

// GetPlayerVisibleDynamicRaceCP returns the currently visible dynamic race checkpoint.
func (s *Streamer) GetPlayerVisibleDynamicRaceCP(playerID int32) int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return InvalidID
	}
	return ps.VisibleRaceCheckpoint
}

// CountRaceCheckpoints returns the number of dynamic race checkpoints.
func (s *Streamer) CountRaceCheckpoints() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.raceCheckpoints)
}

// TogglePlayerDynamicRaceCP toggles a dynamic race checkpoint for a specific player.
func (s *Streamer) TogglePlayerDynamicRaceCP(playerID, rcpID int32, toggle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return
	}
	if toggle {
		delete(ps.DisabledItems[EntityTypeRaceCheckpoint], rcpID)
	} else {
		ps.DisabledItems[EntityTypeRaceCheckpoint][rcpID] = struct{}{}
	}
}

// DestroyAllRaceCheckpoints removes all dynamic race checkpoints.
func (s *Streamer) DestroyAllRaceCheckpoints() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ps := range s.playerStates {
		if ps.VisibleRaceCheckpoint != InvalidID {
			racecheckpoint.Disable(ps.Player)
			ps.VisibleRaceCheckpoint = InvalidID
		}
	}
	for id, rcp := range s.raceCheckpoints {
		s.grid.removeRaceCheckpoint(rcp)
		delete(s.raceCheckpoints, id)
	}
}

// CreateMapIconParams holds parameters for creating a dynamic map icon.
type CreateMapIconParams struct {
	X, Y, Z        float32
	Type           int32
	Color          uint32
	Style          int32
	StreamDistance float32
	Worlds         []int32
	Interiors      []int32
	Players        []int32
	Areas          []int32
	Priority       int32
}

// CreateMapIcon creates a new dynamic map icon.
func (s *Streamer) CreateMapIcon(params CreateMapIconParams) (*DynamicMapIcon, error) {
	if params.StreamDistance <= 0 {
		params.StreamDistance = DefaultMapIconStreamDistance
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.idAlloc[EntityTypeMapIcon].Next()
	mi := &DynamicMapIcon{
		baseEntity: newBaseEntity(id, Vector3{X: params.X, Y: params.Y, Z: params.Z}, params.StreamDistance),
		Type:       params.Type,
		Color:      params.Color,
		Style:      params.Style,
	}

	applyFilters(&mi.baseEntity, params.Worlds, params.Interiors, params.Players, params.Areas, params.Priority)
	s.mapIcons[id] = mi
	s.grid.addMapIcon(mi)
	return mi, nil
}

// DestroyMapIcon removes a dynamic map icon.
func (s *Streamer) DestroyMapIcon(id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mi, ok := s.mapIcons[id]
	if !ok {
		return fmt.Errorf("streamer: map icon %d not found", id)
	}

	for _, ps := range s.playerStates {
		if slot, exists := ps.MapIcons[id]; exists {
			players.RemoveMapIcon(ps.Player, slot)
			ps.freeMapIconSlot(slot)
			delete(ps.MapIcons, id)
		}
	}

	s.grid.removeMapIcon(mi)
	delete(s.mapIcons, id)
	return nil
}

// IsValidMapIcon returns true if the map icon ID exists.
func (s *Streamer) IsValidMapIcon(id int32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.mapIcons[id]
	return ok
}

// CountMapIcons returns the number of dynamic map icons.
func (s *Streamer) CountMapIcons() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.mapIcons)
}

// DestroyAllMapIcons removes all dynamic map icons.
func (s *Streamer) DestroyAllMapIcons() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ps := range s.playerStates {
		for id, slot := range ps.MapIcons {
			players.RemoveMapIcon(ps.Player, slot)
			ps.freeMapIconSlot(slot)
			delete(ps.MapIcons, id)
		}
	}
	for id, mi := range s.mapIcons {
		s.grid.removeMapIcon(mi)
		delete(s.mapIcons, id)
	}
}

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

	// Update for all players who have this label streamed in.
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

// CreateCircle creates a dynamic circular area (2D).
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

// CreateRectangle creates a dynamic rectangular area (2D).
func (s *Streamer) CreateRectangle(minX, minY, maxX, maxY float32, opts ...AreaOption) (*DynamicArea, error) {
	return s.createArea(AreaTypeRectangle, &RectangleShape{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}, opts...)
}

// CreateCuboid creates a dynamic cuboid area (3D).
func (s *Streamer) CreateCuboid(minX, minY, minZ, maxX, maxY, maxZ float32, opts ...AreaOption) (*DynamicArea, error) {
	return s.createArea(AreaTypeCuboid, &CuboidShape{MinX: minX, MinY: minY, MinZ: minZ, MaxX: maxX, MaxY: maxY, MaxZ: maxZ}, opts...)
}

// CreatePolygon creates a dynamic polygon area (2D with Z bounds).
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
	for id, a := range s.areas {
		s.grid.removeArea(a)
		delete(s.areas, id)
	}
}

// AreaOption configures area creation.
type AreaOption func(*areaOptions)

type areaOptions struct {
	StreamDistance float32
	Worlds         []int32
	Interiors      []int32
	Players        []int32
	Priority       int32
}

func defaultAreaOptions() *areaOptions {
	return &areaOptions{
		StreamDistance: 200.0, // Areas use a generous default.
	}
}

// WithAreaStreamDistance sets the stream distance for the area.
func WithAreaStreamDistance(dist float32) AreaOption {
	return func(o *areaOptions) {
		if dist > 0 {
			o.StreamDistance = dist
		}
	}
}

// WithAreaWorlds sets the virtual worlds for the area.
func WithAreaWorlds(worlds ...int32) AreaOption {
	return func(o *areaOptions) {
		o.Worlds = worlds
	}
}

// WithAreaInteriors sets the interiors for the area.
func WithAreaInteriors(interiors ...int32) AreaOption {
	return func(o *areaOptions) {
		o.Interiors = interiors
	}
}

// WithAreaPlayers sets the players that can see/trigger the area.
func WithAreaPlayers(playerIDs ...int32) AreaOption {
	return func(o *areaOptions) {
		o.Players = playerIDs
	}
}

// WithAreaPriority sets the priority for the area.
func WithAreaPriority(priority int32) AreaOption {
	return func(o *areaOptions) {
		o.Priority = priority
	}
}

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
	for id, a := range s.actors {
		s.grid.removeActor(a)
		delete(s.actors, id)
	}
}

// SetTickRate updates the global tick rate.
func (s *Streamer) SetTickRate(rate int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rate > 0 {
		s.cfg.TickRate = rate
	}
}

// GetTickRate returns the current global tick rate.
func (s *Streamer) GetTickRate() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.TickRate
}

// SetPlayerTickRate sets a per-player tick rate override.
func (s *Streamer) SetPlayerTickRate(playerID, rate int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ps, ok := s.playerStates[playerID]; ok {
		ps.TickRate = rate
	}
}

// SetRadiusMultiplier sets the radius multiplier for a player and entity type.
func (s *Streamer) SetRadiusMultiplier(playerID int32, entityType EntityType, multiplier float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ps, ok := s.playerStates[playerID]; ok {
		if entityType >= 0 && int(entityType) < MaxEntityTypes {
			ps.RadiusMultiplier[entityType] = multiplier
		}
	}
}

// GetRadiusMultiplier returns the radius multiplier for a player and entity type.
func (s *Streamer) GetRadiusMultiplier(playerID int32, entityType EntityType) float32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ps, ok := s.playerStates[playerID]; ok {
		if entityType >= 0 && int(entityType) < MaxEntityTypes {
			return ps.RadiusMultiplier[entityType]
		}
	}
	return 1.0
}

func (s *Streamer) onTick(_ context.Context, event *omp.TickEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.globalTickCount++

	// Update player positions.
	for _, ps := range s.playerStates {
		tickRate := ps.TickRate
		if tickRate <= 0 {
			tickRate = s.cfg.TickRate
		}
		ps.TickCount++
		if ps.TickCount < tickRate {
			continue
		}
		ps.TickCount = 0

		var x, y, z float32
		players.GetPos(ps.Player, &x, &y, &z)
		ps.Position = Vector3{X: x, Y: y, Z: z}
		ps.WorldID = players.GetVirtualWorld(ps.Player)
		ps.Interior = players.GetInterior(ps.Player)

		s.processPlayerUpdate(ps)
	}
	return nil
}

func (s *Streamer) onPlayerConnect(_ context.Context, event *omp.PlayerConnectEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerID := players.GetID(event.Player)
	s.playerStates[playerID] = newPlayerState(playerID, event.Player)
	return nil
}

func (s *Streamer) onPlayerDisconnect(_ context.Context, event *omp.PlayerDisconnectEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}

	// Clean up streamed entities for this player.
	for id, serverObj := range ps.Objects {
		playerobject.Destroy(ps.Player, serverObj)
		delete(ps.Objects, id)
	}
	for id, serverTL := range ps.TextLabels {
		playertextlabel.Destroy(ps.Player, serverTL)
		delete(ps.TextLabels, id)
	}
	for id, slot := range ps.MapIcons {
		players.RemoveMapIcon(ps.Player, slot)
		delete(ps.MapIcons, id)
	}

	delete(s.playerStates, playerID)
	return nil
}

func (s *Streamer) onPlayerEnterCheckpoint(_ context.Context, event *omp.PlayerEnterCheckpointEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}
	if ps.VisibleCheckpoint != InvalidID {
		s.cfg.EventHandler.OnPlayerEnterDynamicCheckpoint(playerID, ps.VisibleCheckpoint)
	}
	return nil
}

func (s *Streamer) onPlayerLeaveCheckpoint(_ context.Context, event *omp.PlayerLeaveCheckpointEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}
	if ps.VisibleCheckpoint != InvalidID {
		s.cfg.EventHandler.OnPlayerLeaveDynamicCheckpoint(playerID, ps.VisibleCheckpoint)
	}
	return nil
}

func (s *Streamer) onPlayerEnterRaceCheckpoint(_ context.Context, event *omp.PlayerEnterRaceCheckpointEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}
	if ps.VisibleRaceCheckpoint != InvalidID {
		s.cfg.EventHandler.OnPlayerEnterDynamicRaceCP(playerID, ps.VisibleRaceCheckpoint)
	}
	return nil
}

func (s *Streamer) onPlayerLeaveRaceCheckpoint(_ context.Context, event *omp.PlayerLeaveRaceCheckpointEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	ps, ok := s.playerStates[playerID]
	if !ok {
		return nil
	}
	if ps.VisibleRaceCheckpoint != InvalidID {
		s.cfg.EventHandler.OnPlayerLeaveDynamicRaceCP(playerID, ps.VisibleRaceCheckpoint)
	}
	return nil
}

func (s *Streamer) onPlayerPickUpPickup(_ context.Context, event *omp.PlayerPickUpPickupEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	serverPickupID := pickup.GetID(event.Pickup)

	// Find the dynamic pickup that matches this server pickup.
	for dynamicID, sp := range s.activePickups {
		if pickup.GetID(sp) == serverPickupID {
			s.cfg.EventHandler.OnPlayerPickUpDynamicPickup(playerID, dynamicID)
			break
		}
	}
	return nil
}

func (s *Streamer) onPlayerGiveDamageActor(_ context.Context, event *omp.PlayerGiveDamageActorEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playerID := players.GetID(event.Player)
	serverActorID := actor.GetID(event.Actor)

	for dynamicID, sa := range s.activeActors {
		if actor.GetID(sa) == serverActorID {
			s.cfg.EventHandler.OnPlayerGiveDamageDynamicActor(playerID, dynamicID, event.Amount, event.Weapon, event.Part)
			break
		}
	}
	return nil
}

// applyFilters sets the world/interior/player/area filters on a base entity.
func applyFilters(e *baseEntity, worlds, interiors, playerIDs, areaIDs []int32, priority int32) {
	e.Priority = priority

	if len(worlds) > 0 {
		e.Worlds = make(map[int32]struct{}, len(worlds))
		for _, w := range worlds {
			e.Worlds[w] = struct{}{}
		}
	}
	if len(interiors) > 0 {
		e.Interiors = make(map[int32]struct{}, len(interiors))
		for _, i := range interiors {
			e.Interiors[i] = struct{}{}
		}
	}
	if len(playerIDs) > 0 {
		e.Players = make(map[int32]struct{}, len(playerIDs))
		for _, p := range playerIDs {
			e.Players[p] = struct{}{}
		}
	}
	if len(areaIDs) > 0 {
		e.Areas = make(map[int32]struct{}, len(areaIDs))
		for _, a := range areaIDs {
			e.Areas[a] = struct{}{}
		}
	}
}

// TogglePlayerItem toggles a specific item type for a player.
func (s *Streamer) TogglePlayerItem(playerID int32, entityType EntityType, id int32, toggle bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return
	}
	if entityType < 0 || int(entityType) >= MaxEntityTypes {
		return
	}
	if toggle {
		delete(ps.DisabledItems[entityType], id)
	} else {
		ps.DisabledItems[entityType][id] = struct{}{}
	}
}

// GetDistanceToItem returns the distance from a point to a dynamic entity.
func (s *Streamer) GetDistanceToItem(x, y, z float32, entityType EntityType, id int32) (float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pos Vector3
	switch entityType {
	case EntityTypeObject:
		if obj, ok := s.objects[id]; ok {
			pos = obj.Position
		} else {
			return 0, fmt.Errorf("streamer: object %d not found", id)
		}
	case EntityTypePickup:
		if p, ok := s.pickups[id]; ok {
			pos = p.Position
		} else {
			return 0, fmt.Errorf("streamer: pickup %d not found", id)
		}
	case EntityTypeCheckpoint:
		if cp, ok := s.checkpoints[id]; ok {
			pos = cp.Position
		} else {
			return 0, fmt.Errorf("streamer: checkpoint %d not found", id)
		}
	case EntityTypeRaceCheckpoint:
		if rcp, ok := s.raceCheckpoints[id]; ok {
			pos = rcp.Position
		} else {
			return 0, fmt.Errorf("streamer: race checkpoint %d not found", id)
		}
	case EntityTypeMapIcon:
		if mi, ok := s.mapIcons[id]; ok {
			pos = mi.Position
		} else {
			return 0, fmt.Errorf("streamer: map icon %d not found", id)
		}
	case EntityTypeTextLabel:
		if tl, ok := s.textLabels[id]; ok {
			pos = tl.Position
		} else {
			return 0, fmt.Errorf("streamer: text label %d not found", id)
		}
	case EntityTypeArea:
		if a, ok := s.areas[id]; ok {
			pos = a.Shape.Center()
		} else {
			return 0, fmt.Errorf("streamer: area %d not found", id)
		}
	case EntityTypeActor:
		if a, ok := s.actors[id]; ok {
			pos = a.Position
		} else {
			return 0, fmt.Errorf("streamer: actor %d not found", id)
		}
	default:
		return 0, fmt.Errorf("streamer: invalid entity type %d", entityType)
	}

	return distance3D(x, y, z, pos.X, pos.Y, pos.Z), nil
}

// UpdateEx forces an immediate position update and re-stream for a player.
func (s *Streamer) UpdateEx(playerID int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ps, ok := s.playerStates[playerID]
	if !ok {
		return
	}

	var x, y, z float32
	players.GetPos(ps.Player, &x, &y, &z)
	ps.Position = Vector3{X: x, Y: y, Z: z}
	ps.WorldID = players.GetVirtualWorld(ps.Player)
	ps.Interior = players.GetInterior(ps.Player)

	s.processPlayerUpdate(ps)
}

// Update forces an update for all players.
func (s *Streamer) Update() {
	s.mu.Lock()
	defer s.mu.Unlock()

	maxPlayers := core.MaxPlayers()
	for pid := int32(0); pid < maxPlayers; pid++ {
		ps, ok := s.playerStates[pid]
		if !ok {
			continue
		}
		var x, y, z float32
		players.GetPos(ps.Player, &x, &y, &z)
		ps.Position = Vector3{X: x, Y: y, Z: z}
		ps.WorldID = players.GetVirtualWorld(ps.Player)
		ps.Interior = players.GetInterior(ps.Player)
		s.processPlayerUpdate(ps)
	}
}
