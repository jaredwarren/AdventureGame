package world

type ConditionType string

const (
	CondItem           ConditionType = "item"
	CondEnemiesCleared ConditionType = "enemies_cleared"
	CondSwitchActive   ConditionType = "switch_active"
)

// UnlockCondition defines a requirement to open a door, gate, or chest.
type UnlockCondition struct {
	Type   ConditionType
	Target string // Item ID (e.g., "small_key") or Room/Switch identifier
}

// CanUnlock evaluates whether the given condition is met in the current world state.
func (w *World) CanUnlock(cond UnlockCondition) bool {
	if w == nil || cond.Type == "" {
		return true // Default open / no condition
	}
	switch cond.Type {
	case CondItem:
		if cond.Target == "small_key" || cond.Target == "key" {
			return w.SmallKey > 0
		}
		return w.HasItem(cond.Target)
	case CondEnemiesCleared:
		for i := range w.Enemies {
			if w.Enemies[i].HP > 0 {
				return false
			}
		}
		return true
	default:
		return true
	}
}
