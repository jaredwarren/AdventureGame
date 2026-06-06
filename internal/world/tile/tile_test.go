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
