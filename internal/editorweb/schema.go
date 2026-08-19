package editorweb

import (
	"encoding/json"

	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/enemy"
	"github.com/jaredwarren/game-test/internal/world/pickup"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// SchemaVersion is bumped when the payload shape changes incompatibly. The
// client refuses to run against a version it does not understand rather than
// mis-rendering a map.
const SchemaVersion = 1

// Layer naming conventions the loader depends on.
//
// world.BuildFromTiled looks up the object group by the literal name "markers",
// and the maps use "base" (optional, bottom) and "ground" for tile layers.
const (
	MarkersLayerName = "markers"
	BaseLayerName    = "base"
	GroundLayerName  = "ground"
)

// Schema is everything the browser needs to render and edit maps, served once.
//
// Every field here is derived from internal/world at startup. Nothing in it is
// transcribed by hand, which is what lets a change in the game's registries show
// up in the editor without a matching JavaScript edit.
type Schema struct {
	SchemaVersion int `json:"schemaVersion"`
	TileSize      int `json:"tileSize"`

	Constants constants   `json:"constants"`
	Layers    layerSchema `json:"layers"`

	// Colors is the shared color table every tile op indexes into.
	Colors []string  `json:"colors"`
	Tiles  []tileArt `json:"tiles"`

	Palette   []paletteGroup `json:"palette"`
	Favorites []int          `json:"favorites"`

	Markers          []markerSchema `json:"markers"`
	UnknownMarkerHit hitRectModel   `json:"unknownMarkerHit"`

	Pickups       []fieldOption `json:"pickups"`
	MapProperties []markerField `json:"mapProperties"`
}

type constants struct {
	PlayerHitbox  size `json:"playerHitbox"`
	EnemyHitbox   size `json:"enemyHitbox"`
	PickupHitbox  size `json:"pickupHitbox"`
	GridCols      int  `json:"gridCols"`
	GridRows      int  `json:"gridRows"`
	GridCellTiles size `json:"gridCellTiles"`
}

type size struct {
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type layerSchema struct {
	TileLayerNames  []string `json:"tileLayerNames"`
	ObjectGroupName string   `json:"objectGroupName"`
	DefaultBaseGID  int      `json:"defaultBaseGid"`
}

// mapPropertyFields are the map-level properties world.BuildFromTiled reads.
//
// These cannot be probed: nothing in the codebase writes a default map property,
// so there is no template to read them off. TestMapPropertiesMatchLoader pins
// them against the loader.
var mapPropertyFields = []markerField{
	{
		Name: "light_level", Label: "Light level", Type: "float", TiledType: "float",
		Default: 1.0, Widget: "slider", Min: f64(0), Max: f64(1), Step: 0.05,
		Optional: true,
		Note:     "ambient light multiplier; preferred over ambient_light",
	},
	{
		Name: "ambient_light", Label: "Ambient light (legacy)", Type: "float", TiledType: "float",
		Default: 1.0, Widget: "number", Min: f64(0), Max: f64(1), Step: 0.05,
		Optional: true,
		Note:     "legacy fallback: only read when light_level is absent",
	},
}

// BuildSchema derives the full schema, verifying each probe-derived model as it
// goes. An error here is fatal at startup by design — a silently wrong selection
// box or tile is worse than a server that refuses to boot.
func BuildSchema(opts animOpts) (*Schema, error) {
	markers, err := buildMarkerSchemas()
	if err != nil {
		return nil, err
	}

	colors := newColorTable()
	tiles := recordAllTiles(colors, opts)

	return &Schema{
		SchemaVersion: SchemaVersion,
		TileSize:      tile.Size,
		Constants: constants{
			PlayerHitbox:  size{W: world.DefaultPlayerHitboxW, H: world.DefaultPlayerHitboxH},
			EnemyHitbox:   size{W: enemy.HitboxW, H: enemy.HitboxH},
			PickupHitbox:  size{W: 12, H: 12},
			GridCols:      GridCols,
			GridRows:      GridRows,
			GridCellTiles: size{W: 20, H: 15},
		},
		Layers: layerSchema{
			TileLayerNames:  []string{BaseLayerName, GroundLayerName},
			ObjectGroupName: MarkersLayerName,
			DefaultBaseGID:  tile.GIDGrass,
		},
		Colors:           colors.list,
		Tiles:            tiles,
		Palette:          buildPalette(),
		Favorites:        defaultFavorites,
		Markers:          markers,
		UnknownMarkerHit: unknownMarkerHitRect(),
		Pickups:          pickupOptions(),
		MapProperties:    mapPropertyFields,
	}, nil
}

// cachedSchema holds the marshaled schema and its etag. The payload is built
// once at startup and is identical for the process lifetime, so the browser can
// cache it hard.
type cachedSchema struct {
	schema *Schema
	body   []byte
	etag   string
}

func newCachedSchema(opts animOpts) (*cachedSchema, error) {
	s, err := BuildSchema(opts)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return &cachedSchema{schema: s, body: body, etag: etagOf(body)}, nil
}

// pickupTiledNames is a small helper for validation.
func pickupTiledNames() map[string]bool {
	out := make(map[string]bool, len(pickup.All))
	for _, k := range pickup.All {
		out[k.TiledName()] = true
	}
	return out
}
