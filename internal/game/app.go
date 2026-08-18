// Package game is the thin wiring layer: it bundles the Ebiten-backed
// service implementations, Session, Renderer, and SceneManager, then
// satisfies ebiten.Game by routing Update/Draw/Layout through a reusable
// scenes.Context.
//
// Package responsibilities (deliberately narrow):
//
//   - Own the concrete services (Input/Audio/Assets/Renderer) via the
//     Services bundle.
//   - Own a persistent scenes.Context + scenes.Manager + scenes.Session.
//   - Pump the ebiten.Game lifecycle: Update -> weekly seed + audio tick +
//     scene update; Draw -> renderer.BeginFrame + scene draw + EndFrame.
//
// Non-goals:
//
//   - No gameplay. Scenes handle that (internal/scenes).
//   - No rendering logic. Platform renderer handles that
//     (internal/platform/ebiten/renderer.go).
//   - No input/audio decisions. Scenes make those via services.
package game

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	ebitenplat "github.com/jaredwarren/game-test/internal/platform/ebiten"
	"github.com/jaredwarren/game-test/internal/render"
	"github.com/jaredwarren/game-test/internal/run"
	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/scenes"
	"github.com/jaredwarren/game-test/internal/services"
)

// App is the root ebiten.Game implementation. It holds only wiring — no
// gameplay state — so the dependency graph is legible at a glance.
//
// The platform renderer is held concretely (not as services.Renderer)
// because App needs BeginFrame/EndFrame, which are not part of the
// scene-facing interface.
type App struct {
	session  *run.Session
	renderer *ebitenplat.Renderer
	input    *ebitenplat.Input
	manager  *scenes.Manager

	ctx *appContext
}

type appContext struct {
	input     services.Input
	audio     services.Audio
	assets    services.AssetCache
	renderer  services.Renderer
	clipboard services.Clipboard
	session   *run.Session
	manager   *scenes.Manager
}

func (c *appContext) Input() services.Input         { return c.input }
func (c *appContext) Audio() services.Audio         { return c.audio }
func (c *appContext) Assets() services.AssetCache   { return c.assets }
func (c *appContext) Renderer() services.Renderer   { return c.renderer }
func (c *appContext) Clipboard() services.Clipboard { return c.clipboard }
func (c *appContext) Session() *run.Session         { return c.session }
func (c *appContext) Manager() *scenes.Manager      { return c.manager }

// Services bundles the backend ports App requires. Input/Audio/Assets are
// mandatory; Clipboard may be nil on headless builds (scenes treat a nil
// Clipboard as a no-op for best-effort actions like "copy bug digest").
type Services struct {
	Input     *ebitenplat.Input
	Audio     services.Audio
	Assets    services.AssetCache
	Clipboard services.Clipboard
}

// NewApp constructs the app. If editorMapID is non-empty, starts in the
// disk-backed .tmj editor for that map id.
func NewApp(svc Services, editorMapID string) (*App, error) {
	if svc.Input == nil || svc.Audio == nil || svc.Assets == nil {
		return nil, fmt.Errorf("game: services.Input/Audio/Assets must all be non-nil")
	}

	sess := run.NewSession()
	if _, err := save.Load(""); err == nil {
		sess.HasSave = true
	}

	rend := ebitenplat.NewRenderer(render.NewCamera(320, 240), svc.Assets)
	mgr := scenes.NewManager()
	scenes.RegisterAll(mgr)

	ctx := &appContext{
		input:     svc.Input,
		audio:     svc.Audio,
		assets:    svc.Assets,
		renderer:  rend,
		clipboard: svc.Clipboard,
		session:   sess,
		manager:   mgr,
	}

	if editorMapID != "" {
		mgr.Replace(scenes.SceneEditor, map[string]any{"mapID": editorMapID})
	} else {
		mgr.Replace(scenes.SceneTitle, nil)
	}

	return &App{
		session:  sess,
		renderer: rend,
		input:    svc.Input,
		manager:  mgr,
		ctx:      ctx,
	}, nil
}

// Update runs one simulation tick. Ebitengine invokes this at the rate set
// via ebiten.SetTPS (services.TickRate, 60 Hz), independent of Draw's
// monitor-driven FPS. All per-tick counters in the sim (World.Tick,
// DoorCooldown, swing frames, stamina regen) rely on that contract; see
// internal/services/tick.go.
//
// The weekly seed refreshes every tick so the debug HUD and saves agree on
// the current epoch without scenes having to derive it.
func (a *App) Update() error {
	if a.input != nil {
		a.input.BeginFrame()
	}
	a.session.WeeklySeed = time.Now().Unix() / (7 * 24 * 3600)
	a.ctx.audio.Tick()
	return a.manager.Update(a.ctx)
}

// Draw renders the current scene. Renderer.BeginFrame captures the camera
// offset for world-space Draws; EndFrame releases the screen reference so
// post-frame draws nil-deref instead of racing the next frame.
func (a *App) Draw(screen *ebiten.Image) {
	a.renderer.BeginFrame(screen)
	defer a.renderer.EndFrame()
	a.manager.Draw(a.ctx)
	if a.input != nil && a.input.TouchEnabled() {
		a.input.DrawTouchControls(a.renderer)
	}
}

// Layout fixes the internal resolution; Ebiten scales to the window.
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 320, 240
}
