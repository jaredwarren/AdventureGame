// Package world is the gameplay simulation layer: tile collision, player movement, sword hits,
// pickups, and simple enemy AI. It deliberately knows nothing about Ebiten—only numbers and structs—
// so you can unit-test logic without graphics.
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

	"github.com/jaredwarren/game-test/internal/geom"
	"github.com/jaredwarren/game-test/internal/progression"
)

// TileSize must match maps authored in Tiled (see BuildFromTiled guard).
const TileSize = 16

// MaxBombsCarry is the inventory cap for bomb pickups (see Bombs, PickupBomb).
const MaxBombsCarry = 8

// ItemSlot identifies which item is currently active in the player's secondary slot.
type ItemSlot int

const (
	ItemSlotBomb  ItemSlot = iota // default
	ItemSlotTorch
)

// ClampBombsCarry clamps n to [0, MaxBombs] for save load and pickups.
func (w *World) ClampBombsCarry(n int) int {
	if n < 0 {
		return 0
	}
	maxB := w.Player.MaxBombs
	if maxB <= 0 {
		maxB = MaxBombsCarry
	}
	if n > maxB {
		return maxB
	}
	return n
}

type PickupKind int

const (
	PickupCoin PickupKind = iota
	PickupHeart
	PickupBomb
	PickupSmallKey
	PickupTorch
)

// Pickup is a ground item the player can collect by overlap. Transform +
// Hitbox give it its AABB; Kind drives the on-pickup effect in
// systems.PickupSystem.
type Pickup struct {
	ID EntityID
	Transform
	Hitbox

	Kind   PickupKind
	Gone   bool // set true when consumed; kept in the slice for stable indices
	Opened bool // set true when chest is opened

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

	Swing                  int // >0 while sword animation runs; only mid-window counts as hit frames (see SwordHitbox)
	SwingCD                int // frames until another Z press is accepted
	TorchSwing             int // >0 while torch swing runs; same active window as sword (see TorchHitbox)
	TorchSwingCD           int // frames until another torch press is accepted
	Invuln                 int // i-frames after taking damage
	DodgeTimer             int // if >0, enemy contact check skipped (paired with scene dodgeImpulse nudge)
	Stamina                int // drained by sprint; refilled when not sprinting
	SprintHeld             bool
	SprintExhausted        bool
	SwingDuration          int
	MaxSwingCD             int
	SwingActiveStart       int
	SwingActiveEnd         int
	TorchSwingDuration     int
	MaxTorchSwingCD        int
	TorchSwingActiveStart  int
	TorchSwingActiveEnd    int
	BaseSpeed              float64
	SprintSpeed            float64
	DodgeStaminaCost       int
	DodgeDuration          int
	DodgeMaxImpulse        int
	DodgeSpeed             float64
	StaminaRegenInterval       int
	SwordReach                 float64
	SwordThickness             float64
	TorchReach                 float64
	TorchThickness             float64
	InvulnFrames               int
	EnemyKnockbackForce        float64
	PlayerKnockbackForce       float64
	PlayerHazardKnockbackForce float64
	MaxBombs                   int
	BombFuseDuration           int
	BombRadius                 float64
	BombDamage                 int
	TorchBurnDuration          int
	TorchBurnInterval          int
	TorchBurnDamage            int
}

type World struct {
	MapID string
	TileW int
	TileH int
	MapW  int
	MapH  int
	Tiles []int

	// DestroyedTiles records tiles that have been broken/opened by damage
	// (bombs, fire, ...) keyed by tile index. Rendering and collision
	// treat these tiles as their TileDef.DestroyedGID.
	DestroyedTiles map[int]bool
	Doors          []Door
	Shrines        []Shrine
	Pickups        []Pickup
	IsEditor       bool // set true when running in editor mode
	Enemies        []Enemy

	Player   Player
	Stats    progression.Stats
	HP       int
	Currency int
	// Bombs is how many bomb items the player holds; pickups increment (capped at
	// MaxBombsCarry), placing a bomb decrements (see TryDamageFaceTile(DamageBomb)).
	Bombs        int
	HasTorch     bool
	SmallKey     int
	SelectedItem ItemSlot

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

	// DoorCooldown counts down while >0 (systems.TimersSystem); while non-
	// zero, scenes should skip door warp checks so the player does not
	// immediately re-trigger the door they spawned on after a map
	// transition.
	DoorCooldown int

	// nextID feeds allocID(). Unexported so external code can't mint IDs
	// out of band; use the Spawn* methods instead.
	nextID uint32
}

func (w *World) gidAt(tx, ty int) int {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return GIDWall
	}
	return w.Tiles[ty*w.MapW+tx]
}

// GIDAt returns the tile GID at tile coordinates (for debug / tools).
func (w *World) GIDAt(tx, ty int) int {
	return w.gidAt(tx, ty)
}

func (w *World) tileIndex(tx, ty int) int {
	return ty*w.MapW + tx
}

func (w *World) solidTile(tx, ty int) bool {
	g := w.gidAt(tx, ty)
	idx := w.tileIndex(tx, ty)
	return SolidAt(g, idx, w.DestroyedTiles, w.SmallKey > 0)
}

// RectHitTiles returns true if rect overlaps any solid tile (sample corners + center).
func (w *World) RectHitsSolid(r geom.Rect) bool {
	points := [][2]float64{
		{r.X, r.Y},
		{r.X + r.W, r.Y},
		{r.X, r.Y + r.H},
		{r.X + r.W, r.Y + r.H},
		{r.X + r.W*0.5, r.Y + r.H*0.5},
	}
	for _, p := range points {
		tx := int(p[0] / TileSize)
		ty := int(p[1] / TileSize)
		if w.solidTile(tx, ty) {
			return true
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
	activeStart := w.Player.SwingActiveStart
	if activeStart <= 0 {
		activeStart = 2
	}
	activeEnd := w.Player.SwingActiveEnd
	if activeEnd <= 0 {
		activeEnd = 7
	}
	reach := w.Player.SwordReach
	if reach <= 0 {
		reach = 14.0
	}
	thick := w.Player.SwordThickness
	if thick <= 0 {
		thick = 10.0
	}
	return swordSwingHitbox(w.Player.Swing, activeStart, activeEnd, reach, thick, w.Player.Dir, px, py)
}

// TorchHitbox matches SwordHitbox timing/geometry but keys off TorchSwing.
func (w *World) TorchHitbox() (geom.Rect, bool) {
	px := w.Player.X + w.Player.W*0.5
	py := w.Player.Y + w.Player.H*0.5
	activeStart := w.Player.TorchSwingActiveStart
	if activeStart <= 0 {
		activeStart = 2
	}
	activeEnd := w.Player.TorchSwingActiveEnd
	if activeEnd <= 0 {
		activeEnd = 7
	}
	reach := w.Player.TorchReach
	if reach <= 0 {
		reach = 14.0
	}
	thick := w.Player.TorchThickness
	if thick <= 0 {
		thick = 10.0
	}
	return swordSwingHitbox(w.Player.TorchSwing, activeStart, activeEnd, reach, thick, w.Player.Dir, px, py)
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
	return 1 + w.Stats.DamageBonus()
}

// Torch burn DoT (see systems.BurnSystem).
const (
	TorchBurnDuration      = 72
	TorchBurnTickInterval  = 12
	TorchBurnDamagePerTick = 1
)

// IgniteEnemy extends or starts torch burn on the enemy at index i.
func (w *World) IgniteEnemy(i int) {
	if i < 0 || i >= len(w.Enemies) {
		return
	}
	e := &w.Enemies[i]
	if e.HP <= 0 {
		return
	}
	burnDuration := w.Player.TorchBurnDuration
	if burnDuration <= 0 {
		burnDuration = TorchBurnDuration
	}
	burnInterval := w.Player.TorchBurnInterval
	if burnInterval <= 0 {
		burnInterval = TorchBurnTickInterval
	}
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
	if w.Tiles[idx] != GIDLock {
		return false
	}
	w.Tiles[idx] = GIDGrass
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
	duration := w.Player.SwingDuration
	if duration <= 0 {
		duration = 8
	}
	cooldown := w.Player.MaxSwingCD
	if cooldown <= 0 {
		cooldown = 12
	}
	w.Player.Swing = duration
	w.Player.SwingCD = cooldown
	return true
}

// TrySwingTorch starts a torch swing if the player has a torch and no
// sword or torch animation is already running.
func (w *World) TrySwingTorch() bool {
	if !w.HasTorch {
		return false
	}
	if w.Player.TorchSwing > 0 || w.Player.TorchSwingCD > 0 {
		return false
	}
	if w.Player.Swing > 0 || w.Player.SwingCD > 0 {
		return false
	}
	duration := w.Player.TorchSwingDuration
	if duration <= 0 {
		duration = 8
	}
	cooldown := w.Player.MaxTorchSwingCD
	if cooldown <= 0 {
		cooldown = 12
	}
	w.Player.TorchSwing = duration
	w.Player.TorchSwingCD = cooldown
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
func (w *World) TryDamageFaceTile(kind DamageKind) (ok bool, saveKey string) {
	if kind == DamageBomb && w.Bombs <= 0 {
		return false, ""
	}
	pc := w.PlayerRect()
	cx := pc.X + pc.W*0.5
	cy := pc.Y + pc.H*0.5
	tx := int(cx / TileSize)
	ty := int(cy / TileSize)
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
	def := TileDefOf(g)
	if !def.AcceptsDamage(kind) {
		return false, ""
	}
	if w.DestroyedTiles == nil {
		w.DestroyedTiles = make(map[int]bool)
	}
	w.DestroyedTiles[idx] = true
	if kind == DamageBomb {
		w.Bombs--
		if w.Bombs < 0 {
			w.Bombs = 0
		}
	}
	return true, MapTilePersistKey(w.MapID, tx, ty)
}

// ShrineHeal is the “poor” shrine interaction (no coins). Rich interaction is priced in game.Update.
func (w *World) ShrineHeal() {
	if w.HP < w.MaxHP() {
		w.HP += 2
		if w.HP > w.MaxHP() {
			w.HP = w.MaxHP()
		}
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
	// Twilight and night is when w.TimeOfDay is outside [1200, 10800)
	return w.TimeOfDay < 1200 || w.TimeOfDay >= 10800
}

func (w *World) LightMultiplier() float64 {
	if w.HasAmbientLightOverride {
		return w.AmbientLightOverride
	}
	t := float64(w.TimeOfDay)
	if t >= 0 && t < 1200 {
		return 0.2 + (t/1200.0)*0.8
	}
	if t >= 1200 && t < 9600 {
		return 1.0
	}
	if t >= 9600 && t < 10800 {
		return 1.0 - ((t-9600.0)/1200.0)*0.8
	}
	return 0.2
}

// BreakTileAt breaks the tile at tx, ty with the given damage kind.
func (w *World) BreakTileAt(tx, ty int, kind DamageKind) (ok bool, saveKey string) {
	if tx < 0 || ty < 0 || tx >= w.MapW || ty >= w.MapH {
		return false, ""
	}
	idx := w.tileIndex(tx, ty)
	if w.DestroyedTiles != nil && w.DestroyedTiles[idx] {
		return false, ""
	}
	g := w.gidAt(tx, ty)
	def := TileDefOf(g)
	if !def.AcceptsDamage(kind) {
		return false, ""
	}
	if w.DestroyedTiles == nil {
		w.DestroyedTiles = make(map[int]bool)
	}
	w.DestroyedTiles[idx] = true
	return true, MapTilePersistKey(w.MapID, tx, ty)
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
	if g != GIDTree {
		return false
	}
	// Check if already burning
	for _, f := range w.Flames {
		if f.TX == tx && f.TY == ty {
			return false
		}
	}
	bx := float64(tx*TileSize) + TileSize*0.5
	by := float64(ty*TileSize) + TileSize*0.5
	w.Flames = append(w.Flames, ActiveFlame{
		X:     bx,
		Y:     by,
		TX:    tx,
		TY:    ty,
		Timer: 90, // 1.5 seconds at 60fps
	})
	return true
}

