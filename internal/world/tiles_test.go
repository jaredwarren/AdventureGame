package world

import "testing"

func TestSolidAt_RegistryRules(t *testing.T) {
	t.Parallel()
	idx := 5
	// Destroyable tiles are solid until their index is marked destroyed.
	for _, gid := range []int{GIDCracked, GIDTree} {
		if !SolidAt(gid, idx, nil, false) {
			t.Fatalf("unbroken gid=%d should be solid", gid)
		}
		if SolidAt(gid, idx, map[int]bool{idx: true}, false) {
			t.Fatalf("broken gid=%d should be passable", gid)
		}
	}
	// OpenableByKey: solid without key, passable with key.
	if !SolidAt(GIDLock, idx, nil, false) {
		t.Fatal("lock without key should be solid")
	}
	if SolidAt(GIDLock, idx, nil, true) {
		t.Fatal("lock with key should be passable")
	}
	// Plain passable / solid tiles.
	if SolidAt(GIDGrass, idx, nil, false) {
		t.Fatal("grass should be passable")
	}
	if !SolidAt(GIDWater, idx, nil, false) {
		t.Fatal("water should be solid")
	}
	if !SolidAt(GIDWall, idx, nil, false) {
		t.Fatal("wall should be solid")
	}
	// Unknown GIDs fail closed.
	if !SolidAt(9999, idx, nil, false) {
		t.Fatal("unknown GID should default solid")
	}
}

func TestTileDef_DamageKinds(t *testing.T) {
	t.Parallel()
	if !TileDefOf(GIDCracked).AcceptsDamage(DamageBomb) {
		t.Fatal("cracked should accept bomb")
	}
	if TileDefOf(GIDCracked).AcceptsDamage(DamageFire) {
		t.Fatal("cracked should not accept fire")
	}
	if !TileDefOf(GIDTree).AcceptsDamage(DamageFire) {
		t.Fatal("tree should accept fire")
	}
	if TileDefOf(GIDTree).AcceptsDamage(DamageBomb) {
		t.Fatal("tree should NOT accept bomb")
	}
	if TileDefOf(GIDWall).Destroyable() {
		t.Fatal("wall is not destroyable")
	}
}
