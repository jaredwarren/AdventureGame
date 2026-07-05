package world

import "testing"

func TestUnlockCondition(t *testing.T) {
	w := &World{}

	keyCond := UnlockCondition{Type: CondItem, Target: "small_key"}
	if w.CanUnlock(keyCond) {
		t.Error("expected keyCond to be false when w.SmallKey == 0")
	}

	w.SmallKey = 1
	if !w.CanUnlock(keyCond) {
		t.Error("expected keyCond to be true when w.SmallKey > 0")
	}

	bootsCond := UnlockCondition{Type: CondItem, Target: "pegasus_boots"}
	if w.CanUnlock(bootsCond) {
		t.Error("expected bootsCond to be false before granting item")
	}

	w.GrantItem("pegasus_boots")
	if !w.CanUnlock(bootsCond) {
		t.Error("expected bootsCond to be true after granting pegasus_boots")
	}
}
