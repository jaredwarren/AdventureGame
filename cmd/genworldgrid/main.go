// genworldgrid writes a 10×10 overworld grid of minimal .tmj maps (A-1 … J-10)
// with cardinal door links.
//
// Authored overworld rooms are stamped from existing maps onto grid cells:
//
//	field1 → F-5 (start)
//	field2 → F-6 (east of start)
//	maze2  → G-5 (south of start)
//
// Usage:
//
//	go run ./cmd/genworldgrid -out assets/maps
//	go run ./cmd/genworldgrid -out assets/maps -stamp-authored
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

	startID = "F-5"
)

type edge int

const (
	edgeNorth edge = iota
	edgeSouth
	edgeWest
	edgeEast
)

type doorSpec struct {
	X, Y, W, H     float64
	SpawnX, SpawnY float64
}

type authoredCell struct {
	row, col int
	source   string
}

var authoredCells = []authoredCell{
	{row: 5, col: 4, source: "field1.tmj"}, // F-5
	{row: 5, col: 5, source: "field2.tmj"}, // F-6 (east of start)
	{row: 6, col: 4, source: "maze2.tmj"},  // G-5 (south of start)
}

func authoredIDSet() map[string]bool {
	s := make(map[string]bool, len(authoredCells))
	for _, a := range authoredCells {
		s[cellName(a.row, a.col)] = true
	}
	return s
}

func main() {
	outDir := flag.String("out", "assets/maps", "output directory for .tmj files")
	stamp := flag.Bool("stamp-authored", false, "overwrite F-5/F-6/G-5 from field1/field2/maze2")
	flag.Parse()

	if err := run(*outDir, *stamp); err != nil {
		fmt.Fprintf(os.Stderr, "genworldgrid: %v\n", err)
		os.Exit(1)
	}
}

func run(outDir string, stamp bool) error {
	authored := authoredIDSet()
	sizes := map[string][2]int{}
	for _, a := range authoredCells {
		id := cellName(a.row, a.col)
		src := filepath.Join(outDir, a.source)
		m, err := tiled.LoadMap(src)
		if err != nil {
			return fmt.Errorf("load source %s: %w", src, err)
		}
		sizes[id] = [2]int{m.Width, m.Height}
	}

	written := 0
	for row := 0; row < gridRows; row++ {
		for col := 0; col < gridCols; col++ {
			id := cellName(row, col)
			if authored[id] {
				continue
			}
			m := buildMap(row, col, sizes)
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

	stamped := 0
	if stamp {
		for _, a := range authoredCells {
			if err := stampCell(outDir, a, sizes); err != nil {
				return err
			}
			stamped++
		}
	}

	fmt.Fprintf(os.Stderr, "genworldgrid: start %s, wrote %d empty maps, stamped %d authored\n", startID, written, stamped)
	return nil
}

func cellName(row, col int) string {
	return fmt.Sprintf("%c-%d", 'A'+row, col+1)
}

func neighborID(row, col int, e edge) (string, bool) {
	switch e {
	case edgeNorth:
		if row == 0 {
			return "", false
		}
		return cellName(row-1, col), true
	case edgeSouth:
		if row == gridRows-1 {
			return "", false
		}
		return cellName(row+1, col), true
	case edgeWest:
		if col == 0 {
			return "", false
		}
		return cellName(row, col-1), true
	case edgeEast:
		if col == gridCols-1 {
			return "", false
		}
		return cellName(row, col+1), true
	default:
		return "", false
	}
}

func sizeOf(id string, sizes map[string][2]int) (w, h int) {
	if s, ok := sizes[id]; ok {
		return s[0], s[1]
	}
	return mapW, mapH
}

func opposite(e edge) edge {
	switch e {
	case edgeNorth:
		return edgeSouth
	case edgeSouth:
		return edgeNorth
	case edgeWest:
		return edgeEast
	case edgeEast:
		return edgeWest
	default:
		return e
	}
}

func doorRect(w, h int, e edge) (x, y, dw, dh float64) {
	pw := float64(w * tile.Size)
	ph := float64(h * tile.Size)
	switch e {
	case edgeNorth:
		return 160, 0, 32, 16
	case edgeSouth:
		return 160, ph - 16, 32, 16
	case edgeWest:
		return 0, 112, 16, 32
	case edgeEast:
		return pw - 16, 112, 16, 32
	default:
		return 0, 0, 16, 16
	}
}

func spawnAtEntry(w, h int, entry edge) (sx, sy float64) {
	pw := float64(w * tile.Size)
	ph := float64(h * tile.Size)
	switch entry {
	case edgeNorth:
		return 160, 40
	case edgeSouth:
		return 160, ph - 40
	case edgeWest:
		return 32, 112
	case edgeEast:
		return pw - 40, 112
	default:
		return 160, 120
	}
}

func specToNeighbor(srcW, srcH, dstW, dstH int, e edge) doorSpec {
	x, y, dw, dh := doorRect(srcW, srcH, e)
	sx, sy := spawnAtEntry(dstW, dstH, opposite(e))
	return doorSpec{X: x, Y: y, W: dw, H: dh, SpawnX: sx, SpawnY: sy}
}

func buildMap(row, col int, sizes map[string][2]int) *tiled.Map {
	objects := []tiled.Object{
		spawnObject(1),
	}
	nextID := 2
	for _, e := range []edge{edgeNorth, edgeSouth, edgeWest, edgeEast} {
		target, ok := neighborID(row, col, e)
		if !ok {
			continue
		}
		tw, th := sizeOf(target, sizes)
		spec := specToNeighbor(mapW, mapH, tw, th, e)
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
				Data:    groundTiles(row, col),
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

func groundTiles(row, col int) []int {
	data := make([]int, mapW*mapH)
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			gid := tile.GIDGrass
			onBorder := x == 0 || x == mapW-1 || y == 0 || y == mapH-1
			if onBorder && !isDoorGap(row, col, x, y) {
				gid = tile.GIDWall
			}
			data[y*mapW+x] = gid
		}
	}
	return data
}

func isDoorGap(row, col, tx, ty int) bool {
	if ty == 0 && hasEdge(row, col, edgeNorth) && (tx == 10 || tx == 11) {
		return true
	}
	if ty == mapH-1 && hasEdge(row, col, edgeSouth) && (tx == 10 || tx == 11) {
		return true
	}
	if tx == 0 && hasEdge(row, col, edgeWest) && (ty == 7 || ty == 8) {
		return true
	}
	if tx == mapW-1 && hasEdge(row, col, edgeEast) && (ty == 7 || ty == 8) {
		return true
	}
	return false
}

func hasEdge(row, col int, e edge) bool {
	_, ok := neighborID(row, col, e)
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

func stampCell(outDir string, a authoredCell, sizes map[string][2]int) error {
	id := cellName(a.row, a.col)
	srcPath := filepath.Join(outDir, a.source)
	m, err := tiled.LoadMap(srcPath)
	if err != nil {
		return fmt.Errorf("stamp %s: load %s: %w", id, srcPath, err)
	}
	lg := m.ObjectGroupLayer("markers")
	if lg == nil {
		return fmt.Errorf("stamp %s: missing markers layer", id)
	}

	kept := lg.Objects[:0]
	maxID := 0
	for _, o := range lg.Objects {
		if o.ID > maxID {
			maxID = o.ID
		}
		if o.Type == "door" && o.Name != "to_boss" {
			continue
		}
		kept = append(kept, o)
	}
	lg.Objects = kept
	nextID := maxID + 1
	if m.NextObjectID > nextID {
		nextID = m.NextObjectID
	}

	for _, e := range []edge{edgeNorth, edgeSouth, edgeWest, edgeEast} {
		target, ok := neighborID(a.row, a.col, e)
		if !ok {
			continue
		}
		tw, th := sizeOf(target, sizes)
		spec := specToNeighbor(m.Width, m.Height, tw, th, e)
		lg.Objects = append(lg.Objects, doorObject(nextID, edgeName(e), target, spec))
		nextID++
		punchDoorGap(m, e)
	}

	if id == startID {
		lg.Objects = append(lg.Objects, doorObject(nextID, "to_dungeon", "dungeon", doorSpec{
			X: 96, Y: 0, W: 32, H: 16, SpawnX: 160, SpawnY: 200,
		}))
		nextID++
		punchTile(m, 6, 0)
		punchTile(m, 7, 0)
	}

	m.NextObjectID = nextID
	dest := filepath.Join(outDir, id+".tmj")
	if err := m.WriteFile(dest); err != nil {
		return fmt.Errorf("stamp %s: write: %w", id, err)
	}
	if _, err := tiled.LoadMap(dest); err != nil {
		return fmt.Errorf("stamp %s: validate: %w", id, err)
	}
	return nil
}

func punchDoorGap(m *tiled.Map, e edge) {
	switch e {
	case edgeNorth:
		punchTile(m, 10, 0)
		punchTile(m, 11, 0)
	case edgeSouth:
		punchTile(m, 10, m.Height-1)
		punchTile(m, 11, m.Height-1)
	case edgeWest:
		punchTile(m, 0, 7)
		punchTile(m, 0, 8)
	case edgeEast:
		punchTile(m, m.Width-1, 7)
		punchTile(m, m.Width-1, 8)
	}
}

func punchTile(m *tiled.Map, tx, ty int) {
	if tx < 0 || ty < 0 || tx >= m.Width || ty >= m.Height {
		return
	}
	for i := range m.Layers {
		ly := &m.Layers[i]
		if ly.Type != "tilelayer" || len(ly.Data) == 0 {
			continue
		}
		w := ly.Width
		if w <= 0 {
			w = m.Width
		}
		idx := ty*w + tx
		if idx >= 0 && idx < len(ly.Data) {
			ly.Data[idx] = tile.GIDGrass
		}
	}
}
