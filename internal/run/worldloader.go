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
package run

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jaredwarren/game-test/internal/dungeon"
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
	// CarryRunState applies portable run state after BuildFromTiled without
	// changing player position (respawn, optional door carry).
	CarryRunState *RunState
	// CarryStatsFromSession copies run state from the outgoing Session.World.
	// Prefer CarryRunState when the snapshot is already captured.
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
	var tm *tiled.Map
	if strings.HasPrefix(mapID, "dun:") {
		seed := int64(42)
		parts := strings.Split(mapID, ":")
		if len(parts) >= 2 {
			if parsed, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				seed = parsed
			}
		}
		if sess != nil && seed != sess.LastDungeonSeed {
			sess.ClearDungeonProgress()
		}
		libFiles := make(map[string][]byte)
		for _, rName := range []string{"start.tmj", "combat.tmj", "key.tmj", "boss.tmj", "corridor.tmj"} {
			data, err := assets.MapData("rooms/" + strings.TrimSuffix(rName, ".tmj"))
			if err == nil {
				libFiles[rName] = data
			}
		}
		lib, err := dungeon.LoadRoomLibrary(libFiles)
		if err != nil {
			return fmt.Errorf("worldloader: load room library for %q: %w", mapID, err)
		}
		genMap, dunResult, err := dungeon.Generate(seed, lib)
		if err != nil {
			return fmt.Errorf("worldloader: generate dungeon %q: %w", mapID, err)
		}
		tm = genMap
		if sess != nil {
			sess.LastDungeonSeed = seed
			sess.LastDungeonDigest = dunResult.BugDigest()
		}
	} else {
		raw, err := assets.MapData(mapID)
		if err != nil {
			return fmt.Errorf("worldloader: read %q: %w", mapID, err)
		}
		parsedMap, err := tiled.ParseMap(raw)
		if err != nil {
			return fmt.Errorf("worldloader: parse %q: %w", mapID, err)
		}
		tm = parsedMap
	}

	carry := opts.CarryRunState
	if carry == nil && opts.CarryStatsFromSession && sess.World != nil {
		rs := RunStateFromWorld(sess.World)
		carry = &rs
	}

	st := progression.DefaultStats()
	if opts.Save != nil {
		st = StatsFromSave(opts.Save)
	} else if carry != nil {
		st = carry.Stats
	}

	if opts.ReplacePersistedProgress {
		if opts.Save != nil {
			sess.Maps = progressMapsFromSave(opts.Save)
		} else {
			sess.Maps = nil
		}
	} else if opts.Save != nil {
		sess.Maps = mergeSaveIntoMaps(sess.Maps, opts.Save)
	}

	progress := sess.ProgressFor(mapID)
	w, err := world.BuildFromTiled(tm, mapID, st, progress.collectedPickupSet())
	if err != nil {
		return fmt.Errorf("worldloader: build %q: %w", mapID, err)
	}

	if carry != nil && (opts.Save == nil || opts.Save.MapID != mapID) {
		carry.ApplyTo(w)
	}
	w.DoorCooldown = w.EffectiveBalance().DoorCooldowns.MapLoadFrames

	sess.World = w
	ApplyMapProgress(w, progress)

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

	if err := EnterMap(assets, sess, target, MapLoadOpts{Save: carry}); err != nil {
		return err
	}
	sess.World.DoorCooldown = sess.World.EffectiveBalance().DoorCooldowns.WarpFrames
	return nil
}

// Respawn warps to configured respawn map while preserving run progress (stats, currency,
// keys, tuning). On error the World is cleared.
func Respawn(assets services.AssetCache, sess *Session) {
	var carry *RunState
	targetMap := "field1"
	if sess.World != nil {
		rs := RunStateFromWorld(sess.World)
		carry = &rs
		targetMap = sess.World.EffectiveBalance().Respawn.MapID
	}
	if err := EnterMap(assets, sess, targetMap, MapLoadOpts{CarryRunState: carry}); err != nil {
		sess.World = nil
		return
	}
	if sess.World != nil {
		resp := sess.World.EffectiveBalance().Respawn
		sess.World.HP = int(float64(sess.World.MaxHP()) * resp.HPFraction)
		if sess.World.HP < resp.HPMinimum {
			sess.World.HP = resp.HPMinimum
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
	RunStateFromSave(s).ApplyTo(sess.World)
	sess.World.Player.X = s.PlayerX
	sess.World.Player.Y = s.PlayerY
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
	gs := RunStateFromWorld(w).ToGameSave(w.MapID, w.Player.X, w.Player.Y)
	gs.Bombs = w.ClampBombsCarry(gs.Bombs)
	if cam != nil {
		gs.ReduceScreenShake = cam.ReduceShake
	}
	if sess != nil {
		flattenMapsToSave(gs, sess.Maps)
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
	if sess != nil && sess.LastDungeonDigest != "" && strings.HasPrefix(mapID, "dun:") {
		fmt.Fprintf(&b, "dungeon=[%s] ", sess.LastDungeonDigest)
	}
	fmt.Fprintf(&b, "save=save.json")
	return strings.TrimSpace(b.String())
}
