// Package run holds cross-scene run state: Session, per-map progress, portable
// RunState, and map/save/door/respawn orchestration (worldloader).
//
// It is headless — no Ebiten, no scenes — so loader logic can be tested
// without constructing a scene graph.
package run
