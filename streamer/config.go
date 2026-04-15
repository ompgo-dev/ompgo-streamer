package streamer

// Option configures the Streamer.
type Option func(*config)

// config holds the streamer configuration.
type config struct {
	TickRate     int32
	CellSize     float32
	CellDistance float32

	VisibleObjects    int32
	VisiblePickups    int32
	VisibleMapIcons   int32
	VisibleTextLabels int32
	VisibleActors     int32

	TypePriority [MaxEntityTypes]EntityType

	EventHandler EventHandler
}

func defaultConfig() *config {
	return &config{
		TickRate:          DefaultTickRate,
		CellSize:          DefaultCellSize,
		CellDistance:      DefaultCellDistance,
		VisibleObjects:    DefaultVisibleObjects,
		VisiblePickups:    DefaultVisiblePickups,
		VisibleMapIcons:   DefaultVisibleMapIcons,
		VisibleTextLabels: DefaultVisibleTextLabels,
		VisibleActors:     DefaultVisibleActors,
		TypePriority: [MaxEntityTypes]EntityType{
			EntityTypeObject,
			EntityTypePickup,
			EntityTypeCheckpoint,
			EntityTypeRaceCheckpoint,
			EntityTypeMapIcon,
			EntityTypeTextLabel,
			EntityTypeArea,
			EntityTypeActor,
		},
		EventHandler: BaseEventHandler{},
	}
}

// WithTickRate sets the update tick rate.
func WithTickRate(rate int32) Option {
	return func(c *config) {
		if rate > 0 {
			c.TickRate = rate
		}
	}
}

// WithCellSize sets the grid cell size.
func WithCellSize(size float32) Option {
	return func(c *config) {
		if size > 0 {
			c.CellSize = size
		}
	}
}

// WithCellDistance sets the grid cell distance threshold.
func WithCellDistance(dist float32) Option {
	return func(c *config) {
		if dist > 0 {
			c.CellDistance = dist
		}
	}
}

// WithVisibleObjects sets the max visible objects per player.
func WithVisibleObjects(n int32) Option {
	return func(c *config) {
		if n > 0 {
			c.VisibleObjects = n
		}
	}
}

// WithVisiblePickups sets the max visible pickups.
func WithVisiblePickups(n int32) Option {
	return func(c *config) {
		if n > 0 {
			c.VisiblePickups = n
		}
	}
}

// WithVisibleMapIcons sets the max visible map icons per player.
func WithVisibleMapIcons(n int32) Option {
	return func(c *config) {
		if n > 0 {
			c.VisibleMapIcons = n
		}
	}
}

// WithVisibleTextLabels sets the max visible text labels per player.
func WithVisibleTextLabels(n int32) Option {
	return func(c *config) {
		if n > 0 {
			c.VisibleTextLabels = n
		}
	}
}

// WithVisibleActors sets the max visible actors.
func WithVisibleActors(n int32) Option {
	return func(c *config) {
		if n > 0 {
			c.VisibleActors = n
		}
	}
}

// WithTypePriority sets the entity type processing priority order.
func WithTypePriority(priority [MaxEntityTypes]EntityType) Option {
	return func(c *config) {
		c.TypePriority = priority
	}
}

// WithEventHandler registers an event handler for streamer callbacks.
func WithEventHandler(handler EventHandler) Option {
	return func(c *config) {
		if handler != nil {
			c.EventHandler = handler
		}
	}
}
