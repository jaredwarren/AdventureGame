// Package world is the gameplay simulation layer: player movement, sword hits,
// pickups, and simple enemy AI. It deliberately knows nothing about Ebiten—only
// numbers and structs—so you can unit-test logic without graphics.
//
// Domain-specific tables live in subpackages (import them directly when adding
// new content):
//   - world/tile   — GIDs, collision defs, tile persist keys
//   - world/pickup — collectible kinds and rewards
//   - world/enemy  — per-enemy Tiled tuning
//
// The root world package re-exports common symbols via aliases.go so existing
// callers can keep using world.GIDGrass, world.PickupCoin, etc.
//
// Coordinate conventions:
//   - Tile grid: origin top-left, tile size [TileSize] pixels (must match Tiled export).
//   - Player/enemy positions are pixel AABBs (upper-left of hitbox).
//
// Temporary / improvement notes:
//   - Collision uses 5-point sampling on the player rect (corners + center)—fast but can miss thin walls;
//     consider continuous collision or capsule if tunnels get narrower than the probe spacing.
//   - Sword hitbox timing (Swing countdown + “active window”) is hand-tuned feel, not derived from animation.
//   - Enemy chase uses tile A* (see astar.go) toward the player with axis slide; no enemy–enemy avoidance.
//   - Contact damage uses separate cooldowns on player invuln vs enemy HurtCD—tweak together for fairness.
package world

import (
	"math"

	"github.com/jaredwarren/game-test/internal/balance"
	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/world/tile"
)

// ItemSlot identifies which item is currently active in the player's secondary slot.
type ItemSlot int

const (
	ItemSlotBomb ItemSlot = iota // default
	ItemSlotTorch
)

// MaxBombsCarry returns the player's effective bomb inventory cap.
func (w *World) MaxBombsCarry() int {
	return w.Player.EffectiveMaxBombs()
}

// ClampBombsCarry clamps n to [0, MaxBombsCarry] for save load and pickups.
func (w *World) ClampBombsCarry(n int) int {
	if n < 0 {
		return 0
	}
	maxB := w.MaxBombsCarry()
	if n > maxB {
		return maxB
	}
	return n
}

// Pickup is a ground item the player can collect by overlap. Transform +
// Hitbox give it its AABB; Kind drives the on-pickup effect in
// systems.PickupSystem.
type Pickup struct {
	ID EntityID
	Transform
	Hitbox

	Kind           *PickupKind
	Gone           bool // set true when consumed; kept in the slice for stable indices
	Opened         bool // set true when chest is opened
	PendingCollect bool // set true when chest is opened by player intent until PickupSystem processes it

	// PersistentSaveKey is non-empty when this pickup was authored with
	// persistent=true in Tiled; format from PersistentPickupSaveKey. Consumed
	// pickups stay absent across map reloads while this key remains in save.
	PersistentSaveKey string
}

// Enemy is one combat unit with simple homing AI. Components:
//
//   - Transform / Hitbox — position + AABB
//   - Health            — HP + MaxHP
//   - AI                — seek speed
//   - ContactHurt       — on-touch damage + cooldown
//
// IsBoss flags the run's boss (used for the kill bonus); the field could
// migrate to its own marker component once we have more than one role.
type Enemy struct {
	ID EntityID
	Transform
	Hitbox
	Health
	AI AIHomer
	ContactHurt

	// BurnTimer counts down while the enemy is ignited (torch DoT).
	// BurnCD counts down to the next burn tick while BurnTimer > 0.
	BurnTimer int
	BurnCD    int

	Dir    Dir // facing direction for rendering
	IsBoss bool
}

// Door is an authored map transition marker. Rect is baked by Tiled so it
// doesn't use Transform/Hitbox (shape can be non-entity-sized).
//
// SpawnX/SpawnY are interpreted using SpawnStyle (default DoorSpawnFeet).
// Optional Tiled property spawn_anchor: "feet" (default) or "topleft".
type Door struct {
	ID   EntityID
	Rect geom.Rect

	TargetMap  string
	SpawnX     float64
	SpawnY     float64
	SpawnStyle DoorSpawnStyle
}

// Shrine is an authored interactable location.
type Shrine struct {
	ID      EntityID
	TiledID int
	Rect    geom.Rect
	Active  bool
}

// Sign is an authored interactable wooden sign.
type Sign struct {
	ID   EntityID
	Rect geom.Rect
	Text string
}

// Player holds locomotion + combat timers (frame counts, not seconds).
// Player-only state (Swing, Dodge, Stamina) stays inline rather than as
// one-off components; those concerns aren't shared with other entities.
// SprintHeld reserves a future "toggle sprint" UX; today stamina drain
// is keyed off Shift in scenes.PlayScene.
type Player struct {
	ID EntityID
	Transform
	Hitbox
	Facing

	Swing           int // >0 while sword animation runs; only mid-window counts as hit frames (see SwordHitbox)
	SwingCD         int // frames until another Z press is accepted
	TorchSwing      int // >0 while torch swing runs; same active window as sword (see TorchHitbox)
	TorchSwingCD    int // frames until another torch press is accepted
	Invuln          int // i-frames after taking damage
	DodgeTimer      int // if >0, enemy contact check skipped (paired with scene dodgeImpulse nudge)
	Stamina         int // drained by sprint; refilled when not sprinting
	SprintHeld             bool
	SprintExhausted        bool
	IsMoving               bool
	IsSprinting            bool
	WasOnQuicksand         bool
	RunningAcrossQuicksand bool
	VX                     float64
	VY                     float64

	balance.PlayerTuning
}

type World struct {
	MapID string
	TileW int
	TileH int
	MapW  int
	MapH  int

	// Layers holds dynamic 2D tile layers [layerIndex][tileIndex] (bottom-to-top order).
	// Tiles is kept as a convenience reference to the top active layer.
	Layers [][]int
	Tiles  []int

	// ActiveLayerFilter when >= 0 restricts rendering to only that layer index. -1 shows all layers.
	ActiveLayerFilter int

	// DestroyedTiles records tiles that have been broken/opened by damage
	// (bombs, fire, ...) keyed by tile index. Rendering and collision
	// treat these tiles as removed, revealing the layer below.
	DestroyedTiles map[int]bool
	Doors          []Door
	Shrines        []Shrine
	Signs          []Sign
	Pickups        []Pickup
	IsEditor       bool // set true when running in editor mode
	Enemies        []Enemy

	Player   Player
	Stats    progression.Stats
	HP       int
	Currency int
	// Bombs is how many bomb items the player holds; pickups increment (capped at
	// Player.MaxBombs via MaxBombsCarry), placing a bomb decrements (see TryDamageFaceTile(DamageBomb)).
	Bombs           int
	HasTorch        bool
	HasPegasusBoots bool
	OwnedItems      map[string]bool
	ActiveBuffs     []Buff
	SmallKey        int
	SelectedItem    ItemSlot

	// Tick is the sim-frame counter; incremented by systems.TimersSystem so
	// other systems can key periodic effects off a world-owned clock.
	Tick int

	// TimeOfDay is the current frame in the 14,400 frame cycle (4 minutes).
	TimeOfDay int

	// HasAmbientLightOverride is true if the map specified a custom light level override
	HasAmbientLightOverride bool
	AmbientLightOverride    float64

	// Flames is the list of currently active fires burning tiles (e.g. trees)
	Flames []ActiveFlame

	// ActiveBombs is the list of placed bombs counting down fuse timers
	ActiveBombs []ActiveBomb

	// DoorCooldown counts down while >0 (systems.TimersSystem); while non-
	// zero, scenes should skip door warp checks so the player does not
	// immediately re-trigger the door they spawned on after a map
	// transition.
	DoorCooldown int

	// Balance is the active GameBalance configuration; when nil, EffectiveBalance
	// falls back to balance.Default().
	Balance *balance.GameBalance

	// nextID feeds allocID(). Unexported so external code can't mint IDs
	// out of band; use the Spawn* methods instead.
	nextID uint32
}

// EffectiveBalance returns w.Balance if non-nil, or balance.Default() otherwise.
func (w *World) EffectiveBalance() *balance.GameBalance {
	if w != nil && w.Balance != nil {
		return w.Balance
	}
	return balance.Default()
}

func (w *World) gidAt(tx, ty int) int {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return tile.GIDWall
	}
	idx := ty*w.MapW + tx
	if len(w.Layers) > 0 {
		for k := len(w.Layers) - 1; k >= 0; k-- {
			if idx < len(w.Layers[k]) {
				gid := w.Layers[k][idx]
				if gid != tile.GIDEmpty && !w.DestroyedTiles[idx] {
					return gid
				}
				if gid != tile.GIDEmpty && k == 0 {
					return gid
				}
			}
		}
	}
	if idx < len(w.Tiles) {
		return w.Tiles[idx]
	}
	return tile.GIDEmpty
}

// GIDAt returns the tile GID at tile coordinates (for debug / tools).
func (w *World) GIDAt(tx, ty int) int {
	return w.gidAt(tx, ty)
}

// SurfaceAtFeet returns the SurfaceDef for the tile at world position (px, py).
func (w *World) SurfaceAtFeet(px, py float64) tile.SurfaceDef {
	if w == nil || w.TileW <= 0 || w.TileH <= 0 {
		return tile.DefaultSurface
	}
	tx := int(px / float64(w.TileW))
	ty := int(py / float64(w.TileH))
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return tile.DefaultSurface
	}
	idx := ty*w.MapW + tx
	if len(w.Layers) > 0 {
		for k := len(w.Layers) - 1; k >= 0; k-- {
			if idx < len(w.Layers[k]) {
				gid := w.Layers[k][idx]
				if gid != tile.GIDEmpty {
					if w.DestroyedTiles[idx] {
						if def := tile.DefOf(gid); def.Destroyable() || def.OpenableByKey {
							continue
						}
					}
					return tile.SurfaceForGID(gid)
				}
			}
		}
	}
	gid := w.gidAt(tx, ty)
	return tile.SurfaceForGID(gid)
}

func (w *World) tileIndex(tx, ty int) int {
	return ty*w.MapW + tx
}

func (w *World) solidTile(tx, ty int) bool {
	g := w.gidAt(tx, ty)
	idx := w.tileIndex(tx, ty)
	return tile.FullySolidAt(g, idx, w.DestroyedTiles, w.SmallKey > 0)
}

// RectHitsSolid returns true if rect overlaps any solid tile or sub-tile solid region across all layers.
func (w *World) RectHitsSolid(r geom.Rect) bool {
	minTX := int(math.Floor(r.X / tile.Size))
	maxTX := int(math.Floor((r.X + r.W - 0.000001) / tile.Size))
	minTY := int(math.Floor(r.Y / tile.Size))
	maxTY := int(math.Floor((r.Y + r.H - 0.000001) / tile.Size))

	for ty := minTY; ty <= maxTY; ty++ {
		for tx := minTX; tx <= maxTX; tx++ {
			if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
				tileRect := geom.Rect{X: float64(tx * tile.Size), Y: float64(ty * tile.Size), W: tile.Size, H: tile.Size}
				if r.Overlaps(tileRect) {
					return true
				}
				continue
			}
			idx := w.tileIndex(tx, ty)
			if len(w.Layers) > 0 {
				for k := 0; k < len(w.Layers); k++ {
					if idx < len(w.Layers[k]) {
						gid := w.Layers[k][idx]
						if gid == tile.GIDEmpty {
							continue
						}
						solidRects := tile.SolidRectsAt(gid, idx, w.DestroyedTiles, w.SmallKey > 0)
						for _, sr := range solidRects {
							worldSR := geom.Rect{
								X: float64(tx*tile.Size) + sr.X,
								Y: float64(ty*tile.Size) + sr.Y,
								W: sr.W,
								H: sr.H,
							}
							if r.Overlaps(worldSR) {
								return true
							}
						}
					}
				}
			} else {
				g := w.gidAt(tx, ty)
				solidRects := tile.SolidRectsAt(g, idx, w.DestroyedTiles, w.SmallKey > 0)
				for _, sr := range solidRects {
					worldSR := geom.Rect{
						X: float64(tx*tile.Size) + sr.X,
						Y: float64(ty*tile.Size) + sr.Y,
						W: sr.W,
						H: sr.H,
					}
					if r.Overlaps(worldSR) {
						return true
					}
				}
			}
		}
	}
	return false
}

func slideAABBAgnostic(x, y, ww, hh, dx, dy float64, blocked func(geom.Rect) bool) (float64, float64) {
	full := geom.Rect{X: x + dx, Y: y + dy, W: ww, H: hh}
	if !blocked(full) {
		return x + dx, y + dy
	}
	nx, ny := x, y
	if dx != 0 {
		if !blocked(geom.Rect{X: x + dx, Y: y, W: ww, H: hh}) {
			nx = x + dx
		}
	}
	if dy != 0 {
		if !blocked(geom.Rect{X: nx, Y: y + dy, W: ww, H: hh}) {
			ny = y + dy
		}
	}
	return nx, ny
}

// SlideAABB tries to move a rectangle by (dx, dy) with axis-separated sliding
// on solid tiles only (e.g. knockback that should not consider entities).
func (w *World) SlideAABB(x, y, ww, hh, dx, dy float64) (float64, float64) {
	return slideAABBAgnostic(x, y, ww, hh, dx, dy, w.RectHitsSolid)
}

// RectOverlapsAnyLiveEnemy reports whether r overlaps any enemy with HP > 0.
func (w *World) RectOverlapsAnyLiveEnemy(r geom.Rect) bool {
	for i := range w.Enemies {
		e := &w.Enemies[i]
		if e.HP <= 0 {
			continue
		}
		if r.Overlaps(e.Rect()) {
			return true
		}
	}
	return false
}

// RectOverlapsAnyClosedChest reports whether r overlaps any closed chest pickup.
func (w *World) RectOverlapsAnyClosedChest(r geom.Rect) bool {
	for i := range w.Pickups {
		p := &w.Pickups[i]
		if p.PersistentSaveKey != "" && !p.Opened && !p.Gone {
			if r.Overlaps(p.Rect()) {
				return true
			}
		}
	}
	return false
}

// SlidePlayerAABB slides the player's would-be AABB against solids, live
// enemies, and closed chests (no enemy-enemy blocking here).
func (w *World) SlidePlayerAABB(x, y, ww, hh, dx, dy float64) (float64, float64) {
	return slideAABBAgnostic(x, y, ww, hh, dx, dy, func(r geom.Rect) bool {
		return w.RectHitsSolid(r) || w.RectOverlapsAnyLiveEnemy(r) || w.RectOverlapsAnyClosedChest(r)
	})
}

// SlideEnemyAABB slides an enemy AABB against solids, the player hitbox, and closed chests.
func (w *World) SlideEnemyAABB(x, y, ww, hh, dx, dy float64) (float64, float64) {
	pr := w.PlayerRect()
	return slideAABBAgnostic(x, y, ww, hh, dx, dy, func(r geom.Rect) bool {
		return w.RectHitsSolid(r) || r.Overlaps(pr) || w.RectOverlapsAnyClosedChest(r)
	})
}

// LegalizeEnemyOutOfPlayer nudges a live enemy that overlaps the player until
// separated or attempts are exhausted (spawn overlap, knockback quirks).
func (w *World) LegalizeEnemyOutOfPlayer(e *Enemy) {
	if e.HP <= 0 {
		return
	}
	pr := w.PlayerRect()
	pcx, pcy := w.Player.Center()
	for iter := 0; iter < 32; iter++ {
		if !e.Rect().Overlaps(pr) {
			return
		}
		tcx := e.X + e.W*0.5
		tcy := e.Y + e.H*0.5
		dx := tcx - pcx
		dy := tcy - pcy
		d := math.Hypot(dx, dy)
		const step = 2.5
		var nx, ny float64
		if d < 1e-6 {
			nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, step, 0)
			if nx == e.X && ny == e.Y {
				nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, 0, step)
			}
		} else {
			nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, dx/d*step, dy/d*step)
		}
		if nx == e.X && ny == e.Y {
			nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, step, 0)
			if nx == e.X && ny == e.Y {
				nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, -step, 0)
			}
			if nx == e.X && ny == e.Y {
				nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, 0, step)
			}
			if nx == e.X && ny == e.Y {
				nx, ny = w.SlideEnemyAABB(e.X, e.Y, e.W, e.H, 0, -step)
			}
		}
		e.X, e.Y = nx, ny
	}
}

func (w *World) TryMove(dx, dy float64) {
	nx, ny := w.SlidePlayerAABB(w.Player.X, w.Player.Y, w.Player.W, w.Player.H, dx, dy)
	w.Player.X, w.Player.Y = nx, ny
}

func (w *World) SetFacingFromMotion(dx, dy float64) {
	if math.Abs(dx) > math.Abs(dy) {
		if dx > 0 {
			w.Player.Dir = DirRight
		} else if dx < 0 {
			w.Player.Dir = DirLeft
		}
	} else {
		if dy > 0 {
			w.Player.Dir = DirDown
		} else if dy < 0 {
			w.Player.Dir = DirUp
		}
	}
}

func swordSwingHitbox(swing int, activeStart, activeEnd int, reach, thick float64, dir Dir, px, py float64) (geom.Rect, bool) {
	if swing <= 0 {
		return geom.Rect{}, false
	}
	active := swing >= activeStart && swing <= activeEnd
	if !active {
		return geom.Rect{}, false
	}
	switch dir {
	case DirDown:
		return geom.Rect{X: px - thick*0.5, Y: py, W: thick, H: reach}, true
	case DirUp:
		return geom.Rect{X: px - thick*0.5, Y: py - reach, W: thick, H: reach}, true
	case DirLeft:
		return geom.Rect{X: px - reach, Y: py - thick*0.5, W: reach, H: thick}, true
	default: // DirRight
		return geom.Rect{X: px, Y: py - thick*0.5, W: reach, H: thick}, true
	}
}

// SwordHitbox returns the sword arc AABB during the active swing frames.
func (w *World) SwordHitbox() (geom.Rect, bool) {
	px := w.Player.X + w.Player.W*0.5
	py := w.Player.Y + w.Player.H*0.5
	return swordSwingHitbox(
		w.Player.Swing,
		w.Player.EffectiveSwingActiveStart(),
		w.Player.EffectiveSwingActiveEnd(),
		w.Player.EffectiveSwordReach(),
		w.Player.EffectiveSwordThickness(),
		w.Player.Dir, px, py,
	)
}

// TorchHitbox matches SwordHitbox timing/geometry but keys off TorchSwing.
func (w *World) TorchHitbox() (geom.Rect, bool) {
	px := w.Player.X + w.Player.W*0.5
	py := w.Player.Y + w.Player.H*0.5
	return swordSwingHitbox(
		w.Player.TorchSwing,
		w.Player.EffectiveTorchSwingActiveStart(),
		w.Player.EffectiveTorchSwingActiveEnd(),
		w.Player.EffectiveTorchReach(),
		w.Player.EffectiveTorchThickness(),
		w.Player.Dir, px, py,
	)
}

// Rect returns the enemy's AABB using its Transform + Hitbox components.
func (e Enemy) Rect() geom.Rect { return geom.Rect{X: e.X, Y: e.Y, W: e.W, H: e.H} }

// Center returns the enemy's AABB center point.
func (e Enemy) Center() (float64, float64) { return e.X + e.W*0.5, e.Y + e.H*0.5 }

// Rect returns the player's AABB.
func (p Player) Rect() geom.Rect { return geom.Rect{X: p.X, Y: p.Y, W: p.W, H: p.H} }

// Center returns the player's AABB center point.
func (p Player) Center() (float64, float64) { return p.X + p.W*0.5, p.Y + p.H*0.5 }

// Rect returns the pickup's AABB.
func (p Pickup) Rect() geom.Rect { return geom.Rect{X: p.X, Y: p.Y, W: p.W, H: p.H} }

// DamageEnemy applies dmg to the enemy at slice index i. Zero-clamped.
// No-op when the index is out of range or the enemy is already dead.
func (w *World) DamageEnemy(i int, dmg int) {
	if i < 0 || i >= len(w.Enemies) {
		return
	}
	e := &w.Enemies[i]
	if e.HP <= 0 {
		return
	}
	e.HP -= dmg
	if e.HP < 0 {
		e.HP = 0
	}
}

// EnemyRect returns the AABB of the enemy at slice index i.
func (w *World) EnemyRect(i int) geom.Rect { return w.Enemies[i].Rect() }

// PlayerRect returns the player's AABB.
func (w *World) PlayerRect() geom.Rect { return w.Player.Rect() }

func (w *World) MaxHP() int {
	return w.Stats.MaxHP()
}

func (w *World) MaxStamina() int {
	return w.Stats.MaxStamina()
}

func (w *World) SwordDamage() int {
	return w.Stats.SwordDamage()
}

// IgniteEnemy extends or starts torch burn on the enemy at index i.
func (w *World) IgniteEnemy(i int) {
	if i < 0 || i >= len(w.Enemies) {
		return
	}
	e := &w.Enemies[i]
	if e.HP <= 0 {
		return
	}
	burnDuration := w.Player.EffectiveTorchBurnDuration()
	burnInterval := w.Player.EffectiveTorchBurnInterval()
	if e.BurnTimer < burnDuration {
		e.BurnTimer = burnDuration
	}
	if e.BurnCD <= 0 {
		e.BurnCD = burnInterval
	}
}

// ConvertLockToFloor swaps a single GIDLock tile for walkable floor.
// Returns true when the tile at (tx, ty) was a lock and was converted;
// false (no mutation) otherwise. Out-of-bounds coordinates are a no-op.
//
// This is the structural primitive systems.LockSystem uses; keeping the
// Tile[] index math inside World preserves the invariant that external
// code never writes w.Tiles directly.
func (w *World) ConvertLockToFloor(tx, ty int) bool {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return false
	}
	idx := w.tileIndex(tx, ty)
	if w.Tiles[idx] != tile.GIDLock {
		return false
	}
	w.Tiles[idx] = tile.GIDGrass
	return true
}

// TrySwingSword starts a sword swing if not already swinging / on cooldown.
func (w *World) TrySwingSword() bool {
	if w.Player.Swing > 0 || w.Player.SwingCD > 0 {
		return false
	}
	if w.Player.TorchSwing > 0 || w.Player.TorchSwingCD > 0 {
		return false
	}
	w.Player.Swing = w.Player.EffectiveSwingDuration()
	w.Player.SwingCD = w.Player.EffectiveMaxSwingCD()
	return true
}

// TrySwingTorch starts a torch swing if the player has a torch and no
// sword or torch animation is already running.
func (w *World) TrySwingTorch() bool {
	if !w.HasItem("torch") {
		return false
	}
	if w.Player.TorchSwing > 0 || w.Player.TorchSwingCD > 0 {
		return false
	}
	if w.Player.Swing > 0 || w.Player.SwingCD > 0 {
		return false
	}
	w.Player.TorchSwing = w.Player.EffectiveTorchSwingDuration()
	w.Player.TorchSwingCD = w.Player.EffectiveMaxTorchSwingCD()
	return true
}

// TryDamageFaceTile applies the given damage kind to the tile directly
// in front of the player. On success it marks the tile destroyed and
// returns MapTilePersistKey for session/save persistence. A tile is
// only broken when its TileDef.DamageKinds contains kind, so bombs do
// not harm fire-only tiles (e.g. trees) and vice versa.
//
// DamageBomb requires Bombs > 0 and decrements Bombs on success.
//
// NOTE: Uses player center for the "from" tile math, then steps one
// tile in Dir—works for current hitbox sizes.
func (w *World) TryDamageFaceTile(kind tile.DamageKind) (ok bool, saveKey string) {
	if kind == tile.DamageBomb && w.Bombs <= 0 {
		return false, ""
	}
	pc := w.PlayerRect()
	cx := pc.X + pc.W*0.5
	cy := pc.Y + pc.H*0.5
	tx := int(cx / tile.Size)
	ty := int(cy / tile.Size)
	switch w.Player.Dir {
	case DirDown:
		ty++
	case DirUp:
		ty--
	case DirLeft:
		tx--
	case DirRight:
		tx++
	}
	idx := w.tileIndex(tx, ty)
	if w.DestroyedTiles != nil && w.DestroyedTiles[idx] {
		return false, ""
	}
	g := w.gidAt(tx, ty)
	def := tile.DefOf(g)
	if !def.AcceptsDamage(kind) {
		return false, ""
	}
	if w.DestroyedTiles == nil {
		w.DestroyedTiles = make(map[int]bool)
	}
	w.DestroyedTiles[idx] = true
	if kind == tile.DamageBomb {
		w.Bombs--
		if w.Bombs < 0 {
			w.Bombs = 0
		}
	}
	return true, tile.MapTilePersistKey(w.MapID, tx, ty)
}

// ShrineHeal is the “poor” shrine interaction (no coins). Rich interaction is priced in the shop.
func (w *World) ShrineHeal() {
	heal := w.EffectiveBalance().Economy.ShrineHealAmount
	if heal <= 0 || w.HP >= w.MaxHP() {
		return
	}
	w.HP += heal
	if w.HP > w.MaxHP() {
		w.HP = w.MaxHP()
	}
}

// UpgradeRandomStat picks one of the five progression knobs uniformly via rng(n).
// TODO: weight by build / offer UI choice instead of pure random; hook Wits/Fortune into real systems.
func (w *World) UpgradeRandomStat(rng func(int) int) {
	choices := []func(){
		func() { w.Stats.Vitality++ },
		func() { w.Stats.Resolve++ },
		func() { w.Stats.Might++ },
		func() { w.Stats.Wits++ },
		func() { w.Stats.Fortune++ },
	}
	choices[rng(5)]()
	if w.HP > w.MaxHP() {
		w.HP = w.MaxHP()
	}
}

const CycleLength = 14400

func (w *World) IsNight() bool {
	dn := w.EffectiveBalance().DayNight
	return w.TimeOfDay < dn.NightEndTick || w.TimeOfDay >= dn.NightStartTick
}

func (w *World) LightMultiplier() float64 {
	if w.HasAmbientLightOverride {
		return w.AmbientLightOverride
	}
	dn := w.EffectiveBalance().DayNight
	t := float64(w.TimeOfDay)
	if t >= 0 && t < float64(dn.NightEndTick) {
		return dn.MinAmbientLight + (t/float64(dn.NightEndTick))*(1.0-dn.MinAmbientLight)
	}
	if t >= float64(dn.NightEndTick) && t < float64(dn.DuskStartTick) {
		return 1.0
	}
	duskDuration := float64(dn.DuskEndTick - dn.DuskStartTick)
	if duskDuration > 0 && t >= float64(dn.DuskStartTick) && t < float64(dn.DuskEndTick) {
		return 1.0 - ((t-float64(dn.DuskStartTick))/duskDuration)*(1.0-dn.MinAmbientLight)
	}
	return dn.MinAmbientLight
}

// BreakTileAt breaks the tile at tx, ty with the given damage kind.
func (w *World) BreakTileAt(tx, ty int, kind tile.DamageKind) (ok bool, saveKey string) {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return false, ""
	}
	idx := w.tileIndex(tx, ty)
	if w.DestroyedTiles != nil && w.DestroyedTiles[idx] {
		return false, ""
	}
	g := w.gidAt(tx, ty)
	def := tile.DefOf(g)
	if !def.AcceptsDamage(kind) {
		return false, ""
	}
	if w.DestroyedTiles == nil {
		w.DestroyedTiles = make(map[int]bool)
	}
	w.DestroyedTiles[idx] = true
	return true, tile.MapTilePersistKey(w.MapID, tx, ty)
}

type ActiveFlame struct {
	X, Y   float64
	TX, TY int
	Timer  int
}

// TryIgniteTree ignites a tree tile at tx, ty. Returns true if the tree was successfully ignited.
func (w *World) TryIgniteTree(tx, ty int) bool {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return false
	}
	idx := w.tileIndex(tx, ty)
	if w.DestroyedTiles != nil && w.DestroyedTiles[idx] {
		return false
	}
	g := w.gidAt(tx, ty)
	if g != tile.GIDTree {
		return false
	}
	// Check if already burning
	for _, f := range w.Flames {
		if f.TX == tx && f.TY == ty {
			return false
		}
	}
	bx := float64(tx*tile.Size) + tile.Size*0.5
	by := float64(ty*tile.Size) + tile.Size*0.5
	w.Flames = append(w.Flames, ActiveFlame{
		X:     bx,
		Y:     by,
		TX:    tx,
		TY:    ty,
		Timer: w.EffectiveBalance().Hazards.TreeBurnDuration,
	})
	return true
}
