package balance

import (
	"encoding/json"

	"github.com/jaredwarren/game-test/internal/progression"
)

// DayNight controls cycle timing and ambient light boundaries.
type DayNight struct {
	CycleLength     int     `json:"cycle_length"`
	NightStartTick  int     `json:"night_start_tick"`
	NightEndTick    int     `json:"night_end_tick"`
	DuskStartTick   int     `json:"dusk_start_tick"`
	DuskEndTick     int     `json:"dusk_end_tick"`
	MinAmbientLight float64 `json:"min_ambient_light"`
}

// NightBuffs controls enemy stat multipliers when w.IsNight() is true.
type NightBuffs struct {
	AggroMultiplier float64 `json:"aggro_multiplier"`
	SpeedMultiplier float64 `json:"speed_multiplier"`
}

// Contact controls enemy-player contact damage parameters.
type Contact struct {
	EnemyHurtCD   int     `json:"enemy_hurt_cd"`
	ContactMargin float64 `json:"contact_margin"`
}

// Hazards controls environmental hazard timing and damage.
type Hazards struct {
	TreeBurnDuration int `json:"tree_burn_duration"`
	FlameDamage      int `json:"flame_damage"`
	FlameInterval    int `json:"flame_interval"`
}

// Respawn controls death respawn location and health recovery.
type Respawn struct {
	MapID      string  `json:"map_id"`
	HPFraction float64 `json:"hp_fraction"`
	HPMinimum  int     `json:"hp_minimum"`
}

// DoorCooldowns controls door re-entry frame cooldowns.
type DoorCooldowns struct {
	MapLoadFrames int `json:"map_load_frames"`
	WarpFrames    int `json:"warp_frames"`
}

// GameBalance is the root gameplay balance configuration.
type GameBalance struct {
	DayNight      DayNight            `json:"day_night"`
	NightBuffs    NightBuffs          `json:"night_buffs"`
	Contact       Contact             `json:"contact"`
	Hazards       Hazards             `json:"hazards"`
	Respawn       Respawn             `json:"respawn"`
	DoorCooldowns DoorCooldowns       `json:"door_cooldowns"`
	Economy       progression.Economy `json:"economy"`
}

// Default returns a GameBalance struct populated with canonical defaults.
func Default() *GameBalance {
	return &GameBalance{
		DayNight: DayNight{
			CycleLength:     14400,
			NightStartTick:  10800,
			NightEndTick:    1200,
			DuskStartTick:   9600,
			DuskEndTick:     10800,
			MinAmbientLight: 0.2,
		},
		NightBuffs: NightBuffs{
			AggroMultiplier: 1.5,
			SpeedMultiplier: 1.5,
		},
		Contact: Contact{
			EnemyHurtCD:   60,
			ContactMargin: 1.0,
		},
		Hazards: Hazards{
			TreeBurnDuration: 90,
			FlameDamage:      1,
			FlameInterval:    1,
		},
		Respawn: Respawn{
			MapID:      "F-5",
			HPFraction: 0.5,
			HPMinimum:  2,
		},
		DoorCooldowns: DoorCooldowns{
			MapLoadFrames: 60,
			WarpFrames:    90,
		},
		Economy: progression.DefaultEconomy(),
	}
}

// Load unmarshals JSON over a copy of default balance settings.
func Load(data []byte) (*GameBalance, error) {
	gb := Default()
	if err := json.Unmarshal(data, gb); err != nil {
		return nil, err
	}
	return gb, nil
}
