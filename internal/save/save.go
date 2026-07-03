// Package save persists player-facing progress as JSON on disk (default save.json in cwd).
//
// Bump CurrentVersion and add migration branches in Load when fields move or semantics change.
// Older save files may contain extra fields (e.g. world_seed from the procgen
// prototype); unknown JSON fields are ignored by unmarshal and do not need migration.
package save

import (
	"encoding/json"
	"os"

	"github.com/jaredwarren/game-test/internal/world"
)

const CurrentVersion = 2

// GameSave is versioned for migrations.
type GameSave struct {
	Version int `json:"version"`

	MapID    string  `json:"map_id"`
	PlayerX  float64 `json:"player_x"`
	PlayerY  float64 `json:"player_y"`
	HP       int     `json:"hp"`
	Currency int     `json:"currency"`
	// Bombs is the carried bomb count. Older saves used has_bomb (bool);
	// Load maps has_bomb:true to Bombs=1 when bombs is absent/zero.
	Bombs    int  `json:"bombs,omitempty"`
	HasTorch bool `json:"has_torch"`
	SmallKey int  `json:"small_key"`
	Vitality int  `json:"vitality"`
	Resolve  int  `json:"resolve"`
	Might    int  `json:"might"`
	Wits     int  `json:"wits"`
	Fortune  int  `json:"fortune"`

	// CollectedPickupKeys lists persistent pickup save keys already taken;
	// see world.PersistentPickupSaveKey. Omitted when empty.
	CollectedPickupKeys []string `json:"collected_pickups,omitempty"`

	// DestroyedTileKeys lists tiles destroyed by any damage source
	// (bombs, fire, ...); see world.MapTilePersistKey. The JSON tag
	// stays "broken_cracked_tiles" for backward compatibility with
	// existing save files from before generalized tile destruction.
	DestroyedTileKeys []string `json:"broken_cracked_tiles,omitempty"`

	// OpenedLockTileKeys lists GIDLock tiles opened with a small key; same key format as cracked tiles.
	OpenedLockTileKeys []string `json:"opened_lock_tiles,omitempty"`

	// ActivatedShrines lists keys of shrines that the player has touched.
	ActivatedShrines []string `json:"activated_shrines,omitempty"`

	ReduceScreenShake bool           `json:"reduce_screen_shake"`
	TimeOfDay         int            `json:"time_of_day"`
	SelectedItem      world.ItemSlot `json:"selected_item"`

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
}

const defaultPath = "save.json"

func Load(path string) (*GameSave, error) {
	if path == "" {
		path = defaultPath
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	type saveFile struct {
		GameSave
		LegacyHasBomb bool `json:"has_bomb"`
	}
	var sf saveFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return nil, err
	}
	s := sf.GameSave
	if s.Bombs <= 0 && sf.LegacyHasBomb {
		s.Bombs = 1
	}
	if s.Version < 1 {
		s.Version = 1
	}
	if s.Vitality == 0 {
		s.Vitality = 1
	}
	if s.Resolve == 0 {
		s.Resolve = 1
	}
	if s.Might == 0 {
		s.Might = 1
	}
	if s.Bombs < 0 {
		s.Bombs = 0
	}
	return &s, nil
}

func Save(path string, s *GameSave) error {
	if path == "" {
		path = defaultPath
	}
	s.Version = CurrentVersion
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
