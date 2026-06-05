// Package systems hosts the per-tick sim behaviors that used to live in
// world.UpdateSim as one monolithic method.
//
// Design:
//
//   - A System is a stateless unit of work that reads+mutates *world.World
//     and pushes events to the shared *EventBus. Systems never see Ebiten,
//     Audio, Input, Renderer, or the scene graph.
//
//   - A Pipeline runs a fixed ordered list of Systems and drains the bus
//     into a slice for the caller (a scene) to react to. The scene
//     translates events into SFX, camera shake, hit-stop, etc.; this keeps
//     gameplay logic (this package) and framework wiring (platform + scenes)
//     cleanly separated.
//
//   - Events replace the pre-Phase-3 "snapshot enemy HPs before UpdateSim,
//     diff them after" hack in PlayScene. Events are authoritative: a
//     HitEvent is what it means to damage an enemy, no inference needed.
//
// Architecture boundary:
//
//   - internal/systems is pure core (arch-test enforced). It imports only
//     the sim/core packages (world, geom). It MUST NOT import Ebiten,
//     internal/platform, internal/scenes, or internal/game.
//
//   - Time is passed as dt float64 (seconds per tick) so systems can be
//     frame-rate-agnostic in the future. Today every system ignores dt
//     because we pin Update at TickRate Hz (see internal/services/tick.go).
//
// Not yet in scope:
//
//   - Stamina regen stays in PlayScene because drain is input-bound; when
//     we add a Player.Sprinting flag in a later phase it can move here.
//   - Doors / warp / respawn are still scene-driven because the transition
//     needs the AssetCache port. They're orchestration, not simulation.
package systems
