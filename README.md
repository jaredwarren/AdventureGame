# game-test — Ebiten top-down prototype

Small **Zelda-style** overworld plus **seeded procedural dungeon graph** (Go, [Ebiten v2](https://ebitengine.org/)). This repo is a vertical slice for expanding into a full adventure (see [docs/completion-steps.md](docs/completion-steps.md)).

## Requirements

- **Go** 1.21+ (module declares a newer toolchain; use a current stable Go release).
- **macOS / Linux / Windows** desktop (CGO may be used by Ebiten’s graphics backend on some platforms).

## Quick start

```bash
make run
# or: go run ./cmd/game
```

**Build a binary** (output under `bin/`):

```bash
make build
./bin/game-test
```

**Tests and hygiene (pre-commit gate):**

```bash
make check
# or individual targets: make test, make fmt, make vet
```

## How to play (playtester runbook)

| Action | Key |
|--------|-----|
| **New game** | `Enter` (from title) |
| **Load save** | `K` (from title, if `save.json` exists) |
| Move | Arrow keys |
| Sword | `Z` |
| Bomb (after pickup) | `X` (face a cracked wall) |
| Sprint | `Shift` (uses stamina) |
| Dodge | `Alt` |
| Shrine (heal / buy stat) | `E` while overlapping shrine (5 coins upgrades a stat; otherwise small heal) |
| Pause | `P` |
| **Debug overlay** | `F3` (scene, map, player, combat timers, stats, entity counts) |
| **Quicksave** | `S` while paused, or `Ctrl+S` during play |
| **Quickload** | `L` while paused |

**Saving:** progress is written to **`save.json`** in the working directory (ignored by git). Delete it to start fresh on disk.

**Flow (spoiler-free):** explore **field1**, use doors to **field2**, find a **bomb** and clear the **cracked wall**, use the **shrine** with coins. Enter the **dungeon** from the north door on field1; collect the **small key** to pass the **lock tile**, defeat the **boss** for extra coins, leave via the south door.

## Bug reports (seeds & digest)

When reporting a bug:

1. Pause (`P`). The overlay shows player stats, **weekly epoch**, and quicksave status.
2. Press **`C`** to copy a **bug digest** (`map`, `hp`, `weekly` epoch, `dungeon` digest, save path) to the system clipboard—paste it into the issue.

## Editing maps (single asset pipeline)

1. Install [Tiled](https://www.mapeditor.org/) and open the JSON maps under **`assets/maps/`** (`.tmj` is JSON aligned with Tiled’s map format).
2. Edit **only** files under **`assets/maps/`**. They are embedded at compile time via [`assets/embed_maps.go`](assets/embed_maps.go) (`//go:embed maps/*.tmj`).
3. Run **`go build ./cmd/game`** or **`go run ./cmd/game`** — **no copy step**; the binary always embeds whatever is on disk in `assets/maps/` when you build.

**Web editor (recommended):** `make mapeditor` (or `go run ./cmd/mapeditor -open`) serves a local editor at `http://127.0.0.1:7777` with a per-launch access token. It edits the same `assets/maps/*.tmj` files on disk, and everything it draws — tile art, marker hit boxes, default properties, validation rules — is derived from `internal/world` at startup, so it cannot drift from the game.

**Tile art editor:** `make tileeditor` (or `go run ./cmd/mapeditor -open-tiles`) opens the same server on `/tiles` for a WYSIWYG vector editor. Art is stored as `assets/tiles/{gid}_{name}.tile.json` (spatial variants included) and loaded by the game at startup. Grass, tree, dirt path, and cobble path bases are data-driven; other GIDs can be synthesized from their Go drawers and saved.

Over the in-engine editor it adds undo/redo, rectangle and flood fill, mouse-wheel zoom, a **property panel** for enemy stats, door targets, sign text and pickup kinds (previously only editable by hand-editing JSON), a **validation panel**, and a world-grid browser for jumping between adjacent cells. It preserves each file's own formatting, so opening a map and saving it unchanged leaves no diff.

**In-engine editor (dev):** `go run ./cmd/game -edit field1` (or `make edit field1`, `make edit MAP=field2`, or `make edit-field2`) loads `assets/maps/<id>.tmj` from **disk** (not the embed), lets you paint the `ground` layer and add/move/delete `markers` objects with the same Ebiten renderer as play mode, and writes the file with **Ctrl+S**. Run the game again (or rebuild) so embedded maps match what you saved.

Validate every map — the game's loader must accept it, GIDs must be registered, object ids must be unique (duplicates corrupt save keys), and doors must not drop the player inside a wall:

```bash
make maps-validate     # or: go run ./cmd/mapeditor -check
```

`make maps-check` runs the same validation plus the original file-presence check.

**Adding a new map:** place `yourmap.tmj` in `assets/maps/`, reference it from door `target_map` properties as `yourmap` (loader uses `maps/<name>.tmj`).

## Known limitations (prototype)

- **Placeholder art** — colored rectangles and debug font, not final sprites.
- **Death** — respawns on **field1** with partial HP (placeholder checkpoint model).
- **Dungeon layout** — currently uses hand-authored `.tmj` maps. Dungeon generation (`internal/dungeon`) is a CLI tool (`cmd/gentmj`) with no runtime integration.
- **Clipboard** — `C` on pause uses [atotto/clipboard](https://github.com/atotto/clipboard); if it fails on an unusual OS, copy the digest lines manually from the pause screen.
- **Windows clipboard in GLFW** — upstream stack may have limitations; file an issue with the copied text if clipboard fails.

## Project layout

| Path | Role |
|------|------|
| `cmd/game/main.go` | Entry: window, `RunGame` |
| `internal/game` | Thin wiring layer: satisfies `ebiten.Game`, owns concrete services bundle, renderer, and scene manager |
| `internal/scenes` | Scene lifecycle, state management, HUD, and overlays |
| `internal/systems` | Per-tick simulation systems and event pipeline |
| `internal/services` | Backend-agnostic ports (Input, Audio, Assets, Clock, Clipboard) |
| `internal/platform/ebiten` | Ebiten-backed implementations of `services` and renderer |
| `internal/world` | Simulation state, Tiled build, entity management (ebiten-free) |
| `internal/dungeon` | Maze generation algorithms and tile painting (CLI tool `cmd/gentmj`) |
| `internal/tiled` | Tiled JSON parse + encode (`.tmj` round-trip) |
| `internal/save` | Versioned JSON save |
| `internal/replay` | Deterministic input recording and replay validation |
| `internal/input` | Keybindings, input action mapping, and gamepad definitions |
| `internal/geom` | Integer/float geometry primitives (`Rect`, `Point`, `Vec2`) |
| `internal/archtest` | Tests that enforce architectural import rules |
| `assets/maps/` | **Source of truth** for `.tmj` maps (embedded) |
| `assets/sounds/` | Embedded WAV SFX |
| `assets/sprites/` | Embedded sprite sheets + atlases |
| `sprites/` | Raw art source (NES rips, chromakey intermediates; not embedded) |
| `docs/completion-steps.md` | Roadmap toward a full game |

**Reading the code:** each major package has a top-of-file or `package` doc describing intent and data flow. Look for `NOTE:` (behavior quirks / assumptions) and `TODO:` (planned refactors or missing features).

## License

Add a `LICENSE` file when you publish; none is set by default in this template.
