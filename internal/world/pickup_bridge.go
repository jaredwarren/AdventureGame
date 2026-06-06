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
