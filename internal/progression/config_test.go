package progression

import "testing"

func TestDefaultConfig_Formulas(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	s := DefaultStats()

	if got, want := cfg.MaxHP(s), 8; got != want {
		t.Errorf("MaxHP(DefaultStats) = %d, want %d", got, want)
	}
	if got, want := cfg.MaxStamina(s), 80; got != want {
		t.Errorf("MaxStamina(DefaultStats) = %d, want %d", got, want)
	}
	if got, want := cfg.SwordDamage(s), 2; got != want {
		t.Errorf("SwordDamage(DefaultStats) = %d, want %d", got, want)
	}
}

func TestStatsMethodsUseDefaultConfig(t *testing.T) {
	t.Parallel()
	s := DefaultStats()
	cfg := DefaultConfig()

	if s.MaxHP() != cfg.MaxHP(s) {
		t.Errorf("Stats.MaxHP() = %d, want %d", s.MaxHP(), cfg.MaxHP(s))
	}
	if s.MaxStamina() != cfg.MaxStamina(s) {
		t.Errorf("Stats.MaxStamina() = %d, want %d", s.MaxStamina(), cfg.MaxStamina(s))
	}
	if s.DamageBonus() != cfg.DamageBonus(s) {
		t.Errorf("Stats.DamageBonus() = %d, want %d", s.DamageBonus(), cfg.DamageBonus(s))
	}
	if s.SwordDamage() != cfg.SwordDamage(s) {
		t.Errorf("Stats.SwordDamage() = %d, want %d", s.SwordDamage(), cfg.SwordDamage(s))
	}
}
