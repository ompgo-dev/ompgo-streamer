# ompgo-streamer

Minimal open.mp style dynamic streamer package for ompgo.

This package lets an ompgo gamemode create dynamic objects, pickups, checkpoints, race checkpoints, map icons, text labels, areas, and actors, then stream them automatically per player.

## Install

```bash
go get github.com/ompgo-dev/ompgo-streamer
```

## Runtime bootstrap

Register the streamer by passing `streamer.WithStreamer(...)` to `runtime.Bootstrap(...)`. This creates one runtime-managed streamer instance and stops it automatically during component shutdown.

```go
package main

import (
	"context"

	streamer "github.com/ompgo-dev/ompgo-streamer/streamer"
	"github.com/ompgo-dev/ompgo/pkg/omp"
	"github.com/ompgo-dev/ompgo/pkg/runtime"
)

type GameMode struct {
	omp.BaseEventHandler

	streamer *streamer.Streamer
}

func (gm *GameMode) OnLoad(ctx context.Context) error {
	gm.streamer = streamer.GetStreamer()

	_, err := gm.streamer.CreateObject(streamer.CreateObjectParams{
		ModelID:        19379,
		X:              0,
		Y:              0,
		Z:              3,
		RotX:           0,
		RotY:           0,
		RotZ:           0,
		StreamDistance: 300,
		DrawDistance:   300,
	})
	if err != nil {
		return err
	}

	_, err = gm.streamer.CreateCheckpoint(streamer.CreateCheckpointParams{
		X:              10,
		Y:              0,
		Z:              3,
		Size:           3,
		StreamDistance: 200,
	})
	return err
}

func NewGamemode() runtime.Gamemode {
	return &GameMode{}
}

func init() {
	runtime.Bootstrap(
		runtime.WithComponentName("ompgo_streamer_example"),
		runtime.WithGamemode(NewGamemode),
		streamer.WithStreamer(
			streamer.WithTickRate(50),
		),
	)
}

func main() {}
```

Use `streamer.GetStreamer()` from `OnLoad` or later. It returns the runtime-managed streamer after bootstrap setup has run.

## Manual setup

Manual setup still works if you want to own the streamer instance yourself instead of using the runtime-managed one.

```go
func (gm *GameMode) OnLoad(ctx context.Context) error {
	gm.streamer = streamer.New()
	gm.streamer.Start()
	return nil
}
```

If you use the manual path, keep a reference to that instance yourself and call `Stop()` during shutdown.

## Notes

- Use `streamer.WithStreamer(...)` inside `runtime.Bootstrap(...)` when you want the runtime to create and clean up the streamer for you.
- Access the runtime-managed instance with `streamer.GetStreamer()` from `OnLoad` or later.
- Call `Start()` once after creating the streamer yourself when using the manual setup path.
- Create dynamic entities with methods like `CreateObject`, `CreatePickup`, `CreateTextLabel`, and `CreateActor`.
- If your gamemode manages the streamer manually, call `Stop()` in your shutdown path to unregister the streamer's event handlers.