package editorweb

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jaredwarren/game-test/internal/tiled"
)

// ErrMapNotFound is returned when an id resolves cleanly but has no file.
var ErrMapNotFound = errors.New("map not found")

// ErrStale is returned when a save's baseline etag no longer matches disk.
var ErrStale = errors.New("map changed on disk since it was loaded")

// MapStore is the only thing in this package that touches the filesystem.
//
// Root is absolute and symlink-resolved so the containment check in resolve is
// meaningful.
type MapStore struct {
	Root string
}

// NewMapStore resolves and validates a maps directory.
func NewMapStore(dir string) (*MapStore, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	// EvalSymlinks so that Rel-based containment cannot be defeated by a
	// symlinked root, and so the path we log is the real one.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("maps dir %s: %w", abs, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("maps dir %s is not a directory", resolved)
	}
	return &MapStore{Root: resolved}, nil
}

// resolve turns an untrusted map id into an absolute path inside Root.
//
// Two independent layers, both kept on purpose: NormalizeMapID's per-segment
// allowlist is what actually stops traversal, and the Rel containment check
// below is the backstop if that regex is ever loosened.
func (s *MapStore) resolve(id string) (string, error) {
	clean, err := NormalizeMapID(id)
	if err != nil {
		return "", err
	}
	full := filepath.Join(s.Root, filepath.FromSlash(clean)+".tmj")

	rel, err := filepath.Rel(s.Root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrBadMapID
	}
	return full, nil
}

// Path returns the on-disk path for a map id.
func (s *MapStore) Path(id string) (string, error) { return s.resolve(id) }

// Exists reports whether a map file is present. An invalid id is simply absent.
func (s *MapStore) Exists(id string) bool {
	full, err := s.resolve(id)
	if err != nil {
		return false
	}
	info, err := os.Stat(full)
	return err == nil && !info.IsDir()
}

// Read parses a map from disk.
//
// The returned etag is computed over the RE-ENCODED bytes, not the raw file, so
// a client that loads and immediately saves produces a matching etag. See the
// round-trip rule in the package doc.
func (s *MapStore) Read(id string) (m *tiled.Map, raw []byte, etag string, err error) {
	full, err := s.resolve(id)
	if err != nil {
		return nil, nil, "", err
	}
	raw, err = os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, "", fmt.Errorf("%w: %s", ErrMapNotFound, id)
		}
		return nil, nil, "", err
	}
	m, err = tiled.ParseMap(raw)
	if err != nil {
		return nil, raw, "", fmt.Errorf("parse %s: %w", id, err)
	}
	// The etag is over what a save would WRITE, including the file's own data
	// layout, so loading and immediately saving matches instead of 409ing.
	enc, err := encodeWithStyle(m, detectStyle(raw))
	if err != nil {
		return nil, raw, "", err
	}
	return m, raw, etagOf(enc), nil
}

// CurrentETag returns the etag a Write would have to match, or "" if absent.
func (s *MapStore) CurrentETag(id string) (string, bool) {
	m, _, etag, err := s.Read(id)
	if err != nil || m == nil {
		return "", false
	}
	return etag, true
}

// Write saves a map atomically.
//
// baseETag implements the stale-write guard. Because the browser holds the whole
// document and the in-game editor writes the same files, two editors (or a
// `make world-grid` run, or a git checkout) will otherwise silently clobber each
// other. Pass "" to skip the check.
func (s *MapStore) Write(id string, m *tiled.Map, baseETag string) (string, error) {
	full, err := s.resolve(id)
	if err != nil {
		return "", err
	}
	if baseETag != "" {
		current, ok := s.CurrentETag(id)
		if ok && current != baseETag {
			return current, ErrStale
		}
	}
	// No implicit mkdir: refusing to create directories doubles as traversal
	// hardening and prevents a typo'd id from scattering new trees.
	dir := filepath.Dir(full)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("parent directory does not exist: %s", dir)
	}

	// Encode, never the client's bytes. This is the round-trip rule.
	//
	// The layout is read back off disk rather than carried by the client, so a
	// hand-authored row-per-line map keeps its floor-plan formatting even though
	// the browser never knew about it.
	style := fileStyle{}
	if existing, err := os.ReadFile(full); err == nil {
		style = detectStyle(existing)
	}
	b, err := encodeWithStyle(m, style)
	if err != nil {
		return "", err
	}

	// CreateTemp in the TARGET directory so Rename is a same-filesystem atomic
	// replace; a temp in /tmp would degrade to a copy across volumes.
	tmp, err := os.CreateTemp(dir, ".mapeditor-*.tmj.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", err
	}
	// CreateTemp makes 0600; the maps are 0644 like tiled.WriteFile writes them.
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpName, full); err != nil {
		return "", err
	}
	return etagOf(b), nil
}

// MapInfo is one entry in the map browser index.
type MapInfo struct {
	ID           string         `json:"id"`
	File         string         `json:"file"`
	Group        MapGroup       `json:"group"`
	Grid         *GridPos       `json:"grid,omitempty"`
	Width        int            `json:"width"`
	Height       int            `json:"height"`
	TileWidth    int            `json:"tileWidth"`
	TileHeight   int            `json:"tileHeight"`
	Layers       []LayerInfo    `json:"layers"`
	ObjectCount  int            `json:"objectCount"`
	MarkerCounts map[string]int `json:"markerCounts"`
	Bytes        int64          `json:"bytes"`
	ModUnix      int64          `json:"modUnix"`
	ETag         string         `json:"etag"`
	ParseError   string         `json:"parseError,omitempty"`
}

// LayerInfo summarizes one layer for the sidebar and layers panel.
type LayerInfo struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	Visible bool    `json:"visible"`
	Opacity float64 `json:"opacity"`
}

// List walks the maps tree and summarizes every .tmj it finds.
//
// Files that fail to parse are still listed, with ParseError set, so a broken
// map is visible in the browser instead of silently missing.
func (s *MapStore) List() ([]MapInfo, error) {
	var out []MapInfo

	err := filepath.WalkDir(s.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		// Skip dotfiles, which also skips crashed .mapeditor-*.tmj.tmp temps.
		if strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(name, ".tmj") {
			return nil
		}
		rel, err := filepath.Rel(s.Root, p)
		if err != nil {
			return err
		}
		id := strings.TrimSuffix(filepath.ToSlash(rel), ".tmj")
		if _, err := NormalizeMapID(id); err != nil {
			return nil // unreachable by the API anyway; do not advertise it
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, s.describe(id, filepath.ToSlash(rel), info.Size(), info.ModTime().Unix()))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *MapStore) describe(id, file string, size, mod int64) MapInfo {
	mi := MapInfo{
		ID:           id,
		File:         file,
		Group:        GroupOf(id),
		Bytes:        size,
		ModUnix:      mod,
		MarkerCounts: map[string]int{},
		Layers:       []LayerInfo{},
	}
	if g, ok := ParseGridID(id); ok {
		mi.Grid = &g
	}

	m, _, etag, err := s.Read(id)
	if err != nil {
		mi.ParseError = err.Error()
		return mi
	}
	mi.ETag = etag
	mi.Width, mi.Height = m.Width, m.Height
	mi.TileWidth, mi.TileHeight = m.TileWidth, m.TileHeight

	for _, l := range m.Layers {
		mi.Layers = append(mi.Layers, LayerInfo{
			ID: l.ID, Name: l.Name, Type: l.Type, Visible: l.Visible, Opacity: l.Opacity,
		})
		for _, o := range l.Objects {
			mi.ObjectCount++
			mi.MarkerCounts[o.Type]++
		}
	}
	return mi
}

// Reformats reports whether saving this map would rewrite its formatting even
// with no edits.
//
// Round-tripping through tiled.Map is semantically lossless for every shipped
// map, but not byte-identical for all of them: hand-authored files write tile
// `data` several ints per line where json.MarshalIndent writes one, and explicit
// zero-valued object width/height are dropped by omitempty. The result is a
// whitespace-and-defaults diff with no data change.
//
// The in-game editor has always done this silently. Surfacing it lets the web
// editor warn before a no-op save turns into a large diff in a review.
func Reformats(raw []byte, m *tiled.Map) bool {
	enc, err := encodeWithStyle(m, detectStyle(raw))
	if err != nil {
		return true
	}
	return !bytes.Equal(bytes.TrimRight(raw, "\n"), enc)
}

// etagOf is a short content hash. 64 bits is ample for detecting concurrent
// edits on a local filesystem.
func etagOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// unmodeledFields reports JSON paths present in the raw file that tiled.Map does
// not model, and which a save would therefore silently delete.
//
// The corpus has none today (TestCorpusHasNoUnmodeledFields pins that), but the
// maps are also editable in real Tiled, which can add object rotation, polygons,
// layer tint, and more. Detecting that and refusing to save beats eating it.
func unmodeledFields(raw []byte) []string {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	var out []string
	collectUnknown(doc, mapKeys, "", &out)

	var layers []map[string]json.RawMessage
	if v, ok := doc["layers"]; ok {
		_ = json.Unmarshal(v, &layers)
	}
	for i, l := range layers {
		prefix := fmt.Sprintf("layers[%d]", i)
		collectUnknown(l, layerKeys, prefix, &out)

		var objs []map[string]json.RawMessage
		if v, ok := l["objects"]; ok {
			_ = json.Unmarshal(v, &objs)
		}
		for j, o := range objs {
			collectUnknown(o, objectKeys, fmt.Sprintf("%s.objects[%d]", prefix, j), &out)

			var props []map[string]json.RawMessage
			if v, ok := o["properties"]; ok {
				_ = json.Unmarshal(v, &props)
			}
			for k, pr := range props {
				collectUnknown(pr, propertyKeys, fmt.Sprintf("%s.objects[%d].properties[%d]", prefix, j, k), &out)
			}
		}
	}
	sort.Strings(out)
	return out
}

func collectUnknown(obj map[string]json.RawMessage, known map[string]bool, prefix string, out *[]string) {
	for k := range obj {
		if known[k] {
			continue
		}
		if prefix == "" {
			*out = append(*out, k)
		} else {
			*out = append(*out, prefix+"."+k)
		}
	}
}

// Key sets mirror the json tags on tiled.Map/Layer/Object/Property.
// TestModeledKeySetsMatchStructTags keeps them in step by reflection.
var (
	mapKeys = keySet("compressionlevel", "width", "height", "infinite", "tilewidth",
		"tileheight", "type", "version", "tiledversion", "orientation", "renderorder",
		"layers", "nextlayerid", "nextobjectid", "tilesets", "properties")
	layerKeys = keySet("id", "type", "name", "visible", "opacity", "width", "height",
		"x", "y", "data", "objects", "properties")
	objectKeys   = keySet("id", "name", "type", "x", "y", "width", "height", "properties")
	propertyKeys = keySet("name", "type", "value")
)

func keySet(keys ...string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}
