package world

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// MarkerSpawnContext carries per-map state while loading Tiled marker objects.
type MarkerSpawnContext struct {
	CollectedPersistent map[string]struct{}
	Spawned             *bool
}

// MarkerEditorContext carries editor state when creating a new marker object.
type MarkerEditorContext struct {
	TileWidth, TileHeight float64
	ActivePickupTiledName string
}

// MarkerHandler loads and edits one Tiled marker object type.
type MarkerHandler interface {
	Type() string
	SpawnFromTiled(w *World, o *tiled.Object, mapID string, ctx MarkerSpawnContext)
	ObjectHitRect(o tiled.Object) geom.Rect
	InitMarkerObject(o *tiled.Object, wx, wy float64, ctx MarkerEditorContext)
}

type markerRegistry struct {
	byType map[string]MarkerHandler
	order  []string
}

var markers markerRegistry

func registerMarker(h MarkerHandler) {
	if markers.byType == nil {
		markers.byType = make(map[string]MarkerHandler)
	}
	markers.byType[h.Type()] = h
	markers.order = append(markers.order, h.Type())
}

func init() {
	registerMarker(spawnMarker{})
	registerMarker(enemyMarker{})
	registerMarker(pickupMarker{})
	registerMarker(doorMarker{})
	registerMarker(shrineMarker{})
	registerMarker(signMarker{})
}

// MarkerTypeNames returns registered marker types in editor/loader order.
func MarkerTypeNames() []string {
	out := make([]string, len(markers.order))
	copy(out, markers.order)
	return out
}

// MarkerHandlerFor returns the handler for a Tiled object type, or nil.
func MarkerHandlerFor(typ string) MarkerHandler {
	return markers.byType[typ]
}

// MarkerObjectHitRect returns the editor selection rect for a marker object.
func MarkerObjectHitRect(o tiled.Object) geom.Rect {
	if h := MarkerHandlerFor(o.Type); h != nil {
		return h.ObjectHitRect(o)
	}
	return geom.Rect{X: o.X, Y: o.Y, W: 16, H: 16}
}

// InitMarkerObject applies type-specific defaults to a new editor marker object.
func InitMarkerObject(o *tiled.Object, wx, wy float64, ctx MarkerEditorContext) {
	if h := MarkerHandlerFor(o.Type); h != nil {
		h.InitMarkerObject(o, wx, wy, ctx)
	}
}

type spawnMarker struct{}

func (spawnMarker) Type() string { return "spawn" }

func (spawnMarker) SpawnFromTiled(w *World, o *tiled.Object, _ string, ctx MarkerSpawnContext) {
	w.Player.X, w.Player.Y = PlayerTopLeftFromDoorSpawn(o.X, o.Y, DoorSpawnFeet, w.Player.H)
	if ctx.Spawned != nil {
		*ctx.Spawned = true
	}
}

func (spawnMarker) ObjectHitRect(o tiled.Object) geom.Rect {
	return geom.Rect{X: o.X, Y: o.Y - DefaultPlayerHitboxH, W: DefaultPlayerHitboxW, H: DefaultPlayerHitboxH}
}

func (spawnMarker) InitMarkerObject(*tiled.Object, float64, float64, MarkerEditorContext) {}

type enemyMarker struct{}

func (enemyMarker) Type() string { return "enemy" }

func (enemyMarker) SpawnFromTiled(w *World, o *tiled.Object, _ string, _ MarkerSpawnContext) {
	cfg := EnemyConfigFromTiled(o)
	w.SpawnEnemyConfig(o.X, o.Y-defaultEnemyH, cfg)
}

func (enemyMarker) ObjectHitRect(o tiled.Object) geom.Rect {
	return geom.Rect{X: o.X, Y: o.Y - defaultEnemyH, W: defaultEnemyW, H: defaultEnemyH}
}

func (enemyMarker) InitMarkerObject(o *tiled.Object, _ float64, _ float64, _ MarkerEditorContext) {
	o.Properties = DefaultEnemyTiledProperties()
}

type pickupMarker struct{}

func (pickupMarker) Type() string { return "pickup" }

func (pickupMarker) SpawnFromTiled(w *World, o *tiled.Object, mapID string, ctx MarkerSpawnContext) {
	kind := PickupCoin
	if s, ok := tiled.ObjProp(o, "kind"); ok {
		kind = PickupKindFromTiled(s)
	}
	persist, _ := tiled.ObjPropBool(o, "persistent")
	var saveKey string
	if persist {
		if o.ID != 0 {
			saveKey = PersistentPickupSaveKey(mapID, o.ID)
		} else if o.Name != "" {
			saveKey = fmt.Sprintf("%s:name:%s", mapID, o.Name)
		}
	}
	var opened bool
	if saveKey != "" && ctx.CollectedPersistent != nil {
		if _, skip := ctx.CollectedPersistent[saveKey]; skip {
			opened = true
		}
	}
	id := w.SpawnPickup(o.X, o.Y-defaultPickupHitbox, kind, saveKey)
	if opened {
		for i := range w.Pickups {
			if w.Pickups[i].ID == id {
				w.Pickups[i].Opened = true
			}
		}
	}
}

func (pickupMarker) ObjectHitRect(o tiled.Object) geom.Rect {
	return geom.Rect{X: o.X, Y: o.Y - defaultPickupHitbox, W: defaultPickupHitbox, H: defaultPickupHitbox}
}

func (pickupMarker) InitMarkerObject(o *tiled.Object, _ float64, _ float64, ctx MarkerEditorContext) {
	name := ctx.ActivePickupTiledName
	if name == "" {
		name = "coin"
	}
	o.Properties = []tiled.Property{
		{Name: "kind", Type: "string", Value: name},
	}
}

type doorMarker struct{}

func (doorMarker) Type() string { return "door" }

func (doorMarker) SpawnFromTiled(w *World, o *tiled.Object, _ string, _ MarkerSpawnContext) {
	tmap, _ := tiled.ObjProp(o, "target_map")
	sx, _ := tiled.ObjProp(o, "spawn_x")
	sy, _ := tiled.ObjProp(o, "spawn_y")
	fx, _ := strconv.ParseFloat(sx, 64)
	fy, _ := strconv.ParseFloat(sy, 64)
	style := doorSpawnStyleFromTiled(o)
	w.Doors = append(w.Doors, Door{
		ID:         w.allocID(),
		Rect:       geom.Rect{X: o.X, Y: o.Y, W: o.Width, H: o.Height},
		TargetMap:  tmap,
		SpawnX:     fx,
		SpawnY:     fy,
		SpawnStyle: style,
	})
}

func (doorMarker) ObjectHitRect(o tiled.Object) geom.Rect {
	w, h := o.Width, o.Height
	if w <= 0 {
		w = 16
	}
	if h <= 0 {
		h = 16
	}
	return geom.Rect{X: o.X, Y: o.Y, W: w, H: h}
}

func (doorMarker) InitMarkerObject(o *tiled.Object, wx, wy float64, ctx MarkerEditorContext) {
	tw := ctx.TileWidth
	if tw <= 0 {
		tw = tile.Size
	}
	th := ctx.TileHeight
	if th <= 0 {
		th = tile.Size
	}
	o.X = float64(int(wx/tw) * int(tw))
	o.Y = float64(int(wy/th) * int(th))
	o.Width = 16
	o.Height = 32
	o.Properties = []tiled.Property{
		{Name: "target_map", Type: "string", Value: "field1"},
		{Name: "spawn_x", Type: "string", Value: strconv.Itoa(int(o.X))},
		{Name: "spawn_y", Type: "string", Value: strconv.Itoa(int(o.Y))},
	}
}

type shrineMarker struct{}

func (shrineMarker) Type() string { return "shrine" }

func (shrineMarker) SpawnFromTiled(w *World, o *tiled.Object, _ string, _ MarkerSpawnContext) {
	w.Shrines = append(w.Shrines, Shrine{
		ID:      w.allocID(),
		TiledID: o.ID,
		Rect:    geom.Rect{X: o.X, Y: o.Y, W: o.Width, H: o.Height},
	})
}

func (shrineMarker) ObjectHitRect(o tiled.Object) geom.Rect {
	w, h := o.Width, o.Height
	if w <= 0 {
		w = 16
	}
	if h <= 0 {
		h = 16
	}
	return geom.Rect{X: o.X, Y: o.Y, W: w, H: h}
}

func (shrineMarker) InitMarkerObject(o *tiled.Object, _ float64, _ float64, _ MarkerEditorContext) {
	o.Width = 16
	o.Height = 16
}

func doorSpawnStyleFromTiled(o *tiled.Object) DoorSpawnStyle {
	style := DoorSpawnFeet
	if a, ok := tiled.ObjProp(o, "spawn_anchor"); ok {
		switch strings.ToLower(strings.TrimSpace(a)) {
		case "topleft", "top_left", "top-left", "origin":
			style = DoorSpawnTopLeft
		}
	}
	return style
}

type signMarker struct{}

func (signMarker) Type() string { return "sign" }

func (signMarker) SpawnFromTiled(w *World, o *tiled.Object, _ string, _ MarkerSpawnContext) {
	txt, _ := tiled.ObjProp(o, "text")
	if txt == "" {
		txt = "A wooden sign..."
	}

	tx := int(math.Round(o.X / float64(w.TileW)))
	ty := int(math.Round(o.Y / float64(w.TileH)))

	snappedX := float64(tx * w.TileW)
	snappedY := float64(ty * w.TileH)

	w.Signs = append(w.Signs, Sign{
		ID:   w.allocID(),
		Rect: geom.Rect{X: snappedX, Y: snappedY, W: float64(w.TileW), H: float64(w.TileH)},
		Text: txt,
	})

	if tx >= 0 && tx < w.MapW && ty >= 0 && ty < w.MapH {
		idx := ty*w.MapW + tx
		w.Tiles[idx] = tile.GIDSign
	}
}

func (signMarker) ObjectHitRect(o tiled.Object) geom.Rect {
	w, h := o.Width, o.Height
	if w <= 0 {
		w = 16
	}
	if h <= 0 {
		h = 16
	}
	return geom.Rect{X: o.X, Y: o.Y, W: w, H: h}
}

func (signMarker) InitMarkerObject(o *tiled.Object, wx, wy float64, ctx MarkerEditorContext) {
	tw := ctx.TileWidth
	if tw <= 0 {
		tw = tile.Size
	}
	th := ctx.TileHeight
	if th <= 0 {
		th = tile.Size
	}
	o.X = float64(int(wx/tw) * int(tw))
	o.Y = float64(int(wy/th) * int(th))
	o.Width = 16
	o.Height = 16
	o.Properties = []tiled.Property{
		{Name: "text", Type: "string", Value: "A wooden sign..."},
	}
}
