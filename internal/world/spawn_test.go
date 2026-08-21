package world

import "testing"

func TestParseDoorSpawnCoord(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		val  float64
		keep bool
		ok   bool
	}{
		{"160", 160, false, true},
		{" * ", 0, true, true},
		{"*", 0, true, true},
		{"", 0, false, true},
		{"middle", 0, false, false},
	}
	for _, tc := range cases {
		v, keep, ok := ParseDoorSpawnCoord(tc.in)
		if v != tc.val || keep != tc.keep || ok != tc.ok {
			t.Errorf("ParseDoorSpawnCoord(%q) = (%v,%v,%v), want (%v,%v,%v)",
				tc.in, v, keep, ok, tc.val, tc.keep, tc.ok)
		}
	}
}

func TestResolveDoorSpawnKeepAxis(t *testing.T) {
	t.Parallel()
	const px, py, h = 48.0, 80.0, 12.0

	x, y := ResolveDoorSpawn(Door{KeepSpawnX: true, SpawnY: 40, SpawnStyle: DoorSpawnFeet}, px, py, h)
	if x != px || y != 40-h {
		t.Fatalf("keep X feet: got (%v,%v) want (%v,%v)", x, y, px, 40-h)
	}

	x, y = ResolveDoorSpawn(Door{SpawnX: 16, KeepSpawnY: true, SpawnStyle: DoorSpawnFeet}, px, py, h)
	if x != 16 || y != py {
		t.Fatalf("keep Y feet: got (%v,%v) want (16,%v)", x, y, py)
	}

	x, y = ResolveDoorSpawn(Door{KeepSpawnX: true, SpawnY: 20, SpawnStyle: DoorSpawnTopLeft}, px, py, h)
	if x != px || y != 20 {
		t.Fatalf("keep X topleft: got (%v,%v) want (%v,20)", x, y, px)
	}

	x, y = ResolveDoorSpawn(Door{SpawnX: 16, KeepSpawnY: true, SpawnStyle: DoorSpawnTopLeft}, px, py, h)
	if x != 16 || y != py {
		t.Fatalf("keep Y topleft: got (%v,%v) want (16,%v)", x, y, py)
	}

	x, y = ResolveDoorSpawn(Door{KeepSpawnX: true, KeepSpawnY: true, SpawnStyle: DoorSpawnFeet}, px, py, h)
	if x != px || y != py {
		t.Fatalf("keep both feet (runtime fallback): got (%v,%v) want (%v,%v)", x, y, px, py)
	}
}

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
