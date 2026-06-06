package enemy

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/tiled"
)

func TestConfigFromTiled_Defaults(t *testing.T) {
	t.Parallel()
	cfg := ConfigFromTiled(&tiled.Object{Type: "enemy"})
	d := DefaultConfig()
	if cfg != d {
		t.Fatalf("got %+v want %+v", cfg, d)
	}
}

func TestConfigFromTiled_CustomProps(t *testing.T) {
	t.Parallel()
	o := &tiled.Object{
		Type: "enemy",
		Properties: []tiled.Property{
			{Name: "hp", Type: "int", Value: float64(10)},
			{Name: "speed", Type: "float", Value: 0.8},
			{Name: "aggro", Type: "float", Value: 200.0},
			{Name: "damage", Type: "int", Value: float64(5)},
			{Name: "is_boss", Type: "bool", Value: true},
		},
	}
	cfg := ConfigFromTiled(o)
	if cfg.HP != 10 {
		t.Errorf("HP = %d", cfg.HP)
	}
	if cfg.Speed != 0.8 {
		t.Errorf("Speed = %v", cfg.Speed)
	}
	if cfg.AggroRadius != 200 {
		t.Errorf("AggroRadius = %v", cfg.AggroRadius)
	}
	if cfg.ContactDamage != 5 {
		t.Errorf("ContactDamage = %d", cfg.ContactDamage)
	}
	if !cfg.IsBoss {
		t.Error("IsBoss = false, want true")
	}
}
