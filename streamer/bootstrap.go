package streamer

import (
	"context"
	"sync"

	"github.com/ompgo-dev/ompgo/pkg/runtime"
)

var (
	instanceMu sync.RWMutex
	instance   *Streamer
)

// GetStreamer returns the runtime-managed streamer created by WithStreamer.
// It returns nil when the streamer has not been configured through runtime bootstrap.
func GetStreamer() *Streamer {
	instanceMu.RLock()
	defer instanceMu.RUnlock()
	return instance
}

// WithStreamer returns a runtime option that creates, starts, and exposes a
// streamer instance during runtime bootstrap.
func WithStreamer(opts ...Option) runtime.Option {
	return func(cfg *runtime.Config) {
		if cfg == nil {
			return
		}

		prevSetup := cfg.Setup
		cfg.Setup = func(ctx context.Context) error {
			if prevSetup != nil {
				if err := prevSetup(ctx); err != nil {
					return err
				}
			}

			s := New(opts...)
			s.Start()
			replaceInstance(s)
			return nil
		}

		prevOnFree := cfg.Handlers.OnFree
		cfg.Handlers.OnFree = func(ctx context.Context) error {
			s := clearInstance()
			if s != nil {
				s.Stop()
			}

			if prevOnFree != nil {
				return prevOnFree(ctx)
			}
			return nil
		}
	}
}

func replaceInstance(s *Streamer) {
	instanceMu.Lock()
	prev := instance
	instance = s
	instanceMu.Unlock()

	if prev != nil && prev != s {
		prev.Stop()
	}
}

func clearInstance() *Streamer {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	prev := instance
	instance = nil
	return prev
}
