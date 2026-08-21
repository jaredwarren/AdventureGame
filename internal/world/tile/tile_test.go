package tile

import (
	"image/color"
	"strings"
	"testing"
)

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

func TestWallTiles(t *testing.T) {
	t.Parallel()

	wallGIDs := []int{
		GIDWall, GIDWallTop, GIDWallBottom, GIDWallLeft, GIDWallRight,
		GIDWallNE, GIDWallNW, GIDWallSW, GIDWallSE,
		GIDWallNEInner, GIDWallNWInner, GIDWallSWInner, GIDWallSEInner,
	}

	for _, gid := range wallGIDs {
		d := DefOf(gid)
		if !d.Wall() || !d.IsWall() || !d.Solid() || d.IsFloor() {
			t.Errorf("gid %d (%s): expected to be wall & solid, got Wall=%v IsWall=%v Solid=%v IsFloor=%v",
				gid, d.Name, d.Wall(), d.IsWall(), d.Solid(), d.IsFloor())
		}
		var _ Waller = d
	}
}

func TestDirtPathTransitionTiles(t *testing.T) {
	t.Parallel()

	dirtPathGIDs := []int{
		GIDDirtPath,
		GIDDirtPathTop, GIDDirtPathBottom, GIDDirtPathLeft, GIDDirtPathRight,
		GIDDirtPathNE, GIDDirtPathNW, GIDDirtPathSW, GIDDirtPathSE,
		GIDDirtPathNEInner, GIDDirtPathNWInner, GIDDirtPathSWInner, GIDDirtPathSEInner,
	}

	for _, gid := range dirtPathGIDs {
		d := DefOf(gid)
		if d.Name == "unknown" {
			t.Errorf("gid %d is not registered in defs", gid)
		}
		if d.Solid() || d.Wall() || d.Water() || d.WaterShore() || !d.IsFloor() || !d.IsLand() || d.IsWall() || d.IsWater() {
			t.Errorf("gid %d (%s): expected walkable floor & land, got Solid=%v Wall=%v Water=%v IsFloor=%v IsLand=%v IsWall=%v IsWater=%v",
				gid, d.Name, d.Solid(), d.Wall(), d.Water(), d.IsFloor(), d.IsLand(), d.IsWall(), d.IsWater())
		}
		solidRects := SolidRectsAt(gid, 0, nil, false)
		if len(solidRects) != 0 {
			t.Errorf("gid %d (%s): expected no solid rects (fully walkable), got %v", gid, d.Name, solidRects)
		}
		var _ Floorer = d
	}
}

func TestCobblePathTransitionTiles(t *testing.T) {
	t.Parallel()

	for i := 0; i < int(VariantCount); i++ {
		gid := GIDCobblePath + i
		d := DefOf(gid)
		if d.Name == "unknown" {
			t.Errorf("gid %d is not registered in defs", gid)
		}
		if d.Solid() || d.Wall() || d.Water() || d.WaterShore() || !d.IsFloor() || !d.IsLand() || d.IsWall() || d.IsWater() {
			t.Errorf("gid %d (%s): expected walkable floor & land, got Solid=%v Wall=%v Water=%v IsFloor=%v IsLand=%v IsWall=%v IsWater=%v",
				gid, d.Name, d.Solid(), d.Wall(), d.Water(), d.IsFloor(), d.IsLand(), d.IsWall(), d.IsWater())
		}
		solidRects := SolidRectsAt(gid, 0, nil, false)
		if len(solidRects) != 0 {
			t.Errorf("gid %d (%s): expected no solid rects (fully walkable), got %v", gid, d.Name, solidRects)
		}
		var _ Floorer = d

		// Test vector drawing runs and outputs draw operations
		c := &testCanvas{}
		d.DrawVector(c, 0, 0, Size, Size)
		if len(c.ops) == 0 {
			t.Errorf("cobble_path variant %d (gid %d) produced 0 draw operations", i, gid)
		}
	}
}

func TestDirtPathTransitionDrawing(t *testing.T) {
	t.Parallel()

	dirtGIDs := [13]int{
		GIDDirtPath,
		GIDDirtPathTop, GIDDirtPathBottom, GIDDirtPathLeft, GIDDirtPathRight,
		GIDDirtPathNE, GIDDirtPathNW, GIDDirtPathSW, GIDDirtPathSE,
		GIDDirtPathNEInner, GIDDirtPathNWInner, GIDDirtPathSWInner, GIDDirtPathSEInner,
	}

	for i, gid := range dirtGIDs {
		d := DefOf(gid)
		if d.Name == "unknown" {
			t.Errorf("dirt_path variant %d (gid %d) not registered", i, gid)
		}
		c := &testCanvas{}
		d.DrawVector(c, 0, 0, Size, Size)
		if len(c.ops) == 0 {
			t.Errorf("dirt_path variant %d (gid %d) produced 0 draw operations", i, gid)
		}
	}
}

func TestFloorFamilyTransitionTiles(t *testing.T) {
	t.Parallel()

	floorFamilies := []struct {
		name    string
		baseGID int
	}{
		{"sand_path", GIDSandPath},
		{"snow", GIDSnow},
		{"mud_path", GIDMudPath},
		{"dark_grass", GIDDarkGrass},
		{"ice", GIDIceFamily},
		{"quicksand", GIDQuicksandFamily},
	}

	for _, fam := range floorFamilies {
		for i := 0; i < int(VariantCount); i++ {
			gid := fam.baseGID + i
			d := DefOf(gid)
			if d.Name == "unknown" {
				t.Errorf("family %s variant %d (gid %d) is not registered in defs", fam.name, i, gid)
			}
			if d.Solid() || d.Wall() || d.Water() || d.WaterShore() || !d.IsFloor() || !d.IsLand() || d.IsWall() || d.IsWater() {
				t.Errorf("gid %d (%s): expected walkable floor & land, got Solid=%v Wall=%v Water=%v IsFloor=%v IsLand=%v IsWall=%v IsWater=%v",
					gid, d.Name, d.Solid(), d.Wall(), d.Water(), d.IsFloor(), d.IsLand(), d.IsWall(), d.IsWater())
			}
			solidRects := SolidRectsAt(gid, 0, nil, false)
			if len(solidRects) != 0 {
				t.Errorf("gid %d (%s): expected no solid rects (fully walkable), got %v", gid, d.Name, solidRects)
			}
			var _ Floorer = d
		}
	}
}

func TestWaterHazardFamilyTransitionTiles(t *testing.T) {
	t.Parallel()

	waterFamilies := []struct {
		name    string
		baseGID int
	}{
		{"deep_water", GIDDeepWater},
		{"lava_shore", GIDLavaShore},
		{"swamp_water", GIDSwampWater},
	}

	for _, fam := range waterFamilies {
		for i := 0; i < int(VariantCount); i++ {
			gid := fam.baseGID + i
			d := DefOf(gid)
			if d.Name == "unknown" {
				t.Errorf("family %s variant %d (gid %d) is not registered in defs", fam.name, i, gid)
			}
			if !d.Solid() || !d.Water() || !d.IsWater() || d.IsFloor() || d.IsLand() {
				t.Errorf("gid %d (%s): expected solid water properties, got Solid=%v Water=%v IsWater=%v IsFloor=%v IsLand=%v",
					gid, d.Name, d.Solid(), d.Water(), d.IsWater(), d.IsFloor(), d.IsLand())
			}
			if i == 0 {
				if d.WaterShore() {
					t.Errorf("gid %d (%s): base tile should not be shore", gid, d.Name)
				}
			} else {
				if !d.WaterShore() {
					t.Errorf("gid %d (%s): transition variant should be shore", gid, d.Name)
				}
				solidRects := SolidRectsAt(gid, 0, nil, false)
				if len(solidRects) == 0 {
					t.Errorf("gid %d (%s): expected non-empty SolidRects for transition tile", gid, d.Name)
				}
			}
			var _ Waterer = d
		}
	}
}

type testCanvas struct {
	tx, ty int
	ops    []string
}

func (c *testCanvas) Tick() int           { return 0 }
func (c *testCanvas) GridPos() (int, int) { return c.tx, c.ty }
func (c *testCanvas) FillRect(x, y, w, h float32, clr color.RGBA) {
	c.ops = append(c.ops, "fr")
}
func (c *testCanvas) StrokeRect(x, y, w, h float32, sw float32, clr color.RGBA) {
	c.ops = append(c.ops, "sr")
}
func (c *testCanvas) StrokeLine(x1, y1, x2, y2 float32, sw float32, clr color.RGBA) {
	c.ops = append(c.ops, "sl")
}
func (c *testCanvas) FillCircle(cx, cy, r float32, clr color.RGBA) {
	c.ops = append(c.ops, "fc")
}
func (c *testCanvas) StrokeCircle(cx, cy, r float32, sw float32, clr color.RGBA) {
	c.ops = append(c.ops, "sc")
}
func (c *testCanvas) DrawPath(p Path, clr color.RGBA, fill bool, sw float32) {
	c.ops = append(c.ops, "p")
}

func TestSpatialVariationAndWallUniformity(t *testing.T) {
	t.Parallel()

	grass := DefOf(GIDGrass)
	tree := DefOf(GIDTree)
	water := DefOf(GIDWater)
	dirtPath := DefOf(GIDDirtPath)
	cobblePath := DefOf(GIDCobblePath)
	wall := DefOf(GIDWall)

	// Sample 20 grid positions
	grassVariants := make(map[string]int)
	treeVariants := make(map[string]int)
	waterVariants := make(map[string]int)
	dirtVariants := make(map[string]int)
	cobbleVariants := make(map[string]int)
	wallVariants := make(map[string]int)

	for tx := 0; tx < 5; tx++ {
		for ty := 0; ty < 4; ty++ {
			c1 := &testCanvas{tx: tx, ty: ty}
			grass.DrawVector(c1, 0, 0, Size, Size)
			key1 := strings.Join(c1.ops, ",")
			grassVariants[key1]++

			// Test deterministic repeatability for grass
			c2 := &testCanvas{tx: tx, ty: ty}
			grass.DrawVector(c2, 0, 0, Size, Size)
			key2 := strings.Join(c2.ops, ",")
			if key1 != key2 {
				t.Errorf("grass at (%d,%d) is not deterministic", tx, ty)
			}

			// Tree spatial variation
			ct1 := &testCanvas{tx: tx, ty: ty}
			tree.DrawVector(ct1, 0, 0, Size, Size)
			tkey1 := strings.Join(ct1.ops, ",")
			treeVariants[tkey1]++

			ct2 := &testCanvas{tx: tx, ty: ty}
			tree.DrawVector(ct2, 0, 0, Size, Size)
			tkey2 := strings.Join(ct2.ops, ",")
			if tkey1 != tkey2 {
				t.Errorf("tree at (%d,%d) is not deterministic", tx, ty)
			}

			// Water spatial variation
			cw1 := &testCanvas{tx: tx, ty: ty}
			water.DrawVector(cw1, 0, 0, Size, Size)
			wkey1 := strings.Join(cw1.ops, ",")
			waterVariants[wkey1]++

			cw2 := &testCanvas{tx: tx, ty: ty}
			water.DrawVector(cw2, 0, 0, Size, Size)
			wkey2 := strings.Join(cw2.ops, ",")
			if wkey1 != wkey2 {
				t.Errorf("water at (%d,%d) is not deterministic", tx, ty)
			}

			// Dirt path spatial variation
			cd1 := &testCanvas{tx: tx, ty: ty}
			dirtPath.DrawVector(cd1, 0, 0, Size, Size)
			dkey1 := strings.Join(cd1.ops, ",")
			dirtVariants[dkey1]++

			cd2 := &testCanvas{tx: tx, ty: ty}
			dirtPath.DrawVector(cd2, 0, 0, Size, Size)
			dkey2 := strings.Join(cd2.ops, ",")
			if dkey1 != dkey2 {
				t.Errorf("dirtPath at (%d,%d) is not deterministic", tx, ty)
			}

			// Cobblestone path spatial variation
			cc1 := &testCanvas{tx: tx, ty: ty}
			cobblePath.DrawVector(cc1, 0, 0, Size, Size)
			ckey1 := strings.Join(cc1.ops, ",")
			cobbleVariants[ckey1]++

			cc2 := &testCanvas{tx: tx, ty: ty}
			cobblePath.DrawVector(cc2, 0, 0, Size, Size)
			ckey2 := strings.Join(cc2.ops, ",")
			if ckey1 != ckey2 {
				t.Errorf("cobblePath at (%d,%d) is not deterministic", tx, ty)
			}

			// Wall uniformity
			cw := &testCanvas{tx: tx, ty: ty}
			wall.DrawVector(cw, 0, 0, Size, Size)
			wallKey := strings.Join(cw.ops, ",")
			wallVariants[wallKey]++
		}
	}

	// Grass, Tree, Water, Dirt, and Cobble must produce multiple visual variants
	if len(grassVariants) < 2 {
		t.Errorf("grass should produce multiple distinct spatial variations across the grid, got only %d: %v", len(grassVariants), grassVariants)
	}
	if len(treeVariants) < 2 {
		t.Errorf("tree should produce multiple distinct spatial variations across the grid, got only %d: %v", len(treeVariants), treeVariants)
	}
	if len(waterVariants) < 2 {
		t.Errorf("water should produce multiple distinct spatial variations across the grid, got only %d: %v", len(waterVariants), waterVariants)
	}
	if len(dirtVariants) < 2 {
		t.Errorf("dirtPath should produce multiple distinct spatial variations across the grid, got only %d: %v", len(dirtVariants), dirtVariants)
	}
	if len(cobbleVariants) < 2 {
		t.Errorf("cobblePath should produce multiple distinct spatial variations across the grid, got only %d: %v", len(cobbleVariants), cobbleVariants)
	}

	// Wall must produce exactly 1 invariant uniform draw across all grid positions
	if len(wallVariants) != 1 {
		t.Errorf("wall must be 100%% static and uniform across all grid positions, got %d variations: %v", len(wallVariants), wallVariants)
	}
}
