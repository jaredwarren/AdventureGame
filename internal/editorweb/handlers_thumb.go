package editorweb

import (
	"net/http"

	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// Map thumbnails for the browser sidebar.
//
// The 10x10 world grid needs 100 miniatures at once. Drawing them from the tile
// atlas would be 300 drawImage calls per map and a 16x downscale that reads as
// noise; instead the client paints one pixel per tile from Tile.SwatchColor,
// which is exactly what that field exists for.
//
// The server composites the layers and run-length encodes the result, so a
// mostly-uniform grid cell costs a few dozen bytes and the client does no layer
// logic.

// Thumb is one map's miniature.
type Thumb struct {
	ID   string `json:"id"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	ETag string `json:"etag"`
	// RLE is [gid, count, gid, count, ...] in row-major order.
	RLE []int `json:"rle"`
	// Markers are [tileX, tileY, markerTypeIndex] triples, indexing the schema's
	// marker list so the client can color them by type.
	Markers [][3]int `json:"markers"`
}

func (s *Server) handleThumb(w http.ResponseWriter, r *http.Request) {
	id, ok := s.mapID(w, r)
	if !ok {
		return
	}
	m, _, etag, err := s.store.Read(id)
	if err != nil {
		s.writeStoreErr(w, id, err)
		return
	}
	writeJSON(w, http.StatusOK, buildThumb(id, etag, m))
}

func buildThumb(id, etag string, m *tiled.Map) Thumb {
	t := Thumb{ID: id, W: m.Width, H: m.Height, ETag: etag, RLE: []int{}, Markers: [][3]int{}}
	if m.Width <= 0 || m.Height <= 0 {
		return t
	}

	// Composite top-down: the highest layer with a non-empty gid wins, matching
	// how the renderer stacks layers (gid 0 is a hole, not black).
	flat := make([]int, m.Width*m.Height)
	for i := range m.Layers {
		l := m.Layers[i]
		if l.Type != "tilelayer" || len(l.Data) != len(flat) {
			continue
		}
		for j, gid := range l.Data {
			if gid != tile.GIDEmpty {
				flat[j] = gid
			}
		}
	}

	// Run-length encode. An empty grid cell is a wall border around grass, so
	// this collapses to a few dozen ints.
	for i := 0; i < len(flat); {
		gid := flat[i]
		run := 1
		for i+run < len(flat) && flat[i+run] == gid {
			run++
		}
		t.RLE = append(t.RLE, gid, run)
		i += run
	}

	typeIndex := map[string]int{}
	for i, name := range world.MarkerTypeNames() {
		typeIndex[name] = i
	}
	if group := m.ObjectGroupLayer(MarkersLayerName); group != nil {
		for _, o := range group.Objects {
			idx, known := typeIndex[o.Type]
			if !known {
				continue
			}
			tx, ty := int(o.X)/tile.Size, int(o.Y)/tile.Size
			if tx < 0 || ty < 0 || tx >= m.Width || ty >= m.Height {
				continue
			}
			t.Markers = append(t.Markers, [3]int{tx, ty, idx})
		}
	}
	return t
}
