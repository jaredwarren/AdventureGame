package world

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/geom"
)

func TestApplyPersistedShrines(t *testing.T) {
	t.Parallel()
	w := &World{
		MapID: "testmap",
		Shrines: []Shrine{
			{
				ID:      1,
				TiledID: 10,
				Rect:    geom.Rect{X: 0, Y: 0, W: 16, H: 16},
				Active:  false,
			},
			{
				ID:      2,
				TiledID: 11,
				Rect:    geom.Rect{X: 32, Y: 0, W: 16, H: 16},
				Active:  false,
			},
		},
	}

	keys := map[string]struct{}{
		PersistentShrineSaveKey("testmap", 10): {},
		PersistentShrineSaveKey("other", 11):   {},
	}

	ApplyPersistedShrines(w, keys)

	if !w.Shrines[0].Active {
		t.Error("expected shrine 10 to be active")
	}
	if w.Shrines[1].Active {
		t.Error("expected shrine 11 to not be active (different map ID in key)")
	}
}
