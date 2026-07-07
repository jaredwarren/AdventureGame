package tile

import "testing"

func TestSurfaceProperties(t *testing.T) {
	def := SurfaceForGID(0)
	if def.Type != SurfaceNormal || def.SpeedMultiplier != 1.0 {
		t.Errorf("expected default normal surface for GID 0, got %+v", def)
	}

	waterDef := SurfaceForGID(GIDWater)
	if waterDef.Type != SurfaceWater || waterDef.SpeedMultiplier != 1.0 {
		t.Errorf("expected water surface for GIDWater, got %+v", waterDef)
	}

	mudDef := SurfaceForGID(GIDMud)
	if mudDef.Type != SurfaceMud || mudDef.SpeedMultiplier != 0.5 {
		t.Errorf("expected mud surface for GIDMud, got %+v", mudDef)
	}

	iceDef := SurfaceForGID(GIDIce)
	if iceDef.Type != SurfaceIce || iceDef.SpeedMultiplier != 1.1 {
		t.Errorf("expected ice surface for GIDIce, got %+v", iceDef)
	}

	lavaDef := SurfaceForGID(GIDLava)
	if lavaDef.Type != SurfaceLava || lavaDef.SpeedMultiplier != 0.7 {
		t.Errorf("expected lava surface for GIDLava, got %+v", lavaDef)
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
