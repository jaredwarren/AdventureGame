// title.go — idle start screen.
//
// TitleScene is stateless; it owns no scene-local data. Enter/Exit exist so
// a future "fade in title music" hook has a natural home without changing
// the interface.
package scenes

import (
	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/services"
)

// TitleScene is the start screen shown before gameplay begins.
type TitleScene struct{}

func newTitleScene() Scene { return &TitleScene{} }

func (s *TitleScene) ID() SceneID                                     { return SceneTitle }
func (s *TitleScene) Enter(ctx GameContext, params map[string]any) error { return nil }
func (s *TitleScene) Exit(ctx GameContext) error                         { return nil }

func (s *TitleScene) Update(ctx GameContext) error {
	in := ctx.Input()
	sess := ctx.Session()

	if in.JustPressed(services.ActionDebugToggle) {
		sess.ShowDebugOverlay = !sess.ShowDebugOverlay
	}
	if in.JustPressed(services.ActionConfirm) {
		sess.ClearPersistedProgress()
		if err := OpenWorld(ctx.Assets(), sess, "field1", nil); err != nil {
			return err
		}
		ctx.Manager().Replace(ScenePlay, nil)
		return nil
	}
	// QuickLoad from title ("K") only fires when there's a save on disk.
	if sess.HasSave && in.JustPressed(services.ActionQuickLoad) {
		sv, err := save.Load("")
		if err != nil {
			return err
		}
		if err := LoadGameFromSave(ctx.Assets(), sess, ctx.Renderer().Camera(), sv); err != nil {
			return err
		}
		// Extra grace period beyond the worldloader default; the player
		// just pressed a button, they should not also eat a door warp.
		if sess.World != nil {
			sess.World.DoorCooldown = 90
		}
		ctx.Manager().Replace(ScenePlay, nil)
	}
	return nil
}

func (s *TitleScene) Draw(ctx GameContext) {
	r := ctx.Renderer()
	r.DrawText(40, 40, "overworld + proc dungeon slice")
	r.DrawText(40, 72, "ENTER new game   K load save")
	r.DrawText(40, 96, "arrows move  Z sword  X bomb")
	r.DrawText(40, 112, "shift sprint  alt dodge  E shrine")
	r.DrawText(40, 128, "P pause")
	r.DrawText(40, 144, "F3 playtest debug overlay")
	if ctx.Session().ShowDebugOverlay {
		DrawDebugOverlay(r, ctx.Session(), s.ID(), DebugOverlayExtras{})
	}
}
