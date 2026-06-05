package dungeon

import "testing"

func TestStampOrthogonalMaze_SizeAndCenter(t *testing.T) {
	t.Parallel()
	m := GenerateGrowingTree(4, 5, 7, GrowingTreeBacktracker())
	mw, mh, data := StampOrthogonalMaze(m, 1, 2)
	if mw != 2*m.W+1 || mh != 2*m.H+1 {
		t.Fatalf("map size got %dx%d want %dx%d", mw, mh, 2*m.W+1, 2*m.H+1)
	}
	if len(data) != mw*mh {
		t.Fatalf("data len %d want %d", len(data), mw*mh)
	}
	if data[(2*0+1)*mw+(2*0+1)] != 1 {
		t.Fatal("expected grass at first cell center")
	}
}
