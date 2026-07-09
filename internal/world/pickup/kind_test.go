package pickup

import "testing"

type rewardStub struct {
	currency int
}

func (s *rewardStub) PickupAddCurrency(n int) { s.currency += n }
func (s *rewardStub) PickupHeal(int)          {}
func (s *rewardStub) PickupPlayerHP() int     { return 1 }
func (s *rewardStub) PickupMaxHP() int        { return 10 }
func (s *rewardStub) PickupAddBomb()          {}
func (s *rewardStub) PickupAddSmallKey()      {}
func (s *rewardStub) PickupGrantTorch()       {}
func (s *rewardStub) PickupGrantPegasusBoots() {}
func (s *rewardStub) PickupGrantItem(string)   {}

func TestFromTiled(t *testing.T) {
	t.Parallel()
	if got := FromTiled("heart"); got != Heart {
		t.Errorf("heart: got %v", got)
	}
	if got := FromTiled("unknown"); got != Coin {
		t.Errorf("unknown should default to coin, got %v", got)
	}
}

func TestToastMessage(t *testing.T) {
	t.Parallel()
	if msg := Bomb.ToastMessage(); msg != "Found Bomb!" {
		t.Errorf("got %q", msg)
	}
	if msg := (*Kind)(nil).ToastMessage(); msg != "Found Item!" {
		t.Errorf("nil kind: got %q", msg)
	}
}

func TestAll_Count(t *testing.T) {
	t.Parallel()
	if len(All) != 7 {
		t.Fatalf("expected 7 pickups, got %d", len(All))
	}
}

func TestByID(t *testing.T) {
	t.Parallel()
	if k, ok := ByID(2); !ok || k != Bomb {
		t.Errorf("id 2: got %v ok=%v", k, ok)
	}
	if _, ok := ByID(99); ok {
		t.Error("expected unknown id to miss")
	}
}

func TestRegister_CustomKind(t *testing.T) {
	custom := &Kind{
		id: 200, tiledName: "custom_test", toast: "Found Custom!", editorLabel: "Custom",
		apply: func(rt RewardTarget) { rt.PickupAddCurrency(42) },
	}
	Register(custom)
	defer unregister(custom)

	s := &rewardStub{}
	custom.ApplyReward(s)
	if s.currency != 42 {
		t.Errorf("currency = %d, want 42", s.currency)
	}
	if FromTiled("custom_test") != custom {
		t.Error("expected TiledName lookup for custom pickup")
	}
}
