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

	if opts.CarryStatsFromSession && sess.World != nil {
		w.HP = sess.World.HP
		w.Currency = sess.World.Currency
		w.Player.MaxBombs = sess.World.Player.MaxBombs
		w.Player.BombFuseDuration = sess.World.Player.BombFuseDuration
		w.Player.BombRadius = sess.World.Player.BombRadius
		w.Player.BombDamage = sess.World.Player.BombDamage
		w.Player.TorchBurnDuration = sess.World.Player.TorchBurnDuration
		w.Player.TorchBurnInterval = sess.World.Player.TorchBurnInterval
		w.Player.TorchBurnDamage = sess.World.Player.TorchBurnDamage
		w.Bombs = w.ClampBombsCarry(sess.World.Bombs)
		w.HasTorch = sess.World.HasTorch
		w.TimeOfDay = sess.World.TimeOfDay
		w.SelectedItem = sess.World.SelectedItem
		w.Player.SwingDuration = sess.World.Player.SwingDuration
		w.Player.MaxSwingCD = sess.World.Player.MaxSwingCD
		w.Player.SwingActiveStart = sess.World.Player.SwingActiveStart
		w.Player.SwingActiveEnd = sess.World.Player.SwingActiveEnd
		w.Player.TorchSwingDuration = sess.World.Player.TorchSwingDuration
		w.Player.MaxTorchSwingCD = sess.World.Player.MaxTorchSwingCD
		w.Player.TorchSwingActiveStart = sess.World.Player.TorchSwingActiveStart
		w.Player.TorchSwingActiveEnd = sess.World.Player.TorchSwingActiveEnd
		w.Player.SprintHeld = sess.World.Player.SprintHeld
		w.Player.SprintExhausted = sess.World.Player.SprintExhausted
		w.Player.BaseSpeed = sess.World.Player.BaseSpeed
		w.Player.SprintSpeed = sess.World.Player.SprintSpeed
		w.Player.DodgeStaminaCost = sess.World.Player.DodgeStaminaCost
		w.Player.DodgeDuration = sess.World.Player.DodgeDuration
		w.Player.DodgeMaxImpulse = sess.World.Player.DodgeMaxImpulse
		w.Player.DodgeSpeed = sess.World.Player.DodgeSpeed
		w.Player.StaminaRegenInterval = sess.World.Player.StaminaRegenInterval
		w.Player.SwordReach = sess.World.Player.SwordReach
		w.Player.SwordThickness = sess.World.Player.SwordThickness
		w.Player.TorchReach = sess.World.Player.TorchReach
		w.Player.TorchThickness = sess.World.Player.TorchThickness
		w.Player.InvulnFrames = sess.World.Player.InvulnFrames
		w.Player.EnemyKnockbackForce = sess.World.Player.EnemyKnockbackForce
		w.Player.PlayerKnockbackForce = sess.World.Player.PlayerKnockbackForce
		w.Player.PlayerHazardKnockbackForce = sess.World.Player.PlayerHazardKnockbackForce
		if w.HP > w.MaxHP() {
			w.HP = w.MaxHP()
		}
	}
	w.DoorCooldown = 60

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
	if s.MaxBombs > 0 {
		w.Player.MaxBombs = s.MaxBombs
	}
	if s.BombFuseDuration > 0 {
		w.Player.BombFuseDuration = s.BombFuseDuration
	}
	if s.BombRadius > 0 {
		w.Player.BombRadius = s.BombRadius
	}
	if s.BombDamage > 0 {
		w.Player.BombDamage = s.BombDamage
	}
	if s.TorchBurnDuration > 0 {
		w.Player.TorchBurnDuration = s.TorchBurnDuration
	}
	if s.TorchBurnInterval > 0 {
		w.Player.TorchBurnInterval = s.TorchBurnInterval
	}
	if s.TorchBurnDamage > 0 {
		w.Player.TorchBurnDamage = s.TorchBurnDamage
	}
	w.Bombs = w.ClampBombsCarry(s.Bombs)
	w.HasTorch = s.HasTorch
	w.Stats = StatsFromSave(s)
	w.SmallKey = s.SmallKey
	if w.SmallKey < 0 {
		w.SmallKey = 0
	}
	if cam != nil {
		cam.ReduceShake = s.ReduceScreenShake
	}
	w.TimeOfDay = s.TimeOfDay
	w.SelectedItem = s.SelectedItem

	if s.SwingDuration > 0 {
		w.Player.SwingDuration = s.SwingDuration
	}
	if s.MaxSwingCD > 0 {
		w.Player.MaxSwingCD = s.MaxSwingCD
	}
	if s.SwingActiveStart > 0 {
		w.Player.SwingActiveStart = s.SwingActiveStart
	}
	if s.SwingActiveEnd > 0 {
		w.Player.SwingActiveEnd = s.SwingActiveEnd
	}
	if s.TorchSwingDuration > 0 {
		w.Player.TorchSwingDuration = s.TorchSwingDuration
	}
	if s.MaxTorchSwingCD > 0 {
		w.Player.MaxTorchSwingCD = s.MaxTorchSwingCD
	}
	if s.TorchSwingActiveStart > 0 {
		w.Player.TorchSwingActiveStart = s.TorchSwingActiveStart
	}
	if s.TorchSwingActiveEnd > 0 {
		w.Player.TorchSwingActiveEnd = s.TorchSwingActiveEnd
	}

	if s.BaseSpeed > 0 {
		w.Player.BaseSpeed = s.BaseSpeed
	}
	if s.SprintSpeed > 0 {
		w.Player.SprintSpeed = s.SprintSpeed
	}
	if s.DodgeStaminaCost > 0 {
		w.Player.DodgeStaminaCost = s.DodgeStaminaCost
	}
	if s.DodgeDuration > 0 {
		w.Player.DodgeDuration = s.DodgeDuration
	}
	if s.DodgeMaxImpulse > 0 {
		w.Player.DodgeMaxImpulse = s.DodgeMaxImpulse
	}
	if s.DodgeSpeed > 0 {
		w.Player.DodgeSpeed = s.DodgeSpeed
	}
	if s.StaminaRegenInterval > 0 {
		w.Player.StaminaRegenInterval = s.StaminaRegenInterval
	}

	if s.SwordReach > 0 {
		w.Player.SwordReach = s.SwordReach
	}
	if s.SwordThickness > 0 {
		w.Player.SwordThickness = s.SwordThickness
	}
	if s.TorchReach > 0 {
		w.Player.TorchReach = s.TorchReach
	}
	if s.TorchThickness > 0 {
		w.Player.TorchThickness = s.TorchThickness
	}
	if s.InvulnFrames > 0 {
		w.Player.InvulnFrames = s.InvulnFrames
	}
	if s.EnemyKnockbackForce > 0 {
		w.Player.EnemyKnockbackForce = s.EnemyKnockbackForce
	}
	if s.PlayerKnockbackForce > 0 {
		w.Player.PlayerKnockbackForce = s.PlayerKnockbackForce
	}
	if s.PlayerHazardKnockbackForce > 0 {
		w.Player.PlayerHazardKnockbackForce = s.PlayerHazardKnockbackForce
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
		Bombs:             w.ClampBombsCarry(w.Bombs),
		HasTorch:          w.HasTorch,
		SmallKey:          w.SmallKey,
		Vitality:          w.Stats.Vitality,
		Resolve:           w.Stats.Resolve,
		Might:             w.Stats.Might,
		Wits:              w.Stats.Wits,
		Fortune:           w.Stats.Fortune,
		ReduceScreenShake: reduce,
		TimeOfDay:         w.TimeOfDay,
		SelectedItem:      w.SelectedItem,
		SwingDuration:          w.Player.SwingDuration,
		MaxSwingCD:             w.Player.MaxSwingCD,
		SwingActiveStart:       w.Player.SwingActiveStart,
		SwingActiveEnd:         w.Player.SwingActiveEnd,
		TorchSwingDuration:     w.Player.TorchSwingDuration,
		MaxTorchSwingCD:        w.Player.MaxTorchSwingCD,
		TorchSwingActiveStart:  w.Player.TorchSwingActiveStart,
		TorchSwingActiveEnd:    w.Player.TorchSwingActiveEnd,
		BaseSpeed:             w.Player.BaseSpeed,
		SprintSpeed:           w.Player.SprintSpeed,
		DodgeStaminaCost:      w.Player.DodgeStaminaCost,
		DodgeDuration:         w.Player.DodgeDuration,
		DodgeMaxImpulse:       w.Player.DodgeMaxImpulse,
		DodgeSpeed:            w.Player.DodgeSpeed,
		StaminaRegenInterval:  w.Player.StaminaRegenInterval,
		SwordReach:                 w.Player.SwordReach,
		SwordThickness:             w.Player.SwordThickness,
		TorchReach:                 w.Player.TorchReach,
		TorchThickness:             w.Player.TorchThickness,
		InvulnFrames:               w.Player.InvulnFrames,
		EnemyKnockbackForce:        w.Player.EnemyKnockbackForce,
		PlayerKnockbackForce:       w.Player.PlayerKnockbackForce,
		PlayerHazardKnockbackForce: w.Player.PlayerHazardKnockbackForce,
		MaxBombs:                   w.Player.MaxBombs,
		BombFuseDuration:           w.Player.BombFuseDuration,
		BombRadius:                 w.Player.BombRadius,
		BombDamage:                 w.Player.BombDamage,
		TorchBurnDuration:          w.Player.TorchBurnDuration,
		TorchBurnInterval:          w.Player.TorchBurnInterval,
		TorchBurnDamage:            w.Player.TorchBurnDamage,
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
	fmt.Fprintf(&b, "save=save.json")
	return strings.TrimSpace(b.String())
}
