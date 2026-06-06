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
package scenes

import "github.com/jaredwarren/game-test/internal/world"

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

	// WeeklySeed refreshes from time.Now at the top of every Update; feeds
	// any future weekly-rotation content (leaderboard epochs, event maps).
	// Stored on Session so debug HUD + saves agree on the same epoch for a
	// given frame.
	WeeklySeed int64

	// HasSave indicates whether the disk has a load-able save file. Set
	// once at startup; scenes read it to decide whether to offer quick load.
	HasSave bool

	// ShowDebugOverlay toggles the F3 playtest HUD. Lives on Session (not
	// any one scene) because every scene respects it.
	ShowDebugOverlay bool
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

// ClearPersistedProgress resets all per-map durable progress (e.g. new game).
func (s *Session) ClearPersistedProgress() {
	if s == nil {
		return
	}
	s.Maps = nil
}

// NewSession builds an empty session. Callers populate HasSave from disk
// after construction; it's kept out of the constructor to keep Session
// pure data.
func NewSession() *Session { return &Session{} }
