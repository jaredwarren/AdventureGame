package world

// HasCapability reports whether any item currently owned by the player or active buff grants cap.
func (w *World) HasCapability(cap CapabilityID) bool {
	if w == nil {
		return false
	}
	for id, owned := range w.OwnedItems {
		if !owned {
			continue
		}
		itemDef, ok := GetItem(id)
		if !ok {
			continue
		}
		for _, c := range itemDef.Capabilities {
			if c == cap {
				return true
			}
		}
	}
	for i := range w.ActiveBuffs {
		for _, c := range w.ActiveBuffs[i].Capabilities {
			if c == cap {
				return true
			}
		}
	}
	// Fallback compatibility checks for hardcoded flags
	if cap == CapLightSource && w.HasTorch {
		return true
	}
	if cap == CapSprint && w.HasPegasusBoots {
		return true
	}
	return false
}

// EffectiveStat calculates the final effective stat value after applying flat additions,
// percentage multipliers, and overrides from all owned items and active buffs.
func (w *World) EffectiveStat(stat StatID, baseValue float64) float64 {
	if w == nil {
		return baseValue
	}
	flatAdd := 0.0
	percentAdd := 0.0
	overrideVal := baseValue
	hasOverride := false

	applyMod := func(mod StatModifier) {
		if mod.Stat == stat {
			switch mod.Op {
			case OpFlatAdd:
				flatAdd += mod.Value
			case OpPercentMult:
				percentAdd += mod.Value
			case OpOverride:
				overrideVal = mod.Value
				hasOverride = true
			}
		}
	}

	for id, owned := range w.OwnedItems {
		if !owned {
			continue
		}
		itemDef, ok := GetItem(id)
		if !ok {
			continue
		}
		for _, mod := range itemDef.Modifiers {
			applyMod(mod)
		}
	}

	for i := range w.ActiveBuffs {
		for _, mod := range w.ActiveBuffs[i].Modifiers {
			applyMod(mod)
		}
	}

	if hasOverride {
		return overrideVal
	}
	return (baseValue + flatAdd) * (1.0 + percentAdd)
}
