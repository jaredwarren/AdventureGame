# Overworld Asset Manifest & Tile Requirements

Based on the [Overworld Concept Map](file:///Users/jaredwarren/.gemini/antigravity-ide/brain/55495a72-00f0-41d7-9b22-5b65012e3493/overworld_map_concept_1787155401358.jpg) and the [Overworld Reference Guide](file:///Users/jaredwarren/.gemini/antigravity-ide/brain/55495a72-00f0-41d7-9b22-5b65012e3493/overworld_reference_guide.md), this document tracks all ground tile families, hazard surfaces, barrier walls, flora props, and interactive puzzle elements required for the 10×10 overworld grid (`A-1` through `J-10`).

---

## 1. Deterministic Spatial Variation Status (`c.GridPos()` Hashing)

Summary of assets implementing coordinate-seeded organic variations vs. intentionally static/uniform assets:

| Asset Category / Name | GID Range | Feature Status | Variation Details / Plan |
| :--- | :--- | :--- | :--- |
| **Meadow Grass** (`grass`) | 1 | **Data-driven** (`assets/tiles/1_grass.tile.json`) | 5 weighted variants via `gridHash` (40% double tuft, 30% tall cluster, 18% sparse, 6% yellow flower, 6% white flower) |
| **Meadow Oak Tree** (`tree`) | 8 | **Data-driven** (`assets/tiles/8_tree.tile.json`) | 4 variants (40% lush round oak, 30% bushy multi-lobe, 15% slender woodland, 15% ruby berry blossom) |
| **River & Lake Water** (`water`) | 5, 9–20 | **Implemented** (Base, Go drawer) | 4 variants on base (40% dual waves, 30% 3-ripple current, 15% sun glint sparkle, 15% bobbing lily pad); animation still code-driven |
| **Dirt Path** (`dirt_path`) | 21, 53–64 | **Data-driven** (Base: `21_dirt_path.tile.json`) | 4 variants on base; transition GIDs 53–64 remain Go drawers |
| **Cobblestone Path** (`cobble_path`) | 65–77 | **Data-driven** (Base: `65_cobble_path.tile.json`) | 4 variants on base; transition GIDs 66–77 remain Go drawers |
| **Sand & Shoreline** (`sand_path`) | 78–90 | *Pending* | Planned: shifting wind ripples, seashells, coarse sand specks |
| **Snow & Alpine** (`snow`) | 91–103 | *Pending* | Planned: powder snow drifts, sparkle flakes, subtle frost tracks |
| **Mud & Swamp** (`mud_path`) | 104–116 | *Pending* | Planned: bubble spots, dark water reflections, sludge patches |
| **Dark Forest Earth** (`dark_grass`) | 117–129 | *Pending* | Planned: pine needles, deep moss blades, woodland mushrooms |
| **Deep Ocean Water** (`deep_water`) | 130–142 | *Pending* | Planned: ocean swells, foam whitecaps, deep water crests |
| **Lava & Magma Shore** (`lava_shore`) | 143–155 | *Pending* | Planned: magma crust cracks, alternate bubble pop clusters |
| **Murky Swamp Water** (`swamp_water`) | 156–168 | *Pending* | Planned: toxic green scum patches, marsh gas bubbles |
| **Ice & Frozen Lake** (`ice`) | 169–181 | *Pending* | Planned: crystalline fracture lines, specular frost glaze |
| **Quicksand Hazard** (`quicksand`) | 182–194 | *Pending* | Planned: swirling sand vortex rings, sinking pebbles |
| **Flora Props** (Pines, Palms, Bushes) | Standalone | *Pending* | Planned: foliage lobes, needle density, fallen fruit/dates |
| **Walls, Rocks & Cliffs** | 3, 22–46, 195+ | **Static by Design** | 100% uniform and aligned joints (no spatial random drift) |
| **Doors, Locks & Signs** | 6, 7, 51 | **Static by Design** | 100% uniform interactable structures |

---

## 2. Ground & Path Autotile Families (13-Tile Sets)

Walkable floor sets (`FamilyFloor`) that blend natural paths and ground transitions seamlessly into the base grass:

| Asset Family | Registration | Spatial Variation | GID Range | Purpose & Biome Placement |
| :--- | :--- | :--- | :--- | :--- |
| **Dirt Path** (`dirt_path`) | **Implemented** | **Data-driven** (Base) | 21, 53–64 | Central meadows, starter trails, village paths |
| **Cobblestone Path** (`cobble_path`) | **Implemented** | **Data-driven** (Base) | 65–77 | Town square, bridge approaches, ruin pathways |
| **Sand & Shoreline** (`sand_path`) | **Implemented** | *Pending* | 78–90 | Desert dunes (`D-1`–`J-2`), oasis shores, beach transitions |
| **Snow & Alpine** (`snow`) | **Implemented** | *Pending* | 91–103 | Northern alpine peaks (`A-1`–`A-8`), mountain snowdrifts |
| **Mud & Swamp** (`mud_path`) | **Implemented** | *Pending* | 104–116 | Eastern dark woods (`E-9`–`I-10`), riverbanks, delta bogs |
| **Dark Forest Earth** (`dark_grass`) | **Implemented** | *Pending* | 117–129 | Shaded forest floor under dense ancient canopies |
| **Ice & Frozen Lake** (`ice`) | **Implemented** | *Pending* | 169–181 | Northern frozen lakes & icy caverns (`A-3`–`A-6`, `B-4`) |
| **Quicksand Hazard** (`quicksand`) | **Implemented** | *Pending* | 182–194 | Arid desert pits & ancient ruins (`D-1`, `E-1`, `H-2`) |

---

## 3. Water & Hazard Transition Families

Impassable liquid surfaces (`FamilyWater` / `FamilyWall`) with directional edge collision:

| Asset Family | Registration | Spatial Variation | GID Range | Purpose & Biome Placement |
| :--- | :--- | :--- | :--- | :--- |
| **Blue River & Lake Water** (`water`) | **Implemented** | **Implemented** (Base) | 5, 9–20 | Central rivers, southern lake (`G-5`–`J-9`), oasis |
| **Deep Ocean Water** (`deep_water`) | **Implemented** | *Pending* | 130–142 | Dark blue deep water borders on the southern coast (`J-1`–`J-10`) |
| **Lava & Magma Shore** (`lava_shore`) | **Implemented** | *Pending* | 143–155 | Volcanic northern cavern & mountain interior (`A-8`) |
| **Murky Swamp Water** (`swamp_water`) | **Implemented** | *Pending* | 156–168 | Enclosed witch grove / enchanted forest pools (`C-9`) |

---

## 4. Mountain, Cliff & Wall Barrier Families

Solid impassable barriers (`FamilyWall` / `FamilyRock`) providing natural elevation and boundaries (Intentionally **Static & Uniform**):

| Asset Family | Registration | Spatial Variation | GID Range | Purpose & Biome Placement |
| :--- | :--- | :--- | :--- | :--- |
| **Dungeon Wall** (`wall`) | **Implemented** | **Static by Design** | 3, 22–33 | Dungeons, indoor rooms, fortress battlements |
| **Grey Mountain Rock** (`rock`) | **Implemented** | **Static by Design** | 34–46 | Northern mountain ridges (`A-1`–`C-8`), high peaks |
| **Sandstone Canyon Cliffs** (`canyon_wall`) | *Planned* | **Static by Design** | 195–207 | Arid western canyons (`C-2`–`I-2`), sandstone plateaus |
| **Terraced Earth Cliffs** (`earth_cliff`) | *Planned* | **Static by Design** | 208–220 | 1-tier grass elevation steps in central meadows |
| **Ancient Ruin Stone Walls** (`ruin_wall`) | *Planned* | **Static by Design** | 221–233 | Desert tombs, crumbling temples, submerged island keeps |
| **Low Field Stone Wall / Fence** (`stone_fence`) | *Planned* | **Static by Design** | 234–246 | Village boundaries, farm pastures, road barriers |

---

## 5. Flora & Foliage (Trees, Props & Obstacles)

Standalone and multi-tile environment props:

| Asset Name | Footprint | Registration | Spatial Variation | Region / Biome |
| :--- | :--- | :--- | :--- | :--- |
| **Meadow Oak Tree** (`tree`) | 16×16 (GID 8) | **Implemented** | **Implemented** (4 variants) | Central Meadows (`E-4`–`G-7`) |
| **Snowy Alpine Pine** | 16×16 / 16×32 | *Planned* | *Pending* | Northern Mountains (`A-1`–`B-8`) |
| **Dark Autumn Forest Tree** | 16×16 / 16×32 | *Planned* | *Pending* | Eastern Canopy (`B-8`–`H-10`) |
| **Desert Palm Tree** | 16×32 | *Planned* | *Pending* | Western Oasis (`F-2`, `G-2`) |
| **Desert Saguaro Cactus** | 16×16 | *Planned* | *Pending* | Sand Dunes (`D-1`–`J-1`) |
| **Cuttable Bush / Shrub** | 16×16 | *Planned* | *Pending* | Everywhere |
| **Fallen Tree Log / Stump** | 16×16 / 32×16 | *Planned* | *Pending* | Eastern Woods |
| **Flower Patches & River Reeds** | 16×16 | *Planned* | *Pending* | Riverbanks & Starter Village |

---

## 6. Architectural & Structural Assets

### A. Bridges & Crossings (Walkable over water)
- **Wooden Plank Bridge**: Horizontal (16×16) and Vertical (16×16) tiles allowing player traversal over rivers (`G-5`, `H-4`, `I-4`).
- **Stone Arch Bridge**: Heavy cobblestone river crossing (`F-7`–`G-7`).
- **Stepping Stones / Lily Pads**: Jumpable/walkable points over narrow streams.

### B. Buildings & Settlements
- **Village House Tiles**:
  - Thatched & Clay Roof tiles (Top, Bottom, Eaves, Corners)
  - Wood/Stucco Wall Facades + Windows
  - Cottage Doorways (Entrance warps into interiors)
- **Town Well / Fountain**: 32×32 center landmark for the village square.
- **Wooden Picket Fencing**: Posts and horizontal rail segments.

### C. Major Regional Dungeons & Landmarks
- **Mountain Cave Mouth (`A-6` / `B-7`)**: 32×32 stone cavern archway into the North Mountain Dungeon.
- **Desert Pyramid Entrance (`H-1` / `I-1`)**: Massive carved sandstone tomb entrance with steps and hieroglyphs.
- **Island Fortress / Tower (`H-6`, `I-7`)**: Castled battlements, portcullis gate, and stone turrets on the southern lake.
- **Ancient Monolith / Waypoint Shrine**: Central fast-travel / respawn shrine (`F-5`).

---

## 7. Interactive Objects & Puzzle Elements

| Object | Tiled Type / Layer | Behavior |
| :--- | :--- | :--- |
| **Bombable Cracked Cliff** | Tile / Object (GID 4) | Destroyable with bombs (`DamageBomb`) revealing secret caves |
| **Heavy Grey Boulder** | Tile / Object | Obstacle; requires strength glove or bomb |
| **Treasure Chests** | Object | Small (coins/ammo), Large (heart container/major items) |
| **Clay Pots / Jars** | Object / Tile | Breakable with sword or lift/throw for recovery items |
| **Directional Signposts** | Tile (GID 51) | Inspectable dialog sign (Static) |

---

## 8. Phased Implementation Roadmap

1. **Phase 1: Ground Variations & Foliage Polish**
   - Implement spatial micro-variations for Dirt, Sand, Snow, Mud, Dark Grass, and Paths.
   - Implement standalone Flora props (Alpine Pines, Autumn Canopies, Desert Palms, Cacti, Cuttable Bushes).
2. **Phase 2: Canyon & Cliff Barrier Families** (`C-2`–`I-2`, `A-1`–`C-8`)
   - Sandstone Canyon Cliff family (`canyon_wall`, GIDs 195–207)
   - Terraced Earth Cliffs (`earth_cliff`, GIDs 208–220)
   - Ancient Ruin Walls (`ruin_wall`, GIDs 221–233)
   - Field Stone Fencing (`stone_fence`, GIDs 234–246)
3. **Phase 3: Waterways, Bridges & Dungeons** (`G-4`–`J-9`)
   - Wooden & Stone Bridges (Horizontal & Vertical)
   - Mountain Cave Cavern Entrance (32×32)
   - Desert Pyramid Entrance (32×32)
   - Lake Fortress & Battlements
