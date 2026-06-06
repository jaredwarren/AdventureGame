package scenes

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/world"
)

func TestRunState_RoundTripWorld(t *testing.T) {
	t.Parallel()
	w := &world.World{
		HP:       7,
		Currency: 42,
		Bombs:    3,
		HasTorch: true,
		SmallKey: 2,
		Stats:    progression.Stats{Vitality: 3, Resolve: 2, Might: 4},
		TimeOfDay:      1234,
		SelectedItem:   world.ItemSlotTorch,
	}
	w.Player.BaseSpeed = 2.5
	w.Player.MaxBombs = 10

	rs := RunStateFromWorld(w)
	out := &world.World{Stats: progression.DefaultStats()}
	out.Player = world.DefaultPlayerTuning()
	rs.ApplyTo(out)

	if out.Currency != 42 {
		t.Errorf("Currency = %d", out.Currency)
	}
	if out.SmallKey != 2 {
		t.Errorf("SmallKey = %d", out.SmallKey)
	}
	if out.Stats.Might != 4 {
		t.Errorf("Might = %d", out.Stats.Might)
	}
	if out.Player.BaseSpeed != 2.5 {
		t.Errorf("BaseSpeed = %v", out.Player.BaseSpeed)
	}
	if out.Player.MaxBombs != 10 {
		t.Errorf("MaxBombs = %d", out.Player.MaxBombs)
	}
}

func TestRunState_SaveRoundTrip(t *testing.T) {
	t.Parallel()
	w := &world.World{
		HP: 5, Currency: 99, Bombs: 2, HasTorch: true, SmallKey: 1,
		Stats: progression.Stats{Vitality: 2},
	}
	w.Player = world.DefaultPlayerTuning()
	w.Player.SwordReach = 18

	gs := RunStateFromWorld(w).ToGameSave("field1", 10, 20)
	out := &world.World{Stats: progression.DefaultStats()}
	out.Player = world.DefaultPlayerTuning()
	RunStateFromSave(gs).ApplyTo(out)

	if out.Currency != 99 || out.SmallKey != 1 {
		t.Errorf("got Currency=%d SmallKey=%d", out.Currency, out.SmallKey)
	}
	if out.Player.SwordReach != 18 {
		t.Errorf("SwordReach = %v", out.Player.SwordReach)
	}
}

func TestRespawn_PreservesRunProgress(t *testing.T) {
	assets := &mockAssetCache{mapJSON: tinyMapJSON()}
	sess := NewSession()
	if err := EnterMap(assets, sess, "dungeon", MapLoadOpts{}); err != nil {
		t.Fatal(err)
	}

	sess.World.Currency = 77
	sess.World.SmallKey = 3
	sess.World.HasTorch = true
	sess.World.Stats = progression.Stats{Vitality: 4, Might: 3}
	sess.World.HP = 1

	Respawn(assets, sess)
	if sess.World == nil {
		t.Fatal("world nil after respawn")
	}
	if sess.World.MapID != "field1" {
		t.Errorf("MapID = %q", sess.World.MapID)
	}
	if sess.World.Currency != 77 {
		t.Errorf("Currency = %d, want 77", sess.World.Currency)
	}
	if sess.World.SmallKey != 3 {
		t.Errorf("SmallKey = %d, want 3", sess.World.SmallKey)
	}
	if !sess.World.HasTorch {
		t.Error("HasTorch = false, want true")
	}
	if sess.World.Stats.Vitality != 4 || sess.World.Stats.Might != 3 {
		t.Errorf("stats = %+v", sess.World.Stats)
	}
	wantHP := sess.World.MaxHP() / 2
	if wantHP < 2 {
		wantHP = 2
	}
	if sess.World.HP != wantHP {
		t.Errorf("HP = %d, want %d", sess.World.HP, wantHP)
	}
}
