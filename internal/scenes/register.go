// register.go — factory registration.
//
// RegisterAll wires every scene's factory into a Manager. Call this once
// at App boot before the first Replace. Scene factories are package-private
// (newTitleScene etc.) so external code cannot bypass the manager and
// construct scenes directly.
package scenes

// RegisterAll registers every scene factory with mgr. Idempotent at the
// package level — but Manager.Register panics on duplicates, so call it
// exactly once per Manager.
func RegisterAll(mgr *Manager) {
	mgr.Register(SceneTitle, newTitleScene)
	mgr.Register(ScenePlay, newPlayScene)
	mgr.Register(ScenePause, newPauseScene)
	mgr.Register(SceneEditor, newEditorScene)
	mgr.Register(SceneShop, newShopScene)
}
