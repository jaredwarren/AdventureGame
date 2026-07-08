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
)
