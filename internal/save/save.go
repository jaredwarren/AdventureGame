// Package save persists player-facing progress as JSON on disk (default save.json in cwd).
//
// Bump CurrentVersion and add migration branches in Load when fields move or semantics change.
// Older save files may contain extra fields (e.g. world_seed from the procgen
// prototype); unknown JSON fields are ignored by unmarshal and do not need migration.
package save

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jaredwarren/game-test/internal/balance"
)

const CurrentVersion = 3

// GameSave is versioned for migrations.
type GameSave struct {
	Version int `json:"version"`

	MapID           string   `json:"map_id"`
	PlayerX         float64  `json:"player_x"`
	PlayerY         float64  `json:"player_y"`
	HP              int      `json:"hp"`
	Currency        int      `json:"currency"`
	Bombs           int      `json:"bombs,omitempty"`
	HasTorch        bool     `json:"has_torch"`
	HasPegasusBoots bool     `json:"has_pegasus_boots,omitempty"`
	ShieldLevel     int      `json:"shield_level,omitempty"`
	OwnedItems      []string `json:"owned_items,omitempty"`
	SmallKey        int      `json:"small_key"`
	Vitality        int      `json:"vitality"`
	Resolve         int      `json:"resolve"`
	Might           int      `json:"might"`
	Wits            int      `json:"wits"`
	Fortune         int      `json:"fortune"`

	// CollectedPickupKeys lists persistent pickup save keys already taken.
	CollectedPickupKeys []string `json:"collected_pickups,omitempty"`

	// DestroyedTileKeys lists tiles destroyed by any damage source.
	DestroyedTileKeys []string `json:"broken_cracked_tiles,omitempty"`

	// OpenedLockTileKeys lists GIDLock tiles opened with a small key.
	OpenedLockTileKeys []string `json:"opened_lock_tiles,omitempty"`

	// ActivatedShrines lists keys of shrines that the player has touched.
	ActivatedShrines []string `json:"activated_shrines,omitempty"`

	ReduceScreenShake bool                 `json:"reduce_screen_shake"`
	TimeOfDay         int                  `json:"time_of_day"`
	SelectedItem      int                  `json:"selected_item"`
	Tuning            balance.PlayerTuning `json:"tuning"`
}

const defaultPath = "save.json"

// migrateV1toV2 handles legacy v1 save files:
// - Maps legacy `has_bomb: true` to `bombs: 1` if bombs is 0 or absent.
// - Ensures core stats (vitality, resolve, might) default to at least 1.
func migrateV1toV2(raw map[string]any) map[string]any {
	if hb, ok := raw["has_bomb"].(bool); ok && hb {
		b, _ := raw["bombs"].(float64)
		if b <= 0 {
			raw["bombs"] = float64(1)
		}
	}
	for _, stat := range []string{"vitality", "resolve", "might"} {
		if val, ok := raw[stat].(float64); !ok || val < 1 {
			raw[stat] = float64(1)
		}
	}
	raw["version"] = float64(2)
	return raw
}

// migrateV2toV3 handles v2 -> v3 save migration:
// - Lifts flat player tuning JSON fields into the nested "tuning" object.
func migrateV2toV3(raw map[string]any) map[string]any {
	tuning, _ := raw["tuning"].(map[string]any)
	if tuning == nil {
		tuning = make(map[string]any)
	}

	def := balance.DefaultPlayerTuning()
	flatKeys := map[string]any{
		"swing_duration":                def.SwingDuration,
		"max_swing_cd":                  def.MaxSwingCD,
		"swing_active_start":            def.SwingActiveStart,
		"swing_active_end":              def.SwingActiveEnd,
		"torch_swing_duration":          def.TorchSwingDuration,
		"max_torch_swing_cd":            def.MaxTorchSwingCD,
		"torch_swing_active_start":      def.TorchSwingActiveStart,
		"torch_swing_active_end":        def.TorchSwingActiveEnd,
		"base_speed":                    def.BaseSpeed,
		"sprint_speed":                  def.SprintSpeed,
		"dodge_stamina_cost":            def.DodgeStaminaCost,
		"dodge_duration":                def.DodgeDuration,
		"dodge_max_impulse":             def.DodgeMaxImpulse,
		"dodge_speed":                   def.DodgeSpeed,
		"stamina_regen_interval":        def.StaminaRegenInterval,
		"sword_reach":                   def.SwordReach,
		"sword_thickness":               def.SwordThickness,
		"torch_reach":                   def.TorchReach,
		"torch_thickness":               def.TorchThickness,
		"invuln_frames":                 def.InvulnFrames,
		"enemy_knockback_force":         def.EnemyKnockbackForce,
		"player_knockback_force":        def.PlayerKnockbackForce,
		"player_hazard_knockback_force": def.PlayerHazardKnockbackForce,
		"max_bombs":                     def.MaxBombs,
		"bomb_fuse_duration":            def.BombFuseDuration,
		"bomb_radius":                   def.BombRadius,
		"bomb_damage":                   def.BombDamage,
		"torch_burn_duration":           def.TorchBurnDuration,
		"torch_burn_interval":           def.TorchBurnInterval,
		"torch_burn_damage":             def.TorchBurnDamage,
		"torch_light_radius":            def.TorchLightRadius,
		"personal_light_radius":         def.PersonalLightRadius,
	}

	for key, defaultVal := range flatKeys {
		if val, exists := raw[key]; exists {
			tuning[key] = val
			delete(raw, key)
		} else if _, existsInTuning := tuning[key]; !existsInTuning {
			tuning[key] = defaultVal
		}
	}
	raw["tuning"] = tuning
	raw["version"] = float64(3)
	return raw
}

// LoadBytes parses save JSON data, running migrations as needed up to CurrentVersion.
func LoadBytes(data []byte) (*GameSave, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	vFloat, _ := raw["version"].(float64)
	v := int(vFloat)
	if v == 0 {
		v = 1
	}

	if v > CurrentVersion {
		return nil, fmt.Errorf("unsupported save version: %d (max supported is %d)", v, CurrentVersion)
	}

	if v < 2 {
		raw = migrateV1toV2(raw)
		v = 2
	}
	if v < 3 {
		raw = migrateV2toV3(raw)
		v = 3
	}

	migratedBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	var s GameSave
	if err := json.Unmarshal(migratedBytes, &s); err != nil {
		return nil, err
	}
	s.Version = CurrentVersion
	return &s, nil
}

// Load reads and parses a save file from disk.
func Load(path string) (*GameSave, error) {
	if path == "" {
		path = defaultPath
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadBytes(b)
}

// Save writes a GameSave to disk at CurrentVersion.
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
