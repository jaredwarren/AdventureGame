package dungeon_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/jaredwarren/game-test/internal/dungeon"
)

func dummyRoomLibrary() map[string][]byte {
	return map[string][]byte{
		"start.tmj": []byte(`{
			"width": 7, "height": 7, "tilewidth": 16, "tileheight": 16,
			"layers": [
				{"type": "tilelayer", "name": "ground", "width": 7, "height": 7, "data": [
					2,2,7,7,7,2,2, 2,7,7,7,7,7,2, 7,7,7,7,7,7,7, 7,7,7,7,7,7,7, 7,7,7,7,7,7,7, 2,7,7,7,7,7,2, 2,2,7,7,7,2,2
				]},
				{"type": "objectgroup", "name": "markers", "objects": [
					{"id": 1, "name": "spawn", "type": "spawn", "x": 56, "y": 56}
				]}
			]
		}`),
		"combat.tmj": []byte(`{
			"width": 7, "height": 7, "tilewidth": 16, "tileheight": 16,
			"layers": [
				{"type": "tilelayer", "name": "ground", "width": 7, "height": 7, "data": [
					2,2,7,7,7,2,2, 2,7,7,7,7,7,2, 7,7,7,7,7,7,7, 7,7,7,7,7,7,7, 7,7,7,7,7,7,7, 2,7,7,7,7,7,2, 2,2,7,7,7,2,2
				]},
				{"type": "objectgroup", "name": "markers", "objects": [
					{"id": 1, "name": "slime", "type": "enemy", "x": 32, "y": 32}
				]}
			]
		}`),
		"key.tmj": []byte(`{
			"width": 7, "height": 7, "tilewidth": 16, "tileheight": 16,
			"layers": [
				{"type": "tilelayer", "name": "ground", "width": 7, "height": 7, "data": [
					2,2,7,7,7,2,2, 2,7,7,7,7,7,2, 7,7,7,7,7,7,7, 7,7,7,7,7,7,7, 7,7,7,7,7,7,7, 2,7,7,7,7,7,2, 2,2,7,7,7,2,2
				]},
				{"type": "objectgroup", "name": "markers", "objects": [
					{"id": 1, "name": "key", "type": "pickup", "x": 56, "y": 56, "properties": [{"name": "kind", "type": "string", "value": "key"}]}
				]}
			]
		}`),
		"boss.tmj": []byte(`{
			"width": 7, "height": 7, "tilewidth": 16, "tileheight": 16,
			"layers": [
				{"type": "tilelayer", "name": "ground", "width": 7, "height": 7, "data": [
					2,2,7,7,7,2,2, 2,7,7,7,7,7,2, 7,7,7,7,7,7,7, 7,7,7,7,7,7,7, 7,7,7,7,7,7,7, 2,7,7,7,7,7,2, 2,2,7,7,7,2,2
				]},
				{"type": "objectgroup", "name": "markers", "objects": [
					{"id": 1, "name": "boss_slime", "type": "enemy", "x": 56, "y": 48}
				]}
			]
		}`),
	}
}

func TestStitcher_Determinism(t *testing.T) {
	lib, err := dungeon.LoadRoomLibrary(dummyRoomLibrary())
	if err != nil {
		t.Fatalf("LoadRoomLibrary err: %v", err)
	}

	m1, res1, err1 := dungeon.Generate(42, lib)
	if err1 != nil {
		t.Fatalf("Generate m1 err: %v", err1)
	}
	m2, res2, err2 := dungeon.Generate(42, lib)
	if err2 != nil {
		t.Fatalf("Generate m2 err: %v", err2)
	}

	if res1.BugDigest() != res2.BugDigest() {
		t.Errorf("expected bug digest match: %s vs %s", res1.BugDigest(), res2.BugDigest())
	}

	b1, err := m1.Encode()
	if err != nil {
		t.Fatalf("Encode m1 err: %v", err)
	}
	b2, err := m2.Encode()
	if err != nil {
		t.Fatalf("Encode m2 err: %v", err)
	}

	if !bytes.Equal(b1, b2) {
		t.Errorf("stitched maps for seed 42 are not byte-identical")
	}

	if !reflect.DeepEqual(m1, m2) {
		t.Errorf("stitched tiled.Map structs for seed 42 are not equal")
	}
}
