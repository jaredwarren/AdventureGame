package dungeon

import (
	"encoding/json"
	"fmt"

	"github.com/jaredwarren/game-test/internal/tiled"
)

const (
	TileWall  = 2
	TileFloor = 7
	TileLock  = 6
)

type PrefabRoom struct {
	Name    string
	Width   int
	Height  int
	Tiles   []int
	Objects []tiled.Object
}

type RoomLibrary struct {
	Rooms map[string]*PrefabRoom
}

// LoadRoomLibrary parses embedded bytes for room prefabs into a RoomLibrary.
func LoadRoomLibrary(files map[string][]byte) (*RoomLibrary, error) {
	lib := &RoomLibrary{Rooms: make(map[string]*PrefabRoom)}
	for name, b := range files {
		m, err := tiled.ParseMap(b)
		if err != nil {
			return nil, fmt.Errorf("dungeon: parse room library file %s: %w", name, err)
		}
		pr := &PrefabRoom{
			Name:   name,
			Width:  m.Width,
			Height: m.Height,
		}
		tl := m.TileLayer("ground")
		if tl != nil {
			pr.Tiles = append([]int(nil), tl.Data...)
		}
		for _, o := range m.ObjectGroup("markers") {
			pr.Objects = append(pr.Objects, o)
		}
		lib.Rooms[name] = pr
	}
	return lib, nil
}

// Generate creates a room graph and stitches it into a playable tiled.Map.
func Generate(seed int64, lib *RoomLibrary) (*tiled.Map, *Result, error) {
	g, err := GenerateGraph(seed)
	if err != nil {
		return nil, nil, err
	}
	tm, err := Stitch(g, lib)
	if err != nil {
		return nil, nil, err
	}
	res := &Result{Graph: *g}
	return tm, res, nil
}

// Stitch composes a room graph and room library into a tiled.Map.
func Stitch(g *Graph, lib *RoomLibrary) (*tiled.Map, error) {
	if g == nil || len(g.Nodes) == 0 {
		return nil, fmt.Errorf("dungeon: invalid graph")
	}

	minX, maxX := g.Nodes[0].X, g.Nodes[0].X
	minY, maxY := g.Nodes[0].Y, g.Nodes[0].Y
	for _, n := range g.Nodes {
		if n.X < minX {
			minX = n.X
		}
		if n.X > maxX {
			maxX = n.X
		}
		if n.Y < minY {
			minY = n.Y
		}
		if n.Y > maxY {
			maxY = n.Y
		}
	}

	const roomW, roomH = 7, 7
	const cellSpacingW, cellSpacingH = 10, 10
	const pad = 2

	gridW := (maxX - minX + 1)
	gridH := (maxY - minY + 1)
	mapW := gridW*cellSpacingW + pad*2
	mapH := gridH*cellSpacingH + pad*2

	groundData := make([]int, mapW*mapH)
	for i := range groundData {
		groundData[i] = TileWall
	}

	var objects []tiled.Object
	nextObjID := 1

	type pos struct{ ox, oy int }
	nodePos := make(map[int]pos)

	for _, n := range g.Nodes {
		ox := pad + (n.X-minX)*cellSpacingW
		oy := pad + (n.Y-minY)*cellSpacingH
		nodePos[n.ID] = pos{ox, oy}

		prefabKey := selectPrefabForRole(n.Role)
		pr := lib.Rooms[prefabKey]
		if pr == nil && lib != nil {
			for _, r := range lib.Rooms {
				pr = r
				break
			}
		}

		if pr != nil {
			for ry := 0; ry < pr.Height && ry < roomH; ry++ {
				for rx := 0; rx < pr.Width && rx < roomW; rx++ {
					tx := ox + rx
					ty := oy + ry
					if tx < mapW && ty < mapH {
						srcIdx := ry*pr.Width + rx
						if srcIdx < len(pr.Tiles) {
							groundData[ty*mapW+tx] = pr.Tiles[srcIdx]
						}
					}
				}
			}
			for _, o := range pr.Objects {
				isSocket := false
				for _, p := range o.Properties {
					if p.Name == "socket" {
						isSocket = true
						break
					}
				}
				if isSocket {
					continue
				}

				obj := o
				obj.ID = nextObjID
				nextObjID++
				obj.X += float64(ox * 16)
				obj.Y += float64(oy * 16)
				objects = append(objects, obj)
			}
		}
	}

	for _, e := range g.Edges {
		pA := nodePos[e.From]
		pB := nodePos[e.To]

		cA_x, cA_y := pA.ox+roomW/2, pA.oy+roomH/2
		cB_x, cB_y := pB.ox+roomW/2, pB.oy+roomH/2

		stepX := 0
		if cB_x > cA_x {
			stepX = 1
		} else if cB_x < cA_x {
			stepX = -1
		}
		stepY := 0
		if cB_y > cA_y {
			stepY = 1
		} else if cB_y < cA_y {
			stepY = -1
		}

		midX := (cA_x + cB_x) / 2
		midY := (cA_y + cB_y) / 2

		cx, cy := cA_x, cA_y
		for cx != cB_x {
			groundData[cy*mapW+cx] = TileFloor
			if e.IsLocked && cx == midX {
				groundData[cy*mapW+cx] = TileLock
			}
			cx += stepX
		}
		for cy != cB_y {
			groundData[cy*mapW+cx] = TileFloor
			if e.IsLocked && cy == midY {
				groundData[cy*mapW+cx] = TileLock
			}
			cy += stepY
		}
		groundData[cB_y*mapW+cB_x] = TileFloor
	}

	m := &tiled.Map{
		CompressionLevel: -1,
		Width:            mapW,
		Height:           mapH,
		Infinite:         false,
		TileWidth:        16,
		TileHeight:       16,
		Type:             "map",
		Version:          "1.10",
		Orientation:      "orthogonal",
		NextLayerID:      3,
		NextObjectID:     nextObjID,
		Tilesets:         json.RawMessage("[]"),
		Layers: []tiled.Layer{
			{
				ID:      1,
				Type:    "tilelayer",
				Name:    "ground",
				Visible: true,
				Opacity: 1,
				Width:   mapW,
				Height:  mapH,
				Data:    groundData,
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
		Properties: []tiled.Property{
			{Name: "light_level", Type: "float", Value: 0.3},
		},
	}

	return m, nil
}

func selectPrefabForRole(role Role) string {
	switch role {
	case RoleStart:
		return "start.tmj"
	case RoleKey:
		return "key.tmj"
	case RoleBoss:
		return "boss.tmj"
	default:
		return "combat.tmj"
	}
}
