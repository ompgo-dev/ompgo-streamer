package streamer

// candidate is an entity with a computed distance from the player.
type candidate struct {
	ID       int32
	DistSq   float32
	Priority int32
}

// processPlayerUpdate runs the streaming logic for a single player.
// Must be called with s.mu held.
func (s *Streamer) processPlayerUpdate(ps *playerState) {
	cells := s.grid.nearbyCells(ps.Position.X, ps.Position.Y)

	for _, entityType := range s.cfg.TypePriority {
		switch entityType {
		case EntityTypeObject:
			s.processObjects(ps, cells)
		case EntityTypePickup:
			s.processPickups(ps, cells)
		case EntityTypeCheckpoint:
			s.processCheckpoints(ps, cells)
		case EntityTypeRaceCheckpoint:
			s.processRaceCheckpoints(ps, cells)
		case EntityTypeMapIcon:
			s.processMapIcons(ps, cells)
		case EntityTypeTextLabel:
			s.processTextLabels(ps, cells)
		case EntityTypeArea:
			s.processAreas(ps, cells)
		case EntityTypeActor:
			s.processActors(ps, cells)
		}
	}
}
