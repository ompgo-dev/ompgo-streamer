package streamer

import (
	"sync"

	"github.com/ompgo-dev/ompgo/pkg/omp"
	"github.com/ompgo-dev/ompgo/pkg/runtime"
)

// Streamer manages dynamic entity streaming for open.mp via ompgo.
type Streamer struct {
	mu      sync.RWMutex
	cfg     *config
	grid    *grid
	idAlloc [MaxEntityTypes]*IDAllocator

	objects         map[int32]*DynamicObject
	pickups         map[int32]*DynamicPickup
	checkpoints     map[int32]*DynamicCheckpoint
	raceCheckpoints map[int32]*DynamicRaceCheckpoint
	mapIcons        map[int32]*DynamicMapIcon
	textLabels      map[int32]*DynamicTextLabel
	areas           map[int32]*DynamicArea
	actors          map[int32]*DynamicActor

	playerStates map[int32]*playerState

	activePickups map[int32]*omp.Pickup
	activeActors  map[int32]*omp.Actor

	unregisterFns []func()

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
