package tile

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	artMu    sync.RWMutex
	artByGID = map[int]Art{}
)

// ArtOf returns the loaded art for a GID, if any.
func ArtOf(gid int) (Art, bool) {
	artMu.RLock()
	defer artMu.RUnlock()
	a, ok := artByGID[gid]
	return a, ok
}

// HasArt reports whether editable art is loaded for gid.
func HasArt(gid int) bool {
	_, ok := ArtOf(gid)
	return ok
}

// LoadedArtGIDs returns GIDs that have art files applied.
func LoadedArtGIDs() []int {
	artMu.RLock()
	defer artMu.RUnlock()
	out := make([]int, 0, len(artByGID))
	for g := range artByGID {
		out = append(out, g)
	}
	return out
}

// ApplyArt validates art and installs it as the VectorDraw for that GID.
func ApplyArt(art Art) error {
	if err := art.Validate(); err != nil {
		return err
	}
	if art.Size == 0 {
		art.Size = Size
	}

	registryMu.Lock()
	d, ok := defs[art.GID]
	if !ok {
		registryMu.Unlock()
		return fmt.Errorf("gid %d is not registered", art.GID)
	}
	d.VectorDraw = ArtDrawer(art)
	defs[art.GID] = d
	registryMu.Unlock()

	artMu.Lock()
	artByGID[art.GID] = art
	artMu.Unlock()
	return nil
}

// LoadArtFS loads every *.tile.json from fsys (typically assets.TileArtFS).
// Accepts either a flat FS of *.tile.json or an FS with a tiles/ subdirectory.
func LoadArtFS(fsys fs.FS) error {
	if entries, err := fs.ReadDir(fsys, "tiles"); err == nil {
		return loadArtEntries(fsys, "tiles", entries)
	}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("tile art: %w", err)
	}
	return loadArtEntries(fsys, ".", entries)
}

// LoadArtDir loads every *.tile.json from a filesystem directory.
func LoadArtDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	infos := make([]fs.DirEntry, 0, len(entries))
	for _, e := range entries {
		infos = append(infos, e)
	}
	return loadArtEntries(os.DirFS(dir), ".", infos)
}

func loadArtEntries(fsys fs.FS, root string, entries []fs.DirEntry) error {
	var first error
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tile.json") {
			continue
		}
		path := e.Name()
		if root != "." {
			path = root + "/" + e.Name()
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		var art Art
		if err := json.Unmarshal(data, &art); err != nil {
			if first == nil {
				first = fmt.Errorf("%s: %w", path, err)
			}
			continue
		}
		if err := ApplyArt(art); err != nil {
			if first == nil {
				first = fmt.Errorf("%s: %w", path, err)
			}
		}
	}
	return first
}

// SaveArtFile writes art to dir as {gid}_{name}.tile.json.
func SaveArtFile(dir string, art Art) (string, error) {
	if err := art.Validate(); err != nil {
		return "", err
	}
	if art.Size == 0 {
		art.Size = Size
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	safe := sanitizeArtName(art.Name)
	name := fmt.Sprintf("%d_%s.tile.json", art.GID, safe)
	path := filepath.Join(dir, name)

	// Remove older files for the same GID so renames do not leave orphans.
	if entries, err := os.ReadDir(dir); err == nil {
		prefix := fmt.Sprintf("%d_", art.GID)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".tile.json") && e.Name() != name {
				_ = os.Remove(filepath.Join(dir, e.Name()))
			}
		}
	}

	data, err := json.MarshalIndent(art, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	if err := ApplyArt(art); err != nil {
		return path, err
	}
	return path, nil
}

// ArtFilePath returns the on-disk path for a GID if one exists in dir.
func ArtFilePath(dir string, gid int) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	prefix := fmt.Sprintf("%d_", gid)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".tile.json") {
			return filepath.Join(dir, e.Name()), true
		}
	}
	return "", false
}

func sanitizeArtName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('_')
		}
	}
	s := b.String()
	if s == "" {
		return "tile"
	}
	return s
}
