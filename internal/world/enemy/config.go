// Package enemy holds per-enemy tuning from Tiled markers and spawn defaults.
package enemy

import "github.com/jaredwarren/game-test/internal/tiled"

// Config holds per-enemy tuning authored in Tiled or applied at spawn.
type Config struct {
	HP            int
	Speed         float64
	AggroRadius   float64
	ContactDamage int
	IsBoss        bool
}

// DefaultConfig returns canonical enemy tuning when map props are absent.
func DefaultConfig() Config {
	return Config{
		HP:            DefaultHP,
		Speed:         DefaultSeekSpeed,
		AggroRadius:   DefaultAggroRadiusPx,
		ContactDamage: DefaultContactDamage,
		IsBoss:        false,
	}
}

// ConfigFromTiled reads optional marker properties from a Tiled enemy object.
func ConfigFromTiled(o *tiled.Object) Config {
	cfg := DefaultConfig()
	if o == nil {
		return cfg
	}
	if v, ok := tiled.ObjPropInt(o, "hp"); ok && v > 0 {
		cfg.HP = v
	}
	if v, ok := tiled.ObjPropFloat(o, "speed"); ok && v > 0 {
		cfg.Speed = v
	}
	if v, ok := tiled.ObjPropFloat(o, "aggro"); ok && v > 0 {
		cfg.AggroRadius = v
	}
	if v, ok := tiled.ObjPropInt(o, "damage"); ok && v > 0 {
		cfg.ContactDamage = v
	}
	if v, ok := tiled.ObjPropBool(o, "is_boss"); ok {
		cfg.IsBoss = v
	}
	return cfg
}

// DefaultTiledProperties returns Tiled object properties for a new enemy marker.
func DefaultTiledProperties() []tiled.Property {
	return TiledProperties(DefaultConfig())
}

// TiledProperties serializes a Config for Tiled object properties.
func TiledProperties(cfg Config) []tiled.Property {
	return []tiled.Property{
		{Name: "hp", Type: "int", Value: cfg.HP},
		{Name: "speed", Type: "float", Value: cfg.Speed},
		{Name: "aggro", Type: "float", Value: cfg.AggroRadius},
		{Name: "damage", Type: "int", Value: cfg.ContactDamage},
		{Name: "is_boss", Type: "bool", Value: cfg.IsBoss},
	}
}
