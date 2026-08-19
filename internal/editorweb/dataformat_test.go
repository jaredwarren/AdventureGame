package editorweb

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaredwarren/game-test/internal/tiled"
)

// TestDetectStyleOnCorpus pins which shipped maps use which conventions. The
// room prefabs lay their tile data out as a floor plan and write properties
// inline; the generated grid maps do neither.
func TestDetectStyleOnCorpus(t *testing.T) {
	want := map[string]fileStyle{
		"maps/rooms/boss.tmj":  {Data: layoutRowPerLine, InlineProperties: true},
		"maps/rooms/start.tmj": {Data: layoutRowPerLine, InlineProperties: true},
		"maps/F-5.tmj":         {Data: layoutOnePerLine, InlineProperties: false},
		"maps/A-1.tmj":         {Data: layoutOnePerLine, InlineProperties: false},
		"maps/maze2.tmj":       {Data: layoutOnePerLine, InlineProperties: false},
	}
	forEachShippedMap(t, func(t *testing.T, name string, raw []byte) {
		expect, pinned := want[name]
		if !pinned {
			return
		}
		if got := detectStyle(raw); got != expect {
			t.Errorf("style %+v, want %+v", got, expect)
		}
	})
}

// TestRowPerLineLayoutIsPreserved is the point of the whole file: a room
// prefab's data array is a readable floor plan, and a save must not flatten it.
func TestRowPerLineLayoutIsPreserved(t *testing.T) {
	m := &tiled.Map{
		Width: 3, Height: 2, TileWidth: 16, TileHeight: 16,
		NextLayerID: 2, NextObjectID: 1,
		Tilesets: json.RawMessage("[]"),
		Layers: []tiled.Layer{{
			ID: 1, Type: "tilelayer", Name: "ground", Visible: true, Opacity: 1,
			Width: 3, Height: 2, Data: []int{1, 2, 1, 2, 1, 2},
		}},
	}

	grid, err := encodeWithStyle(m, fileStyle{Data: layoutRowPerLine})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(grid), "1, 2, 1,\n") || !strings.Contains(string(grid), "2, 1, 2\n") {
		t.Errorf("data was not laid out one map row per line:\n%s", grid)
	}

	// The written file must advertise the same style it was written in, and
	// re-encoding it must be a no-op. Without that, every save would flip the
	// layout back and forth.
	if got := detectStyle(grid); got.Data != layoutRowPerLine {
		t.Errorf("round-tripped file detects as %v, want layoutRowPerLine", got.Data)
	}
	reparsed, err := tiled.ParseMap(grid)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	again, err := encodeWithStyle(reparsed, detectStyle(grid))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(grid, again) {
		t.Errorf("encoding is not idempotent:\n--- first ---\n%s\n--- second ---\n%s", grid, again)
	}

	// Flattened, the same map must put one value per line.
	flat, err := encodeWithStyle(m, fileStyle{Data: layoutOnePerLine})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(flat, []byte("1, 2, 1")) {
		t.Error("layoutOnePerLine still grouped values onto one line")
	}
	if detectStyle(flat).Data != layoutOnePerLine {
		t.Error("a flattened file should detect as layoutOnePerLine")
	}
}

// TestInlinePropertiesArePreserved covers the other convention the room prefabs
// use.
func TestInlinePropertiesArePreserved(t *testing.T) {
	expanded := []byte(`{
  "properties": [
    {
      "name": "socket",
      "type": "string",
      "value": "N"
    }
  ]
}`)
	got := string(collapseProperties(expanded))
	if !strings.Contains(got, `{ "name": "socket", "type": "string", "value": "N" }`) {
		t.Errorf("property was not collapsed onto one line:\n%s", got)
	}
}

// TestApplyDataLayoutLeavesMalformedArraysAlone: if the data length is not a
// whole number of rows the layer is broken, and inventing a wrap would obscure
// that rather than fix it.
func TestApplyDataLayoutLeavesMalformedArraysAlone(t *testing.T) {
	in := []byte("\"data\": [\n        1,\n        2,\n        3,\n        4,\n        5\n      ]")
	out := applyDataLayout(in, layoutRowPerLine, []int{3}) // 5 values, width 3
	if !bytes.Equal(in, out) {
		t.Errorf("rewrapped a partial grid:\n%s", out)
	}
}

// TestApplyDataLayoutHandlesMultipleLayers checks each data array gets its own
// layer's width.
func TestApplyDataLayoutHandlesMultipleLayers(t *testing.T) {
	in := []byte(
		"\"data\": [\n        1,\n        2,\n        3,\n        4\n      ]" +
			"\n\"data\": [\n        5,\n        6\n      ]")
	out := string(applyDataLayout(in, layoutRowPerLine, []int{2, 2}))
	if !strings.Contains(out, "1, 2") || !strings.Contains(out, "3, 4") {
		t.Errorf("first layer not wrapped at width 2:\n%s", out)
	}
	if !strings.Contains(out, "5, 6") {
		t.Errorf("second layer not wrapped at width 2:\n%s", out)
	}
}

// TestTileLayerWidthsSkipsObjectGroups keeps the widths list aligned with the
// data arrays applyDataLayout will walk.
func TestTileLayerWidthsSkipsObjectGroups(t *testing.T) {
	m := &tiled.Map{
		Width: 9,
		Layers: []tiled.Layer{
			{Type: "tilelayer", Width: 4},
			{Type: "objectgroup"},
			{Type: "tilelayer"}, // no width: falls back to the map's
		},
	}
	got := tileLayerWidths(m)
	if len(got) != 2 || got[0] != 4 || got[1] != 9 {
		t.Errorf("widths %v, want [4 9]", got)
	}
}
