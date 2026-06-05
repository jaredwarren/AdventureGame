# AdventureGame — Asset Generator Prompts Guide

This document lists all the visual (image/sprite sheet) and audio (sound effect) assets required for **AdventureGame**, along with detailed prompts designed to achieve visual and auditory consistency using Generative AI tools (e.g., Midjourney, DALL-E 3, Stable Audio, ElevenLabs, or AudioGen).

---

## 1. Design Style Consistency Guidelines

To ensure all generated assets feel like they belong in the same retro top-down Action-RPG universe (reminiscent of classic 16-bit games like *The Legend of Zelda: A Link to the Past* or *Chrono Trigger*), use the following style modifiers in all your generation requests.

### Visual Art Style (Midjourney / DALL-E 3)
*   **Perspective**: Top-down 2D orthographic projection (flat view, not angled 3D).
*   **Aesthetic**: 16-bit retro pixel art, crisp pixel grids, sharp lines, flat colors, no anti-aliasing.
*   **Palette**: Curated vibrant, high-contrast color palettes (earth tones for overworld, dark/dull stone grays and torch reds for dungeons).
*   **Background**: Solid transparent background (DALL-E: "on a solid pure white background for easy masking", Midjourney: "on a black background, transparent sprite sheets").
*   **Common Modifiers**: `pixel art, 16-bit, 2d game asset, retro video game, game sprite, flat colors, sharp detail, --style raw`

### Auditory Style (Stable Audio / ElevenLabs / Audiobox)
*   **Aesthetic**: 8-bit / 16-bit chiptune sound effects.
*   **Synth Type**: Square waves, triangle waves, noise channels, FM synthesis.
*   **Common Modifiers**: `retro video game sound effect, 8-bit sfx, chiptune synth, sfxr, vintage arcade, short and punchy`

---

## 2. Visual Assets

All tile assets are rendered on a **16×16** pixel grid. Pickup items are typically sized at **12×12** or **16×16** pixels.

### A. Overworld & Dungeon Tileset (`overworld-spritesheet.png`)
This sheet contains the environmental building blocks of the world.

| GID | Tile Name | Description / Details |
| :--- | :--- | :--- |
| **1** | Grass Tile | Walkable overworld ground. Earth-toned green, soft grassy texture. |
| **2** | Wall Tile | Solid barrier. Dark brown stone brick or rocky mountain cliffside. |
| **3** | Cracked Wall | Destructible wall. Same texture as Wall Tile, but with a distinct dark crack. |
| **4** | Door Tile | Walkable floor warp. Stairs leading down into a dark brick-lined dungeon entrance. |
| **5** | Water Tile | Walkable barrier. Blue water tile with light wave reflections. |
| **6** | Lock Tile | Locked gate barrier. Stone gate with a heavy metal keyhole in the center. |
| **7** | Floor 2 Tile | Walkable dungeon ground. Dark grey/blue slate dungeon bricks. |
| **8** | Tree Tile | Destructible forest pine tree or green leafy bush. |
| **9** | Water Shore Top | Water tile with a straight grass shoreline on the top side. |
| **10** | Water Shore Bottom | Water tile with a straight grass shoreline on the bottom side. |
| **11** | Water Shore Left | Water tile with a straight grass shoreline on the left side. |
| **12** | Water Shore Right | Water tile with a straight grass shoreline on the right side. |
| **13** | Water Shore NE | Water tile with a diagonal grass shoreline in the North-East corner. |
| **14** | Water Shore NW | Water tile with a diagonal grass shoreline in the North-West corner. |
| **15** | Water Shore SW | Water tile with a diagonal grass shoreline in the South-West corner. |
| **16** | Water Shore SE | Water tile with a diagonal grass shoreline in the South-East corner. |

#### GenAI Image Prompts for Tilesets:
> **Prompt (Overworld Tileset - Midjourney):**
> `/imagine prompt: A 16-bit retro pixel art tileset for a top-down adventure game. Includes seamless 16x16 tiles of green overworld grass, rocky stone mountain walls, a cracked stone wall, stairs leading down, a stone lock door with keyhole, a leafy pine tree, a deep blue water tile, and 8 water-to-grass shore transition tiles (top shore, bottom shore, left shore, right shore, and diagonal corner shores for NE, NW, SW, SE). Sprites separated on a clean grid, flat colors, transparent background, classic SNES style --style raw --v 6.0`

> **Prompt (Dungeon Floor & Brick Tiles - DALL-E 3):**
> `A 16-bit retro pixel art sprite sheet of dungeon tiles, top-down perspective. Flat dark gray and deep blue stone brick floor tiles, dungeon wall tiles, and lock doors. Clean grid layout, sharp pixel boundaries, high contrast, on a solid white background.`

### A.1 Water-to-Grass Shore Transition Tileset (`water-tileset.png`)
This sheet contains the transition building blocks for water bodies and grass shores.

| GID | Tile Name | Description / Details |
| :--- | :--- | :--- |
| **9** | Water Shore Top | Water tile with a straight grass shoreline on the top side. |
| **10** | Water Shore Bottom | Water tile with a straight grass shoreline on the bottom side. |
| **11** | Water Shore Left | Water tile with a straight grass shoreline on the left side. |
| **12** | Water Shore Right | Water tile with a straight grass shoreline on the right side. |
| **13** | Water Shore NE | Water tile with a diagonal grass shoreline in the North-East corner (convex). |
| **14** | Water Shore NW | Water tile with a diagonal grass shoreline in the North-West corner (convex). |
| **15** | Water Shore SW | Water tile with a diagonal grass shoreline in the South-West corner (convex). |
| **16** | Water Shore SE | Water tile with a diagonal grass shoreline in the South-East corner (convex). |
| **17** | Water Shore NE Inner | Water tile with a concave grass shoreline in the North-East corner. |
| **18** | Water Shore NW Inner | Water tile with a concave grass shoreline in the North-West corner. |
| **19** | Water Shore SW Inner | Water tile with a concave grass shoreline in the South-West corner. |
| **20** | Water Shore SE Inner | Water tile with a concave grass shoreline in the South-East corner. |
| **5**  | Water Tile (Center) | Pure blue water tile with no shoreline. |

#### GenAI Image Prompts for Water-only Tileset:
> **Prompt (Water Tileset - Midjourney):**
> `/imagine prompt: A 16-bit retro pixel art tile sheet for a top-down adventure game, containing 13 water-to-grass transition tiles. The tiles include: 1 center pure blue water tile (no shore), 4 straight shorelines (grass top side, grass bottom side, grass left side, grass right side), 4 convex rounded corners (grass in NE, NW, SW, SE corners, water everywhere else), and 4 concave rounded corners (water in NE, NW, SW, SE corners, grass everywhere else). The transition between grass and water should be rounded and happen exactly in the middle of each tile. All tiles are of equal dimensions (128x128 pixels) arranged on a clean grid, flat vibrant colors, transparent background, classic SNES RPG style --style raw --v 6.0`

---

### B. Inventory & Pickups Sheet (`legendofzelda_items_sheet.png`)
This sheet holds items that the player can pick up, collect, and use.

| Item Name | Description |
| :--- | :--- |
| **Gold Coin** | Spinning gold coin item, bright yellow/gold with high specular sheen. |
| **Heart** | Red recovery heart, standard glowing red container icon. |
| **Bomb** | Rounded black bomb body, grey fuse cap, and curved fuse line. |
| **Key** | Small golden key used to unlock dungeon locks. |
| **Torch** | Lit wooden torch showing fire animation frames. |

#### GenAI Image Prompts for Items:
> **Prompt (Pickups Sprite Sheet - Midjourney):**
> `/imagine prompt: A pixel art sheet of game items: spinning gold coin, glowing red heart, round black cartoon bomb with a fuse, small vintage brass key, and a wooden torch with flickering orange flames. 16-bit retro game assets, flat vibrant colors, sharp pixel outline, transparent background --v 6.0`

> **Prompt (Individual Key / Heart Items - DALL-E 3):**
> `16-bit retro pixel art item sprites: a red health heart container, a classic round black bomb with a lit fuse, and a golden key. Game assets, flat retro colors, transparent background, isolated sprites.`

---

## 3. Audio Assets

The game triggers three core sound effects (`.wav` format) during combat, interactions, and environmental changes.

### A. Pickup Sound (`pickup.wav`)
*   **Trigger**: Picking up coins/hearts/bombs, purchasing upgrades from the Shrine shop, or unlocking a key door.
*   **Sound Profile**: Bright, rising pitch chime. Positive reinforcement.
*   **GenAI Prompt**:
    > `Retro 8-bit video game coin pickup chime, high-pitched rising synthesizer tone, short, clean, arcade sfx, chiptune sound effect, 0.5 seconds`

### B. Hit & Explosion Sound (`hit.wav`)
*   **Trigger**: Landing sword hits on enemies, player taking damage, bomb explosion, block breaking, or tree burning down.
*   **Sound Profile**: Low, punchy impact crunch or white noise explosion.
*   **GenAI Prompt**:
    > `Retro 8-bit explosion sound effect, classic arcade low-frequency crunch impact, chiptune white noise blast, punchy game sound, 0.8 seconds`

### C. Weapon Swing Sound (`swing.wav`)
*   **Trigger**: Swinging the sword, swinging the torch, or throwing/dropping a bomb.
*   **Sound Profile**: Whoosh sound made of filtered white noise or sweep synth pitch.
*   **GenAI Prompt**:
    > `Retro 8-bit sword swing whoosh, chiptune white noise sweep, short vintage arcade blade swing sound, 0.3 seconds`
