// Package progression is the numeric RPG layer (stats → derived max HP/stamina/damage).
//
// Wits/Fortune are partially placeholders until hint/loot systems consume them.
package progression

// Stats are the five orthogonal knobs from the design plan.
type Stats struct {
	Vitality int // max HP contribution
	Resolve  int // stamina max
	Might    int // damage bonus
	Wits     int // hint / scan (reserved)
	Fortune  int // loot quality (affects pickup rolls)
}

func DefaultStats() Stats {
	return Stats{Vitality: 1, Resolve: 1, Might: 1, Wits: 0, Fortune: 0}
}

// MaxHP returns hearts (each heart = 2 HP units for HUD).
func (s Stats) MaxHP() int {
	return DefaultConfig().MaxHP(s)
}

// MaxStamina tick capacity for sprint/dodge.
func (s Stats) MaxStamina() int {
	return DefaultConfig().MaxStamina(s)
}

// DamageBonus added to sword base.
func (s Stats) DamageBonus() int {
	return DefaultConfig().DamageBonus(s)
}

// SwordDamage returns total melee damage (base + Might bonus).
func (s Stats) SwordDamage() int {
	return DefaultConfig().SwordDamage(s)
}
