package world

import (
	"container/heap"

	"github.com/jaredwarren/game-test/internal/world/tile"
)

// EnemyChaseTarget returns pixel coordinates the enemy should move toward this
// tick: the center of the next tile on an A* path to the player's tile, or the
// player center (pcx, pcy) when the player tile is blocked, there is no path,
// or the enemy already shares the player's tile.
func (w *World) EnemyChaseTarget(ecx, ecy, pcx, pcy float64) (tgx, tgy float64) {
	stx := clampInt(int(ecx/tile.Size), 0, w.MapW-1)
	sty := clampInt(int(ecy/tile.Size), 0, w.MapH-1)
	gtx := clampInt(int(pcx/tile.Size), 0, w.MapW-1)
	gty := clampInt(int(pcy/tile.Size), 0, w.MapH-1)
	if !w.enemyTileWalkable(gtx, gty) {
		return pcx, pcy
	}
	path := w.findPathAStar(stx, sty, gtx, gty)
	if len(path) < 2 {
		return pcx, pcy
	}
	nx, ny := path[1].tx, path[1].ty
	return float64(nx)*tile.Size + tile.Size*0.5, float64(ny)*tile.Size + tile.Size*0.5
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (w *World) enemyTileWalkable(tx, ty int) bool {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return false
	}
	return !w.solidTile(tx, ty)
}

func (w *World) packTile(tx, ty int) int { return ty*w.MapW + tx }

func (w *World) unpackTile(p int) (tx, ty int) {
	return p % w.MapW, p / w.MapW
}

func manhattan(tx0, ty0, tx1, ty1 int) int {
	dx := tx0 - tx1
	if dx < 0 {
		dx = -dx
	}
	dy := ty0 - ty1
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

type gridPoint struct{ tx, ty int }

// findPathAStar returns the tile path from (sx,sy) to (gx,gy) inclusive using
// 4-neighbor grid A* and Manhattan heuristic. Nil if start/goal unwalkable or
// unreachable.
func (w *World) findPathAStar(sx, sy, gx, gy int) []gridPoint {
	if !w.enemyTileWalkable(sx, sy) || !w.enemyTileWalkable(gx, gy) {
		return nil
	}
	if sx == gx && sy == gy {
		return []gridPoint{{sx, sy}}
	}
	start := w.packTile(sx, sy)
	goal := w.packTile(gx, gy)

	gScore := make(map[int]int, 128)
	gScore[start] = 0
	f0 := manhattan(sx, sy, gx, gy)

	h := &tileMinHeap{}
	heap.Init(h)
	heap.Push(h, &tileHeapItem{f: f0, g: 0, tx: sx, ty: sy})

	cameFrom := make(map[int]int, 128)
	dirs := [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

	for h.Len() > 0 {
		cur := heap.Pop(h).(*tileHeapItem)
		cp := w.packTile(cur.tx, cur.ty)
		if known, ok := gScore[cp]; !ok || cur.g != known {
			continue
		}
		if cp == goal {
			return w.reconstructPath(cameFrom, start, goal)
		}
		for _, d := range dirs {
			nx, ny := cur.tx+d[0], cur.ty+d[1]
			if !w.enemyTileWalkable(nx, ny) {
				continue
			}
			np := w.packTile(nx, ny)
			tentative := cur.g + 1
			if old, ok := gScore[np]; ok && tentative >= old {
				continue
			}
			gScore[np] = tentative
			cameFrom[np] = cp
			hCost := manhattan(nx, ny, gx, gy)
			heap.Push(h, &tileHeapItem{f: tentative + hCost, g: tentative, tx: nx, ty: ny})
		}
	}
	return nil
}

func (w *World) reconstructPath(cameFrom map[int]int, start, goal int) []gridPoint {
	path := make([]gridPoint, 0, 32)
	for at := goal; ; {
		tx, ty := w.unpackTile(at)
		path = append(path, gridPoint{tx, ty})
		if at == start {
			break
		}
		prev, ok := cameFrom[at]
		if !ok {
			return nil
		}
		at = prev
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

type tileHeapItem struct {
	tx, ty int
	g      int
	f      int
}

type tileMinHeap []*tileHeapItem

func (h tileMinHeap) Len() int { return len(h) }
func (h tileMinHeap) Less(i, j int) bool {
	if h[i].f != h[j].f {
		return h[i].f < h[j].f
	}
	return h[i].g > h[j].g
}
func (h tileMinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *tileMinHeap) Push(x any) { *h = append(*h, x.(*tileHeapItem)) }

func (h *tileMinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
