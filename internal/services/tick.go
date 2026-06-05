// tick.go — simulation tick-rate contract.
//
// Ebitengine drives Update at a fixed TPS (ticks per second) and Draw at
// the monitor's FPS. The two loops are independent: if the GPU stalls,
// Update still runs at TPS; if Update lags, Draw reuses the last tick's
// world state.
//
// All per-tick counters in the simulation (swing frames, dodge timers,
// door cooldown, stamina regen cadence, world.World.Tick) assume this
// contract. If we ever change TickRate we must audit those counters; most
// of them should be re-expressed in seconds via TickSeconds.
//
// The platform entrypoint (cmd/game/main.go) is responsible for calling
// ebiten.SetTPS(services.TickRate) before ebiten.RunGame. Scenes and core
// code MUST NOT depend on ebiten.ActualTPS or wall-clock time for gameplay
// decisions — that would couple sim to frame rate.
package services

// TickRate is the authoritative simulation tick rate in Hz. 60 matches
// Ebiten's default and keeps per-tick counters readable (10 ticks = ~1/6s).
const TickRate = 60

// TickSeconds is the duration of one simulation tick. Exposed so future
// dt-aware systems (Phase 3) can integrate physics in SI units without
// hard-coding 1/60.
const TickSeconds = 1.0 / float64(TickRate)
