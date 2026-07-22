package run

import (
	"github.com/jaredwarren/game-test/internal/balance"
	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/world"
)

// RunState is the portable per-run player payload: stats, inventory, tuning, and world clock.
type RunState struct {
	HP, Currency, Bombs, SmallKey, TimeOfDay int
	HasTorch, HasPegasusBoots                bool
	ShieldLevel                              int
	OwnedItems                               map[string]bool
	SprintHeld, SprintExhausted              bool
	Stats                                    progression.Stats
	SelectedItem                             world.ItemSlot
	PlayerTuning                             balance.PlayerTuning
}

// RunStateFromWorld snapshots portable run state from a live world.
func RunStateFromWorld(w *world.World) RunState {
	if w == nil {
		return RunState{Stats: progression.DefaultStats(), PlayerTuning: balance.DefaultPlayerTuning()}
	}
	owned := make(map[string]bool)
	if w.OwnedItems != nil {
		for id, flag := range w.OwnedItems {
			if flag {
				owned[id] = true
			}
		}
	}
	if w.HasTorch {
		owned["torch"] = true
	}
	if w.HasPegasusBoots {
		owned["pegasus_boots"] = true
	}
	if w.ShieldLevel > 0 {
		owned["shield"] = true
	}
	return RunState{
		HP: w.HP, Currency: w.Currency, Bombs: w.Bombs, HasTorch: w.HasTorch || owned["torch"], HasPegasusBoots: w.HasPegasusBoots || owned["pegasus_boots"], ShieldLevel: w.ShieldLevel,
		OwnedItems: owned, SmallKey: w.SmallKey, Stats: w.Stats, TimeOfDay: w.TimeOfDay, SelectedItem: w.SelectedItem,
		SprintHeld: w.Player.SprintHeld, SprintExhausted: w.Player.SprintExhausted,
		PlayerTuning: w.Player.PlayerTuning,
	}
}

// RunStateFromSave rebuilds run state from a save payload.
func RunStateFromSave(s *save.GameSave) RunState {
	if s == nil {
		return RunState{Stats: progression.DefaultStats(), PlayerTuning: balance.DefaultPlayerTuning()}
	}
	owned := make(map[string]bool)
	for _, id := range s.OwnedItems {
		owned[id] = true
	}
	if s.HasTorch {
		owned["torch"] = true
	}
	if s.HasPegasusBoots {
		owned["pegasus_boots"] = true
	}
	if s.ShieldLevel > 0 {
		owned["shield"] = true
	}
	return RunState{
		HP: s.HP, Currency: s.Currency, Bombs: s.Bombs, HasTorch: s.HasTorch || owned["torch"], HasPegasusBoots: s.HasPegasusBoots || owned["pegasus_boots"], ShieldLevel: s.ShieldLevel,
		OwnedItems: owned, SmallKey: max(0, s.SmallKey), Stats: StatsFromSave(s), TimeOfDay: s.TimeOfDay,
		SelectedItem: world.ItemSlot(s.SelectedItem), PlayerTuning: s.Tuning,
	}
}

// ApplyTo writes run state into w, clamping HP and bomb count.
func (rs RunState) ApplyTo(w *world.World) {
	if w == nil {
		return
	}
	w.HP, w.Currency, w.Bombs, w.HasTorch, w.HasPegasusBoots, w.ShieldLevel, w.SmallKey = rs.HP, rs.Currency, w.ClampBombsCarry(rs.Bombs), rs.HasTorch, rs.HasPegasusBoots, rs.ShieldLevel, max(0, rs.SmallKey)
	w.Stats, w.TimeOfDay, w.SelectedItem = rs.Stats, rs.TimeOfDay, rs.SelectedItem
	w.Player.SprintHeld, w.Player.SprintExhausted, w.Player.PlayerTuning = rs.SprintHeld, rs.SprintExhausted, rs.PlayerTuning
	if w.OwnedItems == nil {
		w.OwnedItems = make(map[string]bool)
	}
	for id, flag := range rs.OwnedItems {
		if flag {
			w.GrantItem(id)
		}
	}
	if w.HasTorch {
		w.GrantItem("torch")
	}
	if w.HasPegasusBoots {
		w.GrantItem("pegasus_boots")
	}
	if w.ShieldLevel > 0 {
		w.GrantItem("shield")
	}
	if w.HP > w.MaxHP() {
		w.HP = w.MaxHP()
	}
}

// ToGameSave serializes run state into a save payload for mapID at (px, py).
func (rs RunState) ToGameSave(mapID string, px, py float64) *save.GameSave {
	var ownedList []string
	for id, flag := range rs.OwnedItems {
		if flag {
			ownedList = append(ownedList, id)
		}
	}
	return &save.GameSave{
		MapID: mapID, PlayerX: px, PlayerY: py, HP: rs.HP, Currency: rs.Currency, Bombs: rs.Bombs,
		HasTorch: rs.HasTorch || rs.OwnedItems["torch"], HasPegasusBoots: rs.HasPegasusBoots || rs.OwnedItems["pegasus_boots"],
		ShieldLevel: rs.ShieldLevel,
		OwnedItems:  ownedList, SmallKey: rs.SmallKey, Vitality: rs.Stats.Vitality, Resolve: rs.Stats.Resolve,
		Might: rs.Stats.Might, Wits: rs.Stats.Wits, Fortune: rs.Stats.Fortune, TimeOfDay: rs.TimeOfDay,
		SelectedItem: int(rs.SelectedItem), Tuning: rs.PlayerTuning,
	}
}
