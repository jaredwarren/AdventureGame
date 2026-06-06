package world

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/progression"
)

func TestApplyPickupReward_CoinAndHeart(t *testing.T) {
	t.Parallel()
	econ := progression.DefaultEconomy()
	w := &World{HP: 1, Currency: 0}
	w.Stats = progression.Stats{Vitality: 1}

	w.ApplyPickupReward(PickupCoin)
	if w.Currency != econ.CoinPickupValue {
		t.Errorf("Currency = %d, want %d", w.Currency, econ.CoinPickupValue)
	}

	w.ApplyPickupReward(PickupHeart)
	wantHP := 1 + econ.HeartPickupHeal
	if w.HP != wantHP {
		t.Errorf("HP = %d, want %d", w.HP, wantHP)
	}
}

func TestRegisterPickup_CustomKind(t *testing.T) {
	custom := &PickupKind{
		id:          200,
		tiledName:   "custom_test",
		toast:       "Found Custom!",
		editorLabel: "Custom",
		apply: func(w *World) {
			w.Currency += 42
		},
	}
	RegisterPickup(custom)
	t.Cleanup(func() { unregisterPickup(custom) })

	w := &World{}
	w.ApplyPickupReward(custom)
	if w.Currency != 42 {
		t.Errorf("Currency = %d, want 42", w.Currency)
	}
	if PickupKindFromTiled("custom_test") != custom {
		t.Error("expected TiledName lookup for custom pickup")
	}
}

func TestApplyPickupReward_NilKindNoOp(t *testing.T) {
	t.Parallel()
	w := &World{Currency: 5}
	w.ApplyPickupReward(nil)
	if w.Currency != 5 {
		t.Errorf("Currency = %d, want unchanged 5", w.Currency)
	}
}

func TestShrineHeal_UsesEconomyConfig(t *testing.T) {
	t.Parallel()
	econ := progression.DefaultEconomy()
	w := &World{HP: 1}
	w.Stats = progression.Stats{Vitality: 1}

	w.ShrineHeal()
	wantHP := 1 + econ.ShrineHealAmount
	if w.HP != wantHP {
		t.Errorf("HP = %d, want %d", w.HP, wantHP)
	}
}
