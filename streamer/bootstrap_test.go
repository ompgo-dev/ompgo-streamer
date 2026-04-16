package streamer

import (
	"context"
	"errors"
	"testing"

	"github.com/ompgo-dev/ompgo/pkg/runtime"
)

func TestWithStreamerCreatesManagedInstanceAndChainsLifecycle(t *testing.T) {
	t.Cleanup(func() {
		if s := clearInstance(); s != nil {
			s.Stop()
		}
	})

	setupCalls := 0
	freeCalls := 0
	cfg := &runtime.Config{
		Setup: func(context.Context) error {
			setupCalls++
			return nil
		},
		Handlers: runtime.ComponentHandlers{
			OnFree: func(context.Context) error {
				freeCalls++
				return nil
			},
		},
	}

	WithStreamer(WithTickRate(25))(cfg)

	if cfg.Setup == nil {
		t.Fatal("expected setup handler")
	}
	if cfg.Handlers.OnFree == nil {
		t.Fatal("expected OnFree handler")
	}

	if err := cfg.Setup(context.Background()); err != nil {
		t.Fatalf("setup returned error: %v", err)
	}

	if setupCalls != 1 {
		t.Fatalf("expected previous setup to run once, got %d", setupCalls)
	}

	s := GetStreamer()
	if s == nil {
		t.Fatal("expected managed streamer instance")
	}
	if got := s.cfg.TickRate; got != 25 {
		t.Fatalf("expected tick rate 25, got %d", got)
	}

	if err := cfg.Handlers.OnFree(context.Background()); err != nil {
		t.Fatalf("on free returned error: %v", err)
	}

	if freeCalls != 1 {
		t.Fatalf("expected previous OnFree to run once, got %d", freeCalls)
	}
	if GetStreamer() != nil {
		t.Fatal("expected managed streamer instance to be cleared")
	}
}

func TestWithStreamerDoesNotCreateInstanceWhenPreviousSetupFails(t *testing.T) {
	t.Cleanup(func() {
		if s := clearInstance(); s != nil {
			s.Stop()
		}
	})

	wantErr := errors.New("setup failed")
	cfg := &runtime.Config{
		Setup: func(context.Context) error {
			return wantErr
		},
	}

	WithStreamer()(cfg)

	err := cfg.Setup(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected setup error %v, got %v", wantErr, err)
	}
	if GetStreamer() != nil {
		t.Fatal("expected no managed streamer instance after setup failure")
	}
}
