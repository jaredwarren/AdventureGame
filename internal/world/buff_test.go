package world

import (
	"testing"
)

func TestBuffSystem(t *testing.T) {
	w := &World{}
	speedBuff := Buff{
		ID:       "speed_potion",
		Name:     "Speed Potion",
		Duration: 2,
		Modifiers: []StatModifier{
			{Stat: StatBaseSpeed, Op: OpPercentMult, Value: 0.50},
		},
	}

	w.ApplyBuff(speedBuff)
	if !w.HasBuff("speed_potion") {
		t.Error("expected world to have speed_potion buff")
	}

	effSpd := w.EffectiveStat(StatBaseSpeed, 2.0)
	if diff := effSpd - 3.0; diff < -0.0001 || diff > 0.0001 {
		t.Errorf("expected effective speed 3.0 with buff, got %f", effSpd)
	}

	// Remove buff manually
	w.RemoveBuff("speed_potion")
	if w.HasBuff("speed_potion") {
		t.Error("expected speed_potion buff to be removed")
	}

	effSpd = w.EffectiveStat(StatBaseSpeed, 2.0)
	if effSpd != 2.0 {
		t.Errorf("expected base speed 2.0 after buff removal, got %f", effSpd)
	}
}
