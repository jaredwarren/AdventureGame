package world

import (
	"github.com/jaredwarren/game-test/internal/geom"
)

// Shrine is an authored interactable location.
type Shrine struct {
	ID      EntityID
	TiledID int
	Rect    geom.Rect
	Active  bool
}

// ShrineHeal is the “poor” shrine interaction (no coins). Rich interaction is priced in the shop.
func (w *World) ShrineHeal() {
	heal := w.EffectiveBalance().Economy.ShrineHealAmount
	if heal <= 0 || w.HP >= w.MaxHP() {
		return
	}
	w.HP += heal
	if w.HP > w.MaxHP() {
		w.HP = w.MaxHP()
	}
}

// UpgradeRandomStat picks one of the five progression knobs uniformly via rng(n).
// TODO: weight by build / offer UI choice instead of pure random; hook Wits/Fortune into real systems.
func (w *World) UpgradeRandomStat(rng func(int) int) {
	choices := []func(){
		func() { w.Stats.Vitality++ },
		func() { w.Stats.Resolve++ },
		func() { w.Stats.Might++ },
		func() { w.Stats.Wits++ },
		func() { w.Stats.Fortune++ },
	}
	choices[rng(5)]()
	if w.HP > w.MaxHP() {
		w.HP = w.MaxHP()
	}
}
