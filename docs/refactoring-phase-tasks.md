# Refactoring Phase Task Plans (agent-ready)

Companion to [architecture-refactoring-plan.md](architecture-refactoring-plan.md). Each phase below is a **self-contained implementation spec** designed to be fed to an AI coding agent one at a time. Feed a phase only after all its prerequisites are merged and green.

## How to use this document

1. Copy the **Shared preamble** below, then append **one phase section**, and give both to the agent as its task.
2. Run phases in order: `P0 → P1 → P2 → P3 → P4`; `P5` requires P3; `P6` requires P2 and can run any time after it.
3. Each phase lists **tasks sized as individual PRs/commits**. Tell the agent to complete them in order and to keep the game building and `go test ./...` green after every task.
4. After the agent finishes a phase, run the phase's **Validation** section yourself before starting the next.

---

## Shared preamble (prepend to every phase prompt)

```
You are working in the Go + Ebiten v2 game repo `github.com/jaredwarren/game-test`
(top-down Zelda-style adventure, 320×240 logical resolution).

Read these before writing any code:
- README.md (package layout, conventions)
- docs/architecture-refactoring-plan.md (the plan this task comes from)
- internal/archtest/imports_test.go (enforced import boundaries)
- internal/systems/doc.go and internal/scenes/scene.go (layer contracts)

Hard rules:
- Ebiten may only be imported by cmd/game, internal/game, internal/platform/... .
- Core packages (world, systems, scenes, run, save, tiled, etc.) must stay headless;
  the archtest enforces this — never weaken it to make a change compile.
- Sim state lives in internal/world; per-tick behavior in internal/systems; scenes
  only convert input → intents and events → feedback (SFX/shake/particles).
- Determinism is a hard requirement: no time.Now, no global rand, no filesystem
  reads in world/systems. Timers are frame-counted ints. Replay tests must pass.
- Preserve behavior unless the task explicitly says it fixes a bug.
- After every task: go build ./... && go vet ./... && go test ./... must pass.
- Keep commits small — one task from the list per commit.
```

---

# Phase 0 — Truth-up: docs + architecture contract

**Effort:** Small (≤1 day). **Prerequisites:** none. **Behavior change:** none.

**Goal:** documentation matches reality; the archtest becomes a full per-package layer contract; a single `make check` gate exists.

### Task 0.1 — Fix documentation drift

Files: `README.md`, `docs/completion-steps.md`, `docs/parameterization-recommendations.md`.

1. In `README.md`:
   - Remove or correct the "Bug reports" claims about `dungeon.Result.BugDigest()`, the pause dungeon-seed digest, and the `Tab` dungeon graph overlay — none of these exist in the code. Verify by grepping for `BugDigest` and `Tab` handling before editing; describe only what the pause overlay actually shows.
   - Fix the project-layout table row for `internal/game` (it says "Scenes, input, draw, SFX"; it is now a thin wiring layer — see the doc comment in `internal/game/app.go`).
   - Add rows for packages missing from the table: `internal/systems`, `internal/scenes`, `internal/services`, `internal/replay`, `internal/input`, `internal/geom`.
2. In `docs/completion-steps.md`: remove references to `internal/dungeon/gen.go` and `dungeon.Result.BugDigest()` (§0 and §1.4); state that dungeon generation is currently a CLI-only tool (`cmd/gentmj`) with no runtime integration.
3. In `docs/parameterization-recommendations.md`: fix the stale `internal/world/tiledef.go` reference (now `internal/world/tile/def.go`).
4. Check `internal/scenes/title.go` for stale on-screen help text (e.g. "TAB in dungeon") and remove anything that no longer works.

**Done when:** every file/symbol referenced in the three docs exists in the repo (spot-check with grep).

### Task 0.2 — Archtest v2: layer allowlist

File: `internal/archtest/imports_test.go` (extend; keep the existing ban test too).

1. Add a new test `TestLayeredImports` driven by a table mapping **every** `internal/` package to its allowed internal imports, per the "Architecture contract → Import rules (archtest v2)" table in `docs/architecture-refactoring-plan.md`. For the current tree (pre-P2/P3) the table is:

   | Package | Allowed internal imports |
   |---|---|
   | `internal/geom`, `internal/tiled`, `internal/progression`, `internal/render` | (none) |
   | `internal/save` | `internal/world` *(temporary; removed in Phase 4 — mark with a TODO)* |
   | `internal/input` | `internal/services` |
   | `internal/replay` | `internal/services`, `internal/input` |
   | `internal/services` | `internal/world`, `internal/render` |
   | `internal/world` (+ `tile`, `pickup`, `enemy` subpackages) | `internal/geom`, `internal/tiled`, `internal/progression`, own subpackages |
   | `internal/systems` | `internal/world`, `internal/geom` |
   | `internal/dungeon` | `internal/tiled`, `internal/geom` |
   | `internal/scenes` | `internal/systems`, `internal/world`, `internal/services`, `internal/render`, `internal/progression`, `internal/geom`, `internal/save`, `internal/tiled`, `internal/dungeon` |
   | `internal/platform/ebiten` | `internal/services`, `internal/world`, `internal/render`, `internal/input` |
   | `internal/game` | `internal/platform/ebiten`, `internal/scenes`, `internal/services`, `internal/render`, `internal/save` |
   | `internal/archtest` | (test-only, exempt) |

   Before finalizing, build the *actual* import list with `go list -deps` or by parsing imports (reuse `readImports`) and reconcile: if a real import is missing from the table above, add it to the table with a `// TODO(phase-N): remove` comment rather than breaking the build — this test must pass on the current tree.
2. The test must **fail for any `internal/` package on disk that is not classified** in the table (walk `internal/` directories and require a table entry).
3. Remove or keep the dormant `internal/core` placeholder in `pureCorePackages` — per the plan, delete it and note why in a comment.

**Done when:** `go test ./internal/archtest/` passes on the current tree; temporarily adding `github.com/hajimehoshi/ebiten/v2` to a file in `internal/world` makes it fail; creating an empty `internal/foo/foo.go` package makes it fail.

### Task 0.3 — `make check` + golden save capture

1. Add a `check` target to `Makefile`: `check: fmt vet test` (reuse existing targets). Document it in README as the pre-commit gate.
2. Capture a golden save for Phase 4 insurance: run the game or write a tiny throwaway test that builds a `save.GameSave` via the current code path with representative non-zero values for **every** field, and commit it as `internal/save/testdata/v_current.json`. Do not add migration logic yet.

### Validation (run after phase)

```bash
make check
go test ./internal/archtest/ -v
```

---

# Phase 1 — Bombs and flames become sim

**Effort:** Medium (2–4 days). **Prerequisites:** Phase 0.

**Goal:** all gameplay-state mutation runs inside `systems.Pipeline`; `PlayScene` shrinks to intents + reactions. This **fixes two real bugs**: (a) bomb kills bypass `HitEvent`, so a bomb kill on the boss never pays the boss coin bonus and produces no kill juice; (b) bombs are scene-local state, so pausing mid-fuse destroys the lit bomb (the item was already decremented — it silently vanishes).

Read first: `internal/scenes/play.go` (the `ActiveBomb` struct, `tryDropBomb`, the fuse/explosion block in `Update` ~lines 114–219, the chest branch of `handleActions` ~lines 262–288, `reactToEvents`), `internal/systems/pipeline.go`, `internal/systems/events.go`, `internal/systems/timers.go` (flame tick damage), `internal/world/world.go` (`TrySwingSword` as the intent-method pattern to copy, `Flames`, `BreakTileAt`, `TryIgniteTree`).

### Task 1.1 — Move bomb state into `world`

1. Create `internal/world/bomb.go`: move the `ActiveBomb` struct from `play.go` (rename fields idiomatically if needed; keep frame-counted timers). Add `World.Bombs []ActiveBomb`.
2. Add intent method `World.TryPlaceBomb() bool` mirroring `TrySwingSword`: checks bomb count > 0 and facing/tile validity exactly as `tryDropBomb` does today, decrements inventory, appends to `w.Bombs`, returns whether it placed. Move any placement math (facing tile computation) with it.
3. In `play.go`, `tryDropBomb` becomes a one-line call to `w.TryPlaceBomb()` (keep the SFX trigger in the scene, keyed on the returned bool for now).

### Task 1.2 — `systems.BombSystem` + `ExplosionEvent`

1. Add `ExplosionEvent{TX, TY int}` (or pixel coords — match what the scene VFX needs) to `internal/systems/events.go`.
2. Create `internal/systems/bomb.go` with a `BombSystem` that per tick: decrements fuses; on zero — calls `World.BreakTileAt` for the cracked-wall destruction, applies radial enemy damage **through the same damage helper `CombatSystem` uses so a `HitEvent` (with `Killed`/`IsBoss`) is emitted per enemy hit**, removes the bomb, and pushes `ExplosionEvent`. If combat's damage path is currently inline, extract a shared unexported helper in the systems package first.
3. Register in `systems.Default()` ordering: after Combat, before Pickup. Document why in the `pipeline.go` header comment (explosions must resolve before pickups so dropped loot from bomb kills is collectible same-tick, consistent with sword kills).
4. Delete the fuse/explosion block from `PlayScene.Update`. In `reactToEvents`, handle `ExplosionEvent` → existing shake/SFX/particle calls that the deleted block performed. Bomb kill juice/boss bonus now arrives via the ordinary `HitEvent` path — delete any bomb-specific duplication.
5. `PlayScene.Draw` keeps drawing bombs but reads `w.Bombs` (drawing moves to the renderer in Phase 6).

### Task 1.3 — `systems.HazardSystem`

1. Create `internal/systems/hazard.go`: move the flame-tick player damage out of `systems/timers.go` and flame lifecycle handling into `HazardSystem`. Keep event emission for player-hurt identical to today.
2. Move `TryIgniteTree`'s inline `90`-frame constant to a package-level named constant with a `// TODO(phase-2): move to balance.GameBalance.Hazards` comment.
3. Pipeline order: Burn → Bomb → Hazard (hazards after explosions so a bomb-lit flame damages same tick as it appears — verify against current behavior and document).

### Task 1.4 — Unify chest-open with the pickup path

1. Add `World.OpenChestAt(...)` intent that flips the chest state and lets `systems.PickupSystem` (or a direct event push) emit the standard `PickupEvent` — the chest branch in `handleActions` (`play.go` ~L262–288) currently duplicates the `PickupEvent` reaction, and its own comment admits it.
2. Delete the duplicated reaction block; chests now flow through `reactToEvents` like every other pickup.

### Task 1.5 — `systems.StaminaSystem`

1. Add `Player.SprintHeld bool` (if not already present — `systems/doc.go` anticipates this migration). `PlayScene.handleMovement` writes it from input; drain/regen logic moves into `internal/systems/stamina.go`.
2. Pipeline position: last (after Lock). Remove the stamina math from the scene.

### Task 1.6 — Tests

1. `systems_test.go` (or new files) table-driven cases:
   - Bomb placed next to boss, fuse expires → events contain `HitEvent{Killed: true, IsBoss: true}` and `ExplosionEvent`.
   - Bomb adjacent to a cracked wall → `TileDestroyedEvent` (or the existing equivalent) emitted and tile changed.
   - Stamina drains while `SprintHeld`, regens at `StaminaRegenInterval` when not.
2. Extend the replay test (`internal/replay/record_replay_test.go` pattern) with an input stream that drops a bomb; assert deterministic outcome.

### Validation

```bash
make check
```
Manual playtest: drop a bomb → pause (`P`) → resume → the bomb still explodes and the count is correct. Kill the dungeon boss with a bomb → coin bonus is paid and kill juice plays.

**Success criteria:** no `DamageEnemy`/`BreakTileAt` calls from `internal/scenes` except via `World` intent methods; `play.go` shrinks by roughly 200 lines.

---

# Phase 2 — `GameBalance` and the last parameterization gaps

**Effort:** Medium (2–4 days). **Prerequisites:** Phase 1.

**Goal:** one config story; the renderer contains zero gameplay numbers. Executes items 2, 3, 5, 6 of the "Recommended implementation order" in `docs/parameterization-recommendations.md`.

Read first: `docs/parameterization-recommendations.md` §5–§9; `internal/platform/ebiten/renderer.go` (torch radius `85`, personal radius `35`, night threshold `9600` at ~L342–370); `internal/world/world.go` (`CycleLength`, `IsNight`, `LightMultiplier`, `ShrineHeal`, `TryIgniteTree`); `internal/systems/enemy_ai.go` (`enemyHurtCD`, `contactMargin`, night `×1.5` multipliers); `internal/scenes/worldloader.go` (respawn map/HP, door cooldowns); `internal/world/player_defaults.go` (the defaults pattern to copy).

### Task 2.1 — Create `internal/balance`

1. New package `internal/balance` with a `GameBalance` struct composed of sub-structs:
   - `DayNight`: cycle length, night start/end tick, dawn/dusk ramp boundaries, min ambient light (from `world.CycleLength`, `IsNight`, `LightMultiplier` literals).
   - `NightBuffs`: aggro multiplier, speed multiplier (the `×1.5`s in `enemy_ai.go`).
   - `Contact`: enemy re-hit cooldown (`enemyHurtCD=60`), contact margin.
   - `Hazards`: tree burn duration (the Phase-1 TODO constant), flame tick damage, flame tick interval.
   - `Respawn`: map ID (`"field1"`), HP fraction, HP minimum.
   - `DoorCooldowns`: map-load (60) and warp (90) frames.
2. `Default() *GameBalance` returns the struct with all current values. `balance` may import `progression` (per the contract) but nothing else internal.
3. Optional JSON overlay: `Load(data []byte) (*GameBalance, error)` that unmarshals over defaults. Wire a dev-only `-balance path` flag in `cmd/game`. **Go struct defaults remain the source of truth; JSON is a playtest overlay only** — no required config file.
4. Add `balance` to the archtest layer table as a leaf (may import `progression` only).

### Task 2.2 — Inject balance into the sim

1. `World` gains `Balance *balance.GameBalance`; `BuildFromTiled` sets it (accept it as a parameter or default to `balance.Default()` when nil — follow the `Effective*()` zero-value-safe pattern so minimal test worlds keep working).
2. Re-point consumers: `IsNight`/`LightMultiplier` read `Balance.DayNight`; `enemy_ai.go` reads `Balance.NightBuffs`/`Balance.Contact` via the world (delete the package consts); `TryIgniteTree` and `HazardSystem` read `Balance.Hazards`; `worldloader.go` respawn + door cooldowns read `Balance.Respawn`/`Balance.DoorCooldowns`.
3. `ShrineHeal`/`UpgradeRandomStat` stop calling `progression.DefaultEconomy()` globals — the configured `Economy` hangs off `GameBalance` (embed or reference `progression.Economy`).
4. Move the boss kill coin bonus from the scene event handler into `CombatSystem`/`BombSystem` (currency is sim state; paying it belongs in sim). The scene keeps only the toast/SFX reaction to the existing event.

### Task 2.3 — Light radii become gameplay state

1. Add `Player.TorchLightRadius`, `Player.PersonalLightRadius` with defaults in `player_defaults.go` (`85`, `35`) and `Effective*()` accessors; persist like other tuning fields (add to `save.GameSave` + `run_state` copies — yes, still triplicated until Phase 4; keep the pattern consistent).
2. Renderer reads both from `World.Player`; delete the `85`/`35` literals.
3. Fix the night-threshold mismatch: delete the renderer's duplicated `9600` threshold condition (~L342) and drive the night overlay purely from `w.LightMultiplier()` (which it already calls for alpha). Note in the commit that visuals may shift slightly at dusk — this is the bug fix (sim says night ends at 10800).

### Task 2.4 — Archtest + tests

1. Archtest: add rule that `internal/platform/...` may NOT import `internal/progression` or `internal/balance` (it shouldn't need them once radii come from World). Update the layer table: `world` and `systems` may import `balance`; `save` will in Phase 4.
2. Re-point existing systems tests at injected config where they relied on package consts. Add one test proving a non-default `NightBuffs` multiplier changes enemy aggro radius at night.

### Validation

```bash
make check
grep -nE '[0-9]{2,}' internal/platform/ebiten/renderer.go   # review: only visual styling numbers remain
```
Visual check: watch a full day/night cycle; dusk transition should be smooth and match sim night timing.

**Success criteria:** every table in parameterization doc §5–§8 has a struct home; renderer has no gameplay constants.

---

# Phase 3 — Scene layer hygiene: extract `internal/run`, split `play.go` / `editor.go`

**Effort:** Medium (2–4 days). **Prerequisites:** Phase 1 (Phase 2 recommended).

**Goal:** `internal/scenes` contains only scenes/overlays/particles; cross-scene run state gets its own headless package; no scene-layer file exceeds ~400 lines. Includes the pause-overlay decision (recommended: implement it).

Read first: `internal/scenes/session.go`, `map_progress.go`, `run_state.go`, `worldloader.go` (+ tests), `register.go`, `scene.go` (Manager), `internal/game/app.go` (constructs Session).

### Task 3.1 — Extract `internal/run` (mostly `git mv`)

1. Create `internal/run`; move `session.go`, `map_progress.go`, `run_state.go`, `worldloader.go` and their tests from `internal/scenes`. Rename exported symbols minimally for the new package name (`run.Session`, `run.EnterMap`, …). Fix all imports (`scenes`, `game`).
2. `internal/game` now constructs `run.Session` and passes it through the scene context (adjust `GameContext` if `Session()` type moves).
3. Archtest: classify `run` per the contract table (`run` may import `world`, `systems`, `save`, `tiled`, `dungeon`, `progression`, `render`, `services`, `balance`, `geom`; `run` must NOT import `scenes`). Update `scenes`' allowlist: it now imports `run` and should drop direct `tiled`/`save` imports where the moved code was the only user — verify with the archtest, keep any imports still genuinely needed.

### Task 3.2 — Split `play.go`

Same package, mechanical split, one `PlayScene` type:

| File | Contents |
|---|---|
| `scenes/play.go` | struct, Enter/Exit, Update skeleton, Draw routing (~150 lines) |
| `scenes/play_input.go` | `handleActions`, `handleMovement`, item menu |
| `scenes/play_juice.go` | `reactToEvents`, toast, event-keyed particle spawns |

### Task 3.3 — Split `editor.go`

Split 762-line `editor.go` into `editor.go` (state + Update routing), `editor_menus.go` (tile/item panels), `editor_draw.go` (drawing). Same package, no behavior change.

### Task 3.4 — Two-slot overlay Manager (pause/shop as overlays)

1. Extend `scenes.Manager` with `PushOverlay(id, params)` / `PopOverlay()` — exactly **two slots** (base + overlay), not a general stack (the plan explicitly rejects one). While an overlay is active: overlay gets `Update`; base scene gets `Draw` first (world visible, dimmed under the overlay), then overlay `Draw`. Base scene does NOT update (game pauses).
2. Convert pause and shop from `Replace` to overlays over play. Title→play, play→editor, death respawn remain `Replace`.
3. This preserves `PlayScene` particles/toast/hit-stop across pause→resume (with Phase 1 done, bombs already survive because they live in `World`).
4. Update the `scene.go` package doc ("single-slot model" section) to describe the two-slot model and why it stops there.

### Task 3.5 — Tests

1. Moved tests compile and pass under `internal/run`.
2. Manager test: push overlay → base does not update, both draw in order → pop → base updates again.
3. Full manual playtest: title → new game → pause → resume → shop → buy → door transition → death → respawn → quicksave/quickload.

### Validation

```bash
make check
wc -l internal/scenes/*.go   # all ≤ ~400 lines
```

**Success criteria:** `internal/run` compiles headless; `internal/scenes` files are all scenes/overlays/particles; pausing mid-game preserves particles, toasts, and bombs.

---

# Phase 4 — Single tuning source + save v3 with migrations

**Effort:** Medium–Large (3–7 days). **Prerequisites:** Phases 2 and 3.

**Goal:** adding a tunable = touching one struct + one default; save format is versioned with a migration table and golden-file tests. Unblocks roadmap M5 ("track owned items in save with migrations").

Read first: `internal/world/world.go` Player fields (~L131–176), `internal/world/player_defaults.go`, `internal/save/save.go` (flat fields ~L57–87, `Load` fallbacks incl. legacy `has_bomb`), `internal/run/run_state.go` (the four copy functions), the golden file captured in Task 0.3.

### Task 4.1 — `balance.PlayerTuning`

1. Define `balance.PlayerTuning` — one struct containing every field currently triplicated across `world.Player` / `save.GameSave` / `run.playerTuning` (use `run_state.go`'s `playerTuning` as the authoritative field list; include Phase 2's light radii).
2. `world.Player` **embeds** `balance.PlayerTuning`. Keep the `Effective*()` accessors working (they now read embedded fields); `DefaultPlayerTuning()` moves to (or is re-exported from) `balance`.
3. `run.RunState` carries `balance.PlayerTuning` by value. Delete the four copy functions in `run_state.go` — copying is now struct assignment.

### Task 4.2 — Save v3 + migration table

1. `save.GameSave` v3: replace the ~30 flat tuning fields with one nested `Tuning balance.PlayerTuning \`json:"tuning"\``. `ItemSlot` stops being a `world` type in the save: serialize as `int` in save's own type (or move the type to `balance`) so **`save` drops its `world` import and becomes a leaf** (importing only `balance`).
2. Bump `CurrentVersion` to 3. Restructure `Load` as a migration table: read `version`, then apply `migrateV1toV2` (formalize the existing ad-hoc fallbacks, e.g. legacy `has_bomb`), then `migrateV2toV3` (lift flat tuning fields into the nested struct). Each migration is a pure `func(map[string]any) map[string]any` or typed-struct step — pick one style and document it.
3. Update archtest: `save` → `balance` only (remove the Phase-0 TODO entry).

### Task 4.3 — Tests

1. `internal/save/testdata/`: golden files per version — `v2.json` is the file captured in Task 0.3; hand-write a minimal `v1.json` if v1 saves are distinguishable, otherwise document that v1 is folded into v2 handling.
2. Tests: round-trip (save → load → deep-equal); migration (load `v2.json` → assert all tuning values landed in the nested struct); unknown-future-version returns a clear error.
3. Manual: copy a real pre-refactor `save.json` into the working dir, launch, press `K` to load — all upgrades/items/stats intact.

### Validation

```bash
make check
wc -l internal/run/run_state.go   # target < 80 lines
```

**Success criteria:** a new tunable field = 1 struct field + 1 default line; `save` is a leaf in the archtest; old saves load correctly.

---

# Phase 5 — Dungeon runtime integration

**Effort:** Large (1–2 weeks). **Prerequisites:** Phase 3 (`run` owns map loading). Unblocks roadmap **M3**.

**Goal:** `internal/dungeon` deterministically produces playable maps in the running game. Today it is imported only by `cmd/gentmj` — nothing at runtime.

Read first: `internal/dungeon/` (growingtree, floorpaint, stamp — note there is **no** room-graph/key/boss generator on disk; check `git log --all --oneline -- internal/dungeon/` for a deleted `gen.go` to recover logic from history), `internal/run/worldloader.go` (`EnterMap`, the "every map runs the same code path" doc comment), `internal/world/marker.go` (marker registry — door sockets will be Tiled object props), `internal/tiled/tiled.go`, `cmd/gentmj/main.go`, `docs/completion-steps.md` §4.

### Task 5.1 — Room-graph generator

1. In `internal/dungeon`, implement (or recover from git history) seeded room-graph generation: tree-shaped graph with start room, boss room, key room, and one locked edge, where the key is reachable without crossing the lock and the locked edge is a bridge. Output a plain `Graph` value (nodes, edges, roles, seed).
2. Add `Result` with a `BugDigest() string` one-liner (seed, room count, start/boss/key ids, lock edge, edge count) — making the README's promised debug surface real.
3. Property tests: solvability (key before lock), lock-is-a-bridge, determinism (same seed → identical graph), and fallback regeneration on constraint failure (bounded retries with derived sub-seeds).

### Task 5.2 — Prefab room library + stitcher

1. Author 4–6 small room prefabs under `assets/maps/rooms/*.tmj` (combat, corridor, key, boss, start; use `cmd/game -edit` or Tiled). Door sockets are marker objects with a `socket` property (`N`/`E`/`S`/`W`).
2. `dungeon` loads the room library from embedded bytes **passed in by the caller** (`dungeon` may import `tiled` + `geom` only — not `world`, not `assets`): `LoadRoomLibrary(files map[string][]byte) (*RoomLibrary, error)`.
3. Stitcher: place each graph node's prefab on a coarse grid, connect matching sockets with corridors (reuse `StampOrthogonalMaze`/floorpaint for corridor fill), place key/lock/boss/door markers, and compose everything into a single in-memory `*tiled.Map`. The output must be indistinguishable from a hand-authored map to `BuildFromTiled`.
4. Determinism test: same seed + same library → byte-identical encoded map.

### Task 5.3 — Runtime wiring in `run`

1. `run.EnterMap` recognizes generated map IDs of the form `"dun:<seed>:<floor>"`: route to `dungeon.Generate(seed, library) → *tiled.Map` instead of `assets.MapData`. `BuildFromTiled` is unchanged.
2. The existing dungeon entrance door (north door on field1) targets a `dun:<seed>:1` id, with the seed sourced from the existing weekly-epoch/last-dungeon-seed logic.
3. Persistence: `MapProgress` already namespaces by map-ID string, so `dun:*` entries work for free. Implement the decision from the plan: **persist within a run, clear `dun:*` entries when a new dungeon seed is generated.**
4. Pause overlay shows the dungeon seed + `Result.BugDigest()` when the current map is generated; the `C` bug-digest copy includes it. (This also un-drifts the README — update it.)

### Task 5.4 — Tooling + replay

1. `cmd/gentmj` grows a `-dungeon -seed N` mode that writes the fully stitched dungeon map to a `.tmj` for eyeballing in Tiled.
2. Extend the replay test with a seeded-dungeon session (enter, grab key, pass lock) to lock in determinism.

### Validation

```bash
make check
go run ./cmd/gentmj -dungeon -seed 42 -o /tmp/d42.tmj   # open in Tiled, inspect
```
Manual: enter dungeon from field1, verify key→lock→boss→exit loop; die and re-enter — layout identical (same seed); regenerate seed — old `dun:*` progress cleared.

**Success criteria:** dungeon is generated at runtime, deterministic per seed, solvable by property test; nothing outside `dungeon` + `run` (+ pause overlay text) changed.

---

# Phase 6 — Presentation scale-up (renderer restructure)

**Effort:** Medium (2–4 days). **Prerequisites:** Phase 2 (light radii on World). Independent of P3–P5; schedule alongside art milestone M4.

**Goal:** the renderer is structured for sprites, Y-sorting, and layers, with viewport-clamped drawing — zero sim changes.

Read first: `internal/platform/ebiten/renderer.go` (716 lines: frame lifecycle, tile loop ~L197–218, enemy draw hardcoded `14, 12` ~L239, swing animation `(8-Swing)/7.0`, lighting/night overlay), `tiledraw.go`, `pickupdraw.go`, `internal/render/camera.go`.

### Task 6.1 — File split (mechanical, same package)

| File | Contents |
|---|---|
| `renderer.go` | frame lifecycle (BeginFrame/EndFrame), primitive draw methods |
| `worlddraw.go` | `DrawWorld` orchestration + draw list |
| `entitydraw.go` | player/sword/torch/enemy/bomb/flame/shrine/chest draws |
| `lighting.go` | night overlay, torch/personal light masks |

`tiledraw.go`/`pickupdraw.go` stay.

### Task 6.2 — Viewport clamp

In the tile loop, compute visible tile bounds `tx0..tx1, ty0..ty1` from the camera position and screen size, and iterate only that range instead of `MapW×MapH`. Guard for map edges. (Required before M2's 5–15-screen overworld.)

### Task 6.3 — Fix sim-coupled draw math

1. Entity draws use each entity's `Hitbox` dimensions — delete the hardcoded `14, 12`.
2. Swing/torch sweep animation derives from normalized progress `1 - float64(Swing)/float64(player.EffectiveSwingDuration())` — delete the `(8-Swing)/7.0` assumption (breaks silently when `SwingDuration ≠ 8`, and it's a persisted tunable).
3. Bomb/flame drawing moves from `PlayScene.Draw` into `entitydraw.go` (reading `w.Bombs`/`w.Flames`), removing the last procedural entity drawing from the scene layer.

### Task 6.4 — Y-sorted draw list

In `worlddraw.go`, build a per-frame draw list of `{footY float64, draw func()}` entries for entities/props, sort by `footY`, then execute. Ground tiles draw before the list; overlay/canopy layers (future) draw after. Preallocate and reuse the slice across frames to avoid per-frame allocation.

### Task 6.5 — Real text rendering

Replace `ebitenutil.DebugPrintAt` behind the existing `Renderer.DrawText` port method with `text/v2` + an embedded font (interface unchanged — scenes untouched). Keep the debug font available behind the F3 overlay if convenient.

### Validation

```bash
make check
```
Screenshot comparison on field1, field2, dungeon before/after (expect identical layout; text glyphs will differ after 6.5). Frame-time check with F3/Ebiten metrics before and after — no regression; tile-loop clamp should show improvement on the largest map.

**Success criteria:** no draw code depends on tuning magic numbers; entities Y-sort correctly (verify player passing in front of/behind a shrine); scene layer contains no entity drawing.

---

## Phase dependency summary

```
P0 ──► P1 ──► P2 ──► P3 ──► P4
                      └────► P5
        P2 ─────────────────────► P6   (independent of P3–P5)
```

## Cross-phase ground rules for the agent (include if issues arise)

- If a task's line-number reference has drifted, locate the code by symbol name instead; the symbols are authoritative.
- If a described symbol does not exist (e.g. an event type with a slightly different name), adapt to the existing name and note the deviation in the commit message — do not invent parallel structures.
- Never delete or weaken an archtest rule to make code compile; restructure the code instead.
- If a migration/move breaks replay determinism tests, stop and fix the determinism issue before proceeding — that test failing is a real bug, not noise.
