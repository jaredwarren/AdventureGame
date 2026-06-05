// pause.go — frozen menu overlay with stats / save / load.
//
// Notes:
//
//   - PauseScene does not run World.UpdateSim. It only processes input and
//     calls the worldloader helpers; the gameplay sim is literally paused.
//   - Clipboard access for "copy bug digest" flows through the
//     services.Clipboard port; on a headless Context (ctx.Clipboard nil)
//     the action silently no-ops.
package scenes

import (
	"fmt"

	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/services"
)

// PauseScene freezes the world and shows stats/save/load options.
type PauseScene struct{}

func newPauseScene() Scene { return &PauseScene{} }

func (s *PauseScene) ID() SceneID                                     { return ScenePause }
func (s *PauseScene) Enter(ctx GameContext, params map[string]any) error { return nil }
func (s *PauseScene) Exit(ctx GameContext) error                         { return nil }

func (s *PauseScene) Update(ctx GameContext) error {
	in := ctx.Input()
	sess := ctx.Session()
	cam := ctx.Renderer().Camera()

	if in.JustPressed(services.ActionDebugToggle) {
		sess.ShowDebugOverlay = !sess.ShowDebugOverlay
	}
	if in.JustPressed(services.ActionPause) {
		ctx.Manager().Replace(ScenePlay, nil)
		return nil
	}
	if in.JustPressed(services.ActionCopyBugDigest) && ctx.Clipboard() != nil {
		_ = ctx.Clipboard().WriteText(BugDigest(sess))
	}
	if in.JustPressed(services.ActionToggleReduceShake) {
		cam.ReduceShake = !cam.ReduceShake
	}
	// Pause save: "S" alone. We explicitly exclude Ctrl+S so QuickLoad
	// ("L") and a chorded save can't both fire on the same frame.
	if in.JustPressed(services.ActionQuickSave) && !in.IsModifierDown(services.ModCtrl) {
		_ = save.Save("", BuildSave(sess, cam))
	}
	if in.JustPressed(services.ActionQuickLoad) {
		if sv, err := save.Load(""); err == nil {
			_ = LoadGameFromSave(ctx.Assets(), sess, cam, sv)
		}
	}
	return nil
}

func (s *PauseScene) Draw(ctx GameContext) {
	sess := ctx.Session()
	r := ctx.Renderer()

	r.DrawWorld(sess.World)
	DrawHUD(r, sess.World, sess)

	r.DrawText(80, 40, "PAUSED (P resume)")
	if sess.World != nil {
		r.DrawText(40, 64, fmt.Sprintf("VIT %d RES %d MIG %d WIT %d FOR %d",
			sess.World.Stats.Vitality, sess.World.Stats.Resolve,
			sess.World.Stats.Might, sess.World.Stats.Wits, sess.World.Stats.Fortune))
	}
	r.DrawText(40, 88, fmt.Sprintf("reduce_shake=%v (R toggle)", r.Camera().ReduceShake))
	r.DrawText(40, 104, "S quicksave  L quickload")
	r.DrawText(40, 120, fmt.Sprintf("weekly_epoch %d", sess.WeeklySeed))
	r.DrawText(40, 144, "C copy bug digest")

	if sess.ShowDebugOverlay {
		DrawDebugOverlay(r, sess, s.ID(), DebugOverlayExtras{})
	}
}
