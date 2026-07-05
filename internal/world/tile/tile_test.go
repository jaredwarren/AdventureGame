package tile

import "testing"

func TestSolidAt_RegistryRules(t *testing.T) {
	t.Parallel()
	idx := 3
	for _, gid := range []int{GIDCracked, GIDTree} {
		if !SolidAt(gid, idx, nil, false) {
			t.Fatalf("gid %d should be solid when not destroyed", gid)
		}
		if SolidAt(gid, idx, map[int]bool{idx: true}, false) {
			t.Fatalf("gid %d should be passable when destroyed", gid)
		}
	}

	if !SolidAt(GIDLock, idx, nil, false) {
		t.Error("lock without key should be solid")
	}
	if SolidAt(GIDLock, idx, nil, true) {
		t.Error("lock with key should be passable")
	}

	if SolidAt(GIDGrass, idx, nil, false) {
		t.Error("grass should be passable")
	}
	if !SolidAt(GIDWater, idx, nil, false) {
		t.Error("water should be solid")
	}
	if !SolidAt(GIDWall, idx, nil, false) {
		t.Error("wall should be solid")
	}

	if !SolidAt(9999, idx, nil, false) {
		t.Error("unknown gid should fail closed as solid")
	}
}

func TestDef_DamageKinds(t *testing.T) {
	t.Parallel()
	if !DefOf(GIDCracked).AcceptsDamage(DamageBomb) {
		t.Error("cracked should accept bomb")
	}
	if DefOf(GIDCracked).AcceptsDamage(DamageFire) {
		t.Error("cracked should not accept fire")
	}
	if !DefOf(GIDTree).AcceptsDamage(DamageFire) {
		t.Error("tree should accept fire")
	}
	if DefOf(GIDTree).AcceptsDamage(DamageBomb) {
		t.Error("tree should not accept bomb")
	}
	if DefOf(GIDWall).Destroyable() {
		t.Error("wall should not be destroyable")
	}
}

func TestMapTilePersistKey_RoundTrip(t *testing.T) {
	t.Parallel()
	key := MapTilePersistKey("field1", 3, 7)
	mid, tx, ty, ok := ParseMapTilePersistKey(key)
	if !ok || mid != "field1" || tx != 3 || ty != 7 {
		t.Fatalf("got %q %d %d %v", mid, tx, ty, ok)
	}
}

func TestWaterShoreTileSolidRects(t *testing.T) {
	t.Parallel()

	// ShoreLeft: left half walkable, right half solid (X 8..16)
	leftRects := SolidRectsAt(GIDWaterShoreLeft, 0, nil, false)
	if len(leftRects) != 1 || leftRects[0].X != 8 || leftRects[0].Y != 0 || leftRects[0].W != 8 || leftRects[0].H != 16 {
		t.Errorf("unexpected GIDWaterShoreLeft solid rects: %v", leftRects)
	}

	// ShoreRight: left half solid (X 0..8), right half walkable
	rightRects := SolidRectsAt(GIDWaterShoreRight, 0, nil, false)
	if len(rightRects) != 1 || rightRects[0].X != 0 || rightRects[0].Y != 0 || rightRects[0].W != 8 || rightRects[0].H != 16 {
		t.Errorf("unexpected GIDWaterShoreRight solid rects: %v", rightRects)
	}

	// ShoreTop: top half walkable (Y 0..8), bottom half solid (Y 8..16)
	topRects := SolidRectsAt(GIDWaterShoreTop, 0, nil, false)
	if len(topRects) != 1 || topRects[0].X != 0 || topRects[0].Y != 8 || topRects[0].W != 16 || topRects[0].H != 8 {
		t.Errorf("unexpected GIDWaterShoreTop solid rects: %v", topRects)
	}

	// ShoreBottom: top half solid (Y 0..8), bottom half walkable (Y 8..16)
	bottomRects := SolidRectsAt(GIDWaterShoreBottom, 0, nil, false)
	if len(bottomRects) != 1 || bottomRects[0].X != 0 || bottomRects[0].Y != 0 || bottomRects[0].W != 16 || bottomRects[0].H != 8 {
		t.Errorf("unexpected GIDWaterShoreBottom solid rects: %v", bottomRects)
	}

	// ShoreSE: top-left quadrant solid water (NW), other 3 walkable grass
	seRects := SolidRectsAt(GIDWaterShoreSE, 0, nil, false)
	if len(seRects) != 1 || seRects[0].X != 0 || seRects[0].Y != 0 || seRects[0].W != 8 || seRects[0].H != 8 {
		t.Errorf("unexpected GIDWaterShoreSE solid rects: %v", seRects)
	}

	// Inner SE: bottom-right quadrant walkable grass (SE: X 8..16, Y 8..16), other 3 solid water
	seInnerRects := SolidRectsAt(GIDWaterShoreSEInner, 0, nil, false)
	if len(seInnerRects) != 2 || seInnerRects[0].X != 0 || seInnerRects[0].Y != 0 || seInnerRects[0].W != 8 || seInnerRects[0].H != 16 || seInnerRects[1].X != 8 || seInnerRects[1].Y != 0 || seInnerRects[1].W != 8 || seInnerRects[1].H != 8 {
		t.Errorf("unexpected GIDWaterShoreSEInner solid rects: %v", seInnerRects)
	}
}

func TestDef_IsWaterAndLand(t *testing.T) {
	t.Parallel()

	// GIDGrass should be land, not water
	grass := DefOf(GIDGrass)
	if !grass.IsLand() || grass.IsWater() || grass.Water() || grass.WaterShore() {
		t.Errorf("expected grass to be land, got IsLand=%v IsWater=%v", grass.IsLand(), grass.IsWater())
	}

	// GIDWater should be water, but not shore
	water := DefOf(GIDWater)
	if water.IsLand() || !water.IsWater() || !water.Water() || water.WaterShore() {
		t.Errorf("expected GIDWater to be water (non-shore), got IsLand=%v IsWater=%v WaterShore=%v", water.IsLand(), water.IsWater(), water.WaterShore())
	}

	// GIDWaterShoreSEInner should be water and shore
	seInner := DefOf(GIDWaterShoreSEInner)
	if seInner.IsLand() || !seInner.IsWater() || !seInner.Water() || !seInner.WaterShore() {
		t.Errorf("expected GIDWaterShoreSEInner to be water shore, got IsLand=%v IsWater=%v WaterShore=%v", seInner.IsLand(), seInner.IsWater(), seInner.WaterShore())
	}

	// Satisfies Waterer interface
	var _ Waterer = seInner
}

