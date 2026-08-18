// session.go — Session is the mutable run-scoped state that outlives any
// one scene but is narrower than "the whole process".
//
// Life-cycle:
//
//   - One Session instance is created by App and reused forever.
//   - Scenes mutate Session.World via the loader helpers in worldloader.go;
//     they never replace the World pointer directly.
//   - HasSave, WeeklySeed, and ShowDebugOverlay survive scene transitions.
//
// Session deliberately does NOT own:
//   - Scene-local state (PlayScene stores hitStop/dodgeImpulse/etc. itself).
//   - Rendering state (Camera lives on Renderer).
//   - Services (Input/Audio/Assets/Renderer flow via Context, not Session).
package run

import (
	"strings"

	"github.com/jaredwarren/game-test/internal/world"
)

// Session is the run-scoped mutable state shared across scenes.
type Session struct {
	// World is the current gameplay sim. Nil on the title screen and editor
	// start; loader helpers (OpenWorld / EnterMap / LoadGameFromSave)
	// replace it atomically from within a scene Update.
	World *world.World

	// Maps records per-map durable progress (pickups, broken tiles, locks,
	// shrines). Keys are map IDs; values persist across revisits and are
	// flattened into save.GameSave on BuildSave.
	Maps map[string]*MapProgress

	// VisitedMaps records all map IDs the player has entered during this run/save.
	VisitedMaps map[string]struct{}

	// WeeklySeed refreshes from time.Now at the top of every Update; feeds
	// any future weekly-rotation content (leaderboard epochs, event maps).
	// Stored on Session so debug HUD + saves agree on the same epoch for a
	// given frame.
	WeeklySeed int64

	// LastDungeonSeed and LastDungeonDigest record the current active dungeon seed & digest.
	LastDungeonSeed   int64
	LastDungeonDigest string

	// HasSave indicates whether the disk has a load-able save file. Set
	// once at startup; scenes read it to decide whether to offer quick load.
	HasSave bool

	// ShowDebugOverlay toggles the F3 playtest HUD. Lives on Session (not
	// any one scene) because every scene respects it.
	ShowDebugOverlay bool

	// ShowDoorHitboxes toggles door warp-rect overlays (F4). Off by default.
	ShowDoorHitboxes bool
}

// ClearDungeonProgress removes all dun:* map progress entries when a new dungeon seed is generated.
func (s *Session) ClearDungeonProgress() {
	if s == nil || s.Maps == nil {
		return
	}
	for mapID := range s.Maps {
		if strings.HasPrefix(mapID, "dun:") {
			delete(s.Maps, mapID)
		}
	}
}

// ProgressFor returns durable progress for mapID, creating an empty bundle if needed.
func (s *Session) ProgressFor(mapID string) *MapProgress {
	if s == nil || mapID == "" {
		return nil
	}
	if s.Maps == nil {
		s.Maps = make(map[string]*MapProgress)
	}
	if p, ok := s.Maps[mapID]; ok {
		return p
	}
	p := &MapProgress{}
	s.Maps[mapID] = p
	return p
}

// MarkPersistentPickupCollected records a consumed persistent pickup on mapID.
func (s *Session) MarkPersistentPickupCollected(mapID, key string) {
	s.ProgressFor(mapID).MarkCollectedPickup(key)
}

// MarkDestroyedTile records a destroyed tile on mapID.
func (s *Session) MarkDestroyedTile(mapID, key string) {
	s.ProgressFor(mapID).MarkDestroyedTile(key)
}

// MarkOpenedLockTile records an opened lock tile on mapID.
func (s *Session) MarkOpenedLockTile(mapID, key string) {
	s.ProgressFor(mapID).MarkOpenedLock(key)
}

// MarkShrineActivated records an activated shrine on mapID.
func (s *Session) MarkShrineActivated(mapID, key string) {
	s.ProgressFor(mapID).MarkActivatedShrine(key)
}

// MarkMapVisited marks mapID as visited by the player.
func (s *Session) MarkMapVisited(mapID string) {
	if s == nil || mapID == "" {
		return
	}
	if s.VisitedMaps == nil {
		s.VisitedMaps = make(map[string]struct{})
	}
	s.VisitedMaps[mapID] = struct{}{}
}

// IsMapVisited reports whether the player has visited mapID.
func (s *Session) IsMapVisited(mapID string) bool {
	if s == nil || s.VisitedMaps == nil || mapID == "" {
		return false
	}
	_, ok := s.VisitedMaps[mapID]
	return ok
}

// VisitedMapList returns a sorted list of visited map IDs.
func (s *Session) VisitedMapList() []string {
	if s == nil || len(s.VisitedMaps) == 0 {
		return nil
	}
	return sortedKeys(s.VisitedMaps)
}

// SetVisitedMaps replaces the visited maps set with the given list.
func (s *Session) SetVisitedMaps(maps []string) {
	if s == nil {
		return
	}
	if len(maps) == 0 {
		s.VisitedMaps = nil
		return
	}
	s.VisitedMaps = make(map[string]struct{}, len(maps))
	for _, m := range maps {
		if strings.TrimSpace(m) != "" {
			s.VisitedMaps[m] = struct{}{}
		}
	}
}

// ClearPersistedProgress resets all per-map durable progress (e.g. new game).
func (s *Session) ClearPersistedProgress() {
	if s == nil {
		return
	}
	s.Maps = nil
	s.VisitedMaps = nil
}

// NewSession builds an empty session. Callers populate HasSave from disk
// after construction; it's kept out of the constructor to keep Session
// pure data.
func NewSession() *Session { return &Session{} }
