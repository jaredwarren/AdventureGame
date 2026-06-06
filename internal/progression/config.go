package progression

// Config holds progression formulas and per-stat shop increments.
// Tuning these values rebalances max HP, stamina, and sword damage globally.
type Config struct {
	BaseMaxHP         int
	HPPerVitality     int
	BaseMaxStamina    int
	StaminaPerResolve int
	SwordBaseDamage   int
	DamagePerMight    int
}

var defaultConfig = Config{
	BaseMaxHP:         6,
	HPPerVitality:     2,
	BaseMaxStamina:    60,
	StaminaPerResolve: 20,
	SwordBaseDamage:   1,
	DamagePerMight:    1,
}

// DefaultConfig returns the canonical progression formula defaults.
func DefaultConfig() Config {
	return defaultConfig
}

// MaxHP returns max HP from Vitality (each heart = 2 HP units for HUD).
func (c Config) MaxHP(s Stats) int {
	return c.BaseMaxHP + s.Vitality*c.HPPerVitality
}

// MaxStamina returns sprint/dodge stamina capacity from Resolve.
func (c Config) MaxStamina(s Stats) int {
	return c.BaseMaxStamina + s.Resolve*c.StaminaPerResolve
}

// DamageBonus returns bonus damage from Might before sword base is added.
func (c Config) DamageBonus(s Stats) int {
	return s.Might * c.DamagePerMight
}

// SwordDamage returns total melee damage (base + Might bonus).
func (c Config) SwordDamage(s Stats) int {
	return c.SwordBaseDamage + c.DamageBonus(s)
}
