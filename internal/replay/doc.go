// Package replay provides input recording and deterministic playback on
// top of services.Input.
//
// # Why this works
//
// The engine's simulation is pure:
//
//   - systems take (w *world.World, bus *EventBus, dt float64) and update
//     state; no time.Now, no filesystem, no network, no RNG on the hot
//     path (dungeon gen is seeded, one-shot, not re-run during play).
//   - Tick rate is pinned at services.TickRate via ebiten.SetTPS.
//   - Entities carry stable EntityIDs; events carry IDs, not slice
//     indices.
//
// That means a tick's transition is fully determined by (pre-state,
// input, dt). If we reproduce the same pre-state and feed back the same
// input stream, we get the same post-state and the same event trace.
//
// What this package provides
//
//   - Stream + Frame: the on-wire format for a recorded input session.
//     Each Frame is a snapshot of "which actions/modifiers/mouse buttons
//     are currently down" at one sim tick. Edge events (JustPressed,
//     JustReleased) are not stored; they're computed at playback from
//     the delta between consecutive frames (same algorithm as inpututil).
//
//   - Recorder: wraps a live services.Input. Callers invoke BeginTick()
//     once per sim tick; the recorder samples every known action and
//     appends a Frame to an accumulating Stream. All services.Input
//     methods pass through to the wrapped implementation during the tick.
//
//   - Playback: implements services.Input by serving from a prerecorded
//     Stream. Callers invoke Step() once per tick to advance the frame
//     pointer. Past the end of the stream, Step returns false and all
//     polling methods return zero values.
//
// Layering
//
//   - This package is pure core: it imports only internal/services (for
//     the Input interface / Action enum) and internal/input (for the
//     Action↔name table). The archtest (internal/archtest) enforces
//     that it never reaches for ebiten.
//
// Scope (Phase 6)
//
//   - Record + Playback are building blocks. Integrating recorder into
//     the live scene graph (a "start/stop recording" hotkey) is future
//     work; it's a scene concern, not a replay-package concern.
//
//   - Starting world state is the caller's responsibility. A replay
//     session is (startingWorld, stream) → eventTrace. If you want to
//     replay across map transitions or save/load, encode the map id
//
//   - seed in your test harness; Stream intentionally does not own
//     that.
//
//   - RNG determinism is not enforced here. Systems today don't use rand
//     during a tick, so this is a non-issue. If a future system does,
//     it must accept a *rand.Rand seeded by the test; Stream has a
//     reserved Seed field for that day.
package replay
