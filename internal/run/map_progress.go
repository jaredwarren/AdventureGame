package run

import (
	"sort"
	"strings"

	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/world"
)

// MapProgress records durable changes to one map (pickups, tiles, locks, shrines).
type MapProgress struct {
	CollectedPickups map[string]struct{}
	DestroyedTiles   map[string]struct{}
	OpenedLocks      map[string]struct{}
	ActivatedShrines map[string]struct{}
}

func (p *MapProgress) MarkCollectedPickup(key string) {
	if p == nil || key == "" {
		return
	}
	if p.CollectedPickups == nil {
		p.CollectedPickups = make(map[string]struct{})
	}
	p.CollectedPickups[key] = struct{}{}
}

func (p *MapProgress) MarkDestroyedTile(key string) {
	if p == nil || key == "" {
		return
	}
	if p.DestroyedTiles == nil {
		p.DestroyedTiles = make(map[string]struct{})
	}
	p.DestroyedTiles[key] = struct{}{}
}

func (p *MapProgress) MarkOpenedLock(key string) {
	if p == nil || key == "" {
		return
	}
	if p.OpenedLocks == nil {
		p.OpenedLocks = make(map[string]struct{})
	}
	p.OpenedLocks[key] = struct{}{}
}

func (p *MapProgress) MarkActivatedShrine(key string) {
	if p == nil || key == "" {
		return
	}
	if p.ActivatedShrines == nil {
		p.ActivatedShrines = make(map[string]struct{})
	}
	p.ActivatedShrines[key] = struct{}{}
}

func (p *MapProgress) collectedPickupSet() map[string]struct{} {
	if p == nil || len(p.CollectedPickups) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(p.CollectedPickups))
	for k := range p.CollectedPickups {
		if strings.TrimSpace(k) != "" {
			out[k] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ApplyMapProgress patches a freshly built world with this map's saved changes.
func ApplyMapProgress(w *world.World, p *MapProgress) {
	if w == nil || p == nil {
		return
	}
	world.ApplyDestroyedTiles(w, p.DestroyedTiles)
	world.ApplyPersistedOpenedLocks(w, p.OpenedLocks)
	world.ApplyPersistedShrines(w, p.ActivatedShrines)
}

func mapIDFromPersistKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if mid, _, _, ok := world.ParseMapTilePersistKey(key); ok {
		return mid
	}
	parts := strings.Split(key, ":")
	if len(parts) >= 3 && parts[len(parts)-2] == "shrine" {
		return strings.Join(parts[:len(parts)-2], ":")
	}
	if len(parts) >= 2 {
		return strings.Join(parts[:len(parts)-1], ":")
	}
	return ""
}

func mergeKeysIntoProgress(maps map[string]*MapProgress, keys []string, apply func(*MapProgress, string)) {
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		mapID := mapIDFromPersistKey(k)
		if mapID == "" {
			continue
		}
		if maps == nil {
			maps = make(map[string]*MapProgress)
		}
		p, ok := maps[mapID]
		if !ok {
			p = &MapProgress{}
			maps[mapID] = p
		}
		apply(p, k)
	}
}

func progressMapsFromSave(s *save.GameSave) map[string]*MapProgress {
	if s == nil {
		return nil
	}
	out := make(map[string]*MapProgress)
	mergeKeysIntoProgress(out, s.CollectedPickupKeys, (*MapProgress).MarkCollectedPickup)
	mergeKeysIntoProgress(out, s.DestroyedTileKeys, (*MapProgress).MarkDestroyedTile)
	mergeKeysIntoProgress(out, s.OpenedLockTileKeys, (*MapProgress).MarkOpenedLock)
	mergeKeysIntoProgress(out, s.ActivatedShrines, (*MapProgress).MarkActivatedShrine)
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeSaveIntoMaps(maps map[string]*MapProgress, s *save.GameSave) map[string]*MapProgress {
	if s == nil {
		return maps
	}
	if maps == nil {
		maps = make(map[string]*MapProgress)
	}
	mergeKeysIntoProgress(maps, s.CollectedPickupKeys, (*MapProgress).MarkCollectedPickup)
	mergeKeysIntoProgress(maps, s.DestroyedTileKeys, (*MapProgress).MarkDestroyedTile)
	mergeKeysIntoProgress(maps, s.OpenedLockTileKeys, (*MapProgress).MarkOpenedLock)
	mergeKeysIntoProgress(maps, s.ActivatedShrines, (*MapProgress).MarkActivatedShrine)
	return maps
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		if strings.TrimSpace(k) != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func flattenMapsToSave(gs *save.GameSave, maps map[string]*MapProgress) {
	if gs == nil || len(maps) == 0 {
		return
	}
	mapIDs := make([]string, 0, len(maps))
	for id := range maps {
		mapIDs = append(mapIDs, id)
	}
	sort.Strings(mapIDs)

	var pickups, destroyed, locks, shrines []string
	for _, id := range mapIDs {
		p := maps[id]
		if p == nil {
			continue
		}
		pickups = append(pickups, sortedKeys(p.CollectedPickups)...)
		destroyed = append(destroyed, sortedKeys(p.DestroyedTiles)...)
		locks = append(locks, sortedKeys(p.OpenedLocks)...)
		shrines = append(shrines, sortedKeys(p.ActivatedShrines)...)
	}
	if len(pickups) > 0 {
		gs.CollectedPickupKeys = pickups
	}
	if len(destroyed) > 0 {
		gs.DestroyedTileKeys = destroyed
	}
	if len(locks) > 0 {
		gs.OpenedLockTileKeys = locks
	}
	if len(shrines) > 0 {
		gs.ActivatedShrines = shrines
	}
}
