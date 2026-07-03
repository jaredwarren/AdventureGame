package dungeon

import (
	"fmt"
	"math/rand"
)

type Role string

const (
	RoleStart  Role = "start"
	RoleBoss   Role = "boss"
	RoleKey    Role = "key"
	RoleNormal Role = "normal"
)

type Node struct {
	ID   int
	Role Role
	X, Y int
}

type Edge struct {
	From, To int
	IsLocked bool
}

type Graph struct {
	Seed     int64
	Nodes    []Node
	Edges    []Edge
	StartID  int
	BossID   int
	KeyID    int
	LockEdge Edge
}

type Result struct {
	Graph Graph
}

func (r Result) BugDigest() string {
	return fmt.Sprintf("seed:%d rooms:%d start:%d boss:%d key:%d lock:%d-%d edges:%d",
		r.Graph.Seed, len(r.Graph.Nodes), r.Graph.StartID, r.Graph.BossID, r.Graph.KeyID,
		r.Graph.LockEdge.From, r.Graph.LockEdge.To, len(r.Graph.Edges))
}

var cardinalDx = [...]int{0, 1, 0, -1}
var cardinalDy = [...]int{-1, 0, 1, 0}

// GenerateGraph constructs a deterministic room graph for the given seed.
func GenerateGraph(seed int64) (*Graph, error) {
	currSeed := seed
	for attempt := 0; attempt < 100; attempt++ {
		rng := rand.New(rand.NewSource(currSeed))
		g, ok := tryGenerateGraph(seed, rng)
		if ok {
			return g, nil
		}
		currSeed = rng.Int63()
	}
	return nil, fmt.Errorf("dungeon: failed to generate valid graph after 100 attempts for seed %d", seed)
}

func tryGenerateGraph(origSeed int64, rng *rand.Rand) (*Graph, bool) {
	nodeCount := 6 + rng.Intn(4) // 6 to 9 rooms
	nodes := make([]Node, 1, nodeCount)
	nodes[0] = Node{ID: 0, Role: RoleStart, X: 0, Y: 0}

	occupied := map[string]int{"0,0": 0}
	var edges []Edge

	for i := 1; i < nodeCount; i++ {
		// Pick an existing node with an open neighbor
		candidates := make([]int, 0)
		for idx, n := range nodes {
			hasOpen := false
			for d := 0; d < 4; d++ {
				key := fmt.Sprintf("%d,%d", n.X+cardinalDx[d], n.Y+cardinalDy[d])
				if _, exists := occupied[key]; !exists {
					hasOpen = true
					break
				}
			}
			if hasOpen {
				candidates = append(candidates, idx)
			}
		}
		if len(candidates) == 0 {
			return nil, false
		}
		parentID := candidates[rng.Intn(len(candidates))]
		parent := nodes[parentID]

		// Pick an open neighbor direction
		dirs := make([]int, 0, 4)
		for d := 0; d < 4; d++ {
			key := fmt.Sprintf("%d,%d", parent.X+cardinalDx[d], parent.Y+cardinalDy[d])
			if _, exists := occupied[key]; !exists {
				dirs = append(dirs, d)
			}
		}
		dir := dirs[rng.Intn(len(dirs))]
		nx, ny := parent.X+cardinalDx[dir], parent.Y+cardinalDy[dir]

		newNode := Node{ID: i, Role: RoleNormal, X: nx, Y: ny}
		nodes = append(nodes, newNode)
		occupied[fmt.Sprintf("%d,%d", nx, ny)] = i
		edges = append(edges, Edge{From: parentID, To: i, IsLocked: false})
	}

	// Calculate distance from start for all nodes
	dist := make([]int, nodeCount)
	for i := range dist {
		dist[i] = -1
	}
	dist[0] = 0
	queue := []int{0}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, e := range edges {
			var next int
			if e.From == curr {
				next = e.To
			} else if e.To == curr {
				next = e.From
			} else {
				continue
			}
			if dist[next] == -1 {
				dist[next] = dist[curr] + 1
				queue = append(queue, next)
			}
		}
	}

	// Pick boss room (furthest node from start)
	bossID := 0
	maxD := -1
	for id := 0; id < nodeCount; id++ {
		d := dist[id]
		if d > maxD {
			maxD = d
			bossID = id
		}
	}
	if bossID == 0 || maxD < 2 {
		return nil, false
	}
	nodes[bossID].Role = RoleBoss

	// Find main path from 0 to bossID
	path := getPath(0, bossID, edges)
	if len(path) < 2 {
		return nil, false
	}

	// Pick lock edge (edge leading into boss room)
	lockEdgeIdx := -1
	for i, e := range edges {
		if (e.From == path[len(path)-2] && e.To == bossID) || (e.To == path[len(path)-2] && e.From == bossID) {
			lockEdgeIdx = i
			break
		}
	}
	if lockEdgeIdx < 0 {
		return nil, false
	}
	edges[lockEdgeIdx].IsLocked = true
	lockEdge := edges[lockEdgeIdx]

	// Key room must be reachable without crossing lockEdge.
	// Find nodes reachable from start without crossing lockEdge.
	reachableBeforeLock := getReachableNodes(0, edges, true)
	keyCandidates := make([]int, 0)
	for id := 0; id < nodeCount; id++ {
		if reachableBeforeLock[id] && id != 0 && id != bossID {
			keyCandidates = append(keyCandidates, id)
		}
	}
	if len(keyCandidates) == 0 {
		return nil, false
	}
	keyID := keyCandidates[rng.Intn(len(keyCandidates))]
	nodes[keyID].Role = RoleKey

	g := &Graph{
		Seed:     origSeed,
		Nodes:    nodes,
		Edges:    edges,
		StartID:  0,
		BossID:   bossID,
		KeyID:    keyID,
		LockEdge: lockEdge,
	}

	// Validate solvability properties
	if !validateGraph(g) {
		return nil, false
	}

	return g, true
}

func getPath(start, target int, edges []Edge) []int {
	parent := make(map[int]int)
	visited := map[int]bool{start: true}
	q := []int{start}

	for len(q) > 0 {
		curr := q[0]
		q = q[1:]
		if curr == target {
			break
		}
		for _, e := range edges {
			var next int
			if e.From == curr {
				next = e.To
			} else if e.To == curr {
				next = e.From
			} else {
				continue
			}
			if !visited[next] {
				visited[next] = true
				parent[next] = curr
				q = append(q, next)
			}
		}
	}

	if !visited[target] {
		return nil
	}
	path := []int{target}
	for curr := target; curr != start; {
		p := parent[curr]
		path = append([]int{p}, path...)
		curr = p
	}
	return path
}

func getReachableNodes(start int, edges []Edge, ignoreLock bool) map[int]bool {
	visited := map[int]bool{start: true}
	q := []int{start}
	for len(q) > 0 {
		curr := q[0]
		q = q[1:]
		for _, e := range edges {
			if ignoreLock && e.IsLocked {
				continue
			}
			var next int
			if e.From == curr {
				next = e.To
			} else if e.To == curr {
				next = e.From
			} else {
				continue
			}
			if !visited[next] {
				visited[next] = true
				q = append(q, next)
			}
		}
	}
	return visited
}

func validateGraph(g *Graph) bool {
	// Key must be reachable from start without crossing locked edge
	reachableNoLock := getReachableNodes(g.StartID, g.Edges, true)
	if !reachableNoLock[g.KeyID] {
		return false
	}

	// Boss must NOT be reachable without crossing locked edge
	if reachableNoLock[g.BossID] {
		return false
	}

	// Boss MUST be reachable when locked edge is included
	reachableAll := getReachableNodes(g.StartID, g.Edges, false)
	if !reachableAll[g.BossID] {
		return false
	}

	return true
}
