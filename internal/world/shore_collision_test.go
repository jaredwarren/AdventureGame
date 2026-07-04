package world_test

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

func TestRectHitsSolid_WaterShoreTiles(t *testing.T) {
	t.Parallel()

	// 3x3 tile world with GIDWaterShoreLeft at (1, 1) (tile origin: x=16, y=16)
	tiles := []int{
		tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
		tile.GIDGrass, tile.GIDWaterShoreLeft, tile.GIDGrass,
		tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
	}
	w := &world.World{MapW: 3, MapH: 3, Tiles: tiles}

	// Left half of GIDWaterShoreLeft (tile x=16..32, y=16..32) is grass (x=16..24).
	// Rect inside the left half: x=18..22, y=18..22 -> should NOT hit solid!
	greenRect := geom.Rect{X: 18, Y: 18, W: 4, H: 4}
	if w.RectHitsSolid(greenRect) {
		t.Errorf("expected left half of GIDWaterShoreLeft to be walkable, but RectHitsSolid returned true")
	}

	// Rect inside the right half (x=24..32) is water -> SHOULD hit solid!
	blueRect := geom.Rect{X: 25, Y: 18, W: 4, H: 4}
	if !w.RectHitsSolid(blueRect) {
		t.Errorf("expected right half of GIDWaterShoreLeft to be solid water, but RectHitsSolid returned false")
	}
}

func TestRectHitsSolid_WaterShoreSE(t *testing.T) {
	t.Parallel()

	// 3x3 tile world with GIDWaterShoreSE at (1, 1) (tile origin: x=16, y=16)
	// Outer corner: NW is solid water, NE, SW, SE are grass/walkable.
	tiles := []int{
		tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
		tile.GIDGrass, tile.GIDWaterShoreSE, tile.GIDGrass,
		tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
	}
	w := &world.World{MapW: 3, MapH: 3, Tiles: tiles}

	// Top-left quadrant (x=16..24, y=16..24) is blue water -> solid
	topLeftRect := geom.Rect{X: 18, Y: 18, W: 4, H: 4}
	if !w.RectHitsSolid(topLeftRect) {
		t.Errorf("expected top-left quadrant of GIDWaterShoreSE to be solid water, got walkable")
	}

	// Bottom-right quadrant (x=24..32, y=24..32) is grass -> walkable
	bottomRightRect := geom.Rect{X: 26, Y: 26, W: 4, H: 4}
	if w.RectHitsSolid(bottomRightRect) {
		t.Errorf("expected bottom-right quadrant of GIDWaterShoreSE to be walkable, got solid")
	}
}

func TestRectHitsSolid_WaterShoreSEInner(t *testing.T) {
	t.Parallel()

	// 3x3 tile world with GIDWaterShoreSEInner at (1, 1) (tile origin: x=16, y=16)
	// Inner corner: SE (x=24..32, y=24..32) is grass/walkable.
	// NW, NE, SW are solid water.
	tiles := []int{
		tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
		tile.GIDGrass, tile.GIDWaterShoreSEInner, tile.GIDGrass,
		tile.GIDGrass, tile.GIDGrass, tile.GIDGrass,
	}
	w := &world.World{MapW: 3, MapH: 3, Tiles: tiles}

	// Bottom-right quadrant (x=24..32, y=24..32) is grass -> walkable
	bottomRightRect := geom.Rect{X: 26, Y: 26, W: 4, H: 4}
	if w.RectHitsSolid(bottomRightRect) {
		t.Errorf("expected bottom-right quadrant of GIDWaterShoreSEInner to be walkable, got solid")
	}

	// Top-left quadrant (x=16..24, y=16..24) is blue water -> solid
	topLeftRect := geom.Rect{X: 18, Y: 18, W: 4, H: 4}
	if !w.RectHitsSolid(topLeftRect) {
		t.Errorf("expected top-left quadrant of GIDWaterShoreSEInner to be solid water, got walkable")
	}
}
