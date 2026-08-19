package editorweb

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/enemy"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// Map validation.
//
// Most of these check for things the game's loader tolerates *silently*: a tile
// layer with the wrong length is dropped, an unregistered GID becomes an
// invisible wall, an unknown pickup kind quietly turns into a coin, an enemy
// with hp <= 0 keeps the default. Each of those is a bug you find at playtest
// time, or never. Naming them here is most of the value of this panel.

type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
	SeverityInfo  Severity = "info"
)

// Issue is one validation finding.
type Issue struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	// ObjectID lets the UI select the offending marker.
	ObjectID int `json:"objectId,omitempty"`
	// TileX/TileY let the UI pan to the offending tile.
	TileX *int `json:"tileX,omitempty"`
	TileY *int `json:"tileY,omitempty"`
	// Field names the property at fault, for the form to highlight.
	Field string `json:"field,omitempty"`
}

// ValidateOpts tunes severity for checks that are project policy rather than
// correctness.
type ValidateOpts struct {
	// Strict promotes missing door targets from a warning to an error.
	Strict bool
	// ResolveTarget loads another map by id, for cross-map door checks. Nil
	// disables them, which keeps Validate unit-testable without a store.
	ResolveTarget func(id string) (*tiled.Map, bool)
}

type issues struct {
	list  []Issue
	opts  ValidateOpts
	group MapGroup
}

func (is *issues) add(sev Severity, code, format string, args ...any) *Issue {
	is.list = append(is.list, Issue{Severity: sev, Code: code, Message: fmt.Sprintf(format, args...)})
	return &is.list[len(is.list)-1]
}

func (is *issues) errf(code, format string, args ...any) *Issue {
	return is.add(SeverityError, code, format, args...)
}

func (is *issues) warnf(code, format string, args ...any) *Issue {
	return is.add(SeverityWarn, code, format, args...)
}

// doorTargetSeverity is warn by default because a work-in-progress map often
// points at a room that does not exist yet.
func (is *issues) doorTargetSeverity() Severity {
	if is.opts.Strict {
		return SeverityError
	}
	return SeverityWarn
}

// Validate checks a map the way the game's loader would, plus the authoring
// invariants the loader is too forgiving about.
func Validate(m *tiled.Map, id string, opts ValidateOpts) []Issue {
	is := &issues{opts: opts, group: GroupOf(id)}
	if m == nil {
		is.errf("parse_failed", "map could not be parsed")
		return is.list
	}

	validateStructure(is, m)
	validateTiles(is, m)
	validateObjects(is, m, id)
	validateMapProperties(is, m)

	// The single highest-value check: run the game's own loader. If this fails,
	// the map will not load in game, whatever else passed.
	if _, err := world.BuildFromTiled(m, id, progression.DefaultStats(), nil); err != nil {
		is.errf("build_failed", "the game's loader rejects this map: %v", err)
	}

	sortIssues(is.list)
	return is.list
}

func validateStructure(is *issues, m *tiled.Map) {
	if m.TileWidth != tile.Size || m.TileHeight != tile.Size {
		is.errf("bad_tile_size", "tiles are %dx%d; the game requires %dx%d",
			m.TileWidth, m.TileHeight, tile.Size, tile.Size)
	}

	var tileLayers, objectGroups int
	var hasGround bool
	for _, l := range m.Layers {
		switch l.Type {
		case "tilelayer":
			tileLayers++
			if l.Name == GroundLayerName {
				hasGround = true
			}
			if l.Name != BaseLayerName && l.Name != GroundLayerName {
				is.warnf("wrong_layer_name", "tile layer %q is not %q or %q; the game reads layers positionally, so this still works but is easy to misread",
					l.Name, BaseLayerName, GroundLayerName)
			}
			// BuildFromTiled silently *drops* layers whose data length does not
			// match, so this catches a class the loader hides entirely.
			if want := m.Width * m.Height; len(l.Data) != want {
				is.errf("bad_layer_len", "tile layer %q has %d tiles but the map is %dx%d (%d); the loader would silently discard this layer",
					l.Name, len(l.Data), m.Width, m.Height, want)
			}
		case "objectgroup":
			objectGroups++
		}
	}
	if !hasGround {
		is.errf("no_ground_layer", "no tile layer named %q", GroundLayerName)
	}
	if tileLayers > 2 {
		is.warnf("wrong_layer_name", "%d tile layers; the game and editor assume at most %q plus %q", tileLayers, BaseLayerName, GroundLayerName)
	}
	if m.ObjectGroupLayer(MarkersLayerName) == nil {
		is.warnf("no_markers_group", "no object group named %q, so this map can have no markers", MarkersLayerName)
	}
	if objectGroups > 1 {
		is.warnf("extra_objectgroup", "%d object groups; only %q is read", objectGroups, MarkersLayerName)
	}
}

func validateTiles(is *issues, m *tiled.Map) {
	// Report each unknown GID once rather than once per cell.
	reported := map[int]bool{}
	for _, l := range m.Layers {
		if l.Type != "tilelayer" || len(l.Data) != m.Width*m.Height {
			continue
		}
		for i, gid := range l.Data {
			if gid == tile.GIDEmpty || reported[gid] {
				continue
			}
			if tile.DefOf(gid).Name == "unknown" {
				reported[gid] = true
				tx, ty := i%m.Width, i/m.Width
				iss := is.errf("unknown_gid", "layer %q uses gid %d, which is not in the tile registry; the game draws it as an invisible solid wall", l.Name, gid)
				iss.TileX, iss.TileY = &tx, &ty
			}
		}
	}
}

func validateObjects(is *issues, m *tiled.Map, id string) {
	seenID := map[int]bool{}
	var spawns int
	pickupKinds := pickupTiledNames()

	group := m.ObjectGroupLayer(MarkersLayerName)
	if group == nil {
		return
	}

	for i := range group.Objects {
		o := group.Objects[i]

		// Duplicate ids corrupt the persistence keys "<mapID>:<objectID>" and
		// "<mapID>:shrine:<objectID>", so two chests share one collected flag.
		if o.ID == 0 {
			is.warnf("zero_object_id", "%s marker has no object id, so it cannot be tracked across saves", o.Type)
		} else if seenID[o.ID] {
			is.errf("duplicate_object_id", "object id %d is used more than once; persistent pickups and shrines key their save state on it, so these entries would share one collected flag", o.ID).ObjectID = o.ID
		}
		seenID[o.ID] = true

		if world.MarkerHandlerFor(o.Type) == nil {
			validateNonMarkerObject(is, &o)
			continue
		}

		if r := world.MarkerObjectHitRect(o); outOfBounds(r.X, r.Y, r.W, r.H, m) {
			is.warnf("marker_out_of_bounds", "%s marker %d sits outside the map", o.Type, o.ID).ObjectID = o.ID
		}

		switch o.Type {
		case "spawn":
			spawns++
		case "enemy":
			validateEnemy(is, &o)
		case "pickup":
			validatePickup(is, &o, pickupKinds)
		case "door":
			validateDoor(is, &o, m)
		case "sign":
			validateSign(is, &o, m)
		}
	}

	switch spawns {
	case 0:
		// Dungeon room prefabs are stamped into a generated map by
		// internal/dungeon/stitcher.go and never loaded standalone, so a spawn
		// marker would be meaningless in them.
		if is.group != GroupRoom {
			is.warnf("no_spawn_marker", "no spawn marker; the loader falls back to tile (2,2), which may be inside a wall")
		}
	case 1:
	default:
		is.warnf("multiple_spawn_markers", "%d spawn markers; the loader applies each in turn, so the last one wins", spawns)
	}

	if m.NextObjectID <= maxObjectID(m) {
		is.warnf("stale_next_object_id", "nextobjectid (%d) is not above the highest object id (%d); it is repaired on load, but new objects authored elsewhere could collide",
			m.NextObjectID, maxObjectID(m))
	}
}

// dungeonSocketDirs are the socket values internal/dungeon/stitcher.go accepts.
var dungeonSocketDirs = map[string]bool{"N": true, "S": true, "W": true, "E": true}

// validateNonMarkerObject handles objects world's marker registry does not own.
//
// The dungeon room prefabs under assets/maps/rooms carry type "marker" objects
// with a "socket" property; the stitcher reads those to join rooms and excludes
// them when stamping, so they are correct rather than unknown. Everything else
// with an unregistered type is silently dropped by BuildFromTiled, which is
// worth saying out loud.
func validateNonMarkerObject(is *issues, o *tiled.Object) {
	socket, isSocket := tiled.ObjProp(o, "socket")
	if !isSocket {
		is.warnf("unknown_marker_type", "object %d has type %q, which the loader ignores silently", o.ID, o.Type).ObjectID = o.ID
		return
	}
	if !dungeonSocketDirs[strings.ToUpper(strings.TrimSpace(socket))] {
		iss := is.warnf("bad_dungeon_socket", "object %d is a dungeon socket with direction %q; the stitcher only understands N, S, W and E", o.ID, socket)
		iss.ObjectID, iss.Field = o.ID, "socket"
	}
}

// validateEnemy reports properties ConfigFromTiled silently discards. Its
// guards are all "ok && v > 0", so an author who sets hp to 0 to make a
// one-shot enemy gets the default 3 instead, with no indication why.
func validateEnemy(is *issues, o *tiled.Object) {
	check := func(name string, positive bool) {
		if !positive {
			iss := is.warnf("enemy_prop_ignored", "enemy %d has %s <= 0, which enemy.ConfigFromTiled discards in favor of the default", o.ID, name)
			iss.ObjectID, iss.Field = o.ID, name
		}
	}
	if v, ok := tiled.ObjPropInt(o, "hp"); ok {
		check("hp", v > 0)
	}
	if v, ok := tiled.ObjPropFloat(o, "speed"); ok {
		check("speed", v > 0)
	}
	if v, ok := tiled.ObjPropFloat(o, "aggro"); ok {
		check("aggro", v > 0)
	}
	if v, ok := tiled.ObjPropInt(o, "damage"); ok {
		check("damage", v > 0)
	}
	// Cross-check the parsed result so a future silent-discard rule is caught.
	if cfg := enemy.ConfigFromTiled(o); cfg.IsArmoredKnight && cfg.ArmorHealth <= 0 {
		is.warnf("enemy_prop_ignored", "enemy %d is an armored knight with no armor hp", o.ID).ObjectID = o.ID
	}
}

func validatePickup(is *issues, o *tiled.Object, known map[string]bool) {
	kind, ok := tiled.ObjProp(o, "kind")
	if !ok {
		is.warnf("unknown_pickup_kind", "pickup %d has no kind property; the loader defaults it to a coin", o.ID).ObjectID = o.ID
		return
	}
	if !known[kind] {
		iss := is.warnf("unknown_pickup_kind", "pickup %d has kind %q, which is not registered; pickup.FromTiled silently substitutes a coin", o.ID, kind)
		iss.ObjectID, iss.Field = o.ID, "kind"
	}
	persistent, _ := tiled.ObjPropBool(o, "persistent")
	chest, _ := tiled.ObjPropBool(o, "chest")
	if (persistent || chest) && o.ID == 0 && o.Name == "" {
		is.warnf("zero_object_id", "pickup is persistent but has neither an object id nor a name, so its collected state cannot be saved")
	}
}

func validateDoor(is *issues, o *tiled.Object, m *tiled.Map) {
	target, _ := tiled.ObjProp(o, "target_map")
	if strings.TrimSpace(target) == "" {
		iss := is.warnf("door_target_missing", "door %d has no target_map", o.ID)
		iss.ObjectID, iss.Field = o.ID, "target_map"
		return
	}

	sx, sxOK := tiled.ObjProp(o, "spawn_x")
	sy, syOK := tiled.ObjProp(o, "spawn_y")
	fx, fy := 0.0, 0.0
	if sxOK {
		v, err := strconv.ParseFloat(sx, 64)
		if err != nil {
			iss := is.warnf("door_spawn_unparseable", "door %d has spawn_x=%q, which is not a number; the game parses it with strconv.ParseFloat and silently gets 0", o.ID, sx)
			iss.ObjectID, iss.Field = o.ID, "spawn_x"
		}
		fx = v
	}
	if syOK {
		v, err := strconv.ParseFloat(sy, 64)
		if err != nil {
			iss := is.warnf("door_spawn_unparseable", "door %d has spawn_y=%q, which is not a number; the game parses it with strconv.ParseFloat and silently gets 0", o.ID, sy)
			iss.ObjectID, iss.Field = o.ID, "spawn_y"
		}
		fy = v
	}

	if anchor, ok := tiled.ObjProp(o, "spawn_anchor"); ok {
		switch strings.ToLower(strings.TrimSpace(anchor)) {
		case "", "feet", "topleft", "top_left", "top-left", "origin":
		default:
			iss := is.warnf("unknown_spawn_anchor", "door %d has spawn_anchor=%q; only feet (default) and topleft are recognized, so this falls back to feet", o.ID, anchor)
			iss.ObjectID, iss.Field = o.ID, "spawn_anchor"
		}
	}

	if is.opts.ResolveTarget == nil {
		return
	}
	tm, ok := is.opts.ResolveTarget(target)
	if !ok {
		iss := is.add(is.doorTargetSeverity(), "door_target_missing", "door %d points at map %q, which does not exist", o.ID, target)
		iss.ObjectID, iss.Field = o.ID, "target_map"
		return
	}
	if outOfBounds(fx, fy, 0, 0, tm) {
		iss := is.warnf("door_spawn_out_of_bounds", "door %d spawns at (%g, %g) in %q, which is outside that map's %dx%d tiles", o.ID, fx, fy, target, tm.Width, tm.Height)
		iss.ObjectID = o.ID
		return
	}
	// "The door drops you inside a wall" is the most common door-graph bug and
	// is invisible until you walk through it.
	if solidAtPixel(tm, fx, fy) {
		iss := is.warnf("door_spawn_in_solid", "door %d spawns at (%g, %g) in %q, which is a solid tile", o.ID, fx, fy, target)
		iss.ObjectID = o.ID
	}
}

// validateSign catches a genuinely surprising behavior: signMarker.SpawnFromTiled
// ROUNDS the sign's position to the nearest tile and overwrites that tile's gid
// with GIDSign at load time. A sign placed off-grid therefore stamps over a
// different tile than the author sees in the editor.
func validateSign(is *issues, o *tiled.Object, m *tiled.Map) {
	if math.Mod(o.X, tile.Size) == 0 && math.Mod(o.Y, tile.Size) == 0 {
		return
	}
	tx := int(math.Round(o.X / tile.Size))
	ty := int(math.Round(o.Y / tile.Size))
	iss := is.warnf("sign_not_tile_aligned", "sign %d is at (%g, %g), off the tile grid; the loader rounds it to tile (%d, %d) and overwrites that tile with the sign graphic", o.ID, o.X, o.Y, tx, ty)
	iss.ObjectID = o.ID
	iss.TileX, iss.TileY = &tx, &ty
}

func validateMapProperties(is *issues, m *tiled.Map) {
	light, hasLight := tiled.MapPropFloat(m, "light_level")
	ambient, hasAmbient := tiled.MapPropFloat(m, "ambient_light")

	if hasLight && (light < 0 || light > 1) {
		is.warnf("light_level_out_of_range", "light_level is %g; the game expects 0..1", light).Field = "light_level"
	}
	if hasAmbient && (ambient < 0 || ambient > 1) {
		is.warnf("light_level_out_of_range", "ambient_light is %g; the game expects 0..1", ambient).Field = "ambient_light"
	}
	// BuildFromTiled prefers light_level and never looks at ambient_light when
	// it is present, so the second value is dead weight that reads as active.
	if hasLight && hasAmbient {
		is.warnf("dead_ambient_light", "both light_level and ambient_light are set; the loader only reads light_level, so ambient_light has no effect").Field = "ambient_light"
	}
}

// ---- helpers ----

func outOfBounds(x, y, w, h float64, m *tiled.Map) bool {
	maxX := float64(m.Width * tile.Size)
	maxY := float64(m.Height * tile.Size)
	return x < 0 || y < 0 || x+w > maxX || y+h > maxY
}

// solidAtPixel reports whether the topmost non-empty tile at a pixel is solid.
func solidAtPixel(m *tiled.Map, px, py float64) bool {
	tx, ty := int(px)/tile.Size, int(py)/tile.Size
	if tx < 0 || ty < 0 || tx >= m.Width || ty >= m.Height {
		return false
	}
	idx := ty*m.Width + tx
	// Walk top-down: the highest layer with a non-empty gid wins, matching how
	// the renderer and collision read stacked layers.
	for i := len(m.Layers) - 1; i >= 0; i-- {
		l := m.Layers[i]
		if l.Type != "tilelayer" || len(l.Data) != m.Width*m.Height {
			continue
		}
		if gid := l.Data[idx]; gid != tile.GIDEmpty {
			return tile.DefOf(gid).Solid()
		}
	}
	return false
}

func maxObjectID(m *tiled.Map) int {
	max := 0
	for _, l := range m.Layers {
		for _, o := range l.Objects {
			if o.ID > max {
				max = o.ID
			}
		}
	}
	return max
}

// sortIssues puts errors first, then groups by code, so the panel reads well.
func sortIssues(list []Issue) {
	rank := map[Severity]int{SeverityError: 0, SeverityWarn: 1, SeverityInfo: 2}
	sort.SliceStable(list, func(i, j int) bool {
		if rank[list[i].Severity] != rank[list[j].Severity] {
			return rank[list[i].Severity] < rank[list[j].Severity]
		}
		return list[i].Code < list[j].Code
	})
}

func filterSeverity(list []Issue, sev Severity) []Issue {
	var out []Issue
	for _, i := range list {
		if i.Severity == sev {
			out = append(out, i)
		}
	}
	return out
}

// CountBySeverity summarizes a list for badges and exit codes.
func CountBySeverity(list []Issue) (errs, warns int) {
	for _, i := range list {
		switch i.Severity {
		case SeverityError:
			errs++
		case SeverityWarn:
			warns++
		}
	}
	return
}
