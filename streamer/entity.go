package streamer

// baseEntity contains fields shared across all dynamic entity types.
type baseEntity struct {
	ID               int32
	Position         Vector3
	StreamDistance   float32
	Worlds           map[int32]struct{}
	Interiors        map[int32]struct{}
	Players          map[int32]struct{}
	Areas            map[int32]struct{}
	Priority         int32
	InverseAreaCheck bool
	StreamCallbacks  bool
	CellID           CellID
	InGlobalCell     bool
}

func newBaseEntity(id int32, pos Vector3, streamDist float32) baseEntity {
	return baseEntity{
		ID:             id,
		Position:       pos,
		StreamDistance: streamDist,
		Worlds:         map[int32]struct{}{AnyWorld: {}},
		Interiors:      map[int32]struct{}{AnyInterior: {}},
		Players:        map[int32]struct{}{AnyPlayer: {}},
		Areas:          make(map[int32]struct{}),
	}
}

// isVisibleTo checks world, interior, and player filters.
func (e *baseEntity) isVisibleTo(playerID, worldID, interiorID int32) bool {
	if _, ok := e.Worlds[AnyWorld]; !ok {
		if _, ok := e.Worlds[worldID]; !ok {
			return false
		}
	}
	if _, ok := e.Interiors[AnyInterior]; !ok {
		if _, ok := e.Interiors[interiorID]; !ok {
			return false
		}
	}
	if _, ok := e.Players[AnyPlayer]; !ok {
		if _, ok := e.Players[playerID]; !ok {
			return false
		}
	}
	return true
}

// streamDistSq returns the squared stream distance.
func (e *baseEntity) streamDistSq() float32 {
	return e.StreamDistance * e.StreamDistance
}

// DynamicObject represents a streamed object.
type DynamicObject struct {
	baseEntity
	ModelID           int32
	Rotation          Vector3
	DrawDistance      float32
	NoCameraCollision bool
	Materials         map[int32]*ObjectMaterial
	MaterialTexts     map[int32]*ObjectMaterialText
	Move              *ObjectMoveData
	AttachedTo        *AttachData
}

// ObjectMaterial represents a texture applied to an object material slot.
type ObjectMaterial struct {
	ModelID     int32
	TXDName     string
	TextureName string
	Color       uint32
}

// ObjectMaterialText represents text applied to an object material slot.
type ObjectMaterialText struct {
	Text          string
	MaterialSize  int32
	FontFace      string
	FontSize      int32
	Bold          bool
	FontColor     uint32
	BackColor     uint32
	TextAlignment int32
}

// ObjectMoveData describes an object movement in progress.
type ObjectMoveData struct {
	Target    Vector3
	TargetRot Vector3
	Speed     float32
}

// AttachData describes how an entity is attached to a player, vehicle, or object.
type AttachData struct {
	Type     AttachType
	ID       int32
	Offset   Vector3
	Rotation Vector3
}

// AttachType identifies what an entity is attached to.
type AttachType int32

const (
	AttachTypeNone    AttachType = 0
	AttachTypePlayer  AttachType = 1
	AttachTypeVehicle AttachType = 2
	AttachTypeObject  AttachType = 3
)

// DynamicPickup represents a streamed pickup.
type DynamicPickup struct {
	baseEntity
	ModelID      int32
	Type         int32
	VirtualWorld int32
}

// DynamicCheckpoint represents a streamed checkpoint.
type DynamicCheckpoint struct {
	baseEntity
	Size float32
}

// DynamicRaceCheckpoint represents a streamed race checkpoint.
type DynamicRaceCheckpoint struct {
	baseEntity
	Type int32
	Next Vector3
	Size float32
}

// DynamicMapIcon represents a streamed map icon.
type DynamicMapIcon struct {
	baseEntity
	Type  int32
	Color uint32
	Style int32
}

// DynamicTextLabel represents a streamed 3D text label.
type DynamicTextLabel struct {
	baseEntity
	Text              string
	Color             uint32
	DrawDistance      float32
	TestLOS           bool
	AttachedPlayerID  int32
	AttachedVehicleID int32
}

// DynamicActor represents a streamed actor.
type DynamicActor struct {
	baseEntity
	ModelID      int32
	Rotation     float32
	Invulnerable bool
	Health       float32
	Animation    *ActorAnimation
}

// ActorAnimation describes an animation applied to an actor.
type ActorAnimation struct {
	Library string
	Name    string
	Delta   float32
	Loop    bool
	LockX   bool
	LockY   bool
	Freeze  bool
	Time    int32
}
