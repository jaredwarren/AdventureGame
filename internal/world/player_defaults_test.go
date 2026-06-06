package world

import "testing"

func TestPlayerEffectiveDefaultsWhenZero(t *testing.T) {
	var p Player
	d := DefaultPlayerTuning()
	if p.EffectiveBaseSpeed() != d.BaseSpeed {
		t.Errorf("BaseSpeed: got %v want %v", p.EffectiveBaseSpeed(), d.BaseSpeed)
	}
	if p.EffectiveInvulnFrames() != d.InvulnFrames {
		t.Errorf("InvulnFrames: got %d want %d", p.EffectiveInvulnFrames(), d.InvulnFrames)
	}
	if p.EffectiveMaxBombs() != d.MaxBombs {
		t.Errorf("MaxBombs: got %d want %d", p.EffectiveMaxBombs(), d.MaxBombs)
	}
	if p.EffectiveTorchBurnInterval() != d.TorchBurnInterval {
		t.Errorf("TorchBurnInterval: got %d want %d", p.EffectiveTorchBurnInterval(), d.TorchBurnInterval)
	}
}

func TestPlayerEffectiveOverridesWhenSet(t *testing.T) {
	p := Player{BaseSpeed: 2.5, InvulnFrames: 30, MaxBombs: 12}
	if p.EffectiveBaseSpeed() != 2.5 {
		t.Errorf("BaseSpeed override: got %v", p.EffectiveBaseSpeed())
	}
	if p.EffectiveInvulnFrames() != 30 {
		t.Errorf("InvulnFrames override: got %d", p.EffectiveInvulnFrames())
	}
	if p.EffectiveMaxBombs() != 12 {
		t.Errorf("MaxBombs override: got %d", p.EffectiveMaxBombs())
	}
}
