package streamer

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
	return &areaOptions{StreamDistance: 200.0}
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

// WithAreaPlayers sets the players that can see or trigger the area.
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
