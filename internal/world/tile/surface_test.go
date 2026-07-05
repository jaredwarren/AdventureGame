package tile

import "testing"

func TestSurfaceProperties(t *testing.T) {
	def := SurfaceForGID(0)
	if def.Type != SurfaceNormal || def.SpeedMultiplier != 1.0 {
		t.Errorf("expected default normal surface for GID 0, got %+v", def)
	}

	mud := MudSurface
	if mud.SpeedMultiplier != 0.5 {
		t.Errorf("mud speed multiplier = %f, want 0.5", mud.SpeedMultiplier)
	}

	ice := IceSurface
	if ice.Friction != 0.1 {
		t.Errorf("ice friction = %f, want 0.1", ice.Friction)
	}
}
