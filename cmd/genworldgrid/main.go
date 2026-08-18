// genworldgrid writes a 10×10 overworld grid of minimal .tmj maps (A-1 … J-10)
// with cardinal door links. Cell F-5 is field1 (patched in place); the other
// 99 cells are new files.
//
// Usage:
//
//	go run ./cmd/genworldgrid -out assets/maps
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

const (
	gridCols = 10
	gridRows = 10

	mapW = 20
	mapH = 15

	centerCol = 5 // F (A=0 … J=9)
	centerRow = 4 // row 5 (1-based)

	centerMapID = "field1"
)

type edge int

const (
	edgeNorth edge = iota
	edgeSouth
	edgeWest
	edgeEast
)

type doorSpec struct {
	X, Y, W, H    float64
	SpawnX, SpawnY float64
}

// Door rects sit on the 16px border tiles with gaps at:
//   N/S: tiles (10,11)  → x=160, w=32
//   W/E: tiles (7,8)    → y=112, h=32
var standardDoors = map[edge]doorSpec{
	edgeNorth: {X: 160, Y: 0, W: 32, H: 16, SpawnX: 160, SpawnY: 200},
	edgeSouth: {X: 160, Y: 224, W: 32, H: 16, SpawnX: 160, SpawnY: 40},
	edgeWest:  {X: 0, Y: 112, W: 16, H: 32, SpawnX: 280, SpawnY: 112},
	edgeEast:  {X: 304, Y: 112, W: 16, H: 32, SpawnX: 32, SpawnY: 112},
}

// field1GridDoors uses offset positions where existing doors occupy standard slots.
var field1GridDoors = map[edge]doorSpec{
	edgeNorth: {X: 96, Y: 0, W: 32, H: 16, SpawnX: 160, SpawnY: 200},
	edgeSouth: standardDoors[edgeSouth],
	edgeWest:  standardDoors[edgeWest],
	edgeEast:  {X: 304, Y: 96, W: 16, H: 32, SpawnX: 32, SpawnY: 112},
}

func main() {
	outDir := flag.String("out", "assets/maps", "output directory for .tmj files")
	flag.Parse()

	if err := run(*outDir); err != nil {
		fmt.Fprintf(os.Stderr, "genworldgrid: %v\n", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	written := 0
	for col := 0; col < gridCols; col++ {
		for row := 0; row < gridRows; row++ {
			id := mapID(col, row)
			if col == centerCol && row == centerRow {
				continue
			}
			m := buildMap(col, row, id)
			path := filepath.Join(outDir, id+".tmj")
			if err := m.WriteFile(path); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
			if _, err := tiled.LoadMap(path); err != nil {
				return fmt.Errorf("validate %s: %w", path, err)
			}
			written++
		}
	}

	field1Path := filepath.Join(outDir, centerMapID+".tmj")
	if err := patchField1(field1Path); err != nil {
		return err
	}
	if _, err := tiled.LoadMap(field1Path); err != nil {
		return fmt.Errorf("validate %s: %w", field1Path, err)
	}

	fmt.Fprintf(os.Stderr, "genworldgrid: field1 @ F-5, wrote %d maps to %s\n", written, outDir)
	return nil
}

func mapID(col, row int) string {
	if col == centerCol && row == centerRow {
		return centerMapID
	}
	return cellName(col, row)
}

func cellName(col, row int) string {
	return fmt.Sprintf("%c-%d", 'A'+col, row+1)
}

func neighborID(col, row int, e edge) (string, bool) {
	switch e {
	case edgeNorth:
		if row == 0 {
			return "", false
		}
		return mapID(col, row-1), true
	case edgeSouth:
		if row == gridRows-1 {
			return "", false
		}
		return mapID(col, row+1), true
	case edgeWest:
		if col == 0 {
			return "", false
		}
		return mapID(col-1, row), true
	case edgeEast:
		if col == gridCols-1 {
			return "", false
		}
		return mapID(col+1, row), true
	default:
		return "", false
	}
}

func buildMap(col, row int, id string) *tiled.Map {
	objects := []tiled.Object{
		spawnObject(1),
	}
	nextID := 2

	for _, e := range []edge{edgeNorth, edgeSouth, edgeWest, edgeEast} {
		target, ok := neighborID(col, row, e)
		if !ok {
			continue
		}
		spec := standardDoors[e]
		objects = append(objects, doorObject(nextID, edgeName(e), target, spec))
		nextID++
	}

	return &tiled.Map{
		CompressionLevel: -1,
		Width:            mapW,
		Height:           mapH,
		Infinite:         false,
		TileWidth:        tile.Size,
		TileHeight:       tile.Size,
		Type:             "map",
		Version:          "1.10",
		TiledVersion:     "1.10.2",
		Orientation:      "orthogonal",
		RenderOrder:      "right-down",
		Layers: []tiled.Layer{
			{
				ID:      1,
				Type:    "tilelayer",
				Name:    "ground",
				Visible: true,
				Opacity: 1,
				Width:   mapW,
				Height:  mapH,
				Data:    groundTiles(col, row),
			},
			{
				ID:      2,
				Type:    "objectgroup",
				Name:    "markers",
				Visible: true,
				Opacity: 1,
				Objects: objects,
			},
		},
		NextLayerID:  3,
		NextObjectID: nextID,
		Tilesets:     json.RawMessage("[]"),
	}
}

func groundTiles(col, row int) []int {
	data := make([]int, mapW*mapH)
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			gid := tile.GIDGrass
			onBorder := x == 0 || x == mapW-1 || y == 0 || y == mapH-1
			if onBorder && !isDoorGap(col, row, x, y) {
				gid = tile.GIDWall
			}
			data[y*mapW+x] = gid
		}
	}
	return data
}

func isDoorGap(col, row, tx, ty int) bool {
	if ty == 0 && hasEdge(col, row, edgeNorth) && (tx == 10 || tx == 11) {
		return true
	}
	if ty == mapH-1 && hasEdge(col, row, edgeSouth) && (tx == 10 || tx == 11) {
		return true
	}
	if tx == 0 && hasEdge(col, row, edgeWest) && (ty == 7 || ty == 8) {
		return true
	}
	if tx == mapW-1 && hasEdge(col, row, edgeEast) && (ty == 7 || ty == 8) {
		return true
	}
	return false
}

func hasEdge(col, row int, e edge) bool {
	_, ok := neighborID(col, row, e)
	return ok
}

func spawnObject(id int) tiled.Object {
	return tiled.Object{
		ID:   id,
		Name: "spawn",
		Type: "spawn",
		X:    160,
		Y:    120,
	}
}

func doorObject(id int, name, target string, spec doorSpec) tiled.Object {
	return tiled.Object{
		ID:     id,
		Name:   name,
		Type:   "door",
		X:      spec.X,
		Y:      spec.Y,
		Width:  spec.W,
		Height: spec.H,
		Properties: []tiled.Property{
			{Name: "target_map", Type: "string", Value: target},
			{Name: "spawn_x", Type: "string", Value: strconv.Itoa(int(spec.SpawnX))},
			{Name: "spawn_y", Type: "string", Value: strconv.Itoa(int(spec.SpawnY))},
		},
	}
}

func edgeName(e edge) string {
	switch e {
	case edgeNorth:
		return "door_N"
	case edgeSouth:
		return "door_S"
	case edgeWest:
		return "door_W"
	case edgeEast:
		return "door_E"
	default:
		return "door"
	}
}

func patchField1(path string) error {
	m, err := tiled.LoadMap(path)
	if err != nil {
		return fmt.Errorf("load field1: %w", err)
	}
	lg := m.ObjectGroupLayer("markers")
	if lg == nil {
		return fmt.Errorf("field1: missing markers layer")
	}

	filtered := lg.Objects[:0]
	for _, o := range lg.Objects {
		if len(o.Name) >= 10 && o.Name[:10] == "grid_door_" {
			continue
		}
		filtered = append(filtered, o)
	}
	lg.Objects = filtered

	nextID := m.NextObjectID
	if nextID <= 0 {
		nextID = 102
	}

	for _, e := range []edge{edgeNorth, edgeSouth, edgeWest, edgeEast} {
		target, ok := neighborID(centerCol, centerRow, e)
		if !ok {
			continue
		}
		spec := field1GridDoors[e]
		lg.Objects = append(lg.Objects, doorObject(nextID, "grid_"+edgeName(e), target, spec))
		nextID++
	}
	m.NextObjectID = nextID

	if err := m.WriteFile(path); err != nil {
		return fmt.Errorf("write field1: %w", err)
	}
	return nil
}
