package balance

// PlayerTuning holds all tunable gameplay and visual parameters for the player.
type PlayerTuning struct {
	SwingDuration              int     `json:"swing_duration,omitempty"`
	MaxSwingCD                 int     `json:"max_swing_cd,omitempty"`
	SwingActiveStart           int     `json:"swing_active_start,omitempty"`
	SwingActiveEnd             int     `json:"swing_active_end,omitempty"`
	TorchSwingDuration         int     `json:"torch_swing_duration,omitempty"`
	MaxTorchSwingCD            int     `json:"max_torch_swing_cd,omitempty"`
	TorchSwingActiveStart      int     `json:"torch_swing_active_start,omitempty"`
	TorchSwingActiveEnd        int     `json:"torch_swing_active_end,omitempty"`
	BaseSpeed                  float64 `json:"base_speed,omitempty"`
	SprintSpeed                float64 `json:"sprint_speed,omitempty"`
	DodgeStaminaCost           int     `json:"dodge_stamina_cost,omitempty"`
	DodgeDuration              int     `json:"dodge_duration,omitempty"`
	DodgeMaxImpulse            int     `json:"dodge_max_impulse,omitempty"`
	DodgeSpeed                 float64 `json:"dodge_speed,omitempty"`
	StaminaRegenInterval       int     `json:"stamina_regen_interval,omitempty"`
	SwordReach                 float64 `json:"sword_reach,omitempty"`
	SwordThickness             float64 `json:"sword_thickness,omitempty"`
	TorchReach                 float64 `json:"torch_reach,omitempty"`
	TorchThickness             float64 `json:"torch_thickness,omitempty"`
	InvulnFrames               int     `json:"invuln_frames,omitempty"`
	EnemyKnockbackForce        float64 `json:"enemy_knockback_force,omitempty"`
	PlayerKnockbackForce       float64 `json:"player_knockback_force,omitempty"`
	PlayerHazardKnockbackForce float64 `json:"player_hazard_knockback_force,omitempty"`
	MaxBombs                   int     `json:"max_bombs,omitempty"`
	BombFuseDuration           int     `json:"bomb_fuse_duration,omitempty"`
	BombRadius                 float64 `json:"bomb_radius,omitempty"`
	BombDamage                 int     `json:"bomb_damage,omitempty"`
	TorchBurnDuration          int     `json:"torch_burn_duration,omitempty"`
	TorchBurnInterval          int     `json:"torch_burn_interval,omitempty"`
	TorchBurnDamage            int     `json:"torch_burn_damage,omitempty"`
	TorchLightRadius           float64 `json:"torch_light_radius,omitempty"`
	PersonalLightRadius        float64 `json:"personal_light_radius,omitempty"`
}

// DefaultPlayerTuning returns a copy of the canonical player tuning defaults.
func DefaultPlayerTuning() PlayerTuning {
	return PlayerTuning{
		SwingDuration:              8,
		MaxSwingCD:                 12,
		SwingActiveStart:           2,
		SwingActiveEnd:             7,
		TorchSwingDuration:         8,
		MaxTorchSwingCD:            12,
		TorchSwingActiveStart:      2,
		TorchSwingActiveEnd:        7,
		BaseSpeed:                  1.35,
		SprintSpeed:                2.15,
		DodgeStaminaCost:           20,
		DodgeDuration:              20,
		DodgeMaxImpulse:            12,
		DodgeSpeed:                 2.8,
		StaminaRegenInterval:       2,
		SwordReach:                 14.0,
		SwordThickness:             10.0,
		TorchReach:                 14.0,
		TorchThickness:             10.0,
		InvulnFrames:               45,
		EnemyKnockbackForce:        20.0,
		PlayerKnockbackForce:       6.0,
		PlayerHazardKnockbackForce: 12.0,
		MaxBombs:                   8,
		BombFuseDuration:           90,
		BombRadius:                 32.0,
		BombDamage:                 4,
		TorchBurnDuration:          72,
		TorchBurnInterval:          12,
		TorchBurnDamage:            1,
		TorchLightRadius:           85.0,
		PersonalLightRadius:        35.0,
	}
}
