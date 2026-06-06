package dungeon

import (
	"math/rand"
	"testing"
)

func TestPaintNaturalFloors_NoTree_AllWalkableKinds(t *testing.T) {
	t.Parallel()
	m := GenerateGrowingTree(3, 3, 1, GrowingTreeBacktracker())
	mw, mh, data := StampOrthogonalMaze(m, 1, 2)
	rng := rand.New(rand.NewSource(42))
	PaintNaturalFloors(mw, mh, data, FloorPaintParams{
		Floors: []FloorTileWeight{
			{GID: 1, Weight: 1, Passable: true},
			{GID: 5, Weight: 1, Passable: false},
			{GID: 7, Weight: 1, Passable: true},
		},
		TreeGID:         8,
		WallGID:         2,
		SpawnX:          1,
		SpawnY:          1,
		TreeDeadEndProb: 0,
	}, rng)
	seen := map[int]bool{}
	for _, g := range data {
		if g == 2 {
			continue
		}
		seen[g] = true
	}
	for _, need := range []int{1, 5, 7} {
		if !seen[need] {
			t.Fatalf("expected at least one tile of gid %d, saw %v", need, seen)
		}
	}
	if seen[8] {
		t.Fatal("did not expect tree when TreeDeadEndProb=0")
	}
}

func TestPaintNaturalFloors_SpawnNotTree(t *testing.T) {
	t.Parallel()
	m := GenerateGrowingTree(2, 2, 0, GrowingTreeBacktracker())
	mw, mh, data := StampOrthogonalMaze(m, 1, 2)
	spawnX, spawnY := 1, 1
	rng := rand.New(rand.NewSource(0))
	PaintNaturalFloors(mw, mh, data, FloorPaintParams{
		Floors: []FloorTileWeight{
			{GID: 1, Weight: 1, Passable: true},
		},
		TreeGID:         8,
		WallGID:         2,
		SpawnX:          spawnX,
		SpawnY:          spawnY,
		TreeDeadEndProb: 1,
	}, rng)
	if data[spawnY*mw+spawnX] == 8 {
		t.Fatal("spawn should not become tree")
	}
}
