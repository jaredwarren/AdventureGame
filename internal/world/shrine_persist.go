package world

import "fmt"

// PersistentShrineSaveKey returns the save/session key for a Tiled shrine object id.
func PersistentShrineSaveKey(mapID string, tiledObjectID int) string {
	return fmt.Sprintf("%s:shrine:%d", mapID, tiledObjectID)
}

// ApplyPersistedShrines sets Shrine.Active to true for any shrines in the world
// whose persistent save key matches a key in the set.
func ApplyPersistedShrines(w *World, keys map[string]struct{}) {
	if w == nil || len(keys) == 0 {
		return
	}
	for i := range w.Shrines {
		sh := &w.Shrines[i]
		saveKey := PersistentShrineSaveKey(w.MapID, sh.TiledID)
		if _, ok := keys[saveKey]; ok {
			sh.Active = true
		}
	}
}
