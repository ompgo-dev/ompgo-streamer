package streamer

// applyFilters sets the world, interior, player, and area filters on a base entity.
func applyFilters(e *baseEntity, worlds, interiors, playerIDs, areaIDs []int32, priority int32) {
	e.Priority = priority

	if len(worlds) > 0 {
		e.Worlds = make(map[int32]struct{}, len(worlds))
		for _, worldID := range worlds {
			e.Worlds[worldID] = struct{}{}
		}
	}
	if len(interiors) > 0 {
		e.Interiors = make(map[int32]struct{}, len(interiors))
		for _, interiorID := range interiors {
			e.Interiors[interiorID] = struct{}{}
		}
	}
	if len(playerIDs) > 0 {
		e.Players = make(map[int32]struct{}, len(playerIDs))
		for _, playerID := range playerIDs {
			e.Players[playerID] = struct{}{}
		}
	}
	if len(areaIDs) > 0 {
		e.Areas = make(map[int32]struct{}, len(areaIDs))
		for _, areaID := range areaIDs {
			e.Areas[areaID] = struct{}{}
		}
	}
}
