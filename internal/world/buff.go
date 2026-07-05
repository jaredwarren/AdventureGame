package world

type BuffID string

// Buff represents a duration-based temporary status effect, stat boost, or capability grant.
type Buff struct {
	ID           BuffID
	Name         string
	Duration     int            // Remaining frame ticks (-1 for permanent/indefinite duration)
	Modifiers    []StatModifier // Temporary stat adjustments applied while active
	Capabilities []CapabilityID // Temporary capabilities granted while active
	OnTick       func(w *World, b *Buff)
	OnExpire     func(w *World, b *Buff)
}

// ApplyBuff adds a new buff or refreshes the duration if a buff with the same ID exists.
func (w *World) ApplyBuff(b Buff) {
	if w == nil || b.ID == "" {
		return
	}
	for i := range w.ActiveBuffs {
		if w.ActiveBuffs[i].ID == b.ID {
			w.ActiveBuffs[i].Duration = b.Duration
			w.ActiveBuffs[i].Modifiers = b.Modifiers
			w.ActiveBuffs[i].Capabilities = b.Capabilities
			return
		}
	}
	w.ActiveBuffs = append(w.ActiveBuffs, b)
}

// RemoveBuff removes a buff by ID and triggers its OnExpire handler if present.
func (w *World) RemoveBuff(id BuffID) {
	if w == nil {
		return
	}
	filtered := w.ActiveBuffs[:0]
	for i := range w.ActiveBuffs {
		if w.ActiveBuffs[i].ID == id {
			if w.ActiveBuffs[i].OnExpire != nil {
				w.ActiveBuffs[i].OnExpire(w, &w.ActiveBuffs[i])
			}
		} else {
			filtered = append(filtered, w.ActiveBuffs[i])
		}
	}
	w.ActiveBuffs = filtered
}

// HasBuff checks if a buff with the given ID is active.
func (w *World) HasBuff(id BuffID) bool {
	if w == nil {
		return false
	}
	for i := range w.ActiveBuffs {
		if w.ActiveBuffs[i].ID == id {
			return true
		}
	}
	return false
}
