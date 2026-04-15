# ompgo-streamer

Minimal open.mp style dynamic streamer package for ompgo.

This package lets an ompgo gamemode create dynamic objects, pickups, checkpoints, race checkpoints, map icons, text labels, areas, and actors, then stream them automatically per player.

## Install

```bash
go get github.com/ompgo-dev/ompgo-streamer
```

## Basic usage

Create one streamer instance when your gamemode loads, call `Start`, and then create dynamic items through that instance.

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
	gm.streamer = streamer.New()
	gm.streamer.Start()

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
	)
}

func main() {}
```

## Notes

- Call `Start()` once after creating the streamer.
- Create dynamic entities with methods like `CreateObject`, `CreatePickup`, `CreateTextLabel`, and `CreateActor`.
- If your gamemode has a shutdown hook, call `Stop()` to unregister the streamer's event handlers.