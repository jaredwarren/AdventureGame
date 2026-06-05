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

**Tests and hygiene:**

```bash
make test
make fmt
make vet
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
| **Debug overlay** | `F3` (FPS, map, player, combat timers, counts, dungeon digest) |
| Dungeon debug (`Tab` in dungeon) | Room **graph** (key/boss) + **Growing Tree** maze preview (top-right; 24×11, blend 0.65, 10% extra passages) |
| **Quicksave** | `S` while paused, or `Ctrl+S` during play |
| **Quickload** | `L` while paused |

**Saving:** progress is written to **`save.json`** in the working directory (ignored by git). Delete it to start fresh on disk.

**Flow (spoiler-free):** explore **field1**, use doors to **field2**, find a **bomb** and clear the **cracked wall**, use the **shrine** with coins. Enter the **dungeon** from the north door on field1; collect the **small key** to pass the **lock tile**, defeat the **boss** for extra coins, leave via the south door.

## Bug reports (seeds & digest)

When reporting a bug:

1. Pause (`P`). The overlay shows **weekly epoch**, **last dungeon seed**, and (in dungeon) a one-line **dungeon graph digest** (`seed`, rooms, start/boss/key, lock edge, edge count).
2. Press **`C`** to copy a **full bug digest** (map id, HP, weekly epoch, last dungeon seed, dungeon line if applicable) to the system clipboard—paste it into the issue.

## Editing maps (single asset pipeline)

1. Install [Tiled](https://www.mapeditor.org/) and open the JSON maps under **`assets/maps/`** (`.tmj` is JSON aligned with Tiled’s map format).
2. Edit **only** files under **`assets/maps/`**. They are embedded at compile time via [`assets/embed_maps.go`](assets/embed_maps.go) (`//go:embed maps/*.tmj`).
3. Run **`go build ./cmd/game`** or **`go run ./cmd/game`** — **no copy step**; the binary always embeds whatever is on disk in `assets/maps/` when you build.

**In-engine editor (dev):** `go run ./cmd/game -edit field1` (or `make edit field1`, `make edit MAP=field2`, or `make edit-field2`) loads `assets/maps/<id>.tmj` from **disk** (not the embed), lets you paint the `ground` layer and add/move/delete `markers` objects with the same Ebiten renderer as play mode, and writes the file with **Ctrl+S**. Run the game again (or rebuild) so embedded maps match what you saved.

Optional sanity check:

```bash
make maps-check
```

**Adding a new map:** place `yourmap.tmj` in `assets/maps/`, reference it from door `target_map` properties as `yourmap` (loader uses `maps/<name>.tmj`).

## Known limitations (prototype)

- **Placeholder art** — colored rectangles and debug font, not final sprites.
- **Death** — respawns on **field1** with partial HP (placeholder checkpoint model).
- **Dungeon layout** — one hand-authored tile shell per run; **graph + key/boss** placement is procedural (tree-shaped graph); see `internal/dungeon`.
- **Clipboard** — `C` on pause uses [atotto/clipboard](https://github.com/atotto/clipboard); if it fails on an unusual OS, copy the digest lines manually from the pause screen.
- **Windows clipboard in GLFW** — upstream stack may have limitations; file an issue with the copied text if clipboard fails.

## Project layout

| Path | Role |
|------|------|
| `cmd/game/main.go` | Entry: window, `RunGame` |
| `internal/game` | Scenes, input, draw, SFX |
| `internal/services` | Backend-agnostic ports (Input, Audio, Assets, Clock) |
| `internal/platform/ebiten` | Ebiten-backed implementations of `services` |
| `internal/world` | Simulation, Tiled build, combat hooks (ebiten-free) |
| `internal/dungeon` | Seeded **room graph** (key/boss) + **Growing Tree** grid mazes |
| `internal/tiled` | Tiled JSON parse + encode (`.tmj` round-trip) |
| `internal/save` | Versioned JSON save |
| `internal/archtest` | Tests that enforce architectural import rules |
| `assets/maps/` | **Source of truth** for `.tmj` maps (embedded) |
| `assets/sounds/` | Embedded WAV SFX |
| `assets/sprites/` | Embedded sprite sheets + atlases |
| `sprites/` | Raw art source (NES rips, chromakey intermediates; not embedded) |
| `docs/completion-steps.md` | Roadmap toward a full game |

**Reading the code:** each major package has a top-of-file or `package` doc describing intent and data flow. Look for `NOTE:` (behavior quirks / assumptions) and `TODO:` (planned refactors or missing features).

## License

Add a `LICENSE` file when you publish; none is set by default in this template.
