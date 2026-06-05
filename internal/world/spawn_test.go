package world

import "testing"

func TestPlayerTopLeftFromDoorSpawn(t *testing.T) {
	t.Parallel()
	x, y := PlayerTopLeftFromDoorSpawn(160, 40, DoorSpawnFeet, 12)
	if x != 160 || y != 28 {
		t.Fatalf("feet spawn: got (%v,%v) want (160,28)", x, y)
	}
	x, y = PlayerTopLeftFromDoorSpawn(10, 20, DoorSpawnTopLeft, 12)
	if x != 10 || y != 20 {
		t.Fatalf("topleft spawn: got (%v,%v) want (10,20)", x, y)
	}
	x, y = PlayerTopLeftFromDoorSpawn(0, 100, DoorSpawnFeet, 0)
	if y != 100-DefaultPlayerHitboxH {
		t.Fatalf("zero hitboxH should use default: got y=%v want %v", y, 100-DefaultPlayerHitboxH)
	}
}
