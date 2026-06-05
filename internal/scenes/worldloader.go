// worldloader.go — map + save orchestration as free functions.
//
// These helpers take a *Session + the AssetCache (for map JSON) + optional
// *render.Camera explicitly. Each helper mutates Session.World atomically
// (success or no-op on error) so callers never see a half-loaded state.
//
// Scope (intentional):
//   - Loading / switching tilemaps (pure .tmj assets — every map runs the
//     same code path; there is no procedural / shell distinction).
//   - Converting save payloads into world state (and back).
//
// Out of scope:
//   - UI / input handling (that's the scenes' job).
//   - Rendering (that's the Renderer's job).
package scenes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/render"
	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/tiled"
	"github.com/jaredwarren/game-test/internal/world"
)

// MapLoadOpts configures EnterMap beyond the map id. Zero value is a
// fine "fresh load, no carry, no save" call.
type MapLoadOpts struct {
	// Save, when non-nil, is applied to the newly-built world after
	// BuildFromTiled if Save.MapID matches the requested mapID. Stats
	// are taken from the save in that case.
	Save *save.GameSave
	// Cam is passed to ApplySave when Save is applied (e.g. load-game
	// shake preference).
	Cam *render.Camera
	// CarryStatsFromSession copies HP, currency, bombs, and stats from
	// the outgoing Session.World (door transitions). When false, stats
	// come from Save if set, otherwise progression defaults.
	CarryStatsFromSession bool
	// ReplacePersistedProgress, when true with a non-nil Save, replaces
	// Session pickup + tile persistence (cracked walls, opened locks) from the save.
	// When false, those save fields are unioned into the session maps.
	ReplacePersistedProgress bool
}

// EnterMap loads mapID from embedded Tiled JSON, builds the world, and
// wires session fields. Optional Save applies when its MapID matches
// mapID. On error Session.World is left as-is.
//
// After a successful load World.DoorCooldown is primed so the player
// cannot immediately re-trigger the door they spawned on.
func EnterMap(assets services.AssetCache, sess *Session, mapID string, opts MapLoadOpts) error {
	raw, err := assets.MapData(mapID)
	if err != nil {
		return fmt.Errorf("worldloader: read %q: %w", mapID, err)
	}
	tm, err := tiled.ParseMap(raw)
	if err != nil {
		return fmt.Errorf("worldloader: parse %q: %w", mapID, err)
	}

	st := progression.DefaultStats()
	if opts.Save != nil {
		st = StatsFromSave(opts.Save)
	} else if opts.CarryStatsFromSession && sess.World != nil {
		st = sess.World.Stats
	}

	if opts.ReplacePersistedProgress {
		if opts.Save != nil {
			sess.CollectedPersistentPickups = stringSetFromSlice(opts.Save.CollectedPickupKeys)
			sess.DestroyedTiles = stringSetFromSlice(opts.Save.DestroyedTileKeys)
			sess.OpenedLockTiles = stringSetFromSlice(opts.Save.OpenedLockTileKeys)
		} else {
			sess.CollectedPersistentPickups = nil
			sess.DestroyedTiles = nil
			sess.OpenedLockTiles = nil
		}
	} else if opts.Save != nil {
		if len(opts.Save.CollectedPickupKeys) > 0 {
			mergePickupKeys(sess, opts.Save.CollectedPickupKeys)
		}
		if len(opts.Save.DestroyedTileKeys) > 0 {
			mergeDestroyedTileKeys(sess, opts.Save.DestroyedTileKeys)
		}
		if len(opts.Save.OpenedLockTileKeys) > 0 {
			mergeOpenedLockKeys(sess, opts.Save.OpenedLockTileKeys)
		}
	}

	w, err := world.BuildFromTiled(tm, mapID, st, sess.persistentPickupKeySet())
	if err != nil {
		return fmt.Errorf("worldloader: build %q: %w", mapID, err)
	}

	if opts.CarryStatsFromSession && sess.World != nil {
		w.HP = sess.World.HP
		w.Currency = sess.World.Currency
		w.Bombs = world.ClampBombsCarry(sess.World.Bombs)
		w.HasTorch = sess.World.HasTorch
		if w.HP > w.MaxHP() {
			w.HP = w.MaxHP()
		}
	}
	w.DoorCooldown = 60

	sess.World = w
	world.ApplyDestroyedTiles(w, sess.DestroyedTiles)
	world.ApplyPersistedOpenedLocks(w, sess.OpenedLockTiles)

	if opts.Save != nil && opts.Save.MapID == mapID {
		ApplySave(sess, opts.Save, opts.Cam)
	}
	return nil
}

// OpenWorld replaces Session.World with a freshly built map by logical id.
// If s is non-nil, stats are taken from the save; when s.MapID matches mapID,
// position and run state are applied.
func OpenWorld(assets services.AssetCache, sess *Session, mapID string, s *save.GameSave) error {
	return EnterMap(assets, sess, mapID, MapLoadOpts{Save: s})
}

// playerHitboxH returns the outgoing world's player height for door math, or
// the engine default if the world is missing or corrupt.
func playerHitboxH(w *world.World) float64 {
	if w != nil && w.Player.H > 0 {
		return w.Player.H
	}
	return world.DefaultPlayerHitboxH
}

// WarpDoor transitions through a door into another map, carrying the
// current save payload. Returns error on map load failure; on success
// the new World is live.
//
// Player placement uses world.PlayerTopLeftFromDoorSpawn with the door's
// SpawnStyle (from optional Tiled spawn_anchor) and the outgoing player's
// hitbox height so door spawns stay aligned with BuildFromTiled "spawn" markers.
func WarpDoor(assets services.AssetCache, sess *Session, cam *render.Camera, d *world.Door) error {
	target := d.TargetMap
	carry := BuildSave(sess, cam)
	if carry == nil {
		return fmt.Errorf("worldloader: warp %q: no world to carry from", target)
	}
	ph := playerHitboxH(sess.World)
	px, py := world.PlayerTopLeftFromDoorSpawn(d.SpawnX, d.SpawnY, d.SpawnStyle, ph)
	carry.MapID = target
	carry.PlayerX = px
	carry.PlayerY = py

	if err := EnterMap(assets, sess, target, MapLoadOpts{Save: carry, CarryStatsFromSession: true}); err != nil {
		return err
	}
	sess.World.DoorCooldown = 90
	return nil
}

// Respawn is the current death handler: warp to field1 with partial HP. On
// error the World is cleared (title screen state).
func Respawn(assets services.AssetCache, sess *Session) {
	if err := OpenWorld(assets, sess, "field1", nil); err != nil {
		sess.World = nil
		return
	}
	if sess.World != nil {
		sess.World.HP = sess.World.MaxHP() / 2
		if sess.World.HP < 2 {
			sess.World.HP = 2
		}
	}
}

// TryDoors checks if the player overlaps any door and warps through it.
// No-op while World.DoorCooldown > 0 so freshly-spawned players don't re-
// trigger their entry door on the next frame.
func TryDoors(assets services.AssetCache, sess *Session, cam *render.Camera) {
	w := sess.World
	if w == nil || w.DoorCooldown > 0 {
		return
	}
	pr := w.PlayerRect()
	for i := range w.Doors {
		d := &w.Doors[i]
		if pr.Overlaps(d.Rect) {
			_ = WarpDoor(assets, sess, cam, d)
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Save / load helpers
// ---------------------------------------------------------------------------

// StatsFromSave copies progression fields out of a save payload, clamping
// each to a sane minimum so a corrupt save can't produce a zero-HP player.
func StatsFromSave(s *save.GameSave) progression.Stats {
	st := progression.DefaultStats()
	if s == nil {
		return st
	}
	st.Vitality = max(1, s.Vitality)
	st.Resolve = max(1, s.Resolve)
	st.Might = max(1, s.Might)
	st.Wits = max(0, s.Wits)
	st.Fortune = max(0, s.Fortune)
	return st
}

// ApplySave overwrites the current World's per-run state (position, HP,
// currency, stats, keys, bomb count) from a save payload. Updates the passed
// Camera's ReduceShake preference if non-nil.
func ApplySave(sess *Session, s *save.GameSave, cam *render.Camera) {
	if sess.World == nil || s == nil {
		return
	}
	w := sess.World
	w.Player.X = s.PlayerX
	w.Player.Y = s.PlayerY
	w.HP = s.HP
	if w.HP > w.MaxHP() {
		w.HP = w.MaxHP()
	}
	w.Currency = s.Currency
	w.Bombs = world.ClampBombsCarry(s.Bombs)
	w.HasTorch = s.HasTorch
	w.Stats = StatsFromSave(s)
	w.SmallKey = s.SmallKey
	if w.SmallKey < 0 {
		w.SmallKey = 0
	}
	if cam != nil {
		cam.ReduceShake = s.ReduceScreenShake
	}
}

// BuildSave snapshots the current World + Session into a save payload. The
// camera's ReduceShake preference is captured so "player turned off screen
// shake" survives a reload.
func BuildSave(sess *Session, cam *render.Camera) *save.GameSave {
	w := sess.World
	if w == nil {
		return nil
	}
	reduce := false
	if cam != nil {
		reduce = cam.ReduceShake
	}
	gs := &save.GameSave{
		MapID:             w.MapID,
		PlayerX:           w.Player.X,
		PlayerY:           w.Player.Y,
		HP:                w.HP,
		Currency:          w.Currency,
		Bombs:             world.ClampBombsCarry(w.Bombs),
		HasTorch:          w.HasTorch,
		SmallKey:          w.SmallKey,
		Vitality:          w.Stats.Vitality,
		Resolve:           w.Stats.Resolve,
		Might:             w.Stats.Might,
		Wits:              w.Stats.Wits,
		Fortune:           w.Stats.Fortune,
		ReduceScreenShake: reduce,
	}
	if sess != nil && len(sess.CollectedPersistentPickups) > 0 {
		keys := make([]string, 0, len(sess.CollectedPersistentPickups))
		for k := range sess.CollectedPersistentPickups {
			if strings.TrimSpace(k) != "" {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		gs.CollectedPickupKeys = keys
	}
	if sess != nil && len(sess.DestroyedTiles) > 0 {
		keys := make([]string, 0, len(sess.DestroyedTiles))
		for k := range sess.DestroyedTiles {
			if strings.TrimSpace(k) != "" {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		gs.DestroyedTileKeys = keys
	}
	if sess != nil && len(sess.OpenedLockTiles) > 0 {
		keys := make([]string, 0, len(sess.OpenedLockTiles))
		for k := range sess.OpenedLockTiles {
			if strings.TrimSpace(k) != "" {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		gs.OpenedLockTileKeys = keys
	}
	return gs
}

// LoadGameFromSave restores a run: rebuilds the map from assets and
// applies the save payload.
func LoadGameFromSave(assets services.AssetCache, sess *Session, cam *render.Camera, s *save.GameSave) error {
	if s == nil {
		return fmt.Errorf("worldloader: nil save")
	}
	return EnterMap(assets, sess, s.MapID, MapLoadOpts{Save: s, Cam: cam, ReplacePersistedProgress: true})
}

func stringSetFromSlice(keys []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			out[k] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergePickupKeys(sess *Session, keys []string) {
	if sess == nil {
		return
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if sess.CollectedPersistentPickups == nil {
			sess.CollectedPersistentPickups = make(map[string]struct{})
		}
		sess.CollectedPersistentPickups[k] = struct{}{}
	}
}

func mergeDestroyedTileKeys(sess *Session, keys []string) {
	if sess == nil {
		return
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if sess.DestroyedTiles == nil {
			sess.DestroyedTiles = make(map[string]struct{})
		}
		sess.DestroyedTiles[k] = struct{}{}
	}
}

func mergeOpenedLockKeys(sess *Session, keys []string) {
	if sess == nil {
		return
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if sess.OpenedLockTiles == nil {
			sess.OpenedLockTiles = make(map[string]struct{})
		}
		sess.OpenedLockTiles[k] = struct{}{}
	}
}

// BugDigest packs map, health, weekly epoch, and save path for pasting
// into bug reports.
func BugDigest(sess *Session) string {
	mapID := "none"
	w := sess.World
	if w != nil {
		mapID = w.MapID
	}
	var b strings.Builder
	fmt.Fprintf(&b, "map=%s ", mapID)
	if w != nil {
		fmt.Fprintf(&b, "hp=%d/%d ", w.HP, w.MaxHP())
	}
	fmt.Fprintf(&b, "weekly=%d ", sess.WeeklySeed)
	fmt.Fprintf(&b, "save=save.json")
	return strings.TrimSpace(b.String())
}
