package dungeon

import (
	"math/rand"
)

// FloorTileWeight pairs a tile GID with its relative paint weight and
// passability. Passable=true means trees can be planted on this tile kind
// (grass, floor2); Passable=false means it is a blocking tile (water).
type FloorTileWeight struct {
	GID      int
	Weight   float64
	Passable bool
}

// FloorPaintParams configures PaintNaturalFloors. TreeDeadEndProb is the
// probability that a dead-end passable tile (exactly one passable neighbor
// among open cells) becomes a tree; spawn is never treed.
type FloorPaintParams struct {
	// Floors lists all tile GIDs (with relative weights) that open cells can
	// become. Add any number of floor or water variants here; no struct
	// changes required.
	Floors []FloorTileWeight

	TreeGID int
	WallGID int

	SpawnX, SpawnY int

	TreeDeadEndProb float64
}

// PaintNaturalFloors replaces stamped corridor tiles with a weighted mix of
// the tiles listed in p.Floors, then optionally places trees on dead ends
// (excluding spawn).
func PaintNaturalFloors(mapW, mapH int, data []int, p FloorPaintParams, rng *rand.Rand) {
	if rng == nil || mapW < 1 || mapH < 1 || len(data) != mapW*mapH {
		return
	}

	// Normalize weights; guard against all-zero.
	floors := make([]FloorTileWeight, len(p.Floors))
	copy(floors, p.Floors)
	sum := 0.0
	for i := range floors {
		if floors[i].Weight < 0 {
			floors[i].Weight = 0
		}
		sum += floors[i].Weight
	}
	if sum <= 0 {
		// Default: first entry at full weight, or nothing.
		if len(floors) > 0 {
			floors[0].Weight = 1
			sum = 1
		} else {
			return
		}
	}
	for i := range floors {
		floors[i].Weight /= sum
	}

	for i := range data {
		if data[i] == p.WallGID {
			continue
		}
		r := rng.Float64()
		acc := 0.0
		for _, f := range floors {
			acc += f.Weight
			if r < acc {
				data[i] = f.GID
				break
			}
		}
	}

	treeP := p.TreeDeadEndProb
	if treeP <= 0 {
		return
	}
	if treeP > 1 {
		treeP = 1
	}

	for ty := 0; ty < mapH; ty++ {
		for tx := 0; tx < mapW; tx++ {
			if tx == p.SpawnX && ty == p.SpawnY {
				continue
			}
			idx := ty*mapW + tx
			if data[idx] == p.WallGID {
				continue
			}
			if !isPassableFloorKind(data[idx], floors) {
				continue
			}
			if passableNeighborCount(mapW, mapH, data, tx, ty, p, floors) != 1 {
				continue
			}
			if rng.Float64() < treeP {
				data[idx] = p.TreeGID
			}
		}
	}
}

// isPassableFloorKind reports whether gid is a passable floor tile (trees can
// be placed on it). Water and other blocking floor kinds are not passable.
func isPassableFloorKind(gid int, floors []FloorTileWeight) bool {
	for _, f := range floors {
		if f.GID == gid {
			return f.Passable
		}
	}
	return false
}

func passableNeighborCount(mapW, mapH int, data []int, tx, ty int, p FloorPaintParams, floors []FloorTileWeight) int {
	dirs := [...]struct{ dx, dy int }{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	n := 0
	for _, d := range dirs {
		nx, ny := tx+d.dx, ty+d.dy
		if nx < 0 || ny < 0 || nx >= mapW || ny >= mapH {
			continue
		}
		g := data[ny*mapW+nx]
		if g == p.WallGID {
			continue
		}
		if isPassableFloorKind(g, floors) {
			n++
		}
	}
	return n
}
