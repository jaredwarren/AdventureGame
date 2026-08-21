package tile

// Size must match maps authored in Tiled (see world.BuildFromTiled guard).
const Size = 16

// GID values match the hand-authored maps under assets/maps (firstgid=1 convention).
const (
	GIDEmpty   = 0
	GIDGrass   = 1
	GIDWall    = 2
	GIDCracked = 3
	GIDDoor    = 4 // warp trigger drawn as floor in collision; doors from objects preferred
	GIDWater   = 5
	GIDLock    = 6 // locked door until small key
	GIDFloor2  = 7
	GIDTree    = 8

	// Water shore transition tiles
	GIDWaterShoreTop    = 9
	GIDWaterShoreBottom = 10
	GIDWaterShoreLeft   = 11
	GIDWaterShoreRight  = 12
	GIDWaterShoreNE     = 13
	GIDWaterShoreNW     = 14
	GIDWaterShoreSW     = 15
	GIDWaterShoreSE     = 16

	// Inner (concave) water shore transition tiles
	GIDWaterShoreNEInner = 17
	GIDWaterShoreNWInner = 18
	GIDWaterShoreSWInner = 19
	GIDWaterShoreSEInner = 20

	GIDDirtPath = 21

	// Wall edge transition tiles
	GIDWallTop    = 22
	GIDWallBottom = 23
	GIDWallLeft   = 24
	GIDWallRight  = 25
	GIDWallNE     = 26
	GIDWallNW     = 27
	GIDWallSW     = 28
	GIDWallSE     = 29

	// Inner (concave) wall transition tiles
	GIDWallNEInner = 30
	GIDWallNWInner = 31
	GIDWallSWInner = 32
	GIDWallSEInner = 33

	// Rock tiles
	GIDRock        = 34
	GIDRockTop     = 35
	GIDRockBottom  = 36
	GIDRockLeft    = 37
	GIDRockRight   = 38
	GIDRockNE      = 39
	GIDRockNW      = 40
	GIDRockSW      = 41
	GIDRockSE      = 42
	GIDRockNEInner = 43
	GIDRockNWInner = 44
	GIDRockSWInner = 45
	GIDRockSEInner = 46
	GIDQuicksand   = 47
	GIDMud         = 48
	GIDIce         = 49
	GIDLava        = 50
	GIDSign        = 51
	GIDSand        = 52

	// Dirt path edge transition tiles
	GIDDirtPathTop     = 53
	GIDDirtPathBottom  = 54
	GIDDirtPathLeft    = 55
	GIDDirtPathRight   = 56
	GIDDirtPathNE      = 57
	GIDDirtPathNW      = 58
	GIDDirtPathSW      = 59
	GIDDirtPathSE      = 60
	GIDDirtPathNEInner = 61
	GIDDirtPathNWInner = 62
	GIDDirtPathSWInner = 63
	GIDDirtPathSEInner = 64

	// Cobblestone path (occupies 65..77: base + 12 transitions)
	GIDCobblePath = 65

	// Sand / shoreline family (occupies 78..90: base + 12 transitions)
	GIDSandPath = 78

	// Snow ground family (occupies 91..103: base + 12 transitions)
	GIDSnow = 91

	// Mud ground family (occupies 104..116: base + 12 transitions)
	GIDMudPath = 104

	// Dark grass / forest earth family (occupies 117..129: base + 12 transitions)
	GIDDarkGrass = 117

	// Deep ocean water family (occupies 130..142: base + 12 transitions)
	GIDDeepWater = 130

	// Lava / magma shore family (occupies 143..155: base + 12 transitions)
	GIDLavaShore = 143

	// Swamp / poison murky water family (occupies 156..168: base + 12 transitions)
	GIDSwampWater = 156

	// Ice & frozen lake family (occupies 169..181: base + 12 transitions)
	GIDIceFamily = 169

	// Quicksand hazard family (occupies 182..194: base + 12 transitions)
	GIDQuicksandFamily = 182
)
