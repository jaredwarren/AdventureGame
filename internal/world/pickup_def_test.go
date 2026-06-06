package world

import "testing"

func TestPickupKindFromTiled(t *testing.T) {
	t.Parallel()
	if got := PickupKindFromTiled("heart"); got != PickupHeart {
		t.Errorf("heart: got %v", got)
	}
	if got := PickupKindFromTiled("unknown"); got != PickupCoin {
		t.Errorf("unknown should default to coin, got %v", got)
	}
}

func TestPickupToastMessage(t *testing.T) {
	t.Parallel()
	if msg := PickupBomb.ToastMessage(); msg != "Found Bomb!" {
		t.Errorf("got %q", msg)
	}
	if msg := (*PickupKind)(nil).ToastMessage(); msg != "Found Item!" {
		t.Errorf("nil kind: got %q", msg)
	}
}

func TestAllPickups_Count(t *testing.T) {
	t.Parallel()
	if len(AllPickups) != 5 {
		t.Fatalf("expected 5 pickups, got %d", len(AllPickups))
	}
}

func TestPickupKindByID(t *testing.T) {
	t.Parallel()
	if k, ok := PickupKindByID(2); !ok || k != PickupBomb {
		t.Errorf("id 2: got %v ok=%v", k, ok)
	}
	if _, ok := PickupKindByID(99); ok {
		t.Error("expected unknown id to miss")
	}
}
