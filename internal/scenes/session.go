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

	// CollectedPersistentPickups records Tiled persistent pickup keys already
	// collected this run; mirrored into save.GameSave.CollectedPickupKeys.
	CollectedPersistentPickups map[string]struct{}

	// DestroyedTiles records tiles destroyed by any damage source
	// (bomb, fire, ...); keys from world.MapTilePersistKey, mirrored
	// into save.GameSave.DestroyedTileKeys.
	DestroyedTiles map[string]struct{}

	// OpenedLockTiles records GIDLock tiles opened with a key; keys from
	// world.MapTilePersistKey, mirrored into save.OpenedLockTileKeys.
	OpenedLockTiles map[string]struct{}

	// ActivatedShrines records Tiled shrine keys already activated this run;
	// keys from world.PersistentShrineSaveKey, mirrored into save.GameSave.ActivatedShrines.
	ActivatedShrines map[string]struct{}
}

// MarkPersistentPickupCollected records a consumed persistent pickup (no-op if key empty).
func (s *Session) MarkPersistentPickupCollected(key string) {
	if s == nil || key == "" {
		return
	}
	if s.CollectedPersistentPickups == nil {
		s.CollectedPersistentPickups = make(map[string]struct{})
	}
	s.CollectedPersistentPickups[key] = struct{}{}
}

// ClearPersistedProgress resets persistent pickups, bomb-broken cracked
// walls, and key-opened lock tiles (e.g. new game from title).
func (s *Session) ClearPersistedProgress() {
	if s == nil {
		return
	}
	s.CollectedPersistentPickups = nil
	s.DestroyedTiles = nil
	s.OpenedLockTiles = nil
	s.ActivatedShrines = nil
}

// MarkDestroyedTile records a tile destroyed by any damage source
// (bomb, fire, ...). No-op when key is empty.
func (s *Session) MarkDestroyedTile(key string) {
	if s == nil || key == "" {
		return
	}
	if s.DestroyedTiles == nil {
		s.DestroyedTiles = make(map[string]struct{})
	}
	s.DestroyedTiles[key] = struct{}{}
}

// MarkOpenedLockTile records a GIDLock tile opened with a small key (no-op if key empty).
func (s *Session) MarkOpenedLockTile(key string) {
	if s == nil || key == "" {
		return
	}
	if s.OpenedLockTiles == nil {
		s.OpenedLockTiles = make(map[string]struct{})
	}
	s.OpenedLockTiles[key] = struct{}{}
}

// MarkShrineActivated records an activated shrine (no-op if key empty).
func (s *Session) MarkShrineActivated(key string) {
	if s == nil || key == "" {
		return
	}
	if s.ActivatedShrines == nil {
		s.ActivatedShrines = make(map[string]struct{})
	}
	s.ActivatedShrines[key] = struct{}{}
}

// persistentPickupKeySet returns a copy of keys suitable for BuildFromTiled, or nil if empty.
func (s *Session) persistentPickupKeySet() map[string]struct{} {
	if s == nil || len(s.CollectedPersistentPickups) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(s.CollectedPersistentPickups))
	for k := range s.CollectedPersistentPickups {
		if strings.TrimSpace(k) != "" {
			out[k] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NewSession builds an empty session. Callers populate HasSave from disk
// after construction; it's kept out of the constructor to keep Session
// pure data.
func NewSession() *Session { return &Session{} }
