package world

// ApplyPickupReward applies the gameplay effect of collecting a ground pickup or chest item.
func (w *World) ApplyPickupReward(kind *PickupKind) {
	kind.ApplyReward(w)
}
