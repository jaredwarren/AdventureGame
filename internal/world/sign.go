package world

import (
	"github.com/jaredwarren/game-test/internal/geom"
)

// Sign is an authored interactable wooden sign.
type Sign struct {
	ID   EntityID
	Rect geom.Rect
	Text string
}
