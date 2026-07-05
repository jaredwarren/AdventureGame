package world

import "testing"

func TestItemEvaluator(t *testing.T) {
	w := &World{}
	if w.HasCapability(CapSprint) {
		t.Error("expected HasCapability(CapSprint) to be false on empty world")
	}

	w.GrantItem("pegasus_boots")
	if !w.HasCapability(CapSprint) {
		t.Error("expected HasCapability(CapSprint) to be true after granting pegasus_boots")
	}

	w.GrantItem("torch")
	if !w.HasCapability(CapLightSource) {
		t.Error("expected HasCapability(CapLightSource) to be true after granting torch")
	}

	// Register a test item with stat modifiers
	testItem := ItemDef{
		ID:       "test_ring",
		Name:     "Ring of Speed",
		Category: CategoryPassive,
		Modifiers: []StatModifier{
			{Stat: StatBaseSpeed, Op: OpFlatAdd, Value: 1.0},
			{Stat: StatBaseSpeed, Op: OpPercentMult, Value: 0.20},
		},
	}
	RegisterItem(testItem)

	baseSpd := 2.0
	effSpd := w.EffectiveStat(StatBaseSpeed, baseSpd)
	if effSpd != baseSpd {
		t.Errorf("unowned test item should not modify stat, got %f, want %f", effSpd, baseSpd)
	}

	w.GrantItem("test_ring")
	// (2.0 + 1.0) * (1.0 + 0.20) = 3.0 * 1.2 = 3.6
	effSpd = w.EffectiveStat(StatBaseSpeed, baseSpd)
	if diff := effSpd - 3.6; diff < -0.0001 || diff > 0.0001 {
		t.Errorf("expected effective speed 3.6, got %f", effSpd)
	}
}
