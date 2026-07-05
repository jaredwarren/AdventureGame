# Gameplay Architecture Design Patterns

This document outlines the data-driven and interface-driven design patterns established in this codebase. These patterns simplify adding new gameplay content (items, buffs, tile surfaces, sound/FX triggers, and door/quest conditions) without touching core system files or writing repetitive switch statements.

---

## 1. Hybrid Data-Driven Items & Modifiers

### Concept
Items carry data declarations of stat adjustments (`StatModifier`) and capabilities (`CapabilityID`), alongside optional event functions (`OnAcquire`, `OnTick`) for unique behaviors.

### Core Structure
```go
type StatID string

const (
	StatBaseSpeed        StatID = "base_speed"
	StatSprintSpeed      StatID = "sprint_speed"
	StatMaxHP            StatID = "max_hp"
	StatMaxStamina       StatID = "max_stamina"
	StatTorchLightRadius StatID = "torch_light_radius"
)

type CapabilityID string

const (
	CapSprint      CapabilityID = "can_sprint"
	CapLightSource CapabilityID = "light_source"
)

type StatModifier struct {
	Stat  StatID
	Op    ModifierOp // OpFlatAdd, OpPercentMult, OpOverride
	Value float64
}

type ItemDef struct {
	ID           string
	Name         string
	Category     ItemCategory
	PickupKind   *PickupKind
	Capabilities []CapabilityID
	Modifiers    []StatModifier
	OnAcquire    func(w *World)
	OnTick       func(w *World)
}
```

### Usage Example
```go
// Registering a Pegasus Boots item with sprint capability and speed modifier
RegisterItem(ItemDef{
	ID:           "pegasus_boots",
	Name:         "Pegasus Boots",
	Category:     CategoryPassive,
	Capabilities: []CapabilityID{CapSprint},
	Modifiers: []StatModifier{
		{Stat: StatBaseSpeed, Op: OpPercentMult, Value: 0.15},
	},
})

// Querying in gameplay systems:
canSprint := w.HasCapability(CapSprint)
effectiveSpeed := w.EffectiveStat(StatBaseSpeed, baseSpeed)
```

---

## 2. Status Effects & Temporary Buff System

### Concept
Extends the stat modifier and capability evaluation engine to support temporary duration-based buffs, debuffs, potions, and damage-over-time (DoT) effects.

### Core Structure
```go
type BuffID string

type Buff struct {
	ID           BuffID
	Name         string
	Duration     int            // Remaining frame ticks (-1 for permanent)
	Modifiers    []StatModifier // Temporary stat adjustments
	Capabilities []CapabilityID // Temporary capabilities (e.g. "invulnerable", "invisible")
	OnTick       func(w *World, b *Buff)
	OnExpire     func(w *World, b *Buff)
}
```

### Integration Guidelines
- `World.ApplyBuff(buff Buff)` adds or refreshes a buff.
- `item_evaluator.go` iterates over both `w.OwnedItems` and `w.ActiveBuffs` to evaluate effective stats and capability tags seamlessly.
- `TimersSystem` ticks down `buff.Duration` and calls `OnTick` / `OnExpire`.

---

## 3. Advanced Data-Driven Tile Pattern (TileTags & Embedded Surfaces)

### Concept
Tile definitions directly embed physical surface attributes (`SurfaceDef`) and categorical capability tags (`TileTag`), making tile registration completely self-contained in `def.go`.

### Core Structure
```go
type TileTag string

const (
	TagSolid        TileTag = "solid"
	TagWall         TileTag = "wall"
	TagWater        TileTag = "water"
	TagWaterShore   TileTag = "water_shore"
	TagDoor         TileTag = "door"
	TagLock         TileTag = "lock"
	TagIgnitable    TileTag = "ignitable"
	TagDestructible TileTag = "destructible"
)

type SurfaceDef struct {
	Type            SurfaceType // SurfaceNormal, SurfaceIce, SurfaceMud, SurfaceLava, SurfaceWater
	SpeedMultiplier float64     // e.g., 0.5 for Mud/Water slowdown, 1.0 for Normal
	Friction        float64     // e.g., 0.1 for Ice (slippery), 1.0 for Normal
	HazardDamage    int         // Damage per tick (e.g., 1 for Lava)
	HazardInterval  int         // Damage tick interval in frames
	Tags            []string
}

type Tile struct {
	GID          int
	Name         string
	Tags         []TileTag
	Surface      SurfaceDef
	SolidRects   []geom.Rect
	DamageKinds  []DamageKind
	DestroyedGID int
	SwatchColor  color.RGBA
	VectorDraw   func(c Canvas, x, y, w, h float32)
}
```

### Self-Contained Tile Example (`def.go`)
```go
// Adding a new environmental hazard tile (e.g. Lava or Mud):
GIDLava: {
	GID: GIDLava, Name: "lava",
	Tags: []TileTag{TagSolid},
	Surface: SurfaceDef{
		Type: SurfaceLava, SpeedMultiplier: 0.7, HazardDamage: 1, HazardInterval: 30,
	},
	VectorDraw: drawLava,
}

// System consumption:
surface := w.SurfaceAtFeet(px, py)
effectiveSpeed *= surface.SpeedMultiplier
```

---

## 4. Event-Driven Audio & Juice Triggers (FX Registry)

### Concept
Decouples sound playback, particle spawning, and screen-shake triggers from scene files by subscribing to simulation events on the `EventBus`.

### Core Structure
```go
type FXRule struct {
	Sound       string  // Sound file name (e.g. "chest_open.wav")
	Volume      float32 // Playback volume (0.0 - 1.0)
	ShakeFrames int     // Camera screen-shake duration
}

var FXRegistry = map[EventType]FXRule{
	EventChestOpen:   {Sound: "chest_open.wav", Volume: 0.8, ShakeFrames: 4},
	EventPlayerDodge: {Sound: "dodge.wav", Volume: 0.3},
	EventEnemyHurt:   {Sound: "hit.wav", Volume: 0.5},
}
```

### Integration Guidelines
- Scene or system event handler receives `Event` from `EventBus`.
- Looks up `FXRule` in `FXRegistry` via `GetFXRule(evt)` and triggers audio/visual feedback automatically.

---

## 5. Generic Unlock & Door Condition Evaluator

### Concept
Replaces hardcoded item/key checks on doors and gates with a data-driven condition evaluator that can check items, enemy clearance, switches, or puzzle states.

### Core Structure
```go
type ConditionType string

const (
	CondItem           ConditionType = "item"
	CondEnemiesCleared ConditionType = "enemies_cleared"
	CondSwitchActive   ConditionType = "switch_active"
)

type UnlockCondition struct {
	Type   ConditionType
	Target string // Item ID (e.g. "small_key") or Room/Switch ID
}
```

### Integration Guidelines
- Tiled map doors specify custom properties: `unlock_type: "item"`, `unlock_target: "small_key"`.
- `(w *World) CanUnlock(cond UnlockCondition) bool` evaluates the condition generically.
