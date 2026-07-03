// gentmj generates a random orthogonal maze as a Tiled .tmj (single ground
// layer + empty markers group) using the Growing Tree algorithm from
// internal/dungeon.
//
// Usage:
//
//	go run ./cmd/gentmj -o assets/maps/proc_maze.tmj -w 10 -h 8 -seed 42
//
// Open tiles are painted with grass / water / floor2 (see -floor-* weights).
// Water is solid in-game; default -floor-water is 0 so mazes stay connected.
// Trees may replace dead ends (-tree-deadend) on grass/floor2 only; spawn is never a tree.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"

	"github.com/jaredwarren/game-test/assets"
	"github.com/jaredwarren/game-test/internal/dungeon"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
)

func main() {
	outPath := flag.String("o", "", "output path for .tmj (required)")
	dungeonMode := flag.Bool("dungeon", false, "generate fully stitched dungeon map")
	cellsW := flag.Int("w", 10, "maze width in cells (>=1)")
	cellsH := flag.Int("h", 10, "maze height in cells (>=1)")
	seed := flag.Int64("seed", 0, "RNG seed")
	style := flag.String("style", "backtracker", "maze style: backtracker, prim, blend")
	blend := flag.Float64("blend", 0.5, "when style=blend, newest fraction [0,1]")
	extra := flag.Float64("extra", 0, "probability to open extra walls after tree (0=perfect maze)")
	floorGrass := flag.Float64("floor-grass", 0.58, "relative weight for GIDGrass on open tiles")
	floorWater := flag.Float64("floor-water", 0, "relative weight for GIDWater (solid; default 0 keeps maze connected)")
	floorFloor2 := flag.Float64("floor-floor2", 0.42, "relative weight for GIDFloor2 on open tiles")
	treeDeadEnd := flag.Float64("tree-deadend", 0.28, "probability a dead-end open tile (not spawn) becomes GIDTree [0,1]")
	flag.Parse()

	if *outPath == "" {
		fmt.Fprintln(os.Stderr, "gentmj: -o output.tmj is required")
		flag.Usage()
		os.Exit(2)
	}

	if *dungeonMode {
		libFiles := make(map[string][]byte)
		for _, name := range []string{"start.tmj", "combat.tmj", "key.tmj", "boss.tmj", "corridor.tmj"} {
			b, err := assets.MapFS.ReadFile("maps/rooms/" + name)
			if err != nil {
				b, err = os.ReadFile("assets/maps/rooms/" + name)
			}
			if err == nil {
				libFiles[name] = b
			}
		}
		lib, err := dungeon.LoadRoomLibrary(libFiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gentmj: load room library: %v\n", err)
			os.Exit(1)
		}
		tm, res, err := dungeon.Generate(*seed, lib)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gentmj: generate dungeon: %v\n", err)
			os.Exit(1)
		}
		if err := tm.WriteFile(*outPath); err != nil {
			fmt.Fprintf(os.Stderr, "gentmj: write %s: %v\n", *outPath, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote stitched dungeon %s (%dx%d tiles, seed=%d, digest=%s)\n", *outPath, tm.Width, tm.Height, *seed, res.BugDigest())
		return
	}
	if *cellsW < 1 || *cellsH < 1 {
		fmt.Fprintln(os.Stderr, "gentmj: -w and -h must be >= 1")
		os.Exit(2)
	}

	cfg := dungeon.GrowingTreeBacktracker()
	switch *style {
	case "backtracker":
		cfg = dungeon.GrowingTreeBacktracker()
	case "prim":
		cfg = dungeon.GrowingTreePrimLike()
	case "blend":
		cfg = dungeon.GrowingTreeBlend(*blend)
	default:
		fmt.Fprintf(os.Stderr, "gentmj: unknown -style %q (use backtracker, prim, blend)\n", *style)
		os.Exit(2)
	}
	if *extra < 0 || *extra > 1 {
		fmt.Fprintln(os.Stderr, "gentmj: -extra must be in [0,1]")
		os.Exit(2)
	}
	if *treeDeadEnd < 0 || *treeDeadEnd > 1 {
		fmt.Fprintln(os.Stderr, "gentmj: -tree-deadend must be in [0,1]")
		os.Exit(2)
	}
	cfg.ExtraPassageProbability = *extra

	m := dungeon.GenerateGrowingTree(*cellsW, *cellsH, *seed, cfg)
	mw, mh, tileData := dungeon.StampOrthogonalMaze(m, world.GIDGrass, world.GIDWall)

	spawnTX, spawnTY := firstGrassTile(mw, mh, tileData, world.GIDGrass)
	paintRNG := rand.New(rand.NewSource(*seed ^ 0x9e3779b9))
	dungeon.PaintNaturalFloors(mw, mh, tileData, dungeon.FloorPaintParams{
		Floors: []dungeon.FloorTileWeight{
			{GID: world.GIDGrass, Weight: *floorGrass, Passable: true},
			{GID: world.GIDWater, Weight: *floorWater, Passable: false},
			{GID: world.GIDFloor2, Weight: *floorFloor2, Passable: true},
		},
		TreeGID:         world.GIDTree,
		WallGID:         world.GIDWall,
		SpawnX:          spawnTX,
		SpawnY:          spawnTY,
		TreeDeadEndProb: *treeDeadEnd,
	}, paintRNG)
	sx := float64(spawnTX*world.TileSize + world.TileSize/2)
	sy := float64(spawnTY*world.TileSize + world.TileSize - 2)

	tmj := map[string]any{
		"compressionlevel": -1,
		"width":            mw,
		"height":           mh,
		"infinite":         false,
		"tilewidth":        world.TileSize,
		"tileheight":       world.TileSize,
		"type":             "map",
		"version":          "1.10",
		"tiledversion":     "1.10.2",
		"orientation":      "orthogonal",
		"renderorder":      "right-down",
		"layers": []any{
			map[string]any{
				"id":         1,
				"name":       "ground",
				"type":       "tilelayer",
				"visible":    true,
				"opacity":    1,
				"width":      mw,
				"height":     mh,
				"x":          0,
				"y":          0,
				"data":       tileData,
				"properties": mazeMetaProps(*seed, *style, *cellsW, *cellsH, *floorGrass, *floorWater, *floorFloor2, *treeDeadEnd),
			},
			map[string]any{
				"id":      2,
				"name":    "markers",
				"type":    "objectgroup",
				"visible": true,
				"opacity": 1,
				"objects": spawnObject(sx, sy),
			},
		},
		"nextlayerid":  3,
		"nextobjectid": 2,
		"tilesets":     []any{},
		"properties":   rootMetaProps(*seed, *style),
	}

	b, err := json.MarshalIndent(tmj, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gentmj: json: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gentmj: write %s: %v\n", *outPath, err)
		os.Exit(1)
	}

	// Validate round-trip for the game loader.
	if _, err := tiled.ParseMap(b); err != nil {
		fmt.Fprintf(os.Stderr, "gentmj: wrote file but tiled.ParseMap failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%dx%d tiles, %dx%d maze cells, seed=%d)\n", *outPath, mw, mh, *cellsW, *cellsH, *seed)
}

func firstGrassTile(mw, mh int, data []int, grass int) (tx, ty int) {
	for ty = 0; ty < mh; ty++ {
		for tx = 0; tx < mw; tx++ {
			if data[ty*mw+tx] == grass {
				return tx, ty
			}
		}
	}
	return 1, 1
}

func spawnObject(sx, sy float64) []any {
	return []any{
		map[string]any{
			"id":   1,
			"name": "spawn",
			"type": "spawn",
			"x":    sx,
			"y":    sy,
		},
	}
}

func mazeMetaProps(seed int64, style string, cw, ch int, wg, ww, wf2, treeP float64) []any {
	return []any{
		map[string]any{"name": "proc_seed", "type": "string", "value": fmt.Sprintf("%d", seed)},
		map[string]any{"name": "proc_style", "type": "string", "value": style},
		map[string]any{"name": "proc_cells_w", "type": "string", "value": fmt.Sprintf("%d", cw)},
		map[string]any{"name": "proc_cells_h", "type": "string", "value": fmt.Sprintf("%d", ch)},
		map[string]any{"name": "proc_floor_grass_w", "type": "string", "value": fmt.Sprintf("%g", wg)},
		map[string]any{"name": "proc_floor_water_w", "type": "string", "value": fmt.Sprintf("%g", ww)},
		map[string]any{"name": "proc_floor_floor2_w", "type": "string", "value": fmt.Sprintf("%g", wf2)},
		map[string]any{"name": "proc_tree_deadend_p", "type": "string", "value": fmt.Sprintf("%g", treeP)},
	}
}

func rootMetaProps(seed int64, style string) []any {
	return []any{
		map[string]any{"name": "generator", "type": "string", "value": "gentmj/growingtree"},
		map[string]any{"name": "seed", "type": "string", "value": fmt.Sprintf("%d", seed)},
		map[string]any{"name": "style", "type": "string", "value": style},
	}
}
