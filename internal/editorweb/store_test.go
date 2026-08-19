package editorweb

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jaredwarren/game-test/assets"
	"github.com/jaredwarren/game-test/internal/tiled"
)

// TestNormalizeMapIDRejectsTraversal is the security test for the one input that
// reaches this package straight off an HTTP path.
func TestNormalizeMapIDRejectsTraversal(t *testing.T) {
	bad := []string{
		"", ".", "..", "/", "//", "   ",
		"../../etc/passwd",
		"/etc/passwd",
		"rooms/../../x",
		"rooms/../../../etc/passwd",
		"a\x00b",
		`rooms\boss`,
		"rooms/",
		"/rooms/boss",
		".hidden",
		"rooms/.hidden",
		"a/b/c",         // deeper than assets/maps ever goes
		"-leading-dash", // must start alphanumeric
		strings.Repeat("a", 201),
	}
	for _, id := range bad {
		if got, err := NormalizeMapID(id); err == nil {
			t.Errorf("NormalizeMapID(%q) = %q, want an error", id, got)
		}
	}

	good := map[string]string{
		"F-5":            "F-5",
		"F-5.tmj":        "F-5",
		"field1":         "field1",
		"rooms/boss":     "rooms/boss",
		"rooms/boss.tmj": "rooms/boss",
		"maze2":          "maze2",
	}
	for in, want := range good {
		got, err := NormalizeMapID(in)
		if err != nil {
			t.Errorf("NormalizeMapID(%q) errored: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeMapID(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestResolveStaysInsideRoot proves the containment backstop holds even for ids
// the allowlist would let through.
func TestResolveStaysInsideRoot(t *testing.T) {
	s := newTempStore(t)
	for _, id := range []string{"../../etc/passwd", "rooms/../../x", "/etc/passwd"} {
		if p, err := s.resolve(id); err == nil {
			t.Errorf("resolve(%q) = %q, want an error", id, p)
		}
	}
	p, err := s.resolve("rooms/boss")
	if err != nil {
		t.Fatalf("resolve(rooms/boss): %v", err)
	}
	if !strings.HasPrefix(p, s.Root+string(filepath.Separator)) {
		t.Errorf("resolve escaped the root: %q not under %q", p, s.Root)
	}
}

// TestWriteIsAtomicAndByteExact checks the save path produces exactly
// Encode() output — no trailing newline, correct mode, no temp files left.
func TestWriteIsAtomicAndByteExact(t *testing.T) {
	s := newTempStore(t)
	m := smallMap()

	etag, err := s.Write("F-5", m, "")
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	want, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	full, _ := s.resolve("F-5")
	got, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("file bytes differ from Encode() output (len %d vs %d)", len(got), len(want))
	}
	if bytes.HasSuffix(got, []byte("\n")) {
		t.Error("save appended a trailing newline; tiled.WriteFile does not, and this would diff every map")
	}
	if etag != etagOf(want) {
		t.Errorf("returned etag %q != etag of written bytes %q", etag, etagOf(want))
	}

	info, err := os.Stat(full)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Errorf("mode %v, want 0644 (os.CreateTemp defaults to 0600)", perm)
	}

	entries, _ := os.ReadDir(s.Root)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".mapeditor-") {
			t.Errorf("temp file %q left behind after a successful write", e.Name())
		}
	}
}

// TestWriteRejectsStaleETag exercises the guard that stops the browser and the
// in-game editor from silently clobbering each other.
func TestWriteRejectsStaleETag(t *testing.T) {
	s := newTempStore(t)
	m := smallMap()
	etag, err := s.Write("F-5", m, "")
	if err != nil {
		t.Fatal(err)
	}

	// Someone else edits the file (the in-game editor, git, genworldgrid).
	other := smallMap()
	other.Width = 25
	other.Height = 21
	other.Layers[0].Width, other.Layers[0].Height = 25, 21
	other.Layers[0].Data = make([]int, 25*21)
	newETag, err := s.Write("F-5", other, "")
	if err != nil {
		t.Fatal(err)
	}
	if newETag == etag {
		t.Fatal("etag did not change after a different map was written")
	}

	// Our stale save must be refused, and must report the current etag.
	current, err := s.Write("F-5", m, etag)
	if !errors.Is(err, ErrStale) {
		t.Fatalf("stale write err = %v, want ErrStale", err)
	}
	if current != newETag {
		t.Errorf("stale write reported etag %q, want the on-disk %q", current, newETag)
	}

	// Saving against the current etag succeeds.
	if _, err := s.Write("F-5", m, newETag); err != nil {
		t.Fatalf("write with a fresh etag: %v", err)
	}
}

// TestWriteRefusesMissingParent documents that ids never create directories.
func TestWriteRefusesMissingParent(t *testing.T) {
	s := newTempStore(t)
	if _, err := s.Write("nosuchdir/thing", smallMap(), ""); err == nil {
		t.Error("write created a map under a nonexistent directory")
	}
}

// TestReadETagIsOfReEncodedBytes is what makes "open, save, zero diff" work: the
// baseline the client holds must be the etag a save would produce.
func TestReadETagIsOfReEncodedBytes(t *testing.T) {
	s := newTempStore(t)
	m := smallMap()
	written, err := s.Write("F-5", m, "")
	if err != nil {
		t.Fatal(err)
	}
	_, _, read, err := s.Read("F-5")
	if err != nil {
		t.Fatal(err)
	}
	if read != written {
		t.Errorf("Read etag %q != Write etag %q; a load-then-save would spuriously 409", read, written)
	}
}

func TestReadMissingMap(t *testing.T) {
	s := newTempStore(t)
	if _, _, _, err := s.Read("nope"); !errors.Is(err, ErrMapNotFound) {
		t.Errorf("Read of a missing map gave %v, want ErrMapNotFound", err)
	}
	if s.Exists("nope") {
		t.Error("Exists reported a missing map as present")
	}
}

// TestListSkipsTempFiles makes sure a crashed write does not show up as a map.
func TestListSkipsTempFiles(t *testing.T) {
	s := newTempStore(t)
	if _, err := s.Write("F-5", smallMap(), ""); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(s.Root, ".mapeditor-crashed.tmj.tmp")
	if err := os.WriteFile(stale, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	infos, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 || infos[0].ID != "F-5" {
		t.Errorf("List returned %d entries, want just F-5: %+v", len(infos), infos)
	}
}

// ---- Corpus fidelity ----

// TestRoundTripPreservesSemantics is the real "no data loss" guarantee: every
// shipped map must survive ParseMap -> Encode -> ParseMap completely unchanged.
// This is what makes the editor safe to save with.
func TestRoundTripPreservesSemantics(t *testing.T) {
	forEachShippedMap(t, func(t *testing.T, name string, raw []byte) {
		before, err := tiled.ParseMap(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		enc, err := before.Encode()
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		after, err := tiled.ParseMap(enc)
		if err != nil {
			t.Fatalf("reparse: %v", err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Error("map does not survive a parse/encode round trip unchanged — saving it from the editor would lose or alter data")
		}
	})
}

// reformattingMaps are the shipped maps that a no-op save would still rewrite.
//
// The writer reproduces a file's own formatting conventions (see dataformat.go),
// so the hand-authored floor-plan layout and inline properties survive. What
// remains here is normalization the parser does on purpose, never a formatting
// choice:
//
//   - rooms/{combat,corridor,key,start}: no "nextobjectid" on disk;
//     normalizeAfterDecode computes and adds one, which is a repair.
//   - maze2, knight_boss: an object carries explicit "width": 0, "height": 0,
//     which tiled.Object drops via omitempty. Absent and zero mean the same.
//   - rooms/boss: both of the above.
//
// The list should only shrink. A new entry means a map regressed or the writer
// lost a convention it used to preserve.
var reformattingMaps = map[string]bool{
	"maps/knight_boss.tmj":    true,
	"maps/maze2.tmj":          true,
	"maps/rooms/boss.tmj":     true,
	"maps/rooms/combat.tmj":   true,
	"maps/rooms/corridor.tmj": true,
	"maps/rooms/key.tmj":      true,
	"maps/rooms/start.tmj":    true,
}

// TestRoundTripByteIdentical pins which maps save cleanly and which reformat.
//
// A clean byte round trip is what makes "open a map, save it, get an empty git
// diff" true, so the editor is safe to leave running. It holds for every
// generated map; the exceptions are enumerated above.
func TestRoundTripByteIdentical(t *testing.T) {
	forEachShippedMap(t, func(t *testing.T, name string, raw []byte) {
		m, err := tiled.ParseMap(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		got := Reformats(raw, m)
		if want := reformattingMaps[name]; got != want {
			if got {
				t.Errorf("saving this map now reformats it, but it used to round-trip cleanly — if that is intended, add %q to reformattingMaps", name)
			} else {
				t.Errorf("this map now round-trips byte-identically; remove %q from reformattingMaps", name)
			}
		}
	})
}

// TestCorpusHasNoUnmodeledFields is the tripwire for round-trip loss. If it
// fires, someone edited a map in real Tiled and added a feature tiled.Map does
// not model; the save path must refuse rather than delete it.
func TestCorpusHasNoUnmodeledFields(t *testing.T) {
	forEachShippedMap(t, func(t *testing.T, name string, raw []byte) {
		if extra := unmodeledFields(raw); len(extra) > 0 {
			t.Errorf("has fields tiled.Map does not model and a save would drop: %v", extra)
		}
	})
}

// TestUnmodeledFieldsDetectsRealTiledFeatures proves the detector actually works,
// using the fields real Tiled adds that this project's structs ignore.
func TestUnmodeledFieldsDetectsRealTiledFeatures(t *testing.T) {
	raw := []byte(`{
	  "width": 2, "height": 1, "tilewidth": 16, "tileheight": 16,
	  "backgroundcolor": "#ff0000",
	  "layers": [
	    {"type":"tilelayer","name":"ground","data":[1,1],"width":2,"height":1,"tintcolor":"#00ff00","parallaxx":0.5},
	    {"type":"objectgroup","name":"markers","draworder":"index","objects":[
	      {"id":1,"type":"enemy","x":0,"y":0,"rotation":45,"polygon":[{"x":0,"y":0}],
	       "properties":[{"name":"hp","type":"int","value":3,"propertytype":"Custom"}]}
	    ]}
	  ]
	}`)
	got := unmodeledFields(raw)
	want := []string{
		"backgroundcolor",
		"layers[0].parallaxx",
		"layers[0].tintcolor",
		"layers[1].draworder",
		"layers[1].objects[0].polygon",
		"layers[1].objects[0].properties[0].propertytype",
		"layers[1].objects[0].rotation",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unmodeledFields =\n  %v\nwant\n  %v", got, want)
	}
}

// TestModeledKeySetsMatchStructTags keeps the key sets above honest: adding a
// field to tiled.Map without updating them would make the detector cry wolf.
func TestModeledKeySetsMatchStructTags(t *testing.T) {
	cases := []struct {
		name string
		typ  reflect.Type
		keys map[string]bool
	}{
		{"Map", reflect.TypeOf(tiled.Map{}), mapKeys},
		{"Layer", reflect.TypeOf(tiled.Layer{}), layerKeys},
		{"Object", reflect.TypeOf(tiled.Object{}), objectKeys},
		{"Property", reflect.TypeOf(tiled.Property{}), propertyKeys},
	}
	for _, tc := range cases {
		tags := map[string]bool{}
		for i := 0; i < tc.typ.NumField(); i++ {
			tag := tc.typ.Field(i).Tag.Get("json")
			name, _, _ := strings.Cut(tag, ",")
			if name != "" && name != "-" {
				tags[name] = true
			}
		}
		if !reflect.DeepEqual(tags, tc.keys) {
			t.Errorf("tiled.%s json tags %v != the key set used by unmodeledFields %v", tc.name, keysOf(tags), keysOf(tc.keys))
		}
	}
}

// ---- helpers ----

func newTempStore(t *testing.T) *MapStore {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "rooms"), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := NewMapStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func smallMap() *tiled.Map {
	return &tiled.Map{
		Width: 2, Height: 1, TileWidth: 16, TileHeight: 16,
		Type: "map", Version: "1.10", Orientation: "orthogonal", RenderOrder: "right-down",
		NextLayerID: 3, NextObjectID: 2,
		Tilesets: json.RawMessage("[]"),
		Layers: []tiled.Layer{
			{ID: 1, Type: "tilelayer", Name: "ground", Visible: true, Opacity: 1,
				Width: 2, Height: 1, Data: []int{1, 2}},
			{ID: 2, Type: "objectgroup", Name: "markers", Visible: true, Opacity: 1,
				Objects: []tiled.Object{{ID: 1, Type: "spawn", X: 16, Y: 16}}},
		},
	}
}

// forEachShippedMap runs fn over every .tmj embedded in the game binary.
func forEachShippedMap(t *testing.T, fn func(t *testing.T, name string, raw []byte)) {
	t.Helper()
	count := 0
	err := fs.WalkDir(assets.MapFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".tmj") {
			return err
		}
		raw, err := assets.MapFS.ReadFile(p)
		if err != nil {
			return err
		}
		count++
		t.Run(p, func(t *testing.T) { fn(t, p, raw) })
		return nil
	})
	if err != nil {
		t.Fatalf("walk maps: %v", err)
	}
	if count == 0 {
		t.Fatal("no maps found in assets.MapFS")
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
