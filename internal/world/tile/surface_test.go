package tile

import "testing"

func TestSurfaceProperties(t *testing.T) {
	def := SurfaceForGID(0)
	if def.Type != SurfaceNormal || def.SpeedMultiplier != 1.0 {
		t.Errorf("expected default normal surface for GID 0, got %+v", def)
	}

	waterDef := SurfaceForGID(GIDWater)
	if waterDef.Type != SurfaceWater || waterDef.SpeedMultiplier != 0.5 {
		t.Errorf("expected water surface for GIDWater, got %+v", waterDef)
	}

	tileWater := DefOf(GIDWater)
	if !tileWater.HasTag(TagWater) || !tileWater.HasTag(TagSolid) {
		t.Errorf("expected GIDWater to have TagWater and TagSolid, got tags %v", tileWater.Tags)
	}

	tileWall := DefOf(GIDWall)
	if !tileWall.IsWall() || tileWall.IsFloor() {
		t.Error("expected GIDWall to be wall and not floor")
	}

	tileGrass := DefOf(GIDGrass)
	if !tileGrass.IsFloor() || tileGrass.IsWall() {
		t.Error("expected GIDGrass to be floor and not wall")
	}
}
