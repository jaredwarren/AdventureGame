package scenes

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/run"
	"github.com/jaredwarren/game-test/internal/services"
)

type stubScene struct {
	id      SceneID
	updates int
	draws   int
}

func (s *stubScene) ID() SceneID                             { return s.id }
func (s *stubScene) Enter(GameContext, map[string]any) error { return nil }
func (s *stubScene) Exit(GameContext) error                  { return nil }
func (s *stubScene) Update(GameContext) error {
	s.updates++
	return nil
}
func (s *stubScene) Draw(GameContext) { s.draws++ }

type noopCtx struct{}

func (noopCtx) Input() services.Input         { return nil }
func (noopCtx) Audio() services.Audio         { return nil }
func (noopCtx) Assets() services.AssetCache   { return nil }
func (noopCtx) Renderer() services.Renderer   { return nil }
func (noopCtx) Clipboard() services.Clipboard { return nil }
func (noopCtx) Session() *run.Session         { return nil }
func (noopCtx) Manager() *Manager             { return nil }

func TestManager_OverlayPausesBaseUpdate(t *testing.T) {
	base := &stubScene{id: "base"}
	overlay := &stubScene{id: "overlay"}

	m := &Manager{
		factories: map[SceneID]Factory{
			"base":    func() Scene { return base },
			"overlay": func() Scene { return overlay },
		},
		current: base,
	}

	ctx := noopCtx{}

	if err := m.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if base.updates != 1 || overlay.updates != 0 {
		t.Fatalf("expected base update only, got base=%d overlay=%d", base.updates, overlay.updates)
	}

	m.PushOverlay("overlay", nil)
	if err := m.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if base.updates != 1 || overlay.updates != 1 {
		t.Fatalf("after push: expected base frozen overlay=1, got base=%d overlay=%d", base.updates, overlay.updates)
	}

	m.Draw(ctx)
	if base.draws != 1 || overlay.draws != 1 {
		t.Fatalf("draw order: expected both drawn once, got base=%d overlay=%d", base.draws, overlay.draws)
	}

	m.PopOverlay()
	if err := m.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if base.updates != 2 || overlay.updates != 1 {
		t.Fatalf("after pop: expected base resumed, got base=%d overlay=%d", base.updates, overlay.updates)
	}
	if m.OverlayActive() {
		t.Fatal("overlay should be cleared after pop")
	}
}
