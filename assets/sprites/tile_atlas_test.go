package sprites

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/png"
	"testing"
)

func TestTileAtlasRectsWithinOverworldPNG(t *testing.T) {
	overworldImg, _, err := image.Decode(bytes.NewReader(OverworldTilePNG))
	if err != nil {
		t.Fatalf("decode OverworldTilePNG: %v", err)
	}
	lakeImg, _, err := image.Decode(bytes.NewReader(LakeIslandScenePNG))
	if err != nil {
		t.Fatalf("decode LakeIslandScenePNG: %v", err)
	}

	var doc struct {
		SpriteSheet string `json:"sprite_sheet"`
		TilePx      int    `json:"tile_px"`
		Frames      []struct {
			Skip        bool   `json:"skip"`
			SpriteSheet string `json:"sprite_sheet"`
			SX          int    `json:"sx"`
			SY          int    `json:"sy"`
			SW          int    `json:"sw"`
			SH          int    `json:"sh"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(TileAtlasJSON, &doc); err != nil {
		t.Fatalf("unmarshal tile atlas: %v", err)
	}
	if doc.SpriteSheet != TileSpriteSheetFile {
		t.Fatalf("atlas sprite_sheet %q must match %q", doc.SpriteSheet, TileSpriteSheetFile)
	}
	if len(doc.Frames) != 34 {
		t.Fatalf("expected 34 frames (GID 0..33), got %d", len(doc.Frames))
	}
	for i, fr := range doc.Frames {
		if fr.Skip {
			continue
		}
		if fr.SW < 1 || fr.SH < 1 {
			t.Fatalf("frame %d: sw/sh must be >= 1 unless skip", i)
		}
		currentImg := overworldImg
		if fr.SpriteSheet == "lake_island_scene.png" {
			currentImg = lakeImg
		}
		b := currentImg.Bounds()
		r := image.Rect(fr.SX, fr.SY, fr.SX+fr.SW, fr.SY+fr.SH)
		if !r.In(b) {
			t.Fatalf("frame %d rect %v not inside sheet bounds %v", i, r, b)
		}
	}
	if doc.TilePx > 0 {
		for i, fr := range doc.Frames {
			if fr.Skip {
				continue
			}
			if fr.SW != doc.TilePx || fr.SH != doc.TilePx {
				t.Logf("frame %d: sw×sh %d×%d differs from tile_px %d (allowed; game scales to TileSize)", i, fr.SW, fr.SH, doc.TilePx)
			}
		}
	}
}
