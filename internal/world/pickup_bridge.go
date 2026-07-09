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

func (w *World) HasItem(id string) bool {
	if w == nil {
		return false
	}
	if w.OwnedItems != nil && w.OwnedItems[id] {
		return true
	}
	if id == "torch" {
		return w.HasTorch
	}
	if id == "pegasus_boots" {
		return w.HasPegasusBoots
	}
	if id == "shield" {
		return w.ShieldLevel > 0
	}
	return false
}

func (w *World) GrantItem(id string) {
	if w == nil {
		return
	}
	if w.OwnedItems == nil {
		w.OwnedItems = make(map[string]bool)
	}
	w.OwnedItems[id] = true
	if id == "torch" {
		w.HasTorch = true
	}
	if id == "pegasus_boots" {
		w.HasPegasusBoots = true
	}
	if id == "shield" {
		if w.ShieldLevel == 0 {
			w.ShieldLevel = 1
		} else if w.ShieldLevel < 2 {
			w.ShieldLevel = 2
		}
	}
}

func (w *World) PickupGrantItem(itemID string) { w.GrantItem(itemID) }
func (w *World) PickupGrantTorch()             { w.GrantItem("torch") }
func (w *World) PickupGrantPegasusBoots()      { w.GrantItem("pegasus_boots") }

// TryOpenChest checks if the player is overlapping an unopened chest (persistent pickup with save key)
// and marks it opened and pending collection for PickupSystem to process.
func (w *World) TryOpenChest() bool {
	pr := w.PlayerRect()
	for i := range w.Pickups {
		p := &w.Pickups[i]
		if p.IsChest && !p.Opened && !p.Gone && pr.OverlapsExpanded(p.Rect(), 2.0) {
			p.Opened = true
			p.PendingCollect = true
			return true
		}
	}
	return false
}
