package streamer

// EntityType represents the type of a dynamic entity.
type EntityType int32

const (
	EntityTypeObject         EntityType = 0
	EntityTypePickup         EntityType = 1
	EntityTypeCheckpoint     EntityType = 2
	EntityTypeRaceCheckpoint EntityType = 3
	EntityTypeMapIcon        EntityType = 4
	EntityTypeTextLabel      EntityType = 5
	EntityTypeArea           EntityType = 6
	EntityTypeActor          EntityType = 7
)

const MaxEntityTypes = 8

// AreaType represents the shape type of a dynamic area.
type AreaType int32

const (
	AreaTypeCircle    AreaType = 0
	AreaTypeCylinder  AreaType = 1
	AreaTypeSphere    AreaType = 2
	AreaTypeRectangle AreaType = 3
	AreaTypeCuboid    AreaType = 4
	AreaTypePolygon   AreaType = 5
)

const MaxAreaTypes = 6

// InvalidID represents an invalid streamer entity ID.
const InvalidID int32 = 0

// Default streaming distances.
const (
	DefaultObjectStreamDistance  float32 = 300.0
	DefaultObjectDrawDistance    float32 = 0.0
	DefaultPickupStreamDistance  float32 = 200.0
	DefaultCPStreamDistance      float32 = 200.0
	DefaultRaceCPStreamDistance  float32 = 200.0
	DefaultMapIconStreamDistance float32 = 200.0
	DefaultTextLabelStreamDistce float32 = 200.0
	DefaultActorStreamDistance   float32 = 200.0
)

// Default limits.
const (
	DefaultVisibleObjects    int32 = 1000
	DefaultVisiblePickups    int32 = 4096
	DefaultVisibleMapIcons   int32 = 100
	DefaultVisibleTextLabels int32 = 2048
	DefaultVisibleActors     int32 = 1000
	DefaultTickRate          int32 = 50
)

// Default grid settings.
const (
	DefaultCellSize     float32 = 300.0
	DefaultCellDistance float32 = 600.0
)

// AnyWorld indicates the entity is visible in all virtual worlds.
const AnyWorld int32 = -1

// AnyInterior indicates the entity is visible in all interiors.
const AnyInterior int32 = -1

// AnyPlayer indicates the entity is visible to all players.
const AnyPlayer int32 = -1

// Vector3 represents a 3D position or direction.
type Vector3 struct {
	X float32
	Y float32
	Z float32
}

// Vector2 represents a 2D position.
type Vector2 struct {
	X float32
	Y float32
}

// DistanceSquared returns the squared Euclidean distance between two 3D points.
func DistanceSquared(a, b Vector3) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return dx*dx + dy*dy + dz*dz
}

// DistanceSquared2D returns the squared Euclidean distance between two 2D points.
func DistanceSquared2D(a, b Vector2) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}
