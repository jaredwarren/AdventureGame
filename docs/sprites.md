# Drawable sizes (reference for sprites)

Logical game resolution is **320×240** (`internal/game/game.go` `Layout`). World drawing subtracts camera origin and shake (`drawWorld`).

Source of truth in code: `internal/world` (hitboxes, tile GIDs), `internal/game/draw.go` (placeholder draws).

---

## Tiles (ground layer)

All tiles are **16×16** pixels in the world (`world.TileSize`). In-game, when `useTileSprites` is true in `internal/game/draw.go`, ground tiles render from **`assets/sprites/overworld-spritesheet.png`** using **`assets/sprites/tile_atlas.json`** (`internal/game/tileset_sprites.go`). Otherwise `tileColor` placeholders fill each cell.

Legacy **`assets/sprites/tileset_alpha.png`** remains embedded for `scripts/find_tileset_rects` only.

Note: the top-level `sprites/` directory holds raw art source (originals, chromakey intermediates). It is **not** embedded — only files under `assets/sprites/` are compiled into the binary.

### Tile atlas (`tile_atlas.json`)

| Field | Meaning |
|-------|--------|
| `sprite_sheet` | **Required.** Basename paired with `sprites.TileSpriteSheetFile` / `sprites.OverworldTilePNG`. |
| `tile_px` | Optional hint (e.g. **16**); rects may use other sizes—the game scales each crop to `world.TileSize`. |
| `frames` | One entry per GID, index **`i` == GID `i`** (see `world.GID*` in `internal/world/tiles.go`, including `GIDTree` = 8). |

Per frame:

| Field | Meaning |
|-------|--------|
| `skip` | If true, no sprite for that GID (`GIDEmpty` should use this). Nothing is drawn for empty tiles when using sprites. |
| `sx`, `sy`, `sw`, `sh` | Source crop in sheet pixels (required when not `skip`; `sw`/`sh` ≥ 1). |

| GID | Constant     | Notes |
|-----|--------------|--------|
| 0   | `GIDEmpty`   | Often off-map |
| 1   | `GIDGrass`   | Same placeholder color as `GIDFloor2` today |
| 2   | `GIDWall`    | |
| 3   | `GIDCracked` | Destroyable by **bomb** (`world.DamageBomb`) |
| 4   | `GIDDoor`    | Floor tile; warps usually use door **objects** |
| 5   | `GIDWater`   | |
| 6   | `GIDLock`    | Opens with small key |
| 7   | `GIDFloor2`  | |
| 8   | `GIDTree`    | Destroyable by **fire** (`world.DamageFire`); bombs do not affect it |
| 9–16| `GIDWaterShore*` | Straight and outer corner water shore transition tiles |
| 17–20| `GIDWaterShore*Inner` | Inner corner water shore transition tiles |
| 21  | `GIDDirtPath` | Full dirt path tile (walkable floor) |
| 22–25| `GIDWallTop` / `Bottom` / `Left` / `Right` | Straight wall edge transition tiles |
| 26–29| `GIDWallNE` / `NW` / `SW` / `SE` | Outer corner wall transition tiles |
| 30–33| `GIDWall*Inner` | Inner corner wall transition tiles |
| 34–46| `GIDRock*`   | Rock base, edge, and corner transition tiles |
| 47–52| `GIDQuicksand`, `GIDMud`, `GIDIce`, `GIDLava`, `GIDSign`, `GIDSand` | Terrain and surface tiles |
| 53–56| `GIDDirtPathTop` / `Bottom` / `Left` / `Right` | Straight dirt path edge transition tiles (half-dirt, half-transparent, fully walkable) |
| 57–60| `GIDDirtPathNE` / `NW` / `SW` / `SE` | Outer corner dirt path transition tiles (fully walkable) |
| 61–64| `GIDDirtPath*Inner` | Inner corner dirt path transition tiles (fully walkable) |
| 65–77| `GIDCobblePath` (65) + 12 transitions | Cobblestone path autotile family (base + 12 transitions, fully walkable) |
| 78–90| `GIDSandPath` (78) + 12 transitions | Sand & beach autotile family (base + 12 transitions, fully walkable) |
| 91–103| `GIDSnow` (91) + 12 transitions | Snow & alpine autotile family (base + 12 transitions, fully walkable) |
| 104–116| `GIDMudPath` (104) + 12 transitions | Mud & swamp autotile family (base + 12 transitions, fully walkable, mud surface) |
| 117–129| `GIDDarkGrass` (117) + 12 transitions | Dark forest earth autotile family (base + 12 transitions, fully walkable) |
| 130–142| `GIDDeepWater` (130) + 12 transitions | Deep ocean water autotile family (base + 12 transitions, solid water) |
| 143–155| `GIDLavaShore` (143) + 12 transitions | Lava & magma shore autotile family (base + 12 transitions, solid hazard water) |
| 156–168| `GIDSwampWater` (156) + 12 transitions | Murky swamp water autotile family (base + 12 transitions, solid water) |
| 169–181| `GIDIceFamily` (169) + 12 transitions | Ice & frozen lake autotile family (base + 12 transitions, slippery ice surface) |
| 182–194| `GIDQuicksandFamily` (182) + 12 transitions | Quicksand hazard autotile family (base + 12 transitions, quicksand surface) |

Per-tile behavior (solid, destroyable-by-what, openable-by-key, fallback swatch color, becomes-what-when-destroyed) is declared in one place: the `tileDefs` registry in `internal/world/tiledef.go`. Adding a new ground tile type is:

1. New `GID*` constant in `internal/world/tiles.go`.
2. New `TileDef` entry in `tiledef.go` (pick which `DamageKind`s destroy it, set `DestroyedGID` if it should leave something other than grass, provide a `SwatchColor`).
3. Optional new frame in `assets/sprites/tile_atlas.json` (frame index = GID).

Destruction is unified: any weapon calls `World.TryDamageFaceTile(kind)`, and destroyed tiles persist via `save.GameSave.DestroyedTileKeys` (JSON tag kept as `broken_cracked_tiles` for save compatibility). Adding a new weapon means adding a new `DamageKind` constant and calling `TryDamageFaceTile` with it — no changes to collision, renderer, persistence, or editor are required.

See `internal/world/tiles.go`, `internal/world/tiledef.go`, `internal/game/tileset_sprites.go`, and `internal/game/draw.go` (`tileColor` fallback when a frame has no sub-image).

---

## Player

| Part        | Size (px) | Directions |
|-------------|-----------|------------|
| Body/hitbox | **12×12** | **4** facings: down (0), up (1), left (2), right (3) |

- `world.Player.W`, `H` default **12×12** from `BuildFromTiled` (`internal/world/from_tiled.go`).
- Facing: `world.Player.Dir` — constants in `internal/world/direction.go` (`DirDown`, `DirUp`, `DirLeft`, `DirRight`).

---

## Sword (swing / hitbox)

Active only during the middle of the swing (`world.SwordHitbox`). Anchor from **player center** `(Player.X + W/2, Player.Y + H/2)`.

| Facing | Hitbox width × height (px) |
|--------|----------------------------|
| Down   | **10 × 14** |
| Up     | **10 × 14** |
| Left   | **14 × 10** |
| Right  | **14 × 10** |

Constants in code: `reach = 14`, `thick = 10` (`internal/world/world.go` `SwordHitbox`).

Draw today: that rectangle, semi-transparent (`drawWorld`).

---

## Enemies

| Variant | Size (px) | Notes |
|---------|-----------|--------|
| Normal  | **14×12** | `world.EnemyRect` |
| Boss    | **14×12** | Same box; `Enemy.IsBoss` — separate sprite recommended |

---

## Pickups

Collision / spawn overlap stays a **~12×12** AABB at the pickup’s world position (`internal/world/world.go`). **On-screen size and alignment** come from the embedded atlas so each kind can crop/scale independently without recompiling Go.

- Sheet: `assets/sprites/legendofzelda_items_sheet.png` (embedded as `sprites.PickupPNG`; basename constant `sprites.PickupSpriteSheetFile`).
- Atlas: `assets/sprites/pickup_atlas.json` (embedded as `sprites.PickupAtlasJSON`), loaded in `internal/game/pickup_sprites.go`. The atlas **must** name the sheet it was authored against (see `sprite_sheet` below); if it does not match `PickupSpriteSheetFile`, pickup art does not load (fallback draw).

### Pickup atlas (`pickup_atlas.json`)

| Field | Meaning |
|-------|--------|
| `sprite_sheet` | **Required.** Basename of the PNG this atlas crops (e.g. `legendofzelda_items_sheet.png`). Must equal `sprites.PickupSpriteSheetFile` so embedded JSON and embedded bytes stay paired. |

The `frames` array has **exactly five** entries in **`PickupKind` iota order**:

1. coin — `PickupCoin`
2. heart — `PickupHeart`
3. bomb — `PickupBomb`
4. small key — `PickupSmallKey`
5. torch — `PickupTorch` (sets `World.HasTorch`; swing with `ActionTorch` / default key **T**)

| Field | Meaning |
|-------|--------|
| `sx`, `sy` | Top-left of the source rectangle in **sheet pixels** (for the file named by `sprite_sheet`). |
| `sw`, `sh` | Source width and height (**must be ≥ 1**). The rectangle must lie fully inside the PNG bounds (see `assets/sprites/pickup_atlas_test.go`). |
| `dw`, `dh` | Draw size in **world pixels**. If omitted or zero, defaults to **12×12** (same as historical `pickupDrawPx`). |
| `ox`, `oy` | Optional **draw offset** in world pixels, applied after camera subtraction: final position is `(p.X - camX + ox, p.Y - camY + oy)`. Does not move the pickup hitbox. |

Extra JSON keys (e.g. per-frame `"comment"`) are ignored by the loader.

| Kind       | `PickupKind` constant   |
|------------|-------------------------|
| Coin       | `PickupCoin`            |
| Heart      | `PickupHeart`           |
| Bomb       | `PickupBomb`            |
| Small key  | `PickupSmallKey`        |
| Torch      | `PickupTorch`           |

Tiled marker `type: pickup` + property `kind: torch` spawns one. Defined in `internal/world/world.go`.

---

## Doors and shrines (Tiled objects)

Sizes are **per object** from map data (`Door.Rect`, `Shrine.Rect`): width × height in **pixels**, not fixed globally.

Examples from authored maps: door rects **16×32**, **32×8**, etc.

Draw today:

- Door: filled rect + **2px** stroke outline.
- Shrine: **2px** stroke only (no fill).

---

## HUD (screen space)

| Element        | Size / behavior |
|----------------|-----------------|
| Heart pip      | **7×8** per unit; count = `MaxHP()` (varies with stats) |
| Stamina track  | **60×4**; fill width scales with stamina |
| Coin / key / bomb / text | Bitmap font (`DebugPrintAt`), not fixed pixel art size |

`internal/game/draw.go` (`drawHUD`).

---

## Debug / dev-only (optional)

| Element           | Approx size | Where |
|-------------------|-------------|--------|
| Dungeon graph node | **12×12** | `drawDungeonGraph` |
| Maze preview cell  | **4×4** per cell | `drawMazePreview` |
| F3 debug panel     | **214×188** translucent | `drawDebugOverlay` |

---

## Sprite sheet checklist

1. **Tiles:** 16×16; 8 GIDs (grass/floor2 can share art); cracked optional “broken” variant.
2. **Player:** 12×12 × **4** walk facings; sword **4** directional frames (or one slash centered on player center).
3. **Enemy:** 14×12; **boss** variant same footprint.
4. **Pickups:** 12×12 × **4** kinds (coin, heart, bomb, key).
5. **Doors / shrines:** pick standard sizes in Tiled or author per-map sprites matching each object’s `width`/`height`.

When you change any of these numbers, update the matching constants in `internal/world` / `draw.go` or the art will misalign with collision.
