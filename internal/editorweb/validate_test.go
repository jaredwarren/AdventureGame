package editorweb

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jaredwarren/game-test/internal/tiled"
)

func codes(list []Issue) []string {
	var out []string
	for _, i := range list {
		out = append(out, i.Code)
	}
	return out
}

func hasCode(list []Issue, code string) bool { return slices.Contains(codes(list), code) }

// validMap is a minimal map that must produce no issues at all, so every test
// below is asserting the check it names rather than incidental noise.
func validMap() *tiled.Map {
	const w, h = 4, 3
	ground := make([]int, w*h)
	for i := range ground {
		ground[i] = 1 // grass
	}
	return &tiled.Map{
		Width: w, Height: h, TileWidth: 16, TileHeight: 16,
		Type: "map", Version: "1.10", Orientation: "orthogonal", RenderOrder: "right-down",
		NextLayerID: 3, NextObjectID: 2,
		Tilesets: json.RawMessage("[]"),
		Layers: []tiled.Layer{
			{ID: 1, Type: "tilelayer", Name: "ground", Visible: true, Opacity: 1,
				Width: w, Height: h, Data: ground},
			{ID: 2, Type: "objectgroup", Name: "markers", Visible: true, Opacity: 1,
				Objects: []tiled.Object{{ID: 1, Type: "spawn", X: 16, Y: 16}}},
		},
	}
}

// addObjects appends markers to a fixture and keeps nextobjectid consistent, so
// tests assert the check they name rather than tripping stale_next_object_id.
func addObjects(m *tiled.Map, objs ...tiled.Object) {
	g := m.ObjectGroupLayer("markers")
	g.Objects = append(g.Objects, objs...)
	for _, o := range objs {
		if o.ID >= m.NextObjectID {
			m.NextObjectID = o.ID + 1
		}
	}
}

func TestValidateAcceptsAGoodMap(t *testing.T) {
	if got := Validate(validMap(), "test", ValidateOpts{}); len(got) != 0 {
		t.Errorf("a valid map produced issues: %v", got)
	}
}

// TestValidateChecks drives each check through a targeted mutation, so a check
// that silently stops working is caught.
func TestValidateChecks(t *testing.T) {
	tests := []struct {
		name string
		code string
		sev  Severity
		mut  func(m *tiled.Map)
	}{
		{
			name: "duplicate object ids corrupt persistence keys",
			code: "duplicate_object_id", sev: SeverityError,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 1, Type: "shrine", X: 0, Y: 0, Width: 16, Height: 16})
			},
		},
		{
			// BuildFromTiled silently drops a layer whose length is wrong, so
			// without this check the tiles just vanish in game.
			name: "wrong layer length",
			code: "bad_layer_len", sev: SeverityError,
			mut: func(m *tiled.Map) { m.Layers[0].Data = m.Layers[0].Data[:2] },
		},
		{
			name: "missing ground layer",
			code: "no_ground_layer", sev: SeverityError,
			mut: func(m *tiled.Map) { m.Layers[0].Name = "floor" },
		},
		{
			name: "wrong tile size",
			code: "bad_tile_size", sev: SeverityError,
			mut: func(m *tiled.Map) { m.TileWidth, m.TileHeight = 32, 32 },
		},
		{
			// DefOf returns a solid grey "unknown" tile instead of failing, so a
			// bad gid becomes an invisible wall.
			name: "unregistered gid",
			code: "unknown_gid", sev: SeverityError,
			mut: func(m *tiled.Map) { m.Layers[0].Data[0] = 9999 },
		},
		{
			name: "no spawn marker",
			code: "no_spawn_marker", sev: SeverityWarn,
			mut: func(m *tiled.Map) { m.ObjectGroupLayer("markers").Objects = nil },
		},
		{
			name: "two spawn markers",
			code: "multiple_spawn_markers", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "spawn", X: 32, Y: 32})
			},
		},
		{
			name: "unknown marker type is dropped by the loader",
			code: "unknown_marker_type", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "teleporter", X: 0, Y: 16})
			},
		},
		{
			// pickup.FromTiled silently substitutes a coin.
			name: "unknown pickup kind",
			code: "unknown_pickup_kind", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "pickup", X: 0, Y: 16,
					Properties: []tiled.Property{{Name: "kind", Type: "string", Value: "sword"}}})
			},
		},
		{
			// ConfigFromTiled guards on "ok && v > 0", so hp: 0 silently keeps
			// the default rather than making a one-hit enemy.
			name: "enemy hp <= 0 is silently discarded",
			code: "enemy_prop_ignored", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "enemy", X: 0, Y: 16,
					Properties: []tiled.Property{{Name: "hp", Type: "int", Value: 0}}})
			},
		},
		{
			name: "door spawn coordinate is not a number",
			code: "door_spawn_unparseable", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "door", X: 0, Y: 0, Width: 16, Height: 16,
					Properties: []tiled.Property{
						{Name: "target_map", Type: "string", Value: "elsewhere"},
						{Name: "spawn_x", Type: "string", Value: "middle"},
					}})
			},
		},
		{
			name: "door spawn keep on both axes",
			code: "door_spawn_keep_both", sev: SeverityError,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "door", X: 0, Y: 0, Width: 16, Height: 16,
					Properties: []tiled.Property{
						{Name: "target_map", Type: "string", Value: "elsewhere"},
						{Name: "spawn_x", Type: "string", Value: "*"},
						{Name: "spawn_y", Type: "string", Value: " * "},
					}})
			},
		},
		{
			name: "unrecognized spawn anchor falls back to feet",
			code: "unknown_spawn_anchor", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "door", X: 0, Y: 0, Width: 16, Height: 16,
					Properties: []tiled.Property{
						{Name: "target_map", Type: "string", Value: "elsewhere"},
						{Name: "spawn_anchor", Type: "string", Value: "centre"},
					}})
			},
		},
		{
			// The loader ROUNDS a sign to the nearest tile and overwrites that
			// tile's gid, so an off-grid sign stamps somewhere unexpected.
			name: "off-grid sign stamps a different tile",
			code: "sign_not_tile_aligned", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "sign", X: 20, Y: 20, Width: 16, Height: 16})
			},
		},
		{
			name: "marker outside the map",
			code: "marker_out_of_bounds", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "door", X: -8, Y: 0, Width: 16, Height: 16,
					Properties: []tiled.Property{{Name: "target_map", Type: "string", Value: "elsewhere"}}})
			},
		},
		{
			// BuildFromTiled reads light_level and never looks at ambient_light
			// when it is present, so the second value silently does nothing.
			name: "ambient_light is dead when light_level is set",
			code: "dead_ambient_light", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				m.Properties = []tiled.Property{
					{Name: "light_level", Type: "float", Value: 0.3},
					{Name: "ambient_light", Type: "float", Value: 0.9},
				}
			},
		},
		{
			name: "light level out of range",
			code: "light_level_out_of_range", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				m.Properties = []tiled.Property{{Name: "light_level", Type: "float", Value: 4.0}}
			},
		},
		{
			name: "missing markers object group",
			code: "no_markers_group", sev: SeverityWarn,
			mut: func(m *tiled.Map) { m.Layers = m.Layers[:1] },
		},
		{
			name: "dungeon socket with a bad direction",
			code: "bad_dungeon_socket", sev: SeverityWarn,
			mut: func(m *tiled.Map) {
				addObjects(m, tiled.Object{ID: 2, Type: "marker", X: 0, Y: 0,
					Properties: []tiled.Property{{Name: "socket", Type: "string", Value: "NORTHWEST"}}})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := validMap()
			tc.mut(m)
			got := Validate(m, "test", ValidateOpts{})
			if !hasCode(got, tc.code) {
				t.Fatalf("expected issue %q, got %v", tc.code, codes(got))
			}
			for _, i := range got {
				if i.Code == tc.code && i.Severity != tc.sev {
					t.Errorf("%s has severity %q, want %q", tc.code, i.Severity, tc.sev)
				}
			}
		})
	}
}

// TestValidateAcceptsDungeonSockets is the counterpart to the check above: the
// rooms/*.tmj prefabs legitimately carry type "marker" socket objects that
// internal/dungeon/stitcher.go consumes, and they must not read as errors.
func TestValidateAcceptsDungeonSockets(t *testing.T) {
	m := validMap()
	for i, dir := range []string{"N", "S", "W", "E"} {
		addObjects(m, tiled.Object{ID: 10 + i, Type: "marker", X: 0, Y: 0,
			Properties: []tiled.Property{{Name: "socket", Type: "string", Value: dir}}})
	}
	if got := Validate(m, "rooms/test", ValidateOpts{}); len(got) != 0 {
		t.Errorf("dungeon sockets flagged as problems: %v", got)
	}
}

// TestValidateSkipsSpawnCheckForRoomPrefabs — prefabs are stamped into a
// generated dungeon, never loaded standalone.
func TestValidateSkipsSpawnCheckForRoomPrefabs(t *testing.T) {
	m := validMap()
	m.ObjectGroupLayer("markers").Objects = nil

	if got := Validate(m, "rooms/combat", ValidateOpts{}); hasCode(got, "no_spawn_marker") {
		t.Error("room prefab was asked for a spawn marker")
	}
	if got := Validate(m, "F-5", ValidateOpts{}); !hasCode(got, "no_spawn_marker") {
		t.Error("a normal map with no spawn marker was not flagged")
	}
}

// TestValidateCrossMapDoorChecks covers the checks that need a second map, and
// the strict-mode promotion.
func TestValidateCrossMapDoorChecks(t *testing.T) {
	target := validMap()
	target.Layers[0].Data[0] = 2 // wall at tile (0,0)

	resolve := func(id string) (*tiled.Map, bool) {
		if id == "target" {
			return target, true
		}
		return nil, false
	}

	door := func(props ...tiled.Property) *tiled.Map {
		m := validMap()
		addObjects(m, tiled.Object{ID: 2, Type: "door", X: 0, Y: 0, Width: 16, Height: 16, Properties: props})
		return m
	}
	p := func(n, v string) tiled.Property { return tiled.Property{Name: n, Type: "string", Value: v} }

	t.Run("missing target is a warning by default", func(t *testing.T) {
		got := Validate(door(p("target_map", "nowhere")), "test", ValidateOpts{ResolveTarget: resolve})
		if !hasCode(got, "door_target_missing") {
			t.Fatalf("got %v", codes(got))
		}
		for _, i := range got {
			if i.Code == "door_target_missing" && i.Severity != SeverityWarn {
				t.Errorf("severity %q, want warn", i.Severity)
			}
		}
	})

	t.Run("strict mode promotes it to an error", func(t *testing.T) {
		got := Validate(door(p("target_map", "nowhere")), "test", ValidateOpts{ResolveTarget: resolve, Strict: true})
		for _, i := range got {
			if i.Code == "door_target_missing" && i.Severity != SeverityError {
				t.Errorf("severity %q, want error under -strict", i.Severity)
			}
		}
	})

	t.Run("spawning inside a wall", func(t *testing.T) {
		m := door(p("target_map", "target"), p("spawn_x", "4"), p("spawn_y", "4"))
		if got := Validate(m, "test", ValidateOpts{ResolveTarget: resolve}); !hasCode(got, "door_spawn_in_solid") {
			t.Fatalf("got %v", codes(got))
		}
	})

	t.Run("spawning outside the target map", func(t *testing.T) {
		m := door(p("target_map", "target"), p("spawn_x", "9999"), p("spawn_y", "0"))
		if got := Validate(m, "test", ValidateOpts{ResolveTarget: resolve}); !hasCode(got, "door_spawn_out_of_bounds") {
			t.Fatalf("got %v", codes(got))
		}
	})

	t.Run("a good door is silent", func(t *testing.T) {
		m := door(p("target_map", "target"), p("spawn_x", "32"), p("spawn_y", "32"))
		if got := Validate(m, "test", ValidateOpts{ResolveTarget: resolve}); len(got) != 0 {
			t.Errorf("a valid door produced issues: %v", got)
		}
	})

	t.Run("keep axis is not unparseable", func(t *testing.T) {
		m := door(p("target_map", "target"), p("spawn_x", "*"), p("spawn_y", "32"))
		got := Validate(m, "test", ValidateOpts{ResolveTarget: resolve})
		if hasCode(got, "door_spawn_unparseable") || hasCode(got, "door_spawn_keep_both") {
			t.Fatalf("got %v", codes(got))
		}
	})

	t.Run("keep axis skips solid check", func(t *testing.T) {
		m := door(p("target_map", "target"), p("spawn_x", "*"), p("spawn_y", "4"))
		if got := Validate(m, "test", ValidateOpts{ResolveTarget: resolve}); hasCode(got, "door_spawn_in_solid") {
			t.Fatalf("got %v", codes(got))
		}
	})

	t.Run("pinned axis still checked for bounds", func(t *testing.T) {
		m := door(p("target_map", "target"), p("spawn_x", "*"), p("spawn_y", "9999"))
		if got := Validate(m, "test", ValidateOpts{ResolveTarget: resolve}); !hasCode(got, "door_spawn_out_of_bounds") {
			t.Fatalf("got %v", codes(got))
		}
	})

	t.Run("cross-map checks are skipped without a resolver", func(t *testing.T) {
		m := door(p("target_map", "nowhere"))
		if got := Validate(m, "test", ValidateOpts{}); hasCode(got, "door_target_missing") {
			t.Error("resolved a door target with no resolver configured")
		}
	})
}

func TestValidateNilMap(t *testing.T) {
	if got := Validate(nil, "test", ValidateOpts{}); !hasCode(got, "parse_failed") {
		t.Errorf("got %v", codes(got))
	}
}

func TestIssuesSortErrorsFirst(t *testing.T) {
	m := validMap()
	m.Layers[0].Data[0] = 9999                                                        // error
	m.Properties = []tiled.Property{{Name: "light_level", Value: 4.0, Type: "float"}} // warn

	got := Validate(m, "test", ValidateOpts{})
	if len(got) < 2 {
		t.Fatalf("expected at least two issues, got %v", codes(got))
	}
	if got[0].Severity != SeverityError {
		t.Errorf("first issue is %q, want errors sorted first", got[0].Severity)
	}
}
