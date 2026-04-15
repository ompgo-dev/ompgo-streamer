package streamer

import "sync/atomic"

// IDAllocator generates unique sequential IDs starting from 1.
type IDAllocator struct {
	next atomic.Int32
}

// NewIDAllocator creates a new allocator starting at 1.
func NewIDAllocator() *IDAllocator {
	a := &IDAllocator{}
	a.next.Store(1)
	return a
}

// Next returns the next available ID.
func (a *IDAllocator) Next() int32 {
	return a.next.Add(1) - 1
}
