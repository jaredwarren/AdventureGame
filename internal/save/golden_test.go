package save_test

import (
	"path/filepath"
	"testing"

	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/world"
)

func TestGoldenSaveCurrentVersion(t *testing.T) {
	path := filepath.Join("testdata", "v_current.json")
	sv, err := save.Load(path)
	if err != nil {
		t.Fatalf("failed to load golden save %s: %v", path, err)
	}

	if sv.Version != save.CurrentVersion {
		t.Errorf("expected version %d, got %d", save.CurrentVersion, sv.Version)
	}
	if sv.MapID != "field1" {
		t.Errorf("expected MapID field1, got %q", sv.MapID)
	}
	if sv.PlayerX == 0 || sv.PlayerY == 0 {
		t.Errorf("expected non-zero player position, got (%f, %f)", sv.PlayerX, sv.PlayerY)
	}
	if sv.HP <= 0 || sv.Currency <= 0 || sv.Bombs <= 0 || !sv.HasTorch || sv.SmallKey <= 0 {
		t.Errorf("expected non-zero core player state, got HP=%d Currency=%d Bombs=%d Torch=%v Key=%d",
			sv.HP, sv.Currency, sv.Bombs, sv.HasTorch, sv.SmallKey)
	}
	if sv.Vitality <= 0 || sv.Resolve <= 0 || sv.Might <= 0 || sv.Wits <= 0 || sv.Fortune <= 0 {
		t.Errorf("expected non-zero stats")
	}
	if len(sv.CollectedPickupKeys) == 0 || len(sv.DestroyedTileKeys) == 0 || len(sv.OpenedLockTileKeys) == 0 || len(sv.ActivatedShrines) == 0 {
		t.Errorf("expected non-empty persistent key slices")
	}
	if !sv.ReduceScreenShake || sv.TimeOfDay == 0 || sv.SelectedItem != world.ItemSlotTorch {
		t.Errorf("expected non-zero config/state fields")
	}
	if sv.SwingDuration == 0 || sv.BaseSpeed == 0 || sv.MaxBombs == 0 || sv.TorchBurnDuration == 0 {
		t.Errorf("expected non-zero tuning parameters")
	}
}
