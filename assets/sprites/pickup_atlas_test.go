package sprites

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/png"
	"testing"
)

func TestPickupAtlasRectsWithinPickupPNG(t *testing.T) {
	img, _, err := image.Decode(bytes.NewReader(PickupPNG))
	if err != nil {
		t.Fatalf("decode PickupPNG: %v", err)
	}
	b := img.Bounds()

	var doc struct {
		SpriteSheet string `json:"sprite_sheet"`
		Frames      []struct {
			SX int `json:"sx"`
			SY int `json:"sy"`
			SW int `json:"sw"`
			SH int `json:"sh"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(PickupAtlasJSON, &doc); err != nil {
		t.Fatalf("unmarshal atlas: %v", err)
	}
	if doc.SpriteSheet != PickupSpriteSheetFile {
		t.Fatalf("atlas sprite_sheet %q must match embedded pickup PNG %q", doc.SpriteSheet, PickupSpriteSheetFile)
	}
	if len(doc.Frames) != 6 {
		t.Fatalf("expected 6 frames (PickupKind order: coin, heart, bomb, small_key, torch, pegasus_boots), got %d", len(doc.Frames))
	}
	for i, fr := range doc.Frames {
		if fr.SW < 1 || fr.SH < 1 {
			t.Fatalf("frame %d: sw/sh must be >= 1", i)
		}
		r := image.Rect(fr.SX, fr.SY, fr.SX+fr.SW, fr.SY+fr.SH)
		if !r.In(b) {
			t.Fatalf("frame %d rect %v not inside sheet bounds %v", i, r, b)
		}
	}
}
