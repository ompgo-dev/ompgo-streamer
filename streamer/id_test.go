package streamer

import "testing"

func TestIDAllocator(t *testing.T) {
	alloc := NewIDAllocator()

	tests := []struct {
		name     string
		expected int32
	}{
		{"first ID", 1},
		{"second ID", 2},
		{"third ID", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := alloc.Next()
			if got != tt.expected {
				t.Errorf("Next() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestIDAllocatorConcurrency(t *testing.T) {
	alloc := NewIDAllocator()
	done := make(chan int32, 100)

	for i := 0; i < 100; i++ {
		go func() {
			done <- alloc.Next()
		}()
	}

	seen := make(map[int32]struct{})
	for i := 0; i < 100; i++ {
		id := <-done
		if _, exists := seen[id]; exists {
			t.Errorf("duplicate ID: %d", id)
		}
		seen[id] = struct{}{}
	}

	if len(seen) != 100 {
		t.Errorf("expected 100 unique IDs, got %d", len(seen))
	}
}
