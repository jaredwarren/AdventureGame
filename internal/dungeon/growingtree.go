// Package dungeon holds procedural helpers. growingtree.go implements orthogonal
// maze generation (Growing Tree family) from commit history, revived for CLI
// and tooling (see cmd/gentmj).
//
// Passages are stored per-cell booleans for N/E/S/W.
package dungeon

import (
	"math/rand"
)

// Direction bits for orthogonal moves from a cell.
const (
	DirN = iota
	DirE
	DirS
	DirW
)

var dirDx = [...]int{0, 1, 0, -1}
var dirDy = [...]int{-1, 0, 1, 0}
var dirOpposite = [...]int{DirS, DirW, DirN, DirE}

// GrowingTreeConfig controls how the next “active” cell is chosen while carving.
// Weights are normalized to probabilities; if the sum is <= 0, pure newest (recursive backtracker).
//
//   - WeightNewest: always extend from the last carved cell → long corridors (recursive backtracker).
//   - WeightRandom: uniform among all active frontier cells → bushier (Prim-like).
//   - WeightOldest: always extend from the first cell still in the set → different bias.
//
// ExtraPassageProbability: after the spanning tree is complete, each remaining interior wall
// is opened independently with this probability (0 = perfect maze; small values add loops).
type GrowingTreeConfig struct {
	WeightNewest            float64
	WeightRandom            float64
	WeightOldest            float64
	ExtraPassageProbability float64
}

// DefaultGrowingTreeConfig is pure recursive backtracker (newest-only, perfect maze).
func DefaultGrowingTreeConfig() GrowingTreeConfig {
	return GrowingTreeConfig{WeightNewest: 1, WeightRandom: 0, WeightOldest: 0}
}

// GrowingTreeBacktracker is an alias for DefaultGrowingTreeConfig (long corridors).
func GrowingTreeBacktracker() GrowingTreeConfig {
	return DefaultGrowingTreeConfig()
}

// GrowingTreePrimLike favors random frontier selection (shorter branches, more “bushy”).
func GrowingTreePrimLike() GrowingTreeConfig {
	return GrowingTreeConfig{WeightNewest: 0, WeightRandom: 1, WeightOldest: 0}
}

// GrowingTreeBlend returns a simple two-knob mix: newestFrac toward backtracker, rest split
// evenly between random and oldest. newestFrac should be in [0, 1].
func GrowingTreeBlend(newestFrac float64) GrowingTreeConfig {
	if newestFrac < 0 {
		newestFrac = 0
	}
	if newestFrac > 1 {
		newestFrac = 1
	}
	rest := 1 - newestFrac
	return GrowingTreeConfig{
		WeightNewest: newestFrac,
		WeightRandom: rest * 0.5,
		WeightOldest: rest * 0.5,
	}
}

func (c GrowingTreeConfig) normalizedWeights() (newest, random, oldest float64) {
	nw, rw, ow := c.WeightNewest, c.WeightRandom, c.WeightOldest
	if nw < 0 {
		nw = 0
	}
	if rw < 0 {
		rw = 0
	}
	if ow < 0 {
		ow = 0
	}
	sum := nw + rw + ow
	if sum <= 0 {
		return 1, 0, 0
	}
	return nw / sum, rw / sum, ow / sum
}

// pickActiveIndex chooses an index into the active cell list [0, n).
func pickActiveIndex(rng *rand.Rand, n int, newest, random, oldest float64) int {
	if n <= 1 {
		return 0
	}
	r := rng.Float64()
	if r < newest {
		return n - 1
	}
	if r < newest+random {
		return rng.Intn(n)
	}
	return 0
}

// Maze is a rectangular grid where Pass[y][x][d] is true if you can step from (x,y) in direction d.
type Maze struct {
	W, H int
	Pass [][][4]bool
}

// NewMaze allocates an empty maze (all walls closed).
func NewMaze(w, h int) *Maze {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m := &Maze{W: w, H: h}
	m.Pass = make([][][4]bool, h)
	for y := range m.Pass {
		m.Pass[y] = make([][4]bool, w)
	}
	return m
}

func (m *Maze) inBounds(x, y int) bool {
	return x >= 0 && x < m.W && y >= 0 && y < m.H
}

// Carve opens the wall between (x,y) and the neighbor in direction d.
func (m *Maze) Carve(x, y, d int) {
	if !m.inBounds(x, y) {
		return
	}
	nx := x + dirDx[d]
	ny := y + dirDy[d]
	if !m.inBounds(nx, ny) {
		return
	}
	m.Pass[y][x][d] = true
	m.Pass[ny][nx][dirOpposite[d]] = true
}

// Open returns whether movement from (x,y) in direction d is allowed.
func (m *Maze) Open(x, y, d int) bool {
	if !m.inBounds(x, y) {
		return false
	}
	return m.Pass[y][x][d]
}

// GenerateGrowingTree builds a maze on a w×h cell grid using the Growing Tree algorithm.
// The RNG stream is derived from seed. The result is a perfect maze (spanning tree of the grid)
// unless ExtraPassageProbability > 0, in which case random extra openings are added.
func GenerateGrowingTree(w, h int, seed int64, cfg GrowingTreeConfig) *Maze {
	rng := rand.New(rand.NewSource(seed))
	m := NewMaze(w, h)
	if w == 1 && h == 1 {
		return m
	}

	visited := make([][]bool, h)
	for y := range visited {
		visited[y] = make([]bool, w)
	}

	sx := rng.Intn(w)
	sy := rng.Intn(h)
	type pt struct{ x, y int }
	active := make([]pt, 0, w*h)
	active = append(active, pt{sx, sy})
	visited[sy][sx] = true

	nw, rw, ow := cfg.normalizedWeights()

	for len(active) > 0 {
		i := pickActiveIndex(rng, len(active), nw, rw, ow)
		c := active[i]

		var dirs []int
		for d := 0; d < 4; d++ {
			nx := c.x + dirDx[d]
			ny := c.y + dirDy[d]
			if !m.inBounds(nx, ny) || visited[ny][nx] {
				continue
			}
			dirs = append(dirs, d)
		}

		if len(dirs) == 0 {
			active[i] = active[len(active)-1]
			active = active[:len(active)-1]
			continue
		}

		d := dirs[rng.Intn(len(dirs))]
		nx := c.x + dirDx[d]
		ny := c.y + dirDy[d]
		m.Carve(c.x, c.y, d)
		visited[ny][nx] = true
		active = append(active, pt{nx, ny})
	}

	if cfg.ExtraPassageProbability > 0 {
		m.addRandomPassages(rng, cfg.ExtraPassageProbability)
	}
	return m
}

type wall struct {
	x, y, d int
}

func (m *Maze) closedInteriorWalls() []wall {
	var walls []wall
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			if x+1 < m.W && !m.Pass[y][x][DirE] {
				walls = append(walls, wall{x, y, DirE})
			}
			if y+1 < m.H && !m.Pass[y][x][DirS] {
				walls = append(walls, wall{x, y, DirS})
			}
		}
	}
	return walls
}

func (m *Maze) addRandomPassages(rng *rand.Rand, p float64) {
	if p <= 0 {
		return
	}
	walls := m.closedInteriorWalls()
	rng.Shuffle(len(walls), func(i, j int) { walls[i], walls[j] = walls[j], walls[i] })
	for _, w := range walls {
		if rng.Float64() < p {
			m.Carve(w.x, w.y, w.d)
		}
	}
}

// ReachableCells returns how many cells are reachable from (sx,sy) using open passages.
func (m *Maze) ReachableCells(sx, sy int) int {
	if !m.inBounds(sx, sy) {
		return 0
	}
	seen := make([][]bool, m.H)
	for y := range seen {
		seen[y] = make([]bool, m.W)
	}
	q := []struct{ x, y int }{{sx, sy}}
	seen[sy][sx] = true
	cnt := 0
	for head := 0; head < len(q); head++ {
		c := q[head]
		cnt++
		for d := 0; d < 4; d++ {
			if !m.Open(c.x, c.y, d) {
				continue
			}
			nx := c.x + dirDx[d]
			ny := c.y + dirDy[d]
			if !m.inBounds(nx, ny) || seen[ny][nx] {
				continue
			}
			seen[ny][nx] = true
			q = append(q, struct{ x, y int }{nx, ny})
		}
	}
	return cnt
}

// OpenPassageCount counts undirected open edges (each passage once).
func (m *Maze) OpenPassageCount() int {
	n := 0
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			if m.Pass[y][x][DirE] {
				n++
			}
			if m.Pass[y][x][DirS] {
				n++
			}
		}
	}
	return n
}
