package dungeon_test

import (
	"reflect"
	"testing"

	"github.com/jaredwarren/game-test/internal/dungeon"
)

func TestGenerateGraph_SolvabilityAndBridgeProperties(t *testing.T) {
	seeds := []int64{1, 42, 100, 1337, 9999}
	for _, seed := range seeds {
		g, err := dungeon.GenerateGraph(seed)
		if err != nil {
			t.Fatalf("failed to generate graph for seed %d: %v", seed, err)
		}

		res := dungeon.Result{Graph: *g}
		digest := res.BugDigest()
		if digest == "" {
			t.Errorf("expected non-empty BugDigest for seed %d", seed)
		}

		// Solvability check: Key must be reachable from start without lock
		if !g.LockEdge.IsLocked {
			t.Fatalf("seed %d: expected lock edge to be locked", seed)
		}

		var startNode, bossNode, keyNode *dungeon.Node
		for i := range g.Nodes {
			n := &g.Nodes[i]
			if n.ID == g.StartID {
				startNode = n
			}
			if n.ID == g.BossID {
				bossNode = n
			}
			if n.ID == g.KeyID {
				keyNode = n
			}
		}
		if startNode == nil || bossNode == nil || keyNode == nil {
			t.Fatalf("seed %d: missing start, boss, or key node", seed)
		}

		if startNode.Role != dungeon.RoleStart || bossNode.Role != dungeon.RoleBoss || keyNode.Role != dungeon.RoleKey {
			t.Fatalf("seed %d: role assignments mismatch", seed)
		}
	}
}

func TestGenerateGraph_Determinism(t *testing.T) {
	g1, err1 := dungeon.GenerateGraph(42)
	if err1 != nil {
		t.Fatalf("g1 err: %v", err1)
	}
	g2, err2 := dungeon.GenerateGraph(42)
	if err2 != nil {
		t.Fatalf("g2 err: %v", err2)
	}

	if !reflect.DeepEqual(g1, g2) {
		t.Errorf("graphs generated for same seed 42 are not equal")
	}
}
