// Package scenes contains the scene-graph abstractions (Scene, Context,
// Manager) and the concrete scene implementations (title, play, pause,
// editor) plus overlays and particles.
//
// Architecture boundary:
//
//   - This package is pure core: it imports the simulation/core packages
//     (world, systems, run, services, render) and NEVER imports anything
//     under internal/platform or github.com/hajimehoshi/ebiten.
//     The arch-guard test (internal/archtest) enforces this.
//
//   - Scenes never see a *ebiten.Image. All drawing flows through
//     services.Renderer primitives; the platform renderer in
//     internal/platform/ebiten is the only place Ebiten draw APIs live.
//
// Scene lifecycle:
//
//   - Scenes implement the Scene interface. Update may call
//     ctx.Manager.Replace(nextID, params) or PushOverlay; transitions are
//     deferred until the current Update returns so Draw always observes a
//     consistent scene.
//
//   - Enter runs exactly once when a scene becomes active. Exit runs once
//     when it is replaced out. Neither runs during normal Update/Draw ticks.
//
// Two-slot model:
//
//   - The manager keeps a base scene plus at most one overlay (pause, shop).
//     PushOverlay / PopOverlay are intentionally not a general stack.
//     While an overlay is active: only the overlay Update runs; the base
//     scene Draw runs first, then the overlay Draw. This preserves base
//     scene-local state (particles, toasts) across pause and shop.
package scenes

import (
	"fmt"

	"github.com/jaredwarren/game-test/internal/run"
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

	Session() *run.Session
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

// Manager routes Update/Draw to the base scene and optional overlay.
type Manager struct {
	factories map[SceneID]Factory
	current   Scene
	overlay   Scene

	// pending captures a Replace request issued during Update.
	pending *transition
	// overlayPending captures PushOverlay / PopOverlay requests.
	overlayPending *overlayTransition

	transitions int
}

type overlayTransition struct {
	push   bool
	id     SceneID
	params map[string]any
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

// Replace queues a full scene swap (title, play, editor). Clears any overlay.
func (m *Manager) Replace(id SceneID, params map[string]any) {
	m.pending = &transition{id: id, params: params}
}

// PushOverlay queues a single overlay scene over the current base scene.
// Only one overlay is supported; a second push replaces the pending request.
func (m *Manager) PushOverlay(id SceneID, params map[string]any) {
	m.overlayPending = &overlayTransition{push: true, id: id, params: params}
}

// PopOverlay queues removal of the active overlay.
func (m *Manager) PopOverlay() {
	m.overlayPending = &overlayTransition{push: false}
}

// OverlayActive reports whether an overlay scene is showing.
func (m *Manager) OverlayActive() bool { return m.overlay != nil }

// apply performs any pending transition. Safe to call multiple times per
// frame; it's a no-op when nothing is queued.
func (m *Manager) apply(ctx GameContext) error {
	if m.pending != nil {
		t := m.pending
		m.pending = nil
		if m.overlay != nil {
			if err := m.overlay.Exit(ctx); err != nil {
				return fmt.Errorf("overlay %s.Exit: %w", m.overlay.ID(), err)
			}
			m.overlay = nil
		}
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
	}
	return m.applyOverlay(ctx)
}

func (m *Manager) applyOverlay(ctx GameContext) error {
	if m.overlayPending == nil {
		return nil
	}
	ot := m.overlayPending
	m.overlayPending = nil
	if !ot.push {
		if m.overlay != nil {
			if err := m.overlay.Exit(ctx); err != nil {
				return fmt.Errorf("overlay %s.Exit: %w", m.overlay.ID(), err)
			}
			m.overlay = nil
		}
		return nil
	}
	factory, ok := m.factories[ot.id]
	if !ok {
		return fmt.Errorf("scene manager: no factory registered for overlay %q", ot.id)
	}
	next := factory()
	if m.overlay != nil {
		if err := m.overlay.Exit(ctx); err != nil {
			return fmt.Errorf("overlay %s.Exit: %w", m.overlay.ID(), err)
		}
	}
	m.overlay = next
	if err := next.Enter(ctx, ot.params); err != nil {
		return fmt.Errorf("overlay %s.Enter: %w", next.ID(), err)
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
	if m.overlay != nil {
		if err := m.overlay.Update(ctx); err != nil {
			return err
		}
		return m.applyOverlay(ctx)
	}
	if err := m.current.Update(ctx); err != nil {
		return err
	}
	if err := m.apply(ctx); err != nil {
		return err
	}
	return m.applyOverlay(ctx)
}

// Draw delegates to the base scene, then any active overlay.
func (m *Manager) Draw(ctx GameContext) {
	if m.current == nil {
		return
	}
	m.current.Draw(ctx)
	if m.overlay != nil {
		m.overlay.Draw(ctx)
	}
}

// Transitions reports how many Enter/Exit cycles have completed since the
// manager was created. Handy for debug overlays and tests.
func (m *Manager) Transitions() int { return m.transitions }
