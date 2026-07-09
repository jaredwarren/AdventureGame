package world

type ItemCategory int

const (
	CategoryPassive ItemCategory = iota
	CategorySelectable
)

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

type ModifierOp int

const (
	OpFlatAdd ModifierOp = iota
	OpPercentMult
	OpOverride
)

// StatModifier defines a data-driven stat adjustment.
type StatModifier struct {
	Stat  StatID
	Op    ModifierOp
	Value float64
}

// ItemDef defines static metadata, stat modifiers, capabilities, and event hooks for a game item.
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

var (
	ItemBomb = ItemDef{
		ID:         "bomb",
		Name:       "Bomb",
		Category:   CategorySelectable,
		PickupKind: PickupBomb,
	}
	ItemTorch = ItemDef{
		ID:           "torch",
		Name:         "Torch",
		Category:     CategorySelectable,
		PickupKind:   PickupTorch,
		Capabilities: []CapabilityID{CapLightSource},
	}
	ItemPegasusBoots = ItemDef{
		ID:           "pegasus_boots",
		Name:         "Pegasus Boots",
		Category:     CategoryPassive,
		PickupKind:   PickupPegasusBoots,
		Capabilities: []CapabilityID{CapSprint},
	}
	ItemShield = ItemDef{
		ID:         "shield",
		Name:       "Shield",
		Category:   CategoryPassive,
		PickupKind: PickupShield,
	}

	itemsByID = map[string]ItemDef{
		"bomb":          ItemBomb,
		"torch":         ItemTorch,
		"pegasus_boots": ItemPegasusBoots,
		"shield":        ItemShield,
	}

	allItems = []ItemDef{
		ItemBomb,
		ItemTorch,
		ItemPegasusBoots,
		ItemShield,
	}
)

// RegisterItem adds an item definition to the global item registry.
func RegisterItem(def ItemDef) {
	if def.ID == "" {
		return
	}
	if _, exists := itemsByID[def.ID]; !exists {
		allItems = append(allItems, def)
	}
	itemsByID[def.ID] = def
}

// GetItem retrieves an item definition by ID.
func GetItem(id string) (ItemDef, bool) {
	def, ok := itemsByID[id]
	return def, ok
}

// RegisteredItems returns a slice of all registered items.
func RegisteredItems() []ItemDef {
	result := make([]ItemDef, len(allItems))
	copy(result, allItems)
	return result
}
