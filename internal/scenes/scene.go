// Package scenes contains the scene-graph abstractions (Scene, Context,
// Manager, Session) and the concrete scene implementations (title, play,
// pause, editor) plus their orchestration helpers (worldloader, overlays).
//
// Architecture boundary:
//
//   - This package is pure core: it imports the simulation/core packages
//     (world, dungeon, save, tiled, geom, progression, render) and the
//     service ports (services.Input/Audio/AssetCache/Renderer) but NEVER
//     imports anything under internal/platform or github.com/hajimehoshi/ebiten.
//     The arch-guard test (internal/archtest) enforces this.
//
//   - Scenes never see a *ebiten.Image. All drawing flows through
//     services.Renderer primitives; the platform renderer in
//     internal/platform/ebiten is the only place Ebiten draw APIs live.
//
// Scene lifecycle:
//
//   - Scenes implement the Scene interface. Update may call
//     ctx.Manager.Replace(nextID, params); the transition is deferred until
//     the current Update returns so Draw always observes a consistent scene.
//
//   - Enter runs exactly once when a scene becomes active. Exit runs once
//     when it is replaced out. Neither runs during normal Update/Draw ticks.
//
// Single-slot model:
//
//   - The manager keeps a single active scene. Push/Pop stacks (pause-over-
//     play overlays) are a future extension; today every transition is a
//     Replace and a stack would be dead weight.
package scenes

import (
	"fmt"

	"github.com/jaredwarren/game-test/internal/services"
)

// SceneID is a stable identifier for a scene. Add a new constant when you
// introduce a new scene so the manager can build it without a concrete type
// reference.
type SceneID string

const (
	SceneTitle  SceneID = "title"
	ScenePlay   SceneID = "play"
	ScenePause  SceneID = "pause"
	SceneEditor SceneID = "editor"
	SceneShop   SceneID = "shop"
)

// GameContext bundles the state and services a scene is allowed to touch.
// It is defined as an interface to decouple scenes from concrete platform
// implementations and support mock-testing.
type GameContext interface {
	Input() services.Input
	Audio() services.Audio
	Assets() services.AssetCache
	Renderer() services.Renderer
	Clipboard() services.Clipboard

	Session() *Session
	Manager() *Manager
}

// Scene is one mode of the game loop. Only Update may trigger transitions;
// Draw must be side-effect free with respect to scene/session state.
type Scene interface {
	// ID returns this scene's registered identifier (used for debug output
	// and transition bookkeeping).
	ID() SceneID

	// Enter runs exactly once when this scene becomes active, after the
	// outgoing scene's Exit. Params carries optional transition args from
	// Replace(id, params).
	Enter(ctx GameContext, params map[string]any) error

	// Exit runs exactly once when this scene is about to be replaced. Use
	// it to release scene-local resources and snapshots.
	Exit(ctx GameContext) error

	// Update advances the scene one tick. Returning an error halts the game
	// loop; Ebiten surfaces it via RunGame.
	Update(ctx GameContext) error

	// Draw renders the scene. Renderer.BeginFrame has already been called
	// by App; scenes draw via ctx.Renderer().* primitives only.
	Draw(ctx GameContext)
}

// Factory builds a fresh Scene instance on demand. Factories enable lazy
// construction so a scene's setup cost is only paid when we actually
// transition to it.
type Factory func() Scene

// Manager routes Update/Draw to the current scene and applies deferred
// transitions at safe points.
type Manager struct {
	factories map[SceneID]Factory
	current   Scene

	// pending captures a Replace request issued during Update. It is
	// consumed between the current frame's Update and Draw so Draw always
	// sees the post-transition scene.
	pending     *transition
	transitions int
}

type transition struct {
	id     SceneID
	params map[string]any
}

// NewManager returns an empty manager. Register factories via Register
// before calling Replace.
func NewManager() *Manager {
	return &Manager{factories: make(map[SceneID]Factory)}
}

// Register associates a factory with an ID. Panics on duplicate
// registration because re-registering a scene is almost always a bug.
func (m *Manager) Register(id SceneID, f Factory) {
	if _, ok := m.factories[id]; ok {
		panic(fmt.Sprintf("scene manager: duplicate registration for %q", id))
	}
	m.factories[id] = f
}

// Current returns the active scene, or nil before any transition has been
// applied. Exposed for debug overlays only.
func (m *Manager) Current() Scene { return m.current }

// Replace queues a transition to id with optional params. The swap is
// deferred; the currently-running Update will finish before the new scene's
// Enter runs. Calling Replace twice in one frame keeps the last request.
func (m *Manager) Replace(id SceneID, params map[string]any) {
	m.pending = &transition{id: id, params: params}
}

// apply performs any pending transition. Safe to call multiple times per
// frame; it's a no-op when nothing is queued.
func (m *Manager) apply(ctx GameContext) error {
	if m.pending == nil {
		return nil
	}
	t := m.pending
	m.pending = nil

	factory, ok := m.factories[t.id]
	if !ok {
		return fmt.Errorf("scene manager: no factory registered for %q", t.id)
	}
	next := factory()

	if m.current != nil {
		if err := m.current.Exit(ctx); err != nil {
			return fmt.Errorf("scene %s.Exit: %w", m.current.ID(), err)
		}
	}
	m.current = next
	m.transitions++
	if err := next.Enter(ctx, t.params); err != nil {
		return fmt.Errorf("scene %s.Enter: %w", next.ID(), err)
	}
	return nil
}

// Update advances the active scene. It applies any pending transition first
// (so the very first frame sees the initial scene after Enter), runs
// Scene.Update, then applies any transition the scene queued.
func (m *Manager) Update(ctx GameContext) error {
	if err := m.apply(ctx); err != nil {
		return err
	}
	if m.current == nil {
		return fmt.Errorf("scene manager: no scene set; call Replace before Update")
	}
	if err := m.current.Update(ctx); err != nil {
		return err
	}
	return m.apply(ctx)
}

// Draw delegates to the active scene. If no scene is set (e.g. an early
// transition errored), draws nothing rather than calling a half-alive scene.
func (m *Manager) Draw(ctx GameContext) {
	if m.current == nil {
		return
	}
	m.current.Draw(ctx)
}

// Transitions reports how many Enter/Exit cycles have completed since the
// manager was created. Handy for debug overlays and tests.
func (m *Manager) Transitions() int { return m.transitions }
