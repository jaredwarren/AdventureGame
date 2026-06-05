package dungeon

import (
	"math/rand"
)

// FloorMixWeights are relative weights for open-tile decoration (grass / water /
// floor2). Water is still a blocking tile in-game; default gentmj keeps water
// weight at 0 so mazes stay connected. They are normalized; if all zero, grass
// is used everywhere.
type FloorMixWeights struct {
	Grass  float64
	Water  float64
	Floor2 float64
}

// FloorPaintParams configures PaintNaturalFloors. TreeDeadEndProb is the
// probability that a **dead-end** passable tile (exactly one grass/floor2
// neighbor among open cells) becomes a tree; spawn is never treed. Water does
// not count as passable for this graph. Trees block until burned.
type FloorPaintParams struct {
	Mix FloorMixWeights

	GrassGID  int
	WaterGID  int
	Floor2GID int
	TreeGID   int
	WallGID   int

	SpawnX, SpawnY int

	TreeDeadEndProb float64
}

// PaintNaturalFloors replaces stamped corridor tiles (currently all grassGID
// from StampOrthogonalMaze) with a weighted mix of grass / water / floor2,
// then optionally places trees on dead ends (excluding spawn).
func PaintNaturalFloors(mapW, mapH int, data []int, p FloorPaintParams, rng *rand.Rand) {
	if rng == nil || mapW < 1 || mapH < 1 || len(data) != mapW*mapH {
		return
	}
	g, w, f2 := p.Mix.Grass, p.Mix.Water, p.Mix.Floor2
	if g < 0 {
		g = 0
	}
	if w < 0 {
		w = 0
	}
	if f2 < 0 {
		f2 = 0
	}
	sum := g + w + f2
	if sum <= 0 {
		g, w, f2 = 1, 0, 0
		sum = 1
	}
	g /= sum
	w /= sum
	f2 /= sum

	for i := range data {
		if data[i] == p.WallGID {
			continue
		}
		r := rng.Float64()
		if r < g {
			data[i] = p.GrassGID
		} else if r < g+w {
			data[i] = p.WaterGID
		} else {
			data[i] = p.Floor2GID
		}
	}

	treeP := p.TreeDeadEndProb
	if treeP < 0 {
		treeP = 0
	}
	if treeP > 1 {
		treeP = 1
	}
	if treeP <= 0 {
		return
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
			if !isPassableFloorKind(data[idx], p) {
				continue
			}
			if passableNeighborCount(mapW, mapH, data, tx, ty, p) != 1 {
				continue
			}
			if rng.Float64() < treeP {
				data[idx] = p.TreeGID
			}
		}
	}
}

// isPassableFloorKind is grass or floor2 (player can walk; water is blocking).
func isPassableFloorKind(gid int, p FloorPaintParams) bool {
	switch gid {
	case p.GrassGID, p.Floor2GID:
		return true
	default:
		return false
	}
}

func passableNeighborCount(mapW, mapH int, data []int, tx, ty int, p FloorPaintParams) int {
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
		if isPassableFloorKind(g, p) {
			n++
		}
	}
	return n
}
