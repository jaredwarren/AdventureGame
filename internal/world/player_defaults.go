package world

// defaultPlayerTuning is the single source of truth for upgradeable player
// properties. BuildFromTiled seeds new worlds from this; systems resolve
// zero-valued Player fields through Effective* accessors below.
var defaultPlayerTuning = Player{
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

// DefaultPlayerTuning returns a copy of the canonical player tuning defaults.
func DefaultPlayerTuning() Player {
	return defaultPlayerTuning
}

func intOrDefault(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}

func floatOrDefault(v, d float64) float64 {
	if v > 0 {
		return v
	}
	return d
}

func (p Player) EffectiveSwingDuration() int {
	return intOrDefault(p.SwingDuration, defaultPlayerTuning.SwingDuration)
}

func (p Player) EffectiveMaxSwingCD() int {
	return intOrDefault(p.MaxSwingCD, defaultPlayerTuning.MaxSwingCD)
}

func (p Player) EffectiveSwingActiveStart() int {
	return intOrDefault(p.SwingActiveStart, defaultPlayerTuning.SwingActiveStart)
}

func (p Player) EffectiveSwingActiveEnd() int {
	return intOrDefault(p.SwingActiveEnd, defaultPlayerTuning.SwingActiveEnd)
}

func (p Player) EffectiveTorchSwingDuration() int {
	return intOrDefault(p.TorchSwingDuration, defaultPlayerTuning.TorchSwingDuration)
}

func (p Player) EffectiveMaxTorchSwingCD() int {
	return intOrDefault(p.MaxTorchSwingCD, defaultPlayerTuning.MaxTorchSwingCD)
}

func (p Player) EffectiveTorchSwingActiveStart() int {
	return intOrDefault(p.TorchSwingActiveStart, defaultPlayerTuning.TorchSwingActiveStart)
}

func (p Player) EffectiveTorchSwingActiveEnd() int {
	return intOrDefault(p.TorchSwingActiveEnd, defaultPlayerTuning.TorchSwingActiveEnd)
}

func (p Player) EffectiveBaseSpeed() float64 {
	return floatOrDefault(p.BaseSpeed, defaultPlayerTuning.BaseSpeed)
}

func (p Player) EffectiveSprintSpeed() float64 {
	return floatOrDefault(p.SprintSpeed, defaultPlayerTuning.SprintSpeed)
}

func (p Player) EffectiveDodgeStaminaCost() int {
	return intOrDefault(p.DodgeStaminaCost, defaultPlayerTuning.DodgeStaminaCost)
}

func (p Player) EffectiveDodgeDuration() int {
	return intOrDefault(p.DodgeDuration, defaultPlayerTuning.DodgeDuration)
}

func (p Player) EffectiveDodgeMaxImpulse() int {
	return intOrDefault(p.DodgeMaxImpulse, defaultPlayerTuning.DodgeMaxImpulse)
}

func (p Player) EffectiveDodgeSpeed() float64 {
	return floatOrDefault(p.DodgeSpeed, defaultPlayerTuning.DodgeSpeed)
}

func (p Player) EffectiveStaminaRegenInterval() int {
	return intOrDefault(p.StaminaRegenInterval, defaultPlayerTuning.StaminaRegenInterval)
}

func (p Player) EffectiveSwordReach() float64 {
	return floatOrDefault(p.SwordReach, defaultPlayerTuning.SwordReach)
}

func (p Player) EffectiveSwordThickness() float64 {
	return floatOrDefault(p.SwordThickness, defaultPlayerTuning.SwordThickness)
}

func (p Player) EffectiveTorchReach() float64 {
	return floatOrDefault(p.TorchReach, defaultPlayerTuning.TorchReach)
}

func (p Player) EffectiveTorchThickness() float64 {
	return floatOrDefault(p.TorchThickness, defaultPlayerTuning.TorchThickness)
}

func (p Player) EffectiveInvulnFrames() int {
	return intOrDefault(p.InvulnFrames, defaultPlayerTuning.InvulnFrames)
}

func (p Player) EffectiveEnemyKnockbackForce() float64 {
	return floatOrDefault(p.EnemyKnockbackForce, defaultPlayerTuning.EnemyKnockbackForce)
}

func (p Player) EffectivePlayerKnockbackForce() float64 {
	return floatOrDefault(p.PlayerKnockbackForce, defaultPlayerTuning.PlayerKnockbackForce)
}

func (p Player) EffectivePlayerHazardKnockbackForce() float64 {
	return floatOrDefault(p.PlayerHazardKnockbackForce, defaultPlayerTuning.PlayerHazardKnockbackForce)
}

func (p Player) EffectiveMaxBombs() int {
	return intOrDefault(p.MaxBombs, defaultPlayerTuning.MaxBombs)
}

func (p Player) EffectiveBombFuseDuration() int {
	return intOrDefault(p.BombFuseDuration, defaultPlayerTuning.BombFuseDuration)
}

func (p Player) EffectiveBombRadius() float64 {
	return floatOrDefault(p.BombRadius, defaultPlayerTuning.BombRadius)
}

func (p Player) EffectiveBombDamage() int {
	return intOrDefault(p.BombDamage, defaultPlayerTuning.BombDamage)
}

func (p Player) EffectiveTorchBurnDuration() int {
	return intOrDefault(p.TorchBurnDuration, defaultPlayerTuning.TorchBurnDuration)
}

func (p Player) EffectiveTorchBurnInterval() int {
	return intOrDefault(p.TorchBurnInterval, defaultPlayerTuning.TorchBurnInterval)
}

func (p Player) EffectiveTorchBurnDamage() int {
	return intOrDefault(p.TorchBurnDamage, defaultPlayerTuning.TorchBurnDamage)
}

func (p Player) EffectiveTorchLightRadius() float64 {
	return floatOrDefault(p.TorchLightRadius, defaultPlayerTuning.TorchLightRadius)
}

func (p Player) EffectivePersonalLightRadius() float64 {
	return floatOrDefault(p.PersonalLightRadius, defaultPlayerTuning.PersonalLightRadius)
}
