package world

func (w *World) PickupAddCurrency(amount int) { w.Currency += amount }

func (w *World) PickupHeal(heal int) {
	if heal <= 0 || w.HP >= w.MaxHP() {
		return
	}
	w.HP += heal
	if w.HP > w.MaxHP() {
		w.HP = w.MaxHP()
	}
}

func (w *World) PickupPlayerHP() int { return w.HP }
func (w *World) PickupMaxHP() int    { return w.MaxHP() }

func (w *World) PickupAddBomb() { w.Bombs = w.ClampBombsCarry(w.Bombs + 1) }

func (w *World) PickupAddSmallKey() { w.SmallKey++ }

func (w *World) PickupGrantTorch() { w.HasTorch = true }

// TryOpenChest checks if the player is overlapping an unopened chest (persistent pickup with save key)
// and marks it opened and pending collection for PickupSystem to process.
func (w *World) TryOpenChest() bool {
	pr := w.PlayerRect()
	for i := range w.Pickups {
		p := &w.Pickups[i]
		if p.PersistentSaveKey != "" && !p.Opened && !p.Gone && pr.OverlapsExpanded(p.Rect(), 2.0) {
			p.Opened = true
			p.PendingCollect = true
			return true
		}
	}
	return false
}
