package streamer

import "testing"

func TestGridAddRemoveObject(t *testing.T) {
	g := newGrid(300, 600)

	obj := &DynamicObject{
		baseEntity: newBaseEntity(1, Vector3{150, 150, 0}, 200),
		ModelID:    1234,
	}

	g.addObject(obj)

	cid := g.getCellID(150, 150)
	c, ok := g.cells[cid]
	if !ok {
		t.Fatal("cell not created")
	}
	if _, ok := c.Objects[1]; !ok {
		t.Fatal("object not in cell")
	}
	if obj.InGlobalCell {
		t.Fatal("should not be in global cell")
	}

	g.removeObject(obj)
	if _, ok := g.cells[cid]; ok {
		t.Fatal("empty cell not cleaned up")
	}
}

func TestGridGlobalCell(t *testing.T) {
	g := newGrid(300, 600)

	obj := &DynamicObject{
		baseEntity: newBaseEntity(1, Vector3{0, 0, 0}, 1000),
		ModelID:    1234,
	}

	g.addObject(obj)
	if !obj.InGlobalCell {
		t.Fatal("object with large stream distance should be global")
	}
	if _, ok := g.globalCell.Objects[1]; !ok {
		t.Fatal("object not in global cell")
	}

	g.removeObject(obj)
	if _, ok := g.globalCell.Objects[1]; ok {
		t.Fatal("object not removed from global cell")
	}
}

func TestGridNearbyCells(t *testing.T) {
	g := newGrid(300, 600)

	// Add objects in different cells.
	obj1 := &DynamicObject{baseEntity: newBaseEntity(1, Vector3{0, 0, 0}, 200), ModelID: 1}
	obj2 := &DynamicObject{baseEntity: newBaseEntity(2, Vector3{150, 150, 0}, 200), ModelID: 2}
	obj3 := &DynamicObject{baseEntity: newBaseEntity(3, Vector3{5000, 5000, 0}, 200), ModelID: 3}

	g.addObject(obj1)
	g.addObject(obj2)
	g.addObject(obj3)

	cells := g.nearbyCells(100, 100)

	// Should always include global cell.
	foundGlobal := false
	for _, c := range cells {
		if c == g.globalCell {
			foundGlobal = true
		}
	}
	if !foundGlobal {
		t.Fatal("nearbyCells should include global cell")
	}

	// Should find obj1 and obj2 (nearby), but not obj3 (far away).
	foundObj1, foundObj2, foundObj3 := false, false, false
	for _, c := range cells {
		if _, ok := c.Objects[1]; ok {
			foundObj1 = true
		}
		if _, ok := c.Objects[2]; ok {
			foundObj2 = true
		}
		if _, ok := c.Objects[3]; ok {
			foundObj3 = true
		}
	}

	if !foundObj1 {
		t.Error("should find nearby obj1")
	}
	if !foundObj2 {
		t.Error("should find nearby obj2")
	}
	if foundObj3 {
		t.Error("should not find distant obj3")
	}
}

func TestGridCellID(t *testing.T) {
	g := newGrid(300, 600)

	tests := []struct {
		name string
		x, y float32
		want CellID
	}{
		{"origin", 0, 0, CellID{0, 0}},
		{"positive", 350, 150, CellID{1, 0}},
		{"negative", -100, -400, CellID{-1, -2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.getCellID(tt.x, tt.y)
			if got != tt.want {
				t.Errorf("getCellID(%f, %f) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestGridCleanupCell(t *testing.T) {
	g := newGrid(300, 600)

	p := &DynamicPickup{baseEntity: newBaseEntity(1, Vector3{0, 0, 0}, 200), ModelID: 1}
	g.addPickup(p)

	cid := p.CellID
	if _, ok := g.cells[cid]; !ok {
		t.Fatal("cell should exist after add")
	}

	g.removePickup(p)
	if _, ok := g.cells[cid]; ok {
		t.Fatal("cell should be cleaned up when empty")
	}
}

func TestGridMultipleEntityTypes(t *testing.T) {
	g := newGrid(300, 600)

	obj := &DynamicObject{baseEntity: newBaseEntity(1, Vector3{0, 0, 0}, 200), ModelID: 1}
	p := &DynamicPickup{baseEntity: newBaseEntity(1, Vector3{0, 0, 0}, 200), ModelID: 1}

	g.addObject(obj)
	g.addPickup(p)

	// Both should be in the same cell.
	cid := g.getCellID(0, 0)
	c := g.cells[cid]
	if len(c.Objects) != 1 || len(c.Pickups) != 1 {
		t.Fatalf("expected 1 object and 1 pickup, got %d objects and %d pickups", len(c.Objects), len(c.Pickups))
	}

	// Removing one shouldn't clean up the cell (other entity still present).
	g.removeObject(obj)
	if _, ok := g.cells[cid]; !ok {
		t.Fatal("cell should not be cleaned up with remaining entities")
	}

	g.removePickup(p)
	if _, ok := g.cells[cid]; ok {
		t.Fatal("cell should be cleaned up when fully empty")
	}
}
