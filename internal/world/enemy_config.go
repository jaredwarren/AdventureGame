package world

import "github.com/jaredwarren/game-test/internal/tiled"

// EnemyConfig holds per-enemy tuning authored in Tiled or applied at spawn.
type EnemyConfig struct {
	HP            int
	Speed         float64
	AggroRadius   float64
	ContactDamage int
	IsBoss        bool
}

// DefaultEnemyConfig returns canonical enemy tuning when map props are absent.
func DefaultEnemyConfig() EnemyConfig {
	return EnemyConfig{
		HP:            defaultEnemyHP,
		Speed:         defaultEnemySeekSpeed,
		AggroRadius:   DefaultEnemyAggroRadiusPx,
		ContactDamage: defaultEnemyContactDmg,
		IsBoss:        false,
	}
}

// EnemyConfigFromTiled reads optional marker properties from a Tiled enemy object.
// Missing or invalid values fall back to DefaultEnemyConfig.
func EnemyConfigFromTiled(o *tiled.Object) EnemyConfig {
	cfg := DefaultEnemyConfig()
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

// DefaultEnemyTiledProperties returns Tiled object properties for a new enemy marker.
func DefaultEnemyTiledProperties() []tiled.Property {
	cfg := DefaultEnemyConfig()
	return EnemyTiledProperties(cfg)
}

// EnemyTiledProperties serializes an EnemyConfig for Tiled object properties.
func EnemyTiledProperties(cfg EnemyConfig) []tiled.Property {
	return []tiled.Property{
		{Name: "hp", Type: "int", Value: cfg.HP},
		{Name: "speed", Type: "float", Value: cfg.Speed},
		{Name: "aggro", Type: "float", Value: cfg.AggroRadius},
		{Name: "damage", Type: "int", Value: cfg.ContactDamage},
		{Name: "is_boss", Type: "bool", Value: cfg.IsBoss},
	}
}
