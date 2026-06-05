package dungeon

import (
	"math/rand"
	"testing"
)

func TestGrowingTreeAllReachable(t *testing.T) {
	t.Parallel()
	for _, wh := range []struct{ w, h int }{{5, 5}, {8, 3}, {3, 7}, {12, 12}} {
		for seed := int64(0); seed < 30; seed++ {
			cfg := GrowingTreeBlend(0.5)
			cfg.ExtraPassageProbability = 0
			m := GenerateGrowingTree(wh.w, wh.h, seed, cfg)
			got := m.ReachableCells(0, 0)
			want := wh.w * wh.h
			if got != want {
				t.Fatalf("%dx%d seed %d: reachable %d want %d", wh.w, wh.h, seed, got, want)
			}
		}
	}
}

func TestGrowingTreePerfectEdgeCount(t *testing.T) {
	t.Parallel()
	for _, wh := range []struct{ w, h int }{{4, 4}, {10, 2}} {
		for seed := int64(1); seed < 40; seed++ {
			cfg := GrowingTreePrimLike()
			m := GenerateGrowingTree(wh.w, wh.h, seed, cfg)
			edges := m.OpenPassageCount()
			want := wh.w*wh.h - 1
			if edges != want {
				t.Fatalf("%dx%d seed %d: edges %d want %d (perfect maze)", wh.w, wh.h, seed, edges, want)
			}
		}
	}
}

func TestGrowingTreeExtraPassagesIncreaseEdges(t *testing.T) {
	t.Parallel()
	w, h := 6, 6
	seed := int64(42)
	base := GenerateGrowingTree(w, h, seed, GrowingTreeConfig{
		WeightNewest:            1,
		ExtraPassageProbability: 0,
	})
	with := GenerateGrowingTree(w, h, seed, GrowingTreeConfig{
		WeightNewest:            1,
		ExtraPassageProbability: 0.35,
	})
	if with.OpenPassageCount() < base.OpenPassageCount() {
		t.Fatalf("extra passages should not decrease edge count")
	}
}

func TestPickActiveIndexPureModes(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(123))
	n := 20
	if pickActiveIndex(rng, n, 1, 0, 0) != n-1 {
		t.Fatalf("newest-only should pick last index")
	}
	rng = rand.New(rand.NewSource(123))
	if pickActiveIndex(rng, n, 0, 0, 1) != 0 {
		t.Fatalf("oldest-only should pick index 0")
	}
}

func TestNormalizedWeightsZeroSum(t *testing.T) {
	t.Parallel()
	c := GrowingTreeConfig{WeightNewest: 0, WeightRandom: 0, WeightOldest: 0}
	nw, rw, ow := c.normalizedWeights()
	if nw != 1 || rw != 0 || ow != 0 {
		t.Fatalf("got %v %v %v want 1 0 0", nw, rw, ow)
	}
}
