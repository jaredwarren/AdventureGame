package scenes

import (
	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/save"
	"github.com/jaredwarren/game-test/internal/world"
)

// RunState is the portable per-run player payload: stats, inventory, tuning, and
// world clock. Map transitions and saves marshal through this type so field
// lists stay in one place.
type RunState struct {
	HP       int
	Currency int
	Bombs    int
	HasTorch bool
	SmallKey int
	Stats    progression.Stats
	TimeOfDay         int
	SelectedItem      world.ItemSlot
	SprintHeld        bool
	SprintExhausted   bool
	PlayerTuning playerTuning
}

type playerTuning struct {
	SwingDuration         int
	MaxSwingCD            int
	SwingActiveStart      int
	SwingActiveEnd        int
	TorchSwingDuration    int
	MaxTorchSwingCD       int
	TorchSwingActiveStart int
	TorchSwingActiveEnd   int
	BaseSpeed             float64
	SprintSpeed           float64
	DodgeStaminaCost      int
	DodgeDuration         int
	DodgeMaxImpulse       int
	DodgeSpeed            float64
	StaminaRegenInterval  int
	SwordReach            float64
	SwordThickness        float64
	TorchReach            float64
	TorchThickness        float64
	InvulnFrames          int
	EnemyKnockbackForce   float64
	PlayerKnockbackForce  float64
	PlayerHazardKnockbackForce float64
	MaxBombs              int
	BombFuseDuration      int
	BombRadius            float64
	BombDamage            int
	TorchBurnDuration     int
	TorchBurnInterval     int
	TorchBurnDamage       int
}

// RunStateFromWorld snapshots portable run state from a live world.
func RunStateFromWorld(w *world.World) RunState {
	if w == nil {
		return RunState{Stats: progression.DefaultStats()}
	}
	return RunState{
		HP:              w.HP,
		Currency:        w.Currency,
		Bombs:           w.Bombs,
		HasTorch:        w.HasTorch,
		SmallKey:        w.SmallKey,
		Stats:           w.Stats,
		TimeOfDay:       w.TimeOfDay,
		SelectedItem:    w.SelectedItem,
		SprintHeld:      w.Player.SprintHeld,
		SprintExhausted: w.Player.SprintExhausted,
		PlayerTuning:    playerTuningFromPlayer(w.Player),
	}
}

// RunStateFromSave rebuilds run state from a save payload, applying the same
// optional-field defaults as a live load.
func RunStateFromSave(s *save.GameSave) RunState {
	if s == nil {
		return RunState{Stats: progression.DefaultStats()}
	}
	def := world.DefaultPlayerTuning()
	return RunState{
		HP:            s.HP,
		Currency:      s.Currency,
		Bombs:         s.Bombs,
		HasTorch:      s.HasTorch,
		SmallKey:      max(0, s.SmallKey),
		Stats:         StatsFromSave(s),
		TimeOfDay:     s.TimeOfDay,
		SelectedItem:  s.SelectedItem,
		PlayerTuning:  playerTuningFromSave(s, def),
	}
}

// ApplyTo writes run state into w, clamping HP and bomb count.
func (rs RunState) ApplyTo(w *world.World) {
	if w == nil {
		return
	}
	w.HP = rs.HP
	w.Currency = rs.Currency
	w.Bombs = w.ClampBombsCarry(rs.Bombs)
	w.HasTorch = rs.HasTorch
	w.SmallKey = rs.SmallKey
	if w.SmallKey < 0 {
		w.SmallKey = 0
	}
	w.Stats = rs.Stats
	w.TimeOfDay = rs.TimeOfDay
	w.SelectedItem = rs.SelectedItem
	w.Player.SprintHeld = rs.SprintHeld
	w.Player.SprintExhausted = rs.SprintExhausted
	rs.PlayerTuning.applyTo(&w.Player)
	if w.HP > w.MaxHP() {
		w.HP = w.MaxHP()
	}
}

// ToGameSave serializes run state into a save payload for mapID at (px, py).
func (rs RunState) ToGameSave(mapID string, px, py float64) *save.GameSave {
	gs := &save.GameSave{
		MapID:        mapID,
		PlayerX:      px,
		PlayerY:      py,
		HP:           rs.HP,
		Currency:     rs.Currency,
		Bombs:        rs.Bombs,
		HasTorch:     rs.HasTorch,
		SmallKey:     rs.SmallKey,
		Vitality:     rs.Stats.Vitality,
		Resolve:      rs.Stats.Resolve,
		Might:        rs.Stats.Might,
		Wits:         rs.Stats.Wits,
		Fortune:      rs.Stats.Fortune,
		TimeOfDay:    rs.TimeOfDay,
		SelectedItem: rs.SelectedItem,
	}
	rs.PlayerTuning.fillSave(gs)
	return gs
}

func playerTuningFromPlayer(p world.Player) playerTuning {
	return playerTuning{
		SwingDuration:         p.SwingDuration,
		MaxSwingCD:            p.MaxSwingCD,
		SwingActiveStart:      p.SwingActiveStart,
		SwingActiveEnd:        p.SwingActiveEnd,
		TorchSwingDuration:    p.TorchSwingDuration,
		MaxTorchSwingCD:       p.MaxTorchSwingCD,
		TorchSwingActiveStart: p.TorchSwingActiveStart,
		TorchSwingActiveEnd:   p.TorchSwingActiveEnd,
		BaseSpeed:             p.BaseSpeed,
		SprintSpeed:           p.SprintSpeed,
		DodgeStaminaCost:      p.DodgeStaminaCost,
		DodgeDuration:         p.DodgeDuration,
		DodgeMaxImpulse:       p.DodgeMaxImpulse,
		DodgeSpeed:            p.DodgeSpeed,
		StaminaRegenInterval:  p.StaminaRegenInterval,
		SwordReach:            p.SwordReach,
		SwordThickness:        p.SwordThickness,
		TorchReach:            p.TorchReach,
		TorchThickness:        p.TorchThickness,
		InvulnFrames:          p.InvulnFrames,
		EnemyKnockbackForce:   p.EnemyKnockbackForce,
		PlayerKnockbackForce:  p.PlayerKnockbackForce,
		PlayerHazardKnockbackForce: p.PlayerHazardKnockbackForce,
		MaxBombs:              p.MaxBombs,
		BombFuseDuration:      p.BombFuseDuration,
		BombRadius:            p.BombRadius,
		BombDamage:            p.BombDamage,
		TorchBurnDuration:     p.TorchBurnDuration,
		TorchBurnInterval:     p.TorchBurnInterval,
		TorchBurnDamage:       p.TorchBurnDamage,
	}
}

func playerTuningFromSave(s *save.GameSave, def world.Player) playerTuning {
	return playerTuning{
		SwingDuration:         tuningInt(s.SwingDuration, def.SwingDuration),
		MaxSwingCD:            tuningInt(s.MaxSwingCD, def.MaxSwingCD),
		SwingActiveStart:      tuningInt(s.SwingActiveStart, def.SwingActiveStart),
		SwingActiveEnd:        tuningInt(s.SwingActiveEnd, def.SwingActiveEnd),
		TorchSwingDuration:    tuningInt(s.TorchSwingDuration, def.TorchSwingDuration),
		MaxTorchSwingCD:       tuningInt(s.MaxTorchSwingCD, def.MaxTorchSwingCD),
		TorchSwingActiveStart: tuningInt(s.TorchSwingActiveStart, def.TorchSwingActiveStart),
		TorchSwingActiveEnd:   tuningInt(s.TorchSwingActiveEnd, def.TorchSwingActiveEnd),
		BaseSpeed:             tuningFloat(s.BaseSpeed, def.BaseSpeed),
		SprintSpeed:           tuningFloat(s.SprintSpeed, def.SprintSpeed),
		DodgeStaminaCost:      tuningInt(s.DodgeStaminaCost, def.DodgeStaminaCost),
		DodgeDuration:         tuningInt(s.DodgeDuration, def.DodgeDuration),
		DodgeMaxImpulse:       tuningInt(s.DodgeMaxImpulse, def.DodgeMaxImpulse),
		DodgeSpeed:            tuningFloat(s.DodgeSpeed, def.DodgeSpeed),
		StaminaRegenInterval:  tuningInt(s.StaminaRegenInterval, def.StaminaRegenInterval),
		SwordReach:            tuningFloat(s.SwordReach, def.SwordReach),
		SwordThickness:        tuningFloat(s.SwordThickness, def.SwordThickness),
		TorchReach:            tuningFloat(s.TorchReach, def.TorchReach),
		TorchThickness:        tuningFloat(s.TorchThickness, def.TorchThickness),
		InvulnFrames:          tuningInt(s.InvulnFrames, def.InvulnFrames),
		EnemyKnockbackForce:   tuningFloat(s.EnemyKnockbackForce, def.EnemyKnockbackForce),
		PlayerKnockbackForce:  tuningFloat(s.PlayerKnockbackForce, def.PlayerKnockbackForce),
		PlayerHazardKnockbackForce: tuningFloat(s.PlayerHazardKnockbackForce, def.PlayerHazardKnockbackForce),
		MaxBombs:              tuningInt(s.MaxBombs, def.MaxBombs),
		BombFuseDuration:      tuningInt(s.BombFuseDuration, def.BombFuseDuration),
		BombRadius:            tuningFloat(s.BombRadius, def.BombRadius),
		BombDamage:            tuningInt(s.BombDamage, def.BombDamage),
		TorchBurnDuration:     tuningInt(s.TorchBurnDuration, def.TorchBurnDuration),
		TorchBurnInterval:     tuningInt(s.TorchBurnInterval, def.TorchBurnInterval),
		TorchBurnDamage:       tuningInt(s.TorchBurnDamage, def.TorchBurnDamage),
	}
}

func (t playerTuning) applyTo(p *world.Player) {
	if p == nil {
		return
	}
	p.SwingDuration = t.SwingDuration
	p.MaxSwingCD = t.MaxSwingCD
	p.SwingActiveStart = t.SwingActiveStart
	p.SwingActiveEnd = t.SwingActiveEnd
	p.TorchSwingDuration = t.TorchSwingDuration
	p.MaxTorchSwingCD = t.MaxTorchSwingCD
	p.TorchSwingActiveStart = t.TorchSwingActiveStart
	p.TorchSwingActiveEnd = t.TorchSwingActiveEnd
	p.BaseSpeed = t.BaseSpeed
	p.SprintSpeed = t.SprintSpeed
	p.DodgeStaminaCost = t.DodgeStaminaCost
	p.DodgeDuration = t.DodgeDuration
	p.DodgeMaxImpulse = t.DodgeMaxImpulse
	p.DodgeSpeed = t.DodgeSpeed
	p.StaminaRegenInterval = t.StaminaRegenInterval
	p.SwordReach = t.SwordReach
	p.SwordThickness = t.SwordThickness
	p.TorchReach = t.TorchReach
	p.TorchThickness = t.TorchThickness
	p.InvulnFrames = t.InvulnFrames
	p.EnemyKnockbackForce = t.EnemyKnockbackForce
	p.PlayerKnockbackForce = t.PlayerKnockbackForce
	p.PlayerHazardKnockbackForce = t.PlayerHazardKnockbackForce
	p.MaxBombs = t.MaxBombs
	p.BombFuseDuration = t.BombFuseDuration
	p.BombRadius = t.BombRadius
	p.BombDamage = t.BombDamage
	p.TorchBurnDuration = t.TorchBurnDuration
	p.TorchBurnInterval = t.TorchBurnInterval
	p.TorchBurnDamage = t.TorchBurnDamage
}

func (t playerTuning) fillSave(gs *save.GameSave) {
	gs.SwingDuration = t.SwingDuration
	gs.MaxSwingCD = t.MaxSwingCD
	gs.SwingActiveStart = t.SwingActiveStart
	gs.SwingActiveEnd = t.SwingActiveEnd
	gs.TorchSwingDuration = t.TorchSwingDuration
	gs.MaxTorchSwingCD = t.MaxTorchSwingCD
	gs.TorchSwingActiveStart = t.TorchSwingActiveStart
	gs.TorchSwingActiveEnd = t.TorchSwingActiveEnd
	gs.BaseSpeed = t.BaseSpeed
	gs.SprintSpeed = t.SprintSpeed
	gs.DodgeStaminaCost = t.DodgeStaminaCost
	gs.DodgeDuration = t.DodgeDuration
	gs.DodgeMaxImpulse = t.DodgeMaxImpulse
	gs.DodgeSpeed = t.DodgeSpeed
	gs.StaminaRegenInterval = t.StaminaRegenInterval
	gs.SwordReach = t.SwordReach
	gs.SwordThickness = t.SwordThickness
	gs.TorchReach = t.TorchReach
	gs.TorchThickness = t.TorchThickness
	gs.InvulnFrames = t.InvulnFrames
	gs.EnemyKnockbackForce = t.EnemyKnockbackForce
	gs.PlayerKnockbackForce = t.PlayerKnockbackForce
	gs.PlayerHazardKnockbackForce = t.PlayerHazardKnockbackForce
	gs.MaxBombs = t.MaxBombs
	gs.BombFuseDuration = t.BombFuseDuration
	gs.BombRadius = t.BombRadius
	gs.BombDamage = t.BombDamage
	gs.TorchBurnDuration = t.TorchBurnDuration
	gs.TorchBurnInterval = t.TorchBurnInterval
	gs.TorchBurnDamage = t.TorchBurnDamage
}

func tuningInt(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

func tuningFloat(v, def float64) float64 {
	if v > 0 {
		return v
	}
	return def
}
