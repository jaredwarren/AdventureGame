package save_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jaredwarren/game-test/internal/balance"
	"github.com/jaredwarren/game-test/internal/save"
)

func TestGoldenSaveCurrentVersion(t *testing.T) {
	path := filepath.Join("testdata", "v3.json")
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
	if !sv.ReduceScreenShake || sv.TimeOfDay == 0 || sv.SelectedItem != 1 {
		t.Errorf("expected non-zero config/state fields")
	}
	if sv.Tuning.SwingDuration == 0 || sv.Tuning.BaseSpeed == 0 || sv.Tuning.MaxBombs == 0 || sv.Tuning.TorchBurnDuration == 0 {
		t.Errorf("expected non-zero tuning parameters")
	}
}

func TestMigrationV2ToV3(t *testing.T) {
	path := filepath.Join("testdata", "v2.json")
	sv, err := save.Load(path)
	if err != nil {
		t.Fatalf("failed to load v2 save: %v", err)
	}

	if sv.Version != 3 {
		t.Errorf("expected version to migrate to 3, got %d", sv.Version)
	}
	if sv.Tuning.SwingDuration != 12 {
		t.Errorf("expected SwingDuration 12, got %d", sv.Tuning.SwingDuration)
	}
	if sv.Tuning.BaseSpeed != 1.35 {
		t.Errorf("expected BaseSpeed 1.35, got %f", sv.Tuning.BaseSpeed)
	}
	if sv.Tuning.MaxBombs != 10 {
		t.Errorf("expected MaxBombs 10, got %d", sv.Tuning.MaxBombs)
	}
	if sv.Tuning.TorchBurnDuration != 180 {
		t.Errorf("expected TorchBurnDuration 180, got %d", sv.Tuning.TorchBurnDuration)
	}
}

func TestMigrationV1ToV3(t *testing.T) {
	path := filepath.Join("testdata", "v1.json")
	sv, err := save.Load(path)
	if err != nil {
		t.Fatalf("failed to load v1 save: %v", err)
	}

	if sv.Version != 3 {
		t.Errorf("expected version to migrate to 3, got %d", sv.Version)
	}
	if sv.Bombs != 1 {
		t.Errorf("expected legacy has_bomb:true to map to Bombs=1, got %d", sv.Bombs)
	}
	if sv.Vitality < 1 || sv.Resolve < 1 || sv.Might < 1 {
		t.Errorf("expected min stats >= 1, got Vitality=%d Resolve=%d Might=%d", sv.Vitality, sv.Resolve, sv.Might)
	}
	if sv.Tuning.SwingDuration == 0 {
		t.Errorf("expected default tuning populated for v1 save")
	}
}

func TestRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_save.json")

	original := &save.GameSave{
		Version:           save.CurrentVersion,
		MapID:             "dungeon1",
		PlayerX:           64.0,
		PlayerY:           80.0,
		HP:                6,
		Currency:          100,
		Bombs:             5,
		HasTorch:          true,
		SmallKey:          2,
		Vitality:          3,
		Resolve:           4,
		Might:             2,
		Wits:              1,
		Fortune:           1,
		ReduceScreenShake: true,
		TimeOfDay:         4800,
		SelectedItem:      1,
		Tuning:            balance.DefaultPlayerTuning(),
	}

	if err := save.Save(path, original); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := save.Load(path)
	if err != nil {
		t.Fatalf("failed to load saved file: %v", err)
	}

	if !reflect.DeepEqual(original, loaded) {
		t.Errorf("round trip failed:\nwant: %+v\ngot:  %+v", original, loaded)
	}
}

func TestUnsupportedFutureVersion(t *testing.T) {
	futureJSON := []byte(`{"version": 99, "map_id": "field1"}`)
	_, err := save.LoadBytes(futureJSON)
	if err == nil {
		t.Error("expected error when loading future version 99, got nil")
	}
}
