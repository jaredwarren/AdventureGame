# Gameplay Parameterization Recommendations

This document catalogs hardcoded balance values in the codebase and recommends which ones to parameterize next. The goal is to support upgrade systems, progression tuning, map authoring, and a single source of truth for game balance—without scattering magic numbers across scenes and systems.

Related prior analysis: player locomotion, combat hitboxes, and item capacities (see upgradeable-properties scope below).

---

## Current state

### Already parameterized

Most **player** tuning lives on `world.Player`, initialized in `BuildFromTiled` ([`internal/world/from_tiled.go`](../internal/world/from_tiled.go)), and persisted via `save.GameSave` ([`internal/save/save.go`](../internal/save/save.go)).

| Category | Fields on `Player` | Notes |
|----------|-------------------|-------|
| Locomotion | `BaseSpeed`, `SprintSpeed`, `DodgeStaminaCost`, `DodgeDuration`, `DodgeMaxImpulse`, `DodgeSpeed`, `StaminaRegenInterval` | Used in `play.go` with `0` fallbacks |
| Sword combat | `SwingDuration`, `MaxSwingCD`, `SwingActiveStart/End`, `SwordReach`, `SwordThickness` | |
| Torch combat | `TorchSwingDuration`, `MaxTorchSwingCD`, `TorchSwingActiveStart/End`, `TorchReach`, `TorchThickness`, `TorchBurnDuration/Interval/Damage` | Torch swing params exist; light radius does not |
| Defense | `InvulnFrames`, `EnemyKnockbackForce`, `PlayerKnockbackForce`, `PlayerHazardKnockbackForce` | `InvulnFrames` duplicated as `invulnFrames` const in `enemy_ai.go` |
| Items | `MaxBombs`, `BombFuseDuration`, `BombRadius`, `BombDamage` | `MaxBombsCarry` const also in `world.go` |

**Progression stats** (`internal/progression/stats.go`) drive derived max HP, stamina, and damage bonus. **Tile behavior** is in `tile.Def` ([`internal/world/tile/def.go`](../internal/world/tile/def.go)). **Tiled map properties** already support `light_level`, pickup `kind`, door targets, and shrine persistence.

### Player tuning defaults (section 1 complete)

Canonical defaults live in `player_defaults.go`. Consumers use `Player.Effective*()` accessors so zero-valued fields (e.g. from minimal test worlds) still behave correctly. Future `GameBalance` config can replace or overlay `defaultPlayerTuning` without touching every system.

---

## 1. Player upgrade properties — **done**

These were identified as upgrade candidates and are on `Player`, persisted in save, and resolved through a single default source.

| Parameter | Default | Primary location | Proposed field |
|-----------|---------|------------------|----------------|
| Base move speed | `1.35` | `from_tiled.go`, fallback `play.go` | `Player.BaseSpeed` ✓ |
| Sprint move speed | `2.15` | same | `Player.SprintSpeed` ✓ |
| Dodge stamina cost | `20` | same | `Player.DodgeStaminaCost` ✓ |
| Dodge frame duration | `20` | same | `Player.DodgeDuration` ✓ |
| Dodge velocity boost | `12` frames × `2.8` speed | same | `Player.DodgeMaxImpulse`, `Player.DodgeSpeed` ✓ |
| Stamina recovery rate | `1` per `2` ticks | same | `Player.StaminaRegenInterval` ✓ |
| Sword reach | `14.0` | `from_tiled.go`, `world.go` | `Player.SwordReach` ✓ |
| Sword thickness | `10.0` | same | `Player.SwordThickness` ✓ |
| Post-hurt invuln duration | `45` frames | `from_tiled.go`, `enemy_ai.go`, `timers.go` | `Player.InvulnFrames` ✓ — remove `invulnFrames` const |
| Enemy knockback force | `20.0` | `knockback.go`, `from_tiled.go` | `Player.EnemyKnockbackForce` ✓ |
| Player knockback force | `6.0` | same | `Player.PlayerKnockbackForce` ✓ |
| Max bombs carry | `8` | `world.go`, `from_tiled.go` | `Player.MaxBombs` ✓ |
| Bomb fuse timer | `90` frames | `from_tiled.go`, `play.go` | `Player.BombFuseDuration` ✓ |
| Bomb explosion radius | `32.0` | same | `Player.BombRadius` ✓ |
| Bomb explosion damage | `4` | same | `Player.BombDamage` ✓ |
| Torch burn duration | `72` | `from_tiled.go`, `world.go` consts | `Player.TorchBurnDuration` ✓ |
| Torch burn tick interval | `12` | same | `Player.TorchBurnInterval` ✓ |
| Torch burn tick damage | `1` | same | `Player.TorchBurnDamage` ✓ |

**Implemented:** [`internal/world/player_defaults.go`](../internal/world/player_defaults.go) holds `defaultPlayerTuning` and `DefaultPlayerTuning()`. Each field has a `Player.Effective*()` accessor; systems and scenes call those instead of inline fallbacks. `BuildFromTiled` seeds from `DefaultPlayerTuning()`. `World.MaxBombsCarry()` replaces the old package const. Removed `invulnFrames` from `enemy_ai.go`, knockback literals from `knockback.go`, and torch-burn consts from `world.go`.

---

## 2. Per-enemy authoring — **done**

`Enemy` already has the right components, but every Tiled enemy spawns identically:

```go
// internal/world/from_tiled.go
w.SpawnEnemy(o.X, o.Y-defaultEnemyH, 3, false)
```

| Property | Current default | Struct field | Tiled property (proposed) |
|----------|-----------------|--------------|---------------------------|
| HP | `3` | `Health.MaxHP` | `hp` |
| Speed | `0.55` px/tick | `AIHomer.Speed` | `speed` |
| Aggro radius | `128.0` px (~8 tiles) | `AIHomer.AggroRadius` | `aggro` |
| Contact damage | `2` HP | `ContactHurt.Damage` | `damage` |
| Boss flag | always `false` | `IsBoss` | `is_boss` |

Defaults live in [`internal/world/entity.go`](../internal/world/entity.go) (`NewEnemy`, `DefaultEnemyAggroRadiusPx`).

**Why parameterize:** Enables per-map difficulty, mini-bosses, and unlocks the boss kill coin bonus (`+25` in `play.go`) for authored content.

**Implemented:**

1. [`internal/world/enemy_config.go`](../internal/world/enemy_config.go) — `EnemyConfig`, `DefaultEnemyConfig()`, `EnemyConfigFromTiled()`, `DefaultEnemyTiledProperties()`.
2. `BuildFromTiled` spawns via `SpawnEnemyConfig` using Tiled props: `hp`, `speed`, `aggro`, `damage`, `is_boss`.
3. Map editor writes default props on new enemy markers and shows selected enemy stats in the HUD.
4. [`internal/tiled/tiled.go`](../internal/tiled/tiled.go) — `ObjPropInt` / `ObjPropFloat` for typed property reads.

---

## 3. Progression formulas and sword base damage — **done**

Core RPG curves live in [`internal/progression/config.go`](../internal/progression/config.go):

| Formula | Current expression | Shop increment |
|---------|-------------------|----------------|
| Max HP | `6 + Vitality×2` | Vitality buy grants `+2` HP |
| Max stamina | `60 + Resolve×20` | Resolve buy grants `+20` stamina |
| Damage bonus | `Might` (linear) | Might buy grants `+1` damage |
| Sword base damage | `1 + DamageBonus()` | in `world.SwordDamage()` |

Starting stats: `Vitality=1`, `Resolve=1`, `Might=1`, `Wits=0`, `Fortune=0` (`DefaultStats()`).

**Why parameterize:** Supports rebalance (“Vitality now gives +3 HP”) and future upgrades that modify per-stat scaling without touching shop handlers. `Wits` and `Fortune` are placeholders until hint/loot systems consume them.

**Implemented:** `progression.Config` with `DefaultConfig()`, formula methods (`MaxHP`, `MaxStamina`, `SwordDamage`), and `Stats` helpers that delegate to the default config. Shop descriptions and Vitality purchase HP gain read from `cfg.HPPerVitality`, `cfg.StaminaPerResolve`, and `cfg.DamagePerMight`. `World.SwordDamage()` delegates to `Stats.SwordDamage()`.

---

## 4. Economy and rewards — **done**

Shop prices and reward amounts live in [`internal/progression/economy.go`](../internal/progression/economy.go):

| Item | Cost |
|------|------|
| Upgrade Vitality | 10 |
| Upgrade Resolve | 8 |
| Upgrade Might | 12 |
| Heal full HP | 4 |
| Buy bomb | 3 |
| Buy torch | 5 |

Other economy values:

| Property | Value | Location |
|----------|-------|----------|
| Boss kill coin bonus | `+25` | `internal/scenes/play.go` |
| Coin pickup value | `+1` | `internal/systems/pickup.go` |
| Heart pickup heal | `+1` HP | `internal/systems/pickup.go` |
| Shrine free heal | `+2` HP | `internal/world/world.go` (`ShrineHeal`) |

**Why parameterize:** Progression pacing, Fortune/loot modifiers, and difficulty modes. Heart (`+1`) vs shrine (`+2`) inconsistency should be intentional and configurable.

**Implemented:** `progression.Economy` with `DefaultEconomy()`. Shop costs, boss kill bonus (`play.go`), pickup/chest rewards (`World.ApplyPickupReward` in [`pickup_reward.go`](../internal/world/pickup_reward.go)), and shrine heal (`ShrineHeal`) all read from the economy config.

---

## 5. Night difficulty and day/night cycle

### Night enemy buff

```go
// internal/systems/enemy_ai.go
if w.IsNight() {
    aggroR *= 1.5
    speed *= 1.5
}
```

**Proposed:** `World.NightAggroMultiplier`, `World.NightSpeedMultiplier` (or on `GameBalance`). Enables upgrades like “enemies are less aggressive at night.”

### Cycle timing

| Property | Value | Location |
|----------|-------|----------|
| Cycle length | `14400` frames (4 min @ 60 TPS) | `world.CycleLength` |
| Night window | `TimeOfDay < 1200` or `≥ 10800` | `world.IsNight()` |
| Dawn/dusk ramps | piecewise in `LightMultiplier()` | `internal/world/world.go` |
| Min ambient light | `0.2` | `LightMultiplier()` |
| Renderer night threshold | `< 1200` or `≥ 9600` | `internal/platform/ebiten/renderer.go` |

**Note:** Renderer twilight threshold (`9600`) differs slightly from sim night end (`10800`). Align or document intentionally.

**Proposed:** `DayNightConfig` with cycle length, phase boundaries, and light curve constants. Per-map `light_level` override already works via Tiled.

---

## 6. Lighting (torch visibility gap)

Torch combat is on `Player`; **light radius is renderer-only**:

| Property | Value | Location |
|----------|-------|----------|
| Torch light radius | `85` px | `internal/platform/ebiten/renderer.go` |
| Personal light radius | `35` px | same |
| Night overlay intensity | `(1 - mult) × 1.175`, cap `0.94` | same |

**Why parameterize:** “Brighter torch” upgrades need a gameplay field, not a renderer constant. Move radii to `Player.TorchLightRadius` and `Player.PersonalLightRadius` (or `World` defaults), pass into renderer from world state.

---

## 7. Enemy contact cadence

Separate from player invulnerability:

| Property | Value | Location |
|----------|-------|----------|
| Player invuln after contact | `45` frames | `enemy_ai.go` const (should use `Player.InvulnFrames`) |
| Enemy re-hit cooldown | `60` frames | `enemy_ai.go` (`enemyHurtCD`) |
| Contact hitbox margin | `1.0` px | `enemy_ai.go` (`contactMargin`) |

**Proposed:** Global `GameBalance.EnemyContactCooldown` or per-enemy `ContactHurt.Cooldown` on the `Enemy` struct. Comment in `enemy_ai.go` already notes this migration path.

---

## 8. Environmental hazards

Distinct from player torch DoT (`Player.TorchBurn*`):

| Property | Value | Location |
|----------|-------|----------|
| Tree flame duration | `90` frames | `world.TryIgniteTree()` |
| Flame tile player damage | `1` HP/tick | `internal/systems/timers.go` |
| Flame hazard knockback | `12` px fallback | `timers.go` (fallback; `Player.PlayerHazardKnockbackForce` exists) |

**Proposed:** `HazardConfig` on `World` or `GameBalance`: tree burn timer, flame tick damage, flame tick interval. World-level, not player-upgrade-level, but should share the same config package.

---

## 9. Death, respawn, and transitions

| Property | Value | Location |
|----------|-------|----------|
| Respawn map | `"field1"` | `internal/scenes/worldloader.go` |
| Respawn HP | `MaxHP/2`, minimum `2` | `worldloader.go` |
| Door cooldown (map load) | `60` frames | `worldloader.go` |
| Door cooldown (warp) | `90` frames | `worldloader.go`, `title.go` |
| Chest interact margin | `2.0` px | `internal/scenes/play.go` |

**Priority:** Medium. Death loop is called out as placeholder in [`docs/completion-steps.md`](completion-steps.md). Parameterize when the respawn design is locked.

---

## 10. Combat juice and VFX (low priority)

Hardcoded feel tuning—parameterize only if targeting accessibility presets or modding:

| Property | Examples | Location |
|----------|----------|----------|
| Hit-stop | `5` frames on melee hit | `play.go` |
| Camera shake | `(12, 4.5)` bomb, `(6, 2.5)` hit, etc. | `play.go`, `camera.go` |
| Particle spawn rates | dust `15%`, ember `25%`, bomb spark `60%` | `play.go` |
| Debris counts | `5` hit, `15` kill, `16` explosion | `play.go` |
| Particle physics | gravity, drag, velocity ranges | `internal/scenes/particle.go` |

Reduce-shake accessibility toggle exists in save; multipliers (`/2`, `×0.35`) are still hardcoded in `camera.go`.

---

## 11. Editor and proc-gen notes

| Item | Issue | Priority |
|------|-------|----------|
| Editor enemy placement | Writes default `hp` / `speed` / `aggro` / `damage` / `is_boss` on new markers; edit values in Tiled or `.tmj` | Done (§2) |
| Editor playtest stats | Uses `progression.DefaultStats()` always | Medium |
| Editor preview lighting | Fixed `TimeOfDay=3000`, override `1.0` | Medium — hides night/torch testing |
| `gentmj` CLI weights | Parameterized via flags; metadata written to `.tmj` but **not read** by runtime loader | Low — future biome presets |

---

## Recommended implementation order

1. ~~**Per-enemy Tiled props**~~ — done (§2).
2. **`GameBalance` struct** — night multiplier and any remaining globals; economy (§4) and progression formulas (§3) are done in `progression` package.
3. **Torch light radius on `Player`** — closes combat vs visibility upgrade gap; wire renderer to world state.
4. **Centralize player defaults** — `DefaultPlayerConfig()` used by `BuildFromTiled`, save fallbacks, and systems; remove scattered literals.
5. **Enemy contact cooldown** — global config or `ContactHurt.Cooldown` per enemy.
6. **Hazard and day/night config** — when tuning environmental difficulty.
7. **Death/respawn and juice** — when those designs are finalized.

---

## Architectural pattern

Follow the same pattern already used for swing and movement parameters:

1. **Define** fields on `Player`, `Enemy`, `GameBalance`, or `progression.Config`.
2. **Expose** player-relevant fields in `save.GameSave` for persistence across map loads and quicksaves.
3. **Initialize** defaults in one place (`BuildFromTiled`, `DefaultStats()`, or loaded `balance.json`).
4. **Consume** with `if field == 0 { field = default }` only at the default provider—not in every system.
5. **Author** per-entity overrides via Tiled object properties and the map editor.

Optional future step: a `balance.json` (or `gameplay.toml`) loaded at startup for modding, with struct defaults as fallback when the file is absent.

---

## Config surface today

| File | Gameplay role |
|------|---------------|
| `save.json` | Stats, HP, many `Player` tuning fields, time of day |
| `keybinds.json` | Input only |
| `assets/maps/*.tmj` | Geometry, markers, `light_level`; not enemy stats |
| `assets/sprites/*.json` | Visual atlases only |

There is no dedicated balance config file yet. `Player` + save persistence + Tiled properties are the closest existing patterns.
