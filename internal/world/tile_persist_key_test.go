package world

import "testing"

func TestMapTilePersistKeyRoundTrip(t *testing.T) {
	t.Parallel()
	k := MapTilePersistKey("field1", 3, 7)
	mid, tx, ty, ok := ParseMapTilePersistKey(k)
	if !ok || mid != "field1" || tx != 3 || ty != 7 {
		t.Fatalf("got %q %d %d ok=%v", mid, tx, ty, ok)
	}
}

func TestParseMapTilePersistKey_MapIDWithColons(t *testing.T) {
	t.Parallel()
	k := "weird:map:id:2:9"
	mid, tx, ty, ok := ParseMapTilePersistKey(k)
	if !ok || mid != "weird:map:id" || tx != 2 || ty != 9 {
		t.Fatalf("got %q %d %d ok=%v", mid, tx, ty, ok)
	}
}
