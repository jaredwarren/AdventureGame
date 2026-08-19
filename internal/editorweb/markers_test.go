package editorweb

import (
	"encoding/json"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
	"github.com/jaredwarren/game-test/internal/world/enemy"
	"github.com/jaredwarren/game-test/internal/world/pickup"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// TestHitRectModelMatchesGo is the test that makes probing safe. Every derived
// model must reproduce world.MarkerObjectHitRect exactly across a dense input
// grid, including the negatives and degenerate sizes that trip the "w <= 0
// means 16" clamp.
func TestHitRectModelMatchesGo(t *testing.T) {
	types := append(world.MarkerTypeNames(), "\x00not-a-marker-type")
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			m := probeHitRect(typ)
			if err := verifyHitRectModel(typ, m); err != nil {
				t.Error(err)
			}
		})
	}
}

// TestHitRectModelRejectsNonAffine proves the verification actually bites: a
// model that is wrong must be reported, not quietly accepted. Without this,
// TestHitRectModelMatchesGo could pass against a broken verifier.
func TestHitRectModelRejectsNonAffine(t *testing.T) {
	m := probeHitRect("enemy")
	m.Y.K += 12 // the exact "selection box off by one hitbox height" bug
	if err := verifyHitRectModel("enemy", m); err == nil {
		t.Fatal("verifyHitRectModel accepted a model with a 12px Y offset; the startup guard is not protecting anything")
	}
}

// TestHitRectModelKnownValues documents the shapes rather than only asserting
// self-consistency, so a reviewer can see what the probe recovered.
func TestHitRectModelKnownValues(t *testing.T) {
	tests := []struct {
		typ        string
		obj        tiled.Object
		x, y, w, h float64
		what       string
	}{
		// Feet-anchored: the object's y is the BOTTOM of the box.
		{"spawn", tiled.Object{X: 100, Y: 200}, 100, 200 - world.DefaultPlayerHitboxH, world.DefaultPlayerHitboxW, world.DefaultPlayerHitboxH, "player hitbox above the feet"},
		{"enemy", tiled.Object{X: 100, Y: 200}, 100, 200 - enemy.HitboxH, enemy.HitboxW, enemy.HitboxH, "enemy hitbox above the feet"},
		{"pickup", tiled.Object{X: 100, Y: 200}, 100, 188, 12, 12, "pickup hitbox above the feet"},
		// Object-sized, with the degenerate clamp.
		{"door", tiled.Object{X: 32, Y: 48, Width: 16, Height: 32}, 32, 48, 16, 32, "door uses its own size"},
		{"door", tiled.Object{X: 32, Y: 48}, 32, 48, 16, 16, "zero-size door clamps to 16x16"},
		{"shrine", tiled.Object{X: 8, Y: 8, Width: 16, Height: 16}, 8, 8, 16, 16, "shrine uses its own size"},
		{"sign", tiled.Object{X: 0, Y: 0, Width: -4, Height: 0}, 0, 0, 16, 16, "negative size also clamps"},
		// Unknown types fall back to a 16x16 box at the origin.
		{"\x00nope", tiled.Object{X: 5, Y: 7}, 5, 7, 16, 16, "unknown type fallback"},
	}

	for _, tc := range tests {
		o := tc.obj
		o.Type = tc.typ
		m := probeHitRect(tc.typ)
		x, y, w, h := m.eval(o)
		if !nearlyEqual(x, tc.x) || !nearlyEqual(y, tc.y) || !nearlyEqual(w, tc.w) || !nearlyEqual(h, tc.h) {
			t.Errorf("%s (%s): model gave {%v %v %v %v}, want {%v %v %v %v}",
				tc.typ, tc.what, x, y, w, h, tc.x, tc.y, tc.w, tc.h)
		}
	}
}

// TestMarkerTemplatesMatchInitMarkerObject proves the shipped templates are the
// real InitMarkerObject output rather than a hand-copied approximation.
func TestMarkerTemplatesMatchInitMarkerObject(t *testing.T) {
	ctx := world.MarkerEditorContext{
		TileWidth:             tile.Size,
		TileHeight:            tile.Size,
		ActivePickupTiledName: pickup.All[0].TiledName(),
	}
	for _, typ := range world.MarkerTypeNames() {
		tmpl, _ := probeTemplate(typ)

		want := tiled.Object{Type: typ}
		world.InitMarkerObject(&want, 0, 0, ctx)

		if !reflect.DeepEqual(tmpl, want) {
			t.Errorf("%s: template %+v != InitMarkerObject output %+v", typ, tmpl, want)
		}
	}
}

// TestNewMarkerObjectMatchesInitMarkerObject proves the POST /api/marker path
// delegates rather than reimplementing grid snapping. Probing at a fractional
// coordinate is the point: door and sign snap, the rest do not.
func TestNewMarkerObjectMatchesInitMarkerObject(t *testing.T) {
	const wx, wy = 133.5, 97.25
	ctx := world.MarkerEditorContext{
		TileWidth:             tile.Size,
		TileHeight:            tile.Size,
		ActivePickupTiledName: "heart",
	}
	for _, typ := range world.MarkerTypeNames() {
		got, err := NewMarkerObject(typ, 42, "thing", wx, wy, "heart")
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		want := tiled.Object{ID: 42, Name: "thing", Type: typ, X: wx, Y: wy}
		world.InitMarkerObject(&want, wx, wy, ctx)

		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: NewMarkerObject gave %+v, want %+v", typ, got, want)
		}
	}
	if _, err := NewMarkerObject("nonsense", 1, "", 0, 0, ""); err == nil {
		t.Error("NewMarkerObject accepted an unregistered marker type")
	}
}

// TestSnapsToGridIsDerived checks the snapping flag is probed, not hardcoded, and
// that it reports what the handlers actually do today.
func TestSnapsToGridIsDerived(t *testing.T) {
	want := map[string]bool{
		"spawn": false, "enemy": false, "pickup": false,
		"door": true, "shrine": false, "sign": true,
	}
	for _, typ := range world.MarkerTypeNames() {
		_, got := probeTemplate(typ)
		if got != want[typ] {
			t.Errorf("%s: snapsToGrid=%v, want %v", typ, got, want[typ])
		}
	}
}

// TestMarkerFieldsCoverGameReadProperties is the drift guard for the property
// panel. Each expected key below is a property the game genuinely reads; the
// comment names the reader. If a handler starts reading a new property and no
// field is exposed for it, that property becomes uneditable in the UI and this
// test says so.
func TestMarkerFieldsCoverGameReadProperties(t *testing.T) {
	want := map[string][]string{
		// enemy.ConfigFromTiled
		"enemy": {"hp", "speed", "aggro", "damage", "is_boss", "is_armored_knight", "armor_hp"},
		// pickupMarker.SpawnFromTiled
		"pickup": {"kind", "persistent", "chest"},
		// doorMarker.SpawnFromTiled + doorSpawnStyleFromTiled
		"door": {"target_map", "spawn_x", "spawn_y", "spawn_anchor"},
		// signMarker.SpawnFromTiled
		"sign": {"text"},
		// shrineMarker.SpawnFromTiled reads no properties (it keys off o.ID).
		"shrine": {},
		// spawnMarker.SpawnFromTiled reads no properties.
		"spawn": {},
	}

	schemas, err := buildMarkerSchemas()
	if err != nil {
		t.Fatalf("buildMarkerSchemas: %v", err)
	}
	for _, ms := range schemas {
		var got []string
		for _, f := range ms.Fields {
			got = append(got, f.Name)
		}
		for _, name := range want[ms.Type] {
			if !slices.Contains(got, name) {
				t.Errorf("%s: no editable field for property %q, which the game reads", ms.Type, name)
			}
		}
		for _, name := range got {
			if !slices.Contains(want[ms.Type], name) {
				t.Errorf("%s: exposes field %q that the game never reads — either the game changed (update this table) or the field is dead", ms.Type, name)
			}
		}
	}
}

// TestEnemyFieldsAreDerivedNotHardcoded proves the enemy form comes from
// enemy.TiledProperties, so retuning DefaultConfig changes the editor defaults.
func TestEnemyFieldsAreDerivedNotHardcoded(t *testing.T) {
	tmpl, _ := probeTemplate("enemy")
	fields := deriveFields("enemy", tmpl)
	props := enemy.TiledProperties(enemy.DefaultConfig())

	if len(fields) != len(props) {
		t.Fatalf("%d enemy fields, enemy.TiledProperties has %d", len(fields), len(props))
	}
	for i, p := range props {
		f := fields[i]
		if f.Name != p.Name {
			t.Errorf("field %d: name %q != %q", i, f.Name, p.Name)
		}
		if f.TiledType != p.Type {
			t.Errorf("%s: tiledType %q != %q", f.Name, f.TiledType, p.Type)
		}
		if !reflect.DeepEqual(f.Default, p.Value) {
			t.Errorf("%s: default %v != %v", f.Name, f.Default, p.Value)
		}
	}
}

// TestPickupEnumMatchesRegistry keeps the pickup dropdown in step with the
// registry, in order, so registering a Kind in Go needs no JS or schema edit.
func TestPickupEnumMatchesRegistry(t *testing.T) {
	opts := pickupOptions()
	if len(opts) != len(pickup.All) {
		t.Fatalf("%d options, registry has %d kinds", len(opts), len(pickup.All))
	}
	for i, k := range pickup.All {
		if opts[i].Value != k.TiledName() || opts[i].Label != k.EditorLabel() || opts[i].Sprite != k.ID() {
			t.Errorf("option %d = %+v, want value=%q label=%q sprite=%d",
				i, opts[i], k.TiledName(), k.EditorLabel(), k.ID())
		}
	}
}

// TestDoorSpawnPropsStayStrings guards a subtle round-trip hazard: door
// spawn_x/spawn_y are Tiled strings parsed with ParseFloat. If the schema
// advertised them as numbers the client would write numbers back and change the
// on-disk type.
func TestDoorSpawnPropsStayStrings(t *testing.T) {
	tmpl, _ := probeTemplate("door")
	for _, f := range deriveFields("door", tmpl) {
		if f.Name != "spawn_x" && f.Name != "spawn_y" {
			continue
		}
		if f.TiledType != "string" {
			t.Errorf("%s: tiledType %q, want \"string\" (the game parses it with strconv.ParseFloat)", f.Name, f.TiledType)
		}
		if !f.Numeric {
			t.Errorf("%s: should be flagged Numeric so the form shows a number input", f.Name)
		}
	}
}

// TestSizableIsDerived checks resize handles are offered exactly for the types
// whose on-disk width/height actually drive their hit rect.
func TestSizableIsDerived(t *testing.T) {
	want := map[string]bool{
		"spawn": false, "enemy": false, "pickup": false,
		"door": true, "shrine": true, "sign": true,
	}
	schemas, err := buildMarkerSchemas()
	if err != nil {
		t.Fatalf("buildMarkerSchemas: %v", err)
	}
	for _, ms := range schemas {
		if ms.Sizable != want[ms.Type] {
			t.Errorf("%s: sizable=%v, want %v", ms.Type, ms.Sizable, want[ms.Type])
		}
	}
}

// TestMarkerSchemaJSONIsStable guards ETag caching: a map[...] in the payload
// would make the bytes nondeterministic across builds.
func TestMarkerSchemaJSONIsStable(t *testing.T) {
	a, err := buildMarkerSchemas()
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildMarkerSchemas()
	if err != nil {
		t.Fatal(err)
	}
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Error("marker schema JSON is not deterministic")
	}
}

// TestUnknownMarkerHitRectFallback documents the model shipped for object types
// this build does not know about, so Tiled-authored objects still select.
func TestUnknownMarkerHitRectFallback(t *testing.T) {
	m := unknownMarkerHitRect()
	o := tiled.Object{Type: "future-thing", X: 3, Y: 9, Width: 99, Height: 99}
	x, y, w, h := m.eval(o)
	if x != 3 || y != 9 || w != 16 || h != 16 {
		t.Errorf("unknown fallback gave {%v %v %v %v}, want {3 9 16 16}", x, y, w, h)
	}
	if math.Abs(m.W.Cw) > 1e-9 {
		t.Error("unknown fallback must not track the object's width")
	}
}
