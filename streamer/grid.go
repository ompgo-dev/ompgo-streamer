package streamer

import "math"

// CellID identifies a grid cell by its column and row.
type CellID struct {
	X int32
	Y int32
}

// grid provides spatial partitioning for entities.
type grid struct {
	cellSize     float32
	cellDistance float32

	cells      map[CellID]*cell
	globalCell *cell
}

// cell holds references to dynamic entities within a spatial region.
type cell struct {
	Objects         map[int32]*DynamicObject
	Pickups         map[int32]*DynamicPickup
	Checkpoints     map[int32]*DynamicCheckpoint
	RaceCheckpoints map[int32]*DynamicRaceCheckpoint
	MapIcons        map[int32]*DynamicMapIcon
	TextLabels      map[int32]*DynamicTextLabel
	Areas           map[int32]*DynamicArea
	Actors          map[int32]*DynamicActor
}

func newCell() *cell {
	return &cell{
		Objects:         make(map[int32]*DynamicObject),
		Pickups:         make(map[int32]*DynamicPickup),
		Checkpoints:     make(map[int32]*DynamicCheckpoint),
		RaceCheckpoints: make(map[int32]*DynamicRaceCheckpoint),
		MapIcons:        make(map[int32]*DynamicMapIcon),
		TextLabels:      make(map[int32]*DynamicTextLabel),
		Areas:           make(map[int32]*DynamicArea),
		Actors:          make(map[int32]*DynamicActor),
	}
}

func newGrid(cellSize, cellDistance float32) *grid {
	return &grid{
		cellSize:     cellSize,
		cellDistance: cellDistance,
		cells:        make(map[CellID]*cell),
		globalCell:   newCell(),
	}
}

func (g *grid) getCellID(x, y float32) CellID {
	return CellID{
		X: int32(math.Floor(float64(x / g.cellSize))),
		Y: int32(math.Floor(float64(y / g.cellSize))),
	}
}

func (g *grid) getOrCreateCell(id CellID) *cell {
	c, ok := g.cells[id]
	if !ok {
		c = newCell()
		g.cells[id] = c
	}
	return c
}

// isGlobalRange returns true if the stream distance exceeds cell boundaries.
func (g *grid) isGlobalRange(streamDist float32) bool {
	return streamDist > g.cellDistance
}

// addObject places an object into the appropriate grid cell.
func (g *grid) addObject(obj *DynamicObject) {
	if g.isGlobalRange(obj.StreamDistance) {
		g.globalCell.Objects[obj.ID] = obj
		obj.InGlobalCell = true
		return
	}
	cid := g.getCellID(obj.Position.X, obj.Position.Y)
	c := g.getOrCreateCell(cid)
	c.Objects[obj.ID] = obj
	obj.CellID = cid
	obj.InGlobalCell = false
}

func (g *grid) removeObject(obj *DynamicObject) {
	if obj.InGlobalCell {
		delete(g.globalCell.Objects, obj.ID)
		return
	}
	if c, ok := g.cells[obj.CellID]; ok {
		delete(c.Objects, obj.ID)
		g.cleanupCell(obj.CellID, c)
	}
}

func (g *grid) addPickup(p *DynamicPickup) {
	if g.isGlobalRange(p.StreamDistance) {
		g.globalCell.Pickups[p.ID] = p
		p.InGlobalCell = true
		return
	}
	cid := g.getCellID(p.Position.X, p.Position.Y)
	c := g.getOrCreateCell(cid)
	c.Pickups[p.ID] = p
	p.CellID = cid
	p.InGlobalCell = false
}

func (g *grid) removePickup(p *DynamicPickup) {
	if p.InGlobalCell {
		delete(g.globalCell.Pickups, p.ID)
		return
	}
	if c, ok := g.cells[p.CellID]; ok {
		delete(c.Pickups, p.ID)
		g.cleanupCell(p.CellID, c)
	}
}

func (g *grid) addCheckpoint(cp *DynamicCheckpoint) {
	if g.isGlobalRange(cp.StreamDistance) {
		g.globalCell.Checkpoints[cp.ID] = cp
		cp.InGlobalCell = true
		return
	}
	cid := g.getCellID(cp.Position.X, cp.Position.Y)
	c := g.getOrCreateCell(cid)
	c.Checkpoints[cp.ID] = cp
	cp.CellID = cid
	cp.InGlobalCell = false
}

func (g *grid) removeCheckpoint(cp *DynamicCheckpoint) {
	if cp.InGlobalCell {
		delete(g.globalCell.Checkpoints, cp.ID)
		return
	}
	if c, ok := g.cells[cp.CellID]; ok {
		delete(c.Checkpoints, cp.ID)
		g.cleanupCell(cp.CellID, c)
	}
}

func (g *grid) addRaceCheckpoint(rcp *DynamicRaceCheckpoint) {
	if g.isGlobalRange(rcp.StreamDistance) {
		g.globalCell.RaceCheckpoints[rcp.ID] = rcp
		rcp.InGlobalCell = true
		return
	}
	cid := g.getCellID(rcp.Position.X, rcp.Position.Y)
	c := g.getOrCreateCell(cid)
	c.RaceCheckpoints[rcp.ID] = rcp
	rcp.CellID = cid
	rcp.InGlobalCell = false
}

func (g *grid) removeRaceCheckpoint(rcp *DynamicRaceCheckpoint) {
	if rcp.InGlobalCell {
		delete(g.globalCell.RaceCheckpoints, rcp.ID)
		return
	}
	if c, ok := g.cells[rcp.CellID]; ok {
		delete(c.RaceCheckpoints, rcp.ID)
		g.cleanupCell(rcp.CellID, c)
	}
}

func (g *grid) addMapIcon(mi *DynamicMapIcon) {
	if g.isGlobalRange(mi.StreamDistance) {
		g.globalCell.MapIcons[mi.ID] = mi
		mi.InGlobalCell = true
		return
	}
	cid := g.getCellID(mi.Position.X, mi.Position.Y)
	c := g.getOrCreateCell(cid)
	c.MapIcons[mi.ID] = mi
	mi.CellID = cid
	mi.InGlobalCell = false
}

func (g *grid) removeMapIcon(mi *DynamicMapIcon) {
	if mi.InGlobalCell {
		delete(g.globalCell.MapIcons, mi.ID)
		return
	}
	if c, ok := g.cells[mi.CellID]; ok {
		delete(c.MapIcons, mi.ID)
		g.cleanupCell(mi.CellID, c)
	}
}

func (g *grid) addTextLabel(tl *DynamicTextLabel) {
	if g.isGlobalRange(tl.StreamDistance) {
		g.globalCell.TextLabels[tl.ID] = tl
		tl.InGlobalCell = true
		return
	}
	cid := g.getCellID(tl.Position.X, tl.Position.Y)
	c := g.getOrCreateCell(cid)
	c.TextLabels[tl.ID] = tl
	tl.CellID = cid
	tl.InGlobalCell = false
}

func (g *grid) removeTextLabel(tl *DynamicTextLabel) {
	if tl.InGlobalCell {
		delete(g.globalCell.TextLabels, tl.ID)
		return
	}
	if c, ok := g.cells[tl.CellID]; ok {
		delete(c.TextLabels, tl.ID)
		g.cleanupCell(tl.CellID, c)
	}
}

func (g *grid) addArea(a *DynamicArea) {
	center := a.Shape.Center()
	if g.isGlobalRange(a.StreamDistance) {
		g.globalCell.Areas[a.ID] = a
		a.InGlobalCell = true
		return
	}
	cid := g.getCellID(center.X, center.Y)
	c := g.getOrCreateCell(cid)
	c.Areas[a.ID] = a
	a.CellID = cid
	a.InGlobalCell = false
}

func (g *grid) removeArea(a *DynamicArea) {
	if a.InGlobalCell {
		delete(g.globalCell.Areas, a.ID)
		return
	}
	if c, ok := g.cells[a.CellID]; ok {
		delete(c.Areas, a.ID)
		g.cleanupCell(a.CellID, c)
	}
}

func (g *grid) addActor(a *DynamicActor) {
	if g.isGlobalRange(a.StreamDistance) {
		g.globalCell.Actors[a.ID] = a
		a.InGlobalCell = true
		return
	}
	cid := g.getCellID(a.Position.X, a.Position.Y)
	c := g.getOrCreateCell(cid)
	c.Actors[a.ID] = a
	a.CellID = cid
	a.InGlobalCell = false
}

func (g *grid) removeActor(a *DynamicActor) {
	if a.InGlobalCell {
		delete(g.globalCell.Actors, a.ID)
		return
	}
	if c, ok := g.cells[a.CellID]; ok {
		delete(c.Actors, a.ID)
		g.cleanupCell(a.CellID, c)
	}
}

// nearbyCells returns cells within range of the given position.
func (g *grid) nearbyCells(x, y float32) []*cell {
	centerID := g.getCellID(x, y)
	radius := int32(math.Ceil(float64(g.cellDistance / g.cellSize)))

	result := make([]*cell, 0, (2*radius+1)*(2*radius+1)+1)
	result = append(result, g.globalCell)

	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			cid := CellID{X: centerID.X + dx, Y: centerID.Y + dy}
			if c, ok := g.cells[cid]; ok {
				result = append(result, c)
			}
		}
	}
	return result
}

func (g *grid) cleanupCell(id CellID, c *cell) {
	if len(c.Objects) == 0 && len(c.Pickups) == 0 &&
		len(c.Checkpoints) == 0 && len(c.RaceCheckpoints) == 0 &&
		len(c.MapIcons) == 0 && len(c.TextLabels) == 0 &&
		len(c.Areas) == 0 && len(c.Actors) == 0 {
		delete(g.cells, id)
	}
}
